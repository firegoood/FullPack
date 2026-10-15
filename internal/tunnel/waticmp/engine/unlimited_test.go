package wat

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

type unlimitedPacketIO struct {
	in, out chan []byte
	done    chan struct{}
	once    *sync.Once
}

func (p *unlimitedPacketIO) Send(b []byte, _ int) error {
	select {
	case p.out <- append([]byte(nil), b...):
		return nil
	case <-p.done:
		return net.ErrClosed
	}
}
func (p *unlimitedPacketIO) Recv() ([]byte, int, net.Addr, error) {
	select {
	case b := <-p.in:
		return b, 0, &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}, nil
	case <-p.done:
		return nil, 0, nil, net.ErrClosed
	}
}
func (p *unlimitedPacketIO) SetPeer(net.Addr) {}
func (p *unlimitedPacketIO) Close() error     { p.once.Do(func() { close(p.done) }); return nil }

// Real AEAD packets, pacing, ACK/window and stream code over a bounded channel
// transport. This is an engine test, not a raw-ICMP or filtered-WAN claim.
func TestUnlimitedNativeEngineKeeps1000HeldAuthenticatedStreams(t *testing.T) {
	const count = 1000
	a, b := make(chan []byte, 4096), make(chan []byte, 4096)
	done := make(chan struct{})
	once := &sync.Once{}
	left, right := &unlimitedPacketIO{a, b, done, once}, &unlimitedPacketIO{b, a, done, once}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	zero := 0
	cfg := Config{Key: key, SID: 41, Paths: 1, StreamLimit: &zero, InitBps: 1e6, MinBps: 128000, MaxBps: 4e6}
	client, err := NewSession(cfg, left)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Server = true
	server, err := NewSession(cfg, right)
	if err != nil {
		client.Close()
		t.Fatal(err)
	}
	release := make(chan struct{})
	var releaseOnce sync.Once
	var serverWorkers sync.WaitGroup
	t.Cleanup(func() {
		client.Close()
		server.Close()
		releaseOnce.Do(func() { close(release) })
		serverWorkers.Wait()
	})
	serverWorkers.Add(1)
	go func() {
		defer serverWorkers.Done()
		for range count {
			st, err := server.Accept()
			if err != nil {
				return
			}
			serverWorkers.Add(1)
			go func() {
				defer serverWorkers.Done()
				defer st.Close()
				_ = st.SetReadDeadline(time.Now().Add(30 * time.Second))
				_ = st.SetWriteDeadline(time.Now().Add(30 * time.Second))
				buf := make([]byte, 64)
				for range 2 {
					if _, err := io.ReadFull(st, buf); err != nil {
						return
					}
					if _, err := st.Write(buf); err != nil {
						return
					}
				}
				<-release
			}()
		}
	}()
	var streams []*Stream
	for range count {
		st, err := client.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		streams = append(streams, st)
	}
	for round := range 2 {
		var wg sync.WaitGroup
		errors := make(chan error, count)
		for i, st := range streams {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = st.SetReadDeadline(time.Now().Add(25 * time.Second))
				_ = st.SetWriteDeadline(time.Now().Add(25 * time.Second))
				payload := bytes.Repeat([]byte{byte(i%251 + 1)}, 64)
				payload[0] = byte(round)
				if _, err := st.Write(payload); err != nil {
					errors <- err
					return
				}
				got := make([]byte, 64)
				if _, err := io.ReadFull(st, got); err != nil {
					errors <- err
					return
				}
				if !bytes.Equal(got, payload) {
					errors <- fmt.Errorf("stream %d round %d corrupt", i, round)
				}
			}()
		}
		wg.Wait()
		close(errors)
		for err := range errors {
			t.Error(err)
		}
		if t.Failed() {
			t.FailNow()
		}
		for _, s := range []*Session{client, server} {
			if live, _ := s.Load(); live != count {
				t.Fatalf("only %d held native streams", live)
			}
		}
	}
	releaseOnce.Do(func() { close(release) })
}

