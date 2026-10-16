package peerauto

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/sirupsen/logrus"
)

func controlPair(t *testing.T) (*Manager, *candidate, *wire) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	left, right := net.Pipe()
	base := message{Tunnel: "control-test", Manifest: "manifest", Candidate: "active", Sender: "iran", Receiver: "kharej", Incarnation: "iran-boot", PeerIncarnation: "kharej-boot"}
	peerBase := base
	peerBase.Sender, peerBase.Receiver = base.Receiver, base.Sender
	peerBase.Incarnation, peerBase.PeerIncarnation = base.PeerIncarnation, base.Incarnation
	e := &candidate{spec: config.AutoCandidate{ID: "active"}, ctx: ctx, cancel: cancel, w: &wire{c: left, base: base, key: []byte("test-session-key")}}
	m := &Manager{ctx: ctx, cancel: cancel, log: logrus.New(), policy: Defaults(config.AutoConfig{Selection: &config.AutoSelectionConfig{}}), endpoints: map[string]*candidate{"active": e}}
	m.pool.selected = []string{"active"}
	m.active = &route{endpoint: e, ctx: ctx, generation: 7}
	t.Cleanup(func() { cancel(); left.Close(); right.Close(); m.workers.Wait() })
	return m, e, &wire{c: right, base: peerBase, key: e.w.key}
}

func TestControlLivenessSurvivesBlockedReserveAndAuthority(t *testing.T) {
	m, e, peer := controlPair(t)
	reserve := &candidate{spec: config.AutoCandidate{ID: "reserve"}, ctx: m.ctx}
	// Model a slow isolated transaction. A heartbeat may not queue behind it
	// or depend on the authority finishing its serial scan. Compress only the
	// follower's idle deadline; this test is not a real-hour soak.
	reserve.mu.Lock()
	defer reserve.mu.Unlock()
	m.endpoints["reserve"] = reserve
	m.pool.selected = append(m.pool.selected, "reserve")
	seen := make(chan error, 1)
	go func() {
		for range 4 {
			_ = peer.c.SetDeadline(time.Now().Add(6 * time.Second))
			r, err := peer.receive()
			if err == nil && r.Op != "STATUS" {
				err = errControlProtocol
			}
			if err == nil {
				err = peer.send(message{Op: "STATUS", Active: "active", Generation: 7})
			}
			if err != nil {
				seen <- err
				return
			}
		}
		seen <- nil
	}()
	m.launch(m.controlLoop)
	select {
	case err := <-seen:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(14 * time.Second):
		t.Fatal("reserve transaction starved active control")
	}
	if e.ctx.Err() != nil || m.active.generation != 7 || len(m.status.History) != 0 {
		t.Fatal("independent liveness changed authority, generation or active route")
	}
}

func TestControlTickSkipsBusyTransactionAndKeepsGenerationFencing(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(map[bool]string{false: "matching", true: "foreign-generation"}[mismatch], func(t *testing.T) {
			m, e, peer := controlPair(t)
			e.mu.Lock()
			done := make(chan struct{})
			go func() { m.controlTick(); close(done) }()
			select {
			case <-done:
			case <-time.After(time.Second):
				e.mu.Unlock()
				t.Fatal("heartbeat queued behind an authenticated transaction")
			}
			e.mu.Unlock()
			result := make(chan error, 1)
			go func() {
				_, err := peer.receive()
				generation := uint64(7)
				if mismatch {
					generation++
				}
				if err == nil {
					err = peer.send(message{Op: "STATUS", Active: "active", Generation: generation})
				}
				result <- err
			}()
			m.controlTick()
			if err := <-result; err != nil {
				t.Fatal(err)
			}
			if (e.ctx.Err() != nil) != mismatch {
				t.Fatal("authenticated generation disagreement was not fenced")
			}
		})
	}
}

func TestControlLoopCancellationJoinsInFlightHeartbeat(t *testing.T) {
	m, e, peer := controlPair(t)
	started := make(chan struct{})
	go func() { _, _ = peer.receive(); close(started) }()
	m.launch(m.controlLoop)
	<-started
	m.cancel()
	// runSession's joined cancellation closes the carrier; emulate that here.
	e.w.c.Close()
	done := make(chan struct{})
	go func() { m.workers.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelled manager retained an in-flight heartbeat")
	}
	if len(m.status.History) != 0 {
		t.Fatal("operator shutdown was reported as a carrier failure")
	}
	if _, err := peer.c.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("control connection retained after cancellation: %v", err)
	}
}

func TestControlHeartbeatCannotCrossBilateralCommitBoundary(t *testing.T) {
	m, _, peer := controlPair(t)
	m.controlCommit.Lock()
	done := make(chan struct{})
	go func() { m.controlTick(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		m.controlCommit.Unlock()
		t.Fatal("heartbeat queued behind a two-peer commit")
	}
	m.controlCommit.Unlock()
	request := make(chan struct{})
	reply := make(chan struct{})
	replyDone := make(chan struct{})
	go func() {
		_, _ = peer.receive()
		close(request)
		<-reply
		_ = peer.send(message{Op: "STATUS", Active: "active", Generation: 7})
	}()
	go func() { m.controlTick(); close(replyDone) }()
	<-request
	if m.controlCommit.TryLock() {
		m.controlCommit.Unlock()
		t.Fatal("commit crossed an outstanding authenticated heartbeat")
	}
	close(reply)
	<-replyDone
	if !m.controlCommit.TryLock() {
		t.Fatal("completed heartbeat retained commit ownership")
	}
	m.controlCommit.Unlock()
}

func TestControlStatusSeparatesLiveRouteFromPreparedFencingEpoch(t *testing.T) {
	m, e, _ := followerForState(t)
	gen := m.store.snapshot().Generation + 1
	commit := func(next uint64) message {
		tx := fresh()
		if m.transition(e, message{Op: "PREPARE_SWITCH", Generation: next, Transaction: tx}).Op != "READY" {
			t.Fatal("prepare refused")
		}
		return message{Op: "COMMIT", Generation: next, Transaction: tx, Nonce: fresh()}
	}
	if m.transition(e, commit(gen)).Op != "COMMIT_ACK" {
		t.Fatal("initial commit refused")
	}
	pending := commit(gen + 1)
	if r := m.transition(e, message{Op: "STATUS"}); r.Active != e.spec.ID || r.Generation != gen {
		t.Fatal("pending epoch falsely reports a live route mismatch", r)
	}
	if r := m.transition(e, message{Op: "STATE_SYNC"}); r.Generation != gen+1 {
		t.Fatal("reconciliation lost the durable fencing epoch", r)
	}
	if m.transition(e, pending).Op != "COMMIT_ACK" {
		t.Fatal("higher epoch commit refused")
	}
	if r := m.transition(e, message{Op: "STATUS"}); r.Generation != gen+1 {
		t.Fatal("heartbeat did not report the new committed generation", r)
	}
	m.mu.Lock()
	m.fenceLocked()
	m.mu.Unlock()
	if r := m.transition(e, message{Op: "STATUS"}); r.Active != "" || r.Generation != gen+1 {
		t.Fatal("fenced route was still advertised live or lost its epoch", r)
	}
}
