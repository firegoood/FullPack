package smux

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type pacedBulkConn struct {
	net.Conn
	enabled          atomic.Bool
	once             sync.Once
	entered, release chan struct{}
}

func (c *pacedBulkConn) Write(b []byte) (int, error) {
	if c.enabled.Load() && len(b) > 2048 {
		c.once.Do(func() { close(c.entered); <-c.release })
		time.Sleep(8 * time.Millisecond)
	}
	return c.Conn.Write(b)
}

func TestAuthenticatedControlPassesQueuedBulk(t *testing.T) {
	x, y := net.Pipe()
	raw := &pacedBulkConn{Conn: x, entered: make(chan struct{}), release: make(chan struct{})}
	cfg := DefaultConfig()
	cfg.Version = 2
	cfg.KeepAliveDisabled = true
	a, _ := Client(raw, cfg)
	b, _ := Server(y, cfg)
	defer a.Close()
	defer b.Close()
	control, _ := a.OpenStream()
	peerControl, _ := b.AcceptStream()
	if a.PrioritizeControl(control) == nil {
		t.Fatal("unenrolled stream gained priority")
	}
	if err := a.EnableAuthenticatedFlowControl(); err != nil {
		t.Fatal(err)
	}
	if err := b.EnableAuthenticatedFlowControl(); err != nil {
		t.Fatal(err)
	}
	if err := a.PrioritizeControl(control); err != nil {
		t.Fatal(err)
	}
	if a.PrioritizeControl(peerControl) == nil || a.PrioritizeControl(&Stream{}) == nil {
		t.Fatal("foreign/malformed stream gained control priority")
	}
	var writers, readers sync.WaitGroup
	var data []*Stream
	for range 128 {
		d, _ := a.OpenStream()
		r, _ := b.AcceptStream()
		data = append(data, d)
		readers.Add(1)
		go func() { defer readers.Done(); _, _ = io.ReadFull(r, make([]byte, 32768)) }()
	}
	if a.PrioritizeControl(data[0]) == nil {
		t.Fatal("more than one stream gained control priority")
	}
	defer func() { a.Close(); b.Close(); writers.Wait(); readers.Wait() }()
	raw.enabled.Store(true)
	for _, d := range data {
		writers.Add(1)
		go func() { defer writers.Done(); _, _ = d.Write(make([]byte, 32768)) }()
	}
	<-raw.entered
	until := time.Now().Add(3 * time.Second)
	for a.sq.Len()+len(a.shaper) < 120 && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	if a.sq.Len()+len(a.shaper) < 120 {
		close(raw.release)
		t.Fatal("fixture did not build its bounded bulk queue")
	}
	ack := make(chan struct{})
	go func() { defer close(ack); io.Copy(peerControl, peerControl) }()
	defer func() { b.Close(); <-ack }()
	// Compressed deadline and synthetic byte pacing, not a WAN timing claim.
	_ = control.SetDeadline(time.Now().Add(300 * time.Millisecond))
	start := time.Now()
	close(raw.release)
	_, err := control.Write([]byte("control"))
	if err == nil {
		_, err = io.ReadFull(control, make([]byte, 7))
	}
	if err != nil {
		t.Fatal("control starved behind 128 queued bulk writers", err)
	}
	t.Logf("authenticated control round trip with 128 bulk writers: %s", time.Since(start))
	// New user admission must also pass the existing bulk queue. The initial
	// SYN and bounded application header may not wait behind megabytes of data.
	type opened struct {
		stream *Stream
		err    error
	}
	openedStream := make(chan opened, 1)
	go func() { s, err := a.OpenStream(); openedStream <- opened{s, err} }()
	var fast *Stream
	select {
	case result := <-openedStream:
		fast, err = result.stream, result.err
	case <-time.After(300 * time.Millisecond):
		t.Fatal("new stream SYN starved behind bulk")
	}
	if err != nil {
		t.Fatal(err)
	}
	peerFast, err := b.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	fastDone := make(chan struct{})
	go func() {
		defer close(fastDone)
		var header [43]byte
		if _, err := io.ReadFull(peerFast, header[:]); err == nil {
			_, _ = peerFast.Write([]byte{1})
		}
	}()
	defer func() { b.Close(); <-fastDone }()
	_ = fast.SetDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := fast.Write(make([]byte, 43)); err != nil {
		t.Fatal("new data handshake starved", err)
	}
	var result [1]byte
	if _, err := io.ReadFull(fast, result[:]); err != nil || result[0] != 1 {
		t.Fatal("new data ACK starved", err)
	}
}

func TestControlPriorityRetainsBoundedQueueAndUserFairness(t *testing.T) {
	s := &Session{controlWrites: make(chan writeRequest, 8), handshakeWrites: make(chan writeRequest, 32), sq: NewShaperQueue()}
	for range 32 {
		s.handshakeWrites <- writeRequest{frame: newFrame(2, cmdSYN, 7)}
	}
	for range 8 {
		s.controlWrites <- writeRequest{frame: newFrame(2, cmdPSH, 3)}
	}
	s.sq.Push(writeRequest{frame: newFrame(2, cmdPSH, 5)})
	burst := 0
	for range 16 {
		r, ok := s.nextWrite(&burst)
		if !ok || r.frame.sid != 3 {
			t.Fatal("control lane lost priority")
		}
		s.controlWrites <- r
	}
	r, ok := s.nextWrite(&burst)
	if !ok || r.frame.sid != 5 || burst != 0 || cap(s.controlWrites) != 8 || cap(s.handshakeWrites) != 32 || len(s.handshakeWrites) != 32 {
		t.Fatal("control priority starved queued users or lost its bound")
	}
}

func TestHandshakePriorityIsFiniteAndRequiresEnrollment(t *testing.T) {
	a, _ := flowPair(t, 2)
	s := newStream(3, 32768, a)
	if s.dataClass(1) != CLSDATA || s.priorityRemaining.Load() != 256 {
		t.Fatal("unauthenticated stream received priority")
	}
	if err := a.EnableAuthenticatedFlowControl(); err != nil {
		t.Fatal(err)
	}
	if s.dataClass(128) != CLSCTRL || s.dataClass(128) != CLSCTRL || s.dataClass(1) != CLSDATA {
		t.Fatal("handshake allowance was not bounded to 256 bytes")
	}
	large := newStream(5, 32768, a)
	if large.dataClass(257) != CLSDATA || large.dataClass(1) != CLSDATA {
		t.Fatal("bulk payload retained a handshake allowance")
	}
}
