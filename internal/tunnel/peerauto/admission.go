package peerauto

import (
	"context"
	"errors"
	"sync"
	"time"
)

const admissionWait = 2 * time.Second

var (
	errAdmissionFull    = errors.New("bounded TCP admission queue full")
	errAdmissionTimeout = errors.New("TCP admission queue deadline exceeded")
	errUDPCapacity      = errors.New("UDP admission share exhausted; capacity reserved for TCP")
)

// The enrolled total remains unchanged. UDP associations cannot signal client
// closure, so cap their share at one quarter; TCP may use the entire total.
// Pending TCP sockets have a separate, bounded FIFO and never own a live slot.
// No queue, counter or fairness setting changes the enrollment hash or fencing.
type admissionGate struct {
	mu    sync.Mutex
	live  chan struct{}
	udp   int
	queue []*admissionTicket
}

type admissionTicket struct {
	gate              *admissionGate
	ready             chan struct{}
	udp               bool
	granted, released bool // protected by gate.mu
}

func (g *admissionGate) udpLimit() int   { return min(cap(g.live)-1, max(1, cap(g.live)/4)) }
func (g *admissionGate) queueLimit() int { return 2 * cap(g.live) }

// begin runs in the accept loop, before starting a worker. Overload cannot
// allocate an unbounded goroutine/socket queue, even if no active flow ends.
func (g *admissionGate) begin(ctx context.Context, udp bool) (*admissionTicket, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if udp && (g.udp >= g.udpLimit() || len(g.live) >= cap(g.live) || len(g.queue) != 0) {
		return nil, errUDPCapacity
	}
	t := &admissionTicket{gate: g, ready: make(chan struct{}), udp: udp}
	if len(g.live) < cap(g.live) && len(g.queue) == 0 {
		g.grant(t)
	} else {
		if len(g.queue) >= g.queueLimit() {
			return nil, errAdmissionFull
		}
		g.queue = append(g.queue, t)
	}
	return t, nil
}

func (g *admissionGate) grant(t *admissionTicket) {
	g.live <- struct{}{}
	if t.udp {
		g.udp++
	}
	t.granted = true
	close(t.ready)
}

func (t *admissionTicket) wait(ctx context.Context, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-t.ready:
		if err := ctx.Err(); err != nil {
			t.release()
			return err
		}
		return nil
	case <-ctx.Done():
		t.release()
		return ctx.Err()
	case <-timer.C:
		t.release()
		return errAdmissionTimeout
	}
}

func (t *admissionTicket) release() {
	g := t.gate
	g.mu.Lock()
	defer g.mu.Unlock()
	if t.released {
		return
	}
	t.released = true
	if t.granted {
		<-g.live
		if t.udp {
			g.udp--
		}
	} else {
		for i, q := range g.queue {
			if q == t {
				g.queue = append(g.queue[:i], g.queue[i+1:]...)
				break
			}
		}
	}
	for len(g.queue) != 0 && len(g.live) < cap(g.live) {
		next := g.queue[0]
		g.queue[0] = nil
		g.queue = g.queue[1:]
		g.grant(next)
	}
}

func (g *admissionGate) snapshot() (live, udp, queued int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.live), g.udp, len(g.queue)
}
