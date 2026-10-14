package e2e

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

// Faults close only the selected carrier's network connections. Both real
// managers, other authenticated candidates and the application listener live on.
type autoFaultProxy struct {
	listener net.Listener
	target   string
	mu       sync.Mutex
	cut      bool
	conns    map[net.Conn]bool
	workers  sync.WaitGroup
}

func newAutoFaultProxy(t *testing.T) *autoFaultProxy {
	t.Helper()
	p := &autoFaultProxy{target: fmt.Sprintf("127.0.0.1:%d", freePort(t)), conns: map[net.Conn]bool{}}
	_, port, _ := net.SplitHostPort(p.target)
	var err error
	p.listener, err = net.Listen("tcp", "127.0.0.2:"+port)
	if err != nil {
		t.Fatal(err)
	}
	p.workers.Add(1)
	go func() {
		defer p.workers.Done()
		for {
			c, err := p.listener.Accept()
			if err != nil {
				return
			}
			p.mu.Lock()
			if p.cut {
				p.mu.Unlock()
				c.Close()
				continue
			}
			p.conns[c] = true
			p.workers.Add(1)
			p.mu.Unlock()
			go func() {
				defer p.workers.Done()
				defer func() { c.Close(); p.mu.Lock(); delete(p.conns, c); p.mu.Unlock() }()
				r, err := net.DialTimeout("tcp", p.target, time.Second)
				if err != nil {
					return
				}
				defer r.Close()
				done := make(chan struct{})
				go func() { io.Copy(r, c); r.Close(); c.Close(); close(done) }()
				io.Copy(c, r)
				r.Close()
				c.Close()
				<-done
			}()
		}
	}()
	t.Cleanup(p.fail)
	return p
}

func (p *autoFaultProxy) fail() {
	p.listener.Close()
	p.mu.Lock()
	p.cut = true
	for c := range p.conns {
		c.Close()
	}
	p.mu.Unlock()
	p.workers.Wait()
}

func TestPeerAutoConsecutiveFailuresKeepDefaultsAndReachThirdCarrier(t *testing.T) {
	backend := startEchoBackend(t)
	udpBackend := startUDPEcho(t)
	entry, udpEntry := freePort(t), freePort(t)
	maps := []string{fmt.Sprintf("127.0.0.1:%d=%s", entry, backend.addr), fmt.Sprintf("127.0.0.1:%d=%s", udpEntry, udpBackend)}
	var proxies []*autoFaultProxy
	var candidates []config.AutoCandidate
	for _, tr := range []config.TransportType{config.TCP, config.WSS, config.WSSMUX} {
		p := newAutoFaultProxy(t)
		proxies = append(proxies, p)
		candidates = append(candidates, config.AutoCandidate{ID: string(tr), Transport: tr, Endpoint: p.listener.Addr().String(), LocalBind: p.target})
	}
	a := config.AutoConfig{Enabled: true, ObserveOnly: true, TunnelID: "consecutive-failure-test", IranID: "consecutive-iran", KharejID: "consecutive-kharej", Ports: maps, AcceptUDP: true, Candidates: candidates}
	defaults := peerauto.Defaults(a)
	if defaults.Cooldown != 300 || defaults.MaxSwitchesPerHour != 2 {
		t.Fatal("test must retain the actual default policy")
	}
	token := "consecutive-failure-test-secret-0123456789"
	srv := &config.Config{Auto: a, Server: config.ServerConfig{Transport: config.TCP, BindAddr: candidates[0].Endpoint, Token: token, Ports: maps}}
	cli := &config.Config{Auto: a, Client: config.ClientConfig{Transport: config.TCP, RemoteAddr: candidates[0].Endpoint, Token: token}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := peerauto.New(ctx, srv, filepath.Join(t.TempDir(), "iran.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := peerauto.New(ctx, cli, filepath.Join(t.TempDir(), "kharej.json"))
	if err != nil {
		m.Close()
		t.Fatal(err)
	}
	done := make(chan error, 2)
	go func() { done <- m.Start() }()
	go func() { done <- p.Start() }()
	t.Cleanup(func() {
		cancel()
		m.Close()
		p.Close()
		for range 2 {
			if err := <-done; err != nil {
				t.Error(err)
			}
		}
	})
	wait := func(id string, allReady bool) {
		t.Helper()
		deadline := time.Now().Add(45 * time.Second)
		for time.Now().Before(deadline) {
			left, right := m.Snapshot(), p.Snapshot()
			ready := true
			if allReady {
				for _, s := range append(left.Candidates, right.Candidates...) {
					ready = ready && s.Ready
				}
			}
			if left.Active == id && right.Active == id && left.Generation == right.Generation && ready {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("carrier %s did not agree: Iran=%+v Kharej=%+v", id, m.Snapshot(), p.Snapshot())
	}
	traffic := func() {
		t.Helper()
		tun := &tunnel{Entry: fmt.Sprintf("127.0.0.1:%d", entry)}
		if err := tun.roundTrip(randomPayload(t, 32768)); err != nil {
			t.Fatal(err)
		}
		if err := udpRoundTrip(fmt.Sprintf("127.0.0.1:%d", udpEntry), randomPayload(t, 8000)); err != nil {
			t.Fatal(err)
		}
	}
	wait("tcp", true)
	traffic()
	for _, id := range []string{"wss", "wssmux"} {
		for range 3 {
			probeCtx, cancelProbe := context.WithTimeout(ctx, 20*time.Second)
			err := m.Action(probeCtx, peerauto.Action{Operation: "test-now", Candidate: id})
			cancelProbe()
			if err != nil {
				t.Fatalf("standby %s did not pass independent integrity probes: %v", id, err)
			}
		}
	}
	actionCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	if err := m.Action(actionCtx, peerauto.Action{Operation: "resume"}); err != nil {
		t.Fatal(err)
	}
	previous := m.Snapshot().Generation
	for i, next := range []string{"wss", "wssmux"} {
		start := time.Now()
		proxies[i].fail()
		wait(next, false)
		elapsed := time.Since(start)
		if m.Snapshot().Generation <= previous {
			t.Fatal("failure recovery did not advance its durable generation")
		}
		previous = m.Snapshot().Generation
		traffic()
		t.Logf("confirmed network failure %d reached %s in %s with unchanged 300s/2-per-hour defaults", i+1, next, elapsed)
	}
}
