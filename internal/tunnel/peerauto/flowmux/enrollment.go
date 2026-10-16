package smux

import "errors"

var ErrFlowControlWindow = errors.New("authenticated stream exceeded its receive credit")

// EnableAuthenticatedFlowControl retains the finite pre-enrollment session
// budget until the owner has checked peer identity, manifest and fresh
// challenges. Thereafter memory scales with accepted streams' bounded windows,
// rather than a shared bucket that can deadlock all of them behind slow users.
// It never changes v2 framing, credit values, stream counts or authorization.
func (s *Session) EnableAuthenticatedFlowControl() error {
	if s.config.Version != 2 {
		return errors.New("per-stream flow control requires negotiated smux v2")
	}
	s.authenticatedFlow.Store(true)
	s.notifyBucket()
	return nil
}

func (s *Session) FlowControl() (version int, authenticated bool, streamWindow int) {
	return s.config.Version, s.authenticatedFlow.Load(), max(initialPeerWindow, s.config.MaxStreamBuffer)
}

// BufferedBytes reports queued receive payload, excluding sockets and a frame
// already being copied to the application. It is not an account/user count.
func (s *Session) BufferedBytes() int64 { return s.bufferedBytes.Load() }

// PrioritizeControl reserves a small local write lane for the one authenticated
// control stream. Wire framing and per-user credit are unchanged. A peer cannot
// select a priority stream or enlarge this queue with an on-wire flag.
func (s *Session) PrioritizeControl(control *Stream) error {
	if !s.authenticatedFlow.Load() || s.config.Version != 2 || control == nil || control.stream == nil || control.stream.sess != s || control.ID() == 0 {
		return errors.New("priority requires this session's enrolled v2 control stream")
	}
	id := control.ID()
	if !s.controlStream.CompareAndSwap(0, id) && s.controlStream.Load() != id {
		return errors.New("only one enrolled control stream may receive priority")
	}
	return nil
}

func (s *Session) ControlPriority() bool { return s.controlStream.Load() != 0 }

// nextWrite runs only on sendLoop. At most sixteen control frames pass before an
// already queued user frame, so control liveness does not remove data fairness.
func (s *Session) nextWrite(burst *int) (writeRequest, bool) {
	if *burst < 16 {
		select {
		case r := <-s.controlWrites:
			*burst += 1
			return r, true
		default:
		}
		select {
		case r := <-s.handshakeWrites:
			*burst += 1
			return r, true
		default:
		}
	}
	if r, ok := s.sq.Pop(); ok {
		*burst = 0
		return r, true
	}
	select {
	case r := <-s.controlWrites:
		*burst = 1
		return r, true
	default:
	}
	select {
	case r := <-s.handshakeWrites:
		*burst = 1
		return r, true
	default:
		return writeRequest{}, false
	}
}
