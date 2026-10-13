package peerauto

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/l3"
	"github.com/sirupsen/logrus"
)

type testPacketSession struct {
	carrierSession
	out chan []byte
	in  chan []byte
}

func (s testPacketSession) packetChannel(_ [16]byte) (packetEndpoint, error) {
	return packetEndpoint{send: func(b []byte) error { s.out <- append([]byte(nil), b...); return nil }, receive: func(ctx context.Context) ([]byte, error) {
		select {
		case b := <-s.in:
			return b, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, close: func() {}}, nil
}
func packetPair(t *testing.T, probe bool) (*packetPipe, *packetPipe, chan []byte, chan []byte) {
	t.Helper()
	a, b := make(chan []byte, 64), make(chan []byte, 64)
	w := &wire{key: []byte("0123456789abcdef0123456789abcdef")}
	if probe {
		w.base.Transaction = "probe"
	}
	p, e := newPacketPipe(testPacketSession{out: a, in: b}, w, true)
	if e != nil {
		t.Fatal(e)
	}
	q, e := newPacketPipe(testPacketSession{out: b, in: a}, w, false)
	if e != nil {
		t.Fatal(e)
	}
	return p, q, a, b
}
func TestAutoPacketIntegrityReplayAndGeneration(t *testing.T) {
	p, q, a, _ := packetPair(t, false)
	payload := make([]byte, 9000)
	_, _ = rand.Read(payload)
	if err := p.send(packetIP, 7, payload); err != nil {
		t.Fatal(err)
	}
	frames := make([][]byte, 0, 24)
	for len(a) > 0 {
		frames = append(frames, <-a)
	}
	// Reordering across the entire MTU is harmless; corruption and forged
	// counters do not poison replay or allocate authenticated reassemblies.
	bad := append([]byte(nil), frames[0]...)
	binary.BigEndian.PutUint64(bad[25:33], math.MaxUint64)
	a <- bad
	for i := len(frames) - 1; i >= 0; i-- {
		a <- frames[i]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m, err := q.receive(ctx, func(g uint64) bool { return g == 7 })
	if err != nil || !bytes.Equal(m.data, payload) {
		t.Fatalf("reassembly: %v", err)
	}
	for _, b := range frames {
		a <- b
	}
	if err := p.send(packetIP, 6, []byte("stale")); err != nil {
		t.Fatal(err)
	}
	if err := p.send(packetIP, 7, []byte("fresh")); err != nil {
		t.Fatal(err)
	}
	m, err = q.receive(ctx, func(g uint64) bool { return g == 7 })
	if err != nil || string(m.data) != "fresh" {
		t.Fatalf("replay/generation admitted: %q %v", m.data, err)
	}
	p.sequence = math.MaxUint64
	if p.send(packetIP, 7, []byte("overflow")) == nil {
		t.Fatal("nonce exhaustion accepted")
	}
	if p.send(packetProbe, 0, []byte("probe injection")) == nil {
		t.Fatal("production session accepted probe payload")
	}
}
func TestAutoPacketProbeChecksActualDatagrams(t *testing.T) {
	p, q, _, _ := packetPair(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	finish := respondPacketProbe(ctx, q, 32768)
	_, _, digest, err := exchangePacketProbe(ctx, p, 32768)
	if err != nil {
		t.Fatal(err)
	}
	got, err := finish()
	if err != nil || got != digest {
		t.Fatalf("datagram evidence disagrees: %v", err)
	}
}

func TestAutoDatagramProbeCannotPassOnControlAlone(t *testing.T) {
	p, _, _, _ := packetPair(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, _, err := exchangePacketProbe(ctx, p, 512); err == nil {
		t.Fatal("blocked datagrams qualified a carrier")
	}
}

type testPacketDevice struct {
	in, out chan []byte
	done    chan struct{}
	once    sync.Once
	name    string
}

func (d *testPacketDevice) Read(b [][]byte, s []int) (int, error) {
	select {
	case p := <-d.in:
		s[0] = copy(b[0], p)
		return 1, nil
	case <-d.done:
		return 0, net.ErrClosed
	}
}
func (d *testPacketDevice) Write(b [][]byte) (int, error) {
	for _, p := range b {
		select {
		case d.out <- append([]byte(nil), p...):
		case <-d.done:
			return 0, net.ErrClosed
		}
	}
	return len(b), nil
}
func (d *testPacketDevice) BatchSize() int   { return 1 }
func (d *testPacketDevice) Name() string     { return d.name }
func (d *testPacketDevice) Close() error     { d.once.Do(func() { close(d.done) }); return nil }
func (d *testPacketDevice) MTU() int         { return 9000 }
func (d *testPacketDevice) SetMTU(int) error { return nil }
func newTestPacketDevice(name string) *testPacketDevice {
	return &testPacketDevice{in: make(chan []byte, 8), out: make(chan []byte, 8), done: make(chan struct{}), name: name}
}
func testIP(size int) []byte {
	b := make([]byte, size)
	_, _ = rand.Read(b)
	b[0] = 0x45
	binary.BigEndian.PutUint16(b[2:4], uint16(size))
	return b
}
func freePacketPort(t *testing.T) string {
	t.Helper()
	c, e := net.ListenPacket("udp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := c.LocalAddr().String()
	c.Close()
	return addr
}

func TestAutoL3DirectAndReverseDatagramHandover(t *testing.T) {
	a := config.AutoConfig{Enabled: true, Mode: "l3", TunnelID: "l3-test-enrolled-tunnel", IranID: "l3-test-enrolled-iran", KharejID: "l3-test-enrolled-kharej", L3: &config.AutoL3Config{IranIP: "10.55.0.1/30", KharejIP: "10.55.0.2/30", MTU: 9000}, Candidates: []config.AutoCandidate{{ID: "udp", Transport: config.UDP, Mode: "direct", Endpoint: freePacketPort(t)}, {ID: "quic", Transport: config.QUIC, Endpoint: freePacketPort(t)}}}
	token := "0123456789abcdef0123456789abcdef"
	srv := &config.Config{Auto: a, Server: config.ServerConfig{BindAddr: "127.0.0.1:1", Token: token}}
	cli := &config.Config{Auto: a, Client: config.ClientConfig{RemoteAddr: "127.0.0.1:1", Token: token}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	iran, kharej := newTestPacketDevice("iran"), newTestPacketDevice("kharej")
	makeManager := func(c *config.Config, d *testPacketDevice) *Manager {
		m, e := NewWithPacketDevice(ctx, c, filepath.Join(t.TempDir(), d.name+".json"), func(l3.Config, *logrus.Logger) (l3.PacketDevice, error) { return d, nil })
		if e != nil {
			t.Fatal(e)
		}
		return m
	}
	m, p := makeManager(srv, iran), makeManager(cli, kharej)
	done := make(chan error, 2)
	go func() { done <- m.Start() }()
	go func() { done <- p.Start() }()
	t.Cleanup(func() {
		cancel()
		m.Close()
		p.Close()
		for range 2 {
			select {
			case e := <-done:
				if e != nil {
					t.Error(e)
				}
			case <-time.After(5 * time.Second):
				t.Error("shutdown not joined")
			}
		}
	})
	deadline := time.Now().Add(35 * time.Second)
	for m.Snapshot().Active != "udp" && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if m.Snapshot().Active != "udp" {
		t.Fatalf("bootstrap: %+v %+v", m.Snapshot(), p.Snapshot())
	}
	roundTrip := func(from, to *testPacketDevice) {
		t.Helper()
		packet := testIP(9000)
		from.in <- packet
		select {
		case b := <-to.out:
			if !bytes.Equal(packet, b) {
				t.Fatal("IP payload corrupted")
			}
		case <-time.After(3 * time.Second):
			t.Fatal("IP plane unavailable")
		}
	}
	roundTrip(iran, kharej)
	roundTrip(kharej, iran)
	before := m.Snapshot().Generation
	actionCtx, stop := context.WithTimeout(ctx, 25*time.Second)
	defer stop()
	if e := m.Action(actionCtx, Action{Operation: "test-now", Candidate: "quic"}); e != nil {
		t.Fatal(e)
	}
	if m.Snapshot().Generation != before {
		t.Fatal("isolated datagram probe changed route")
	}
	roundTrip(iran, kharej)
	m.mu.Lock()
	old := m.active
	m.mu.Unlock()
	if e := m.Action(actionCtx, Action{Operation: "switch", Candidate: "quic"}); e != nil {
		t.Fatal(e)
	}
	if m.Snapshot().Generation != p.Snapshot().Generation || m.Snapshot().Active != "quic" {
		t.Fatal("packet generations disagree")
	}
	if e := old.endpoint.w.packet.send(packetIP, old.generation, testIP(512)); e != nil && !errors.Is(e, net.ErrClosed) {
		t.Fatal(e)
	}
	select {
	case <-kharej.out:
		t.Fatal("retired generation entered TUN")
	case <-time.After(100 * time.Millisecond):
	}
	roundTrip(iran, kharej)
	roundTrip(kharej, iran)
	if m.device != iran || p.device != kharej {
		t.Fatal("handover rebuilt TUN")
	}
}
