package network

import (
	"bytes"
	"io"
	"net"

	"github.com/xtaci/smux"
)

const muxOwnedWriteChunk = 32 << 10

// OwnMuxStreamWrites protects callers from upstream smux retaining a queued
// payload after Write returns on session closure or a deadline. Each bounded
// copy stays owned by that pending request, so callers may immediately reuse
// their relay/receive buffers. Stream methods, framing and negotiation remain
// upstream's; this does not enable Auto's optional flow-control extensions.
func OwnMuxStreamWrites(s *smux.Stream) net.Conn { return ownedMuxStream{s} }

type ownedMuxStream struct{ *smux.Stream }

func (s ownedMuxStream) Write(p []byte) (int, error) { return writeOwnedMuxFrames(s.Stream, p) }

func writeOwnedMuxFrames(w io.Writer, p []byte) (total int, err error) {
	if len(p) == 0 {
		return w.Write(p)
	}
	for len(p) > 0 {
		owned := bytes.Clone(p[:min(len(p), muxOwnedWriteChunk)])
		n, err := w.Write(owned)
		total += n
		if err != nil {
			return total, err
		}
		if n != len(owned) {
			return total, io.ErrShortWrite
		}
		p = p[n:]
	}
	return total, nil
}
