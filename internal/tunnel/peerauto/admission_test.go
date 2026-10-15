package peerauto

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestAdmissionUDPShareLeavesTCPRoomAndKeepsTotalBound(t *testing.T) {
	for _, limit := range []int{1, 64, 192} {
		g := &admissionGate{live: make(chan struct{}, limit)}
		var held []*admissionTicket
		for range g.udpLimit() {
			ticket, err := g.begin(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			held = append(held, ticket)
		}
		if _, err := g.begin(context.Background(), true); !errors.Is(err, errUDPCapacity) {
			t.Fatalf("UDP escaped share: %v", err)
		}
		for len(held) < limit {
			ticket, err := g.begin(context.Background(), false)
			if err != nil || ticket.wait(context.Background(), time.Second) != nil {
				t.Fatal("UDP starved TCP", err)
			}
			held = append(held, ticket)
		}
		live, udp, queued := g.snapshot()
		if live != limit || udp != g.udpLimit() || queued != 0 {
			t.Fatalf("bad accounting %d %d %d", live, udp, queued)
		}
		for _, ticket := range held {
			ticket.release()
			ticket.release()
		}
		live, udp, queued = g.snapshot()
		if live != 0 || udp != 0 || queued != 0 {
			t.Fatal("admission resource leak")
		}
	}
}

func TestAdmissionFIFOQueueBoundsCancellationAndTimeout(t *testing.T) {
	g := &admissionGate{live: make(chan struct{}, 1)}
	first, _ := g.begin(context.Background(), false)
	second, _ := g.begin(context.Background(), false)
	third, _ := g.begin(context.Background(), false)
	if _, err := g.begin(context.Background(), false); !errors.Is(err, errAdmissionFull) {
		t.Fatal("unbounded queue", err)
	}
	first.release()
	if err := second.wait(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	select {
	case <-third.ready:
		t.Fatal("FIFO waiter bypassed")
	default:
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := second.wait(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatal("granted cancellation leaked slot", err)
	}
	if err := third.wait(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	fourth, _ := g.begin(context.Background(), false)
	if err := fourth.wait(context.Background(), time.Millisecond); !errors.Is(err, errAdmissionTimeout) {
		t.Fatal(err)
	}
	third.release()
	if live, udp, queued := g.snapshot(); live != 0 || udp != 0 || queued != 0 {
		t.Fatal("cancel/timeout leaked admission")
	}
}

func TestAdmissionConcurrentGrantCancellationAndRelease(t *testing.T) {
	g := &admissionGate{live: make(chan struct{}, 64)}
	var workers sync.WaitGroup
	for i := range 2000 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ticket, err := g.begin(ctx, i%4 == 0)
			if err != nil {
				return
			}
			if i%3 == 0 {
				cancel()
			}
			_ = ticket.wait(ctx, 10*time.Millisecond)
			ticket.release()
			live, udp, queued := g.snapshot()
			if live > 64 || udp > 16 || queued > 128 {
				t.Error("resource bound escaped")
			}
		}()
	}
	workers.Wait()
	if live, udp, queued := g.snapshot(); live != 0 || udp != 0 || queued != 0 {
		t.Fatal("concurrent admission leaked resources")
	}
}
