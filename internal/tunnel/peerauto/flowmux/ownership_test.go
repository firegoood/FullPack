package smux

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

func TestQueuedPayloadSurvivesCallerReuseAfterEarlyReturn(t *testing.T) {
	for _, exit := range []string{"deadline", "closed", "write-error"} {
		t.Run(exit, func(t *testing.T) {
			s := &Session{shaper: make(chan writeRequest), die: make(chan struct{}), chSocketWriteError: make(chan struct{})}
			deadline := make(chan time.Time)
			p := []byte("bytes owned by an outstanding sender")
			want := bytes.Clone(p)
			f := newFrame(2, cmdPSH, 3)
			f.data = p
			done := make(chan error, 1)
			go func() { _, err := s.writeFrameInternal(f, deadline, CLSDATA); done <- err }()
			var request writeRequest
			select {
			case request = <-s.shaper:
			case <-time.After(time.Second):
				t.Fatal("frame did not reach the sender")
			}
			var expected error
			switch exit {
			case "deadline":
				expected = ErrTimeout
				close(deadline)
			case "closed":
				expected = io.ErrClosedPipe
				close(s.die)
			case "write-error":
				expected = io.ErrUnexpectedEOF
				s.socketWriteError.Store(expected)
				close(s.chSocketWriteError)
			}
			select {
			case err := <-done:
				if !errors.Is(err, expected) {
					t.Fatalf("early return: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("write did not cancel promptly")
			}
			clear(p)
			if !bytes.Equal(request.frame.data, want) {
				t.Fatal("sender still borrows the caller's reusable payload")
			}
		})
	}
}
