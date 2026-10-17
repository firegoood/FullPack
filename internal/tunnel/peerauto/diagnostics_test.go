package peerauto

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
	"testing"

	smux "github.com/backpack/backpack/internal/tunnel/peerauto/flowmux"
	"github.com/sirupsen/logrus"
)

func TestCandidateDiagnosticsSeparateTimeoutFromPeerRejectionAndHideSecrets(t *testing.T) {
	m, e, _ := followerForState(t)
	delete(m.endpoints, e.spec.ID) // no enrolled production session owns this attempt
	for _, tc := range []struct {
		code string
		err  error
	}{
		{"timeout", fmt.Errorf("handshake: %w", context.DeadlineExceeded)},
		{"timeout", &net.OpError{Op: "read", Net: "tcp", Err: context.DeadlineExceeded}},
		{"cancelled", context.Canceled},
		{"peer_rejected", errEnrollmentRejected},
		{"peer_rejected", errControlRejected},
		{"protocol_error", errControlProtocol},
		{"flow_control_error", smux.ErrFlowControlWindow},
		{"connection_closed", io.EOF},
		{"connection_closed", syscall.ECONNRESET},
		{"bind_conflict", syscall.EADDRINUSE},
		{"local_denied", syscall.EPERM},
		{"connection_refused", syscall.ECONNREFUSED},
		{"route_unavailable", syscall.ENETUNREACH},
		{"resource_owned", fmt.Errorf("pck: another process may still own this tunnel's firewall rules")},
		{"unclassified", fmt.Errorf("private secret: %s", m.token)},
	} {
		m.candidateFailure(e.spec.ID, "control_auth", "peer control authentication unavailable", tc.err)
		s := m.Snapshot().Candidates[0]
		if s.ErrorCode != tc.code || s.ErrorStage != "control_auth" || s.Ready || strings.Contains(s.Reason, m.token) {
			t.Fatalf("wrong or unsafe diagnostic: %+v", s)
		}
	}
}

func TestFailedExtraAttemptCannotOverwriteLiveSessionDiagnostics(t *testing.T) {
	m, e, _ := followerForState(t)
	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()
	e.ctx = ctx
	m.candidateStatus(e.spec.ID, func(s *CandidateStatus) {
		s.Ready = true
		s.Reason = "authenticated enrolled endpoint"
	})
	before := m.Snapshot().Candidates[0]
	for _, stage := range []string{"carrier_accept", "control_open", "control_type", "control_auth", "mux_enrollment"} {
		m.candidateFailure(e.spec.ID, stage, "extra attempt failed", context.DeadlineExceeded)
		if got := m.Snapshot().Candidates[0]; got != before {
			t.Fatalf("%s changed live session diagnostics: %+v", stage, got)
		}
	}
	cancel()
	m.candidateFailure(e.spec.ID, "control_auth", "new attempt failed", io.EOF)
	got := m.Snapshot().Candidates[0]
	if got.Ready || got.ErrorCode != "connection_closed" || got.ErrorStage != "control_auth" {
		t.Fatalf("closed session concealed a real reconnect failure: %+v", got)
	}
}

func TestAutoUsesRoleSpecificDebugLogLevel(t *testing.T) {
	c := stateConfig()
	c.Server.LogLevel, c.Server.LogFormat = "debug", "json"
	m, err := New(context.Background(), c, t.TempDir()+"/state.json")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if m.log.GetLevel() != logrus.DebugLevel {
		t.Fatal("configured debug level was ignored")
	}
	if _, ok := m.log.Formatter.(*logrus.JSONFormatter); !ok {
		t.Fatal("configured JSON logs were ignored")
	}
}
