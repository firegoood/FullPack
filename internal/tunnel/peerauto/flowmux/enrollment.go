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
