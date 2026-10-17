package peerauto

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"syscall"

	smux "github.com/backpack/backpack/internal/tunnel/peerauto/flowmux"
)

var errControlRejected = errors.New("coordination identity, version, policy or replay check failed")
var errEnrollmentRejected = errors.New("peer enrollment/capabilities mismatch; upgrade and enroll both peers")
var errControlProtocol = errors.New("invalid coordination protocol")

// Public status exposes a stable category and stage, never a frame, MAC,
// identity value or helper secret. A timeout is not evidence of bad enrollment.
func classifyCandidateError(err error) (code, summary string) {
	var timeout net.Error
	switch {
	case errors.Is(err, smux.ErrFlowControlWindow):
		return "flow_control_error", "peer exceeded its bounded stream receive credit"
	case errors.Is(err, errPoolPrerequisite):
		return "prerequisite_missing", "SKIPPED: receiver-local helper or authenticated cover is unavailable"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded), errors.As(err, &timeout) && timeout.Timeout():
		return "timeout", "exchange timed out; network filtering is not established"
	case errors.Is(err, context.Canceled):
		return "cancelled", "attempt cancelled"
	case errors.Is(err, errEnrollmentRejected), errors.Is(err, errControlRejected), errors.Is(err, errPeerRejected):
		return "peer_rejected", "identity, enrollment, protocol or replay checks rejected the exchange"
	case errors.Is(err, errControlProtocol):
		return "protocol_error", "invalid control framing or challenge"
	case errors.Is(err, syscall.EADDRINUSE):
		return "bind_conflict", "local address or port is already reserved"
	case errors.Is(err, os.ErrPermission), errors.Is(err, syscall.EPERM):
		return "local_denied", "local permission or firewall policy denied the operation"
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, net.ErrClosed), errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.EPIPE):
		return "connection_closed", "connection closed before the exchange completed"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "connection_refused", "carrier endpoint refused the connection"
	case errors.Is(err, syscall.ENETUNREACH), errors.Is(err, syscall.EHOSTUNREACH):
		return "route_unavailable", "network route is unavailable"
	case strings.Contains(err.Error(), "another process may still own this tunnel's firewall rules"):
		return "resource_owned", "existing raw-carrier ownership prevents takeover"
	default:
		return "unclassified", "exchange failed; detailed cause requires local debug logs"
	}
}

func (m *Manager) candidateFailure(id, stage, prefix string, err error) {
	code, summary := classifyCandidateError(err)
	// The same listener accepts production and isolated probe sessions. A
	// failed extra handshake must not mark an existing enrolled session dead.
	// Check and publish under one lock so a newly accepted session cannot be
	// overwritten by an older attempt's diagnostic. Its owner handles closure.
	m.mu.Lock()
	e := m.endpoints[id]
	live := e != nil && e.ctx.Err() == nil
	if !live {
		for i := range m.status.Candidates {
			if s := &m.status.Candidates[i]; s.ID == id {
				s.Ready = false
				s.ErrorCode, s.ErrorStage = code, stage
				s.Reason = prefix + ": " + summary
				break
			}
		}
	}
	m.mu.Unlock()
	m.log.WithField("candidate", id).WithField("stage", stage).WithField("errorCode", code).WithField("enrolledSessionRetained", live).WithError(err).Debug("carrier attempt failed")
}