func TestAutoStreamOverrideExceedsLegacyLimitAndKeepsTombstoneGuard(t *testing.T) {
	zero := 0
	s := memorySession()
	s.cfg.StreamLimit = &zero
	s.nextSID = 1
	for range 1200 {
		if _, err := s.OpenStream(); err != nil {
			t.Fatal("unlimited local stream refused", err)
		}
	}
	receiver := memorySession()
	receiver.cfg.Server = true
	receiver.cfg.StreamLimit = &zero
	receiver.acceptC = sync.NewCond(&receiver.mu)
	for i := range 1200 {
		receiver.onStream(uint64(i+1), 0, []byte("data"), false)
	}
	if len(receiver.streams) != 1200 || len(receiver.acceptQ) != 1200 {
		t.Fatal("incoming streams retained the legacy ceiling")
	}
	for i := range 4096 {
		receiver.tomb[uint64(i+2000)] = time.Now()
	}
	receiver.onStream(9999, 0, []byte("new"), false)
	if receiver.streams[9999] != nil {
		t.Fatal("unlimited streams weakened bounded tombstone protection")
	}
	manual := memorySession()
	manual.nextSID = 1
	for range 2 {
		if _, err := manual.OpenStream(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := manual.OpenStream(); err != ErrTooManyConns {
		t.Fatal("manual explicit cap changed", err)
	}
}

func TestAutoRetirementRemainsBoundedAcrossStreamChurnAndNeverResurrects(t *testing.T) {
	zero := 0
	s := memorySession()
	s.cfg.Server = true
	s.cfg.StreamLimit = &zero
	s.acceptC = sync.NewCond(&s.mu)
	for i := uint64(1); i <= 12000; i++ {
		s.onStream(i, 0, []byte("x"), false)
		st := s.streams[i]
		if st == nil {
			t.Fatalf("stream churn stalled at %d", i)
		}
		s.killStream(st, ErrClosed)
		s.acceptQ = nil
	}
	if s.peerSeenThrough != 12000 || len(s.peerSeenAhead) != 0 || len(s.tomb) != 0 {
		t.Fatal("retirement history grew with completed streams")
	}
	s.onStream(42, 0, []byte("replayed in a fresh packet"), false)
	if len(s.streams) != 0 {
		t.Fatal("retired identity resurrected after compaction")
	}
	// Sparse authenticated identities may not allocate unbounded history.
	for i := uint64(1); i <= 4096; i++ {
		if !s.observePeerStream(12000 + 2*i) {
			t.Fatal("bounded ahead window rejected early")
		}
	}
	if s.observePeerStream(30000) || len(s.peerSeenAhead) != 4096 {
		t.Fatal("out-of-order identity budget escaped")
	}
	for i := uint64(1); i <= 4096; i++ {
		if !s.observePeerStream(12000 + 2*i - 1) {
			t.Fatal("gap could not be filled")
		}
	}
	if s.peerSeenThrough != 20192 || len(s.peerSeenAhead) != 0 {
		t.Fatal("filled gaps did not compact")
	}
	if s.observePeerStream(16000) {
		t.Fatal("old ahead identity replay accepted")
	}
}

func TestAutoRetirementTracksBothPeerNamespacesAndRejectsLocalIDs(t *testing.T) {
	zero := 0
	for _, server := range []bool{false, true} {
		s := memorySession()
		s.cfg.Server, s.cfg.BidirectionalStreams, s.cfg.StreamLimit = server, true, &zero
		s.acceptC = sync.NewCond(&s.mu)
		peerFirst, localFirst := uint64(2), uint64(1)
		if server {
			peerFirst, localFirst = 1, 2
		}
		for i := uint64(0); i < 6000; i++ {
			sid := peerFirst + 2*i
			s.onStream(sid, 0, []byte("peer"), false)
			st := s.streams[sid]
			if st == nil {
				t.Fatalf("server=%v peer stream %d refused", server, sid)
			}
			s.killStream(st, ErrClosed)
			s.acceptQ = nil
			if s.observePeerStream(localFirst + 2*i) {
				t.Fatal("local namespace consumed peer retirement history")
			}
		}
		if s.peerSeenThrough != 6000 || len(s.peerSeenAhead) != 0 || len(s.tomb) != 0 {
			t.Fatal("bidirectional retirement failed to compact")
		}
		s.onStream(peerFirst, 0, []byte("replayed"), false)
		s.onStream(localFirst, 0, []byte("wrong namespace"), false)
		if len(s.streams) != 0 {
			t.Fatal("retired or wrong-namespace stream allocated")
		}
	}
}
