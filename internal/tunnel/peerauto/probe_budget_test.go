package peerauto

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
)

func TestProbeReplayWindowAllowsAnHourOfPacedHealthChecks(t *testing.T) {
	m := &Manager{recentTx: map[string]time.Time{}}
	start := time.Now()
	// The global one-second spacing is stricter than this replay-cache model.
	// Preserve every transaction for a full hour, including prepare identities.
	for i := range 3600 {
		if err := m.rememberProbeTransaction(fmt.Sprintf("%064x", i), start.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("valid paced transaction %d rejected: %v", i, err)
		}
	}
	if len(m.recentTx) != 3600 {
		t.Fatal("one-hour replay protection was shortened")
	}
	if err := m.rememberProbeTransaction(fmt.Sprintf("%064x", 0), start.Add(time.Hour)); err == nil || probeSkipped(err) {
		t.Fatal("replayed transaction must still fail at the one-hour boundary")
	}
	if err := m.rememberProbeTransaction(fmt.Sprintf("%064x", 0), start.Add(time.Hour+time.Second)); err != nil {
		t.Fatalf("expired replay entry did not release capacity: %v", err)
	}
}

func TestProbeReplayBudgetStaysBoundedAndDoesNotQuarantineACarrier(t *testing.T) {
	m := &Manager{recentTx: map[string]time.Time{}}
	now := time.Now()
	for i := range maxRecentTransactions {
		m.recentTx[fmt.Sprintf("%064x", i)] = now
	}
	if err := m.rememberProbeTransaction(fmt.Sprintf("%064x", maxRecentTransactions), now); !errors.Is(err, errProbeReplayBudget) {
		t.Fatalf("full replay cache not classified as budget skip: %v", err)
	}
	if len(m.recentTx) != maxRecentTransactions {
		t.Fatal("full replay cache grew")
	}
	if err := m.rememberProbeTransaction(fmt.Sprintf("%064x", 0), now); err == nil || probeSkipped(err) {
		t.Fatal("capacity pressure hid an actual replay")
	}
	e := &candidate{}
	m.probeFailure(e, fmt.Errorf("probe offer: %w", errProbeReplayBudget))
	if e.failedProbes != 0 || !e.quarantineUntil.IsZero() || m.status.History[0].Result != "SKIPPED" {
		t.Fatal("local replay pressure counted as a failed carrier")
	}
}

func TestProbeBudgetRejectKeepsProductionControlUsable(t *testing.T) {
	for _, budget := range []error{errProbePressure, errProbeBusy, errProbeReplayBudget} {
		t.Run(budget.Error(), func(t *testing.T) {
			left, right := net.Pipe()
			defer left.Close()
			defer right.Close()
			base := message{Tunnel: "pair", Manifest: "manifest", Candidate: "tcp", Sender: "iran", Receiver: "kharej", Incarnation: "iran-boot", PeerIncarnation: "kharej-boot"}
			peerBase := base
			peerBase.Sender, peerBase.Receiver = base.Receiver, base.Sender
			peerBase.Incarnation, peerBase.PeerIncarnation = base.PeerIncarnation, base.Incarnation
			w := &wire{c: left, key: []byte("test-session-key"), base: base}
			peer := &wire{c: right, key: w.key, base: peerBase}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := &Manager{ctx: ctx, policy: config.AutoConfig{MaxConnections: 64, MaxMemoryMB: 2048}, admission: &admissionGate{limit: 64}, pendingProbes: map[string]*pendingProbe{}}
			e := &candidate{ctx: ctx, cancel: cancel, w: w, spec: config.AutoCandidate{ID: "tcp", Transport: config.TCP, Mode: "reverse"}}
			done := make(chan error, 1)
			go func() {
				r, err := peer.receive()
				if err == nil {
					err = peer.send(message{Op: "REJECT", Transaction: r.Transaction, Reason: budget.Error()})
				}
				if err == nil {
					r, err = peer.receive()
				}
				if err == nil {
					err = peer.send(message{Op: "STATUS", Transaction: r.Transaction})
				}
				done <- err
			}()
			if err := m.exchangeProbe(e, 512); !errors.Is(err, budget) {
				t.Fatalf("typed budget rejection lost: %v", err)
			}
			if e.ctx.Err() != nil || len(m.pendingProbes) != 0 {
				t.Fatal("budget skip cancelled production or retained a pending probe")
			}
			if _, err := w.ask(message{Op: "STATUS"}, "STATUS"); err != nil {
				t.Fatalf("budget skip damaged subsequent control: %v", err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}
