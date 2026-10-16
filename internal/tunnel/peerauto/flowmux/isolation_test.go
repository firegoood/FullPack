package smux

import (
	"bytes"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func flowPair(t *testing.T, version int) (*Session, *Session) {
	t.Helper()
	a, b := net.Pipe()
	cfg := DefaultConfig()
	cfg.Version = version
	cfg.MaxReceiveBuffer = 256 << 10
	cfg.MaxStreamBuffer = 16 << 10
	x, err := Client(a, cfg)
	if err != nil {
		t.Fatal(err)
	}
	y, err := Server(b, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { x.Close(); y.Close() })
	return x, y
}

func flowStream(t *testing.T, a, b *Session) (*Stream, *Stream) {
	t.Helper()
	x, err := a.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	y, err := b.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	return x, y
}

func TestLegacySharedBucketReproducesControlStarvation(t *testing.T) {
	a, b := flowPair(t, 1)
	control, peerControl := flowStream(t, a, b)
	data, _ := flowStream(t, a, b)
	var sent atomic.Int64
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 32<<10)
		for range 32 {
			n, err := data.Write(buf)
			sent.Add(int64(n))
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() { a.Close(); b.Close(); <-done })
	until := time.Now().Add(3 * time.Second)
	for sent.Load() < 256<<10 && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	if sent.Load() < 256<<10 {
		t.Fatal("fixture did not exhaust the shared receive bucket")
	}
	echo := make(chan struct{})
	go func() { defer close(echo); io.Copy(peerControl, peerControl) }()
	t.Cleanup(func() { a.Close(); b.Close(); <-echo })
	_ = control.SetDeadline(time.Now().Add(300 * time.Millisecond))
	_, err := control.Write([]byte("control"))
	if err == nil {
		_, err = io.ReadFull(control, make([]byte, 7))
	}
	if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
		t.Fatalf("legacy shared-bucket stall not reproduced: %v", err)
	}
}

func TestAuthenticatedWindowsKeepControlAndOtherUsersMoving(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "forward", true: "reverse"}[reverse], func(t *testing.T) {
			a, b := flowPair(t, 2)
			if reverse {
				a, b = b, a
			}
			if err := a.EnableAuthenticatedFlowControl(); err != nil {
				t.Fatal(err)
			}
			if err := b.EnableAuthenticatedFlowControl(); err != nil {
				t.Fatal(err)
			}
			control, peerControl := flowStream(t, a, b)
			var wg sync.WaitGroup
			var sent atomic.Int64
			for range 64 {
				data, _ := flowStream(t, a, b)
				wg.Add(1)
				go func() {
					defer wg.Done()
					buf := make([]byte, 32<<10)
					for range 64 {
						n, err := data.Write(buf)
						sent.Add(int64(n))
						if err != nil {
							return
						}
					}
				}()
			}
			t.Cleanup(func() { a.Close(); b.Close(); wg.Wait() })
			until := time.Now().Add(5 * time.Second)
			for sent.Load() < 2<<20 && time.Now().Before(until) {
				time.Sleep(time.Millisecond)
			}
			if sent.Load() < 2<<20 {
				t.Fatal("slow readers did not exceed the former global budget")
			}
			echo := make(chan struct{})
			go func() { defer close(echo); io.Copy(peerControl, peerControl) }()
			t.Cleanup(func() { a.Close(); b.Close(); <-echo })
			for i := range 8 {
				_ = control.SetDeadline(time.Now().Add(2 * time.Second))
				p := bytes.Repeat([]byte{byte(i + 1)}, 1024)
				if _, err := control.Write(p); err != nil {
					t.Fatal(err)
				}
				out := make([]byte, len(p))
				if _, err := io.ReadFull(control, out); err != nil || !bytes.Equal(out, p) {
					t.Fatalf("control stalled/corrupt behind 64 slow readers: %v", err)
				}
			}
			fast, peerFast := flowStream(t, a, b)
			fastEcho := make(chan struct{})
			go func() { defer close(fastEcho); io.Copy(peerFast, peerFast) }()
			t.Cleanup(func() { a.Close(); b.Close(); <-fastEcho })
			_ = fast.SetDeadline(time.Now().Add(2 * time.Second))
			if _, err := fast.Write([]byte("another user")); err != nil {
				t.Fatal(err)
			}
			out := make([]byte, 12)
			if _, err := io.ReadFull(fast, out); err != nil || string(out) != "another user" {
				t.Fatal("unrelated user starved", err)
			}
		})
	}
}

func TestLegacyCannotDisableItsPreEnrollmentBudget(t *testing.T) {
	a, _ := flowPair(t, 1)
	if a.EnableAuthenticatedFlowControl() == nil || a.authenticatedFlow.Load() {
		t.Fatal("v1 lost its bounded legacy budget")
	}
}
