package peerauto

import (
	"context"
	"errors"
	"io"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestOSResourceExhaustionIsLocalAndNotCarrierOrEnrolledCapacity(t *testing.T) {
	for _, err := range []error{syscall.EMFILE, syscall.ENFILE, syscall.ENOBUFS, syscall.ENOMEM} {
		wrapped := &net.OpError{Op: "accept", Net: "tcp", Err: err}
		if !localResourceError(wrapped) || dataErrorCode(wrapped) != "local_resource" {
			t.Fatal("OS exhaustion misclassified", wrapped)
		}
	}
	if localResourceError(net.ErrClosed) || dataErrorCode(errUDPHandshakeBusy) != "handshake_busy" {
		t.Fatal("shutdown/handshake classification lost")
	}
}

func TestDataHealthSeparatesCapacityBackendAndCarrier(t *testing.T) {
	m, _, _ := followerForState(t)
	m.setState("HEALTHY", "authenticated carrier")
	for _, err := range []error{errAdmissionFull, errAdmissionTimeout, errUDPCapacity, errPeerCapacity, errBackend, errNoDataRoute, errDataPool, io.EOF, context.DeadlineExceeded} {
		m.dataFailure("test", err)
	}
	s := m.Snapshot()
	if s.State != "HEALTHY" || s.Data.Code != "recent_errors" || len(s.Data.Errors) != 9 || s.Data.Errors["backend_unavailable"] != 1 {
		t.Fatalf("incorrect separate data diagnostics: %+v", s)
	}
	s.Data.Errors["backend_unavailable"] = 100
	if m.Snapshot().Data.Errors["backend_unavailable"] != 1 {
		t.Fatal("snapshot aliases diagnostics")
	}
	f := FlowStatus{Live: 64, Limit: 64, QueueLimit: 128}
	if d := m.dataSnapshot(f, time.Now()); d.Code != "capacity" {
		t.Fatal("saturation still green", d)
	}
	f.Live, f.QueuedTCP = 63, 1
	if d := m.dataSnapshot(f, time.Now()); d.Code != "queueing" {
		t.Fatal("waiters not reported", d)
	}
	f.QueuedTCP = 0
	if d := m.dataSnapshot(f, time.Now().Add(time.Minute)); d.Code != "available" || d.Errors["queue_full"] != 1 {
		t.Fatal("stale failure stayed current or counter lost", d)
	}
}

func TestDataAdmissionAcknowledgmentsDistinguishPeerCapacityAndBackend(t *testing.T) {
	for _, tc := range []struct {
		ack  byte
		want error
	}{{0, errBackend}, {2, errPeerCapacity}} {
		m, e, _ := followerForState(t)
		e.data = make(chan net.Conn, 1)
		e.w = &wire{key: make([]byte, 32)}
		m.active = &route{endpoint: e, ctx: m.ctx, generation: 2, ticket: "test-ticket"}
		local, peer := net.Pipe()
		e.data <- local
		done := make(chan error, 1)
		go func() {
			defer peer.Close()
			_ = peer.SetDeadline(time.Now().Add(time.Second))
			var header [requestSize]byte
			_, err := io.ReadFull(peer, header[:])
			if err == nil {
				_, err = peer.Write([]byte{tc.ack})
			}
			done <- err
		}()
		_, err := m.claim(m.ctx, 0, false)
		if !errors.Is(err, tc.want) {
			t.Fatalf("ACK %d: %v", tc.ack, err)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

func TestPeerDataQueueCannotAdmitRetiredGeneration(t *testing.T) {
	m, e, _ := followerForState(t)
	e.w = &wire{key: make([]byte, 32)}
	r := &route{endpoint: e, ctx: m.ctx, generation: 2, ticket: "old-generation"}
	m.active = r
	var held []*admissionTicket
	for range 64 {
		ticket, err := m.admission.begin(m.ctx, false)
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, ticket)
	}
	defer func() {
		for _, ticket := range held {
			ticket.release()
		}
	}()
	server, client := net.Pipe()
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(3 * time.Second))
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer server.Close()
		m.serveData(e, server, make(chan struct{}, 1), func() {})
	}()
	if _, err := client.Write(requestHeader(r, 0, false)); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		_, _, queued := m.admission.snapshot()
		if queued == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if _, _, queued := m.admission.snapshot(); queued != 1 {
		t.Fatal("request was not queued")
	}
	m.mu.Lock()
	r.retired.Store(true) // old established streams can drain, without new admissions
	m.active = &route{endpoint: e, ctx: m.ctx, generation: 3, ticket: "new-generation"}
	m.mu.Unlock()
	held[0].release()
	var ack [1]byte
	if n, err := client.Read(ack[:]); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("retired queued request reached backend/ACK: %d %v", n, err)
	}
	<-done
}
