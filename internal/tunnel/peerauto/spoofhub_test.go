package peerauto

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"net"
	"testing"
	"time"
)

func TestSpoofConversationsAuthenticateBoundAndRetire(t *testing.T) {
	raw, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	peer, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		raw.Close()
		t.Fatal(err)
	}
	defer peer.Close()
	h := &spoofHub{raw: raw, key: []byte("fixed-test-only-secret"), clients: make(map[[16]byte]*spoofSocket), admitted: make(map[[16]byte]bool), done: make(chan struct{}), joined: make(chan struct{})}
	s := &spoofSocket{hub: h, server: true, queue: make(chan spoofDatagram, 64), done: make(chan struct{}), changed: make(chan struct{})}
	h.server = s
	go h.read()
	t.Cleanup(func() { raw.Close(); <-h.joined })
	frame := func(id byte, valid bool) []byte {
		b := make([]byte, spoofHeader+3)
		copy(b, spoofMagic[:])
		b[8] = id
		copy(b[spoofHeader:], "IP!")
		mac := hmac.New(sha256.New, h.key)
		mac.Write(b[:24])
		mac.Write(b[spoofHeader:])
		copy(b[24:spoofHeader], mac.Sum(nil))
		if !valid {
			b[24] ^= 1
		}
		return b
	}
	for i := byte(1); i <= 32; i++ {
		if _, err := peer.WriteTo(frame(i, false), raw.LocalAddr()); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = peer.WriteTo(frame(1, true), raw.LocalAddr())
	b := make([]byte, 64)
	_ = s.SetReadDeadline(time.Now().Add(time.Second))
	n, a, err := s.ReadFrom(b)
	if err != nil || !bytes.Equal(b[:n], []byte("IP!")) {
		t.Fatalf("authenticated conversation: %v", err)
	}
	h.mu.Lock()
	count := len(h.admitted)
	h.mu.Unlock()
	if count != 1 {
		t.Fatal("forged MAC allocated conversation state")
	}
	if _, err := s.WriteTo([]byte("reply"), a); err != nil {
		t.Fatal(err)
	}
	_ = peer.SetReadDeadline(time.Now().Add(time.Second))
	n, _, err = peer.ReadFrom(make([]byte, 128))
	if err != nil || n != spoofHeader+5 {
		t.Fatalf("reply did not preserve conversation framing: %v", err)
	}
	s.retire(a)
	_, _ = peer.WriteTo(frame(1, true), raw.LocalAddr())
	_, _ = peer.WriteTo(frame(2, true), raw.LocalAddr())
	n, a, err = s.ReadFrom(b)
	if err != nil || n != 3 || a.(spoofAddr).id[0] != 2 {
		t.Fatalf("retired session replay admitted: %v", err)
	}
	for i := byte(3); i <= 10; i++ {
		_, _ = peer.WriteTo(frame(i, true), raw.LocalAddr())
	}
	// Packet 10 follows the admitted eight IDs, then the local deadline joins
	// the receive loop; no sleeps or unbounded hostile-identity map are needed.
	_ = s.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for {
		_, _, err = s.ReadFrom(b)
		if err != nil {
			break
		}
	}
	h.mu.Lock()
	count = len(h.admitted)
	h.mu.Unlock()
	if count != 8 {
		t.Fatalf("conversation bound: %d", count)
	}
}
