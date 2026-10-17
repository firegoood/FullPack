package network

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

type retainingMuxWriter struct {
	queued [][]byte
	err    error
	short  bool
}

func (w *retainingMuxWriter) Write(p []byte) (int, error) {
	w.queued = append(w.queued, p)
	if w.err != nil {
		return 0, w.err
	}
	if w.short {
		return len(p) - 1, nil
	}
	return len(p), nil
}

func TestMuxWriteOwnershipSurvivesEarlyReturn(t *testing.T) {
	for _, failure := range []error{nil, io.ErrClosedPipe, os.ErrDeadlineExceeded} {
		name := "completed"
		if failure != nil {
			name = failure.Error()
		}
		t.Run(name, func(t *testing.T) {
			p := bytes.Repeat([]byte{0x7b}, 3*muxOwnedWriteChunk+1)
			w := &retainingMuxWriter{err: failure}
			n, err := writeOwnedMuxFrames(w, p)
			if !errors.Is(err, failure) || failure == nil && n != len(p) || failure != nil && n != 0 {
				t.Fatalf("write count/error: %d %v", n, err)
			}
			clear(p)
			for _, queued := range w.queued {
				if len(queued) > muxOwnedWriteChunk || !bytes.Equal(queued, bytes.Repeat([]byte{0x7b}, len(queued))) {
					t.Fatal("pending writer borrowed reusable caller bytes or an unbounded chunk")
				}
			}
		})
	}
}

func TestOwnedMuxWritePreservesShortWriteError(t *testing.T) {
	w := &retainingMuxWriter{short: true}
	if n, err := writeOwnedMuxFrames(w, []byte("payload")); n != 6 || !errors.Is(err, io.ErrShortWrite) || len(w.queued) != 1 {
		t.Fatalf("short write was hidden: %d %v", n, err)
	}
}
