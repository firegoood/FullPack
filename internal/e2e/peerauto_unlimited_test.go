//go:build linux

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
	"github.com/shirou/gopsutil/v4/process"
)

// These are held application connections, not a count of authenticated users.
// The separate synthetic-Xray audit measures independent accounts as well.
func TestPeerAutoUnlimitedApplicationConnections(t *testing.T) {
	if os.Getenv("FULLPACK_AUTO_UNLIMITED_LOAD") != "1" {
		t.Skip("opt-in held 1000-connection carrier audit")
	}
	for _, carrier := range []config.TransportType{config.TCP, config.WSS, config.STEALTH, config.QUIC} {
		for _, mode := range []string{"reverse", "direct"} {
			t.Run(string(carrier)+"/"+mode, func(t *testing.T) { testUnlimitedConnections(t, carrier, mode, false) })
		}
	}
	t.Run("mixed-tcp-udp", func(t *testing.T) { testUnlimitedConnections(t, config.TCP, "reverse", true) })
}

func testUnlimitedConnections(t *testing.T, carrier config.TransportType, mode string, mixed bool) {
	logUnlimitedFixtureResources(t, "start")
	t.Cleanup(func() { logUnlimitedFixtureResources(t, "after joined cleanup") })
	const count = 1000
	backend := startEchoBackend(t)
	if mixed {
		u, err := net.ListenPacket("udp", backend.addr)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			b := make([]byte, 2048)
			for {
				n, p, e := u.ReadFrom(b)
				if e != nil {
					return
				}
				if _, e = u.WriteTo(b[:n], p); e != nil {
					return
				}
			}
		}()
		t.Cleanup(func() { u.Close(); <-done })
	}
	endpoint, entry := fmt.Sprintf("127.0.0.1:%d", freePort(t)), fmt.Sprintf("127.0.0.1:%d", freePort(t))
	maps := []string{entry + "=" + backend.addr}
	zero := 0
	a := config.AutoConfig{Enabled: true, TunnelID: "unlimited-load-tunnel", IranID: "unlimited-load-iran", KharejID: "unlimited-load-kharej", Ports: maps, ConnectionLimit: &zero,
		AcceptUDP: mixed,
		// Race/preceding subtests share one process. This is a test RSS budget,
		// not a production default or removal of CPU/probe-pressure protection.
		MaxMemoryMB: 2048, Candidates: []config.AutoCandidate{{ID: "route", Transport: carrier, Mode: mode, Endpoint: endpoint}}}
	token := "unlimited-application-load-secret-0123456789"
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	m, err := peerauto.New(ctx, &config.Config{Auto: a, Server: config.ServerConfig{BindAddr: endpoint, Token: token, Ports: maps, LogLevel: "error"}}, filepath.Join(t.TempDir(), "iran.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := peerauto.New(ctx, &config.Config{Auto: a, Client: config.ClientConfig{RemoteAddr: endpoint, Token: token, LogLevel: "error"}}, filepath.Join(t.TempDir(), "peer.json"))
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
			select {
			case err := <-done:
				if err != nil {
					t.Error(err)
				}
			case <-time.After(15 * time.Second):
				t.Error("unlimited workers did not join")
			}
		}
		if m.Snapshot().Flows.Live != 0 || p.Snapshot().Flows.Live != 0 {
			t.Error("unlimited shutdown leaked live flows")
		}
	})
	until := time.Now().Add(45 * time.Second)
	for m.Snapshot().Active != "route" && time.Now().Before(until) {
		time.Sleep(25 * time.Millisecond)
	}
	if m.Snapshot().Active != "route" {
		logUnlimitedFixtureResources(t, "bootstrap failure")
		t.Fatalf("unlimited pair did not bootstrap: %+v", m.Snapshot())
	}
	gen := m.Snapshot().Generation
	var udp []net.Conn
	t.Cleanup(func() {
		for _, c := range udp {
			c.Close()
		}
	})
	checkUDP := func(c net.Conn, i int) error {
		payload := []byte(fmt.Sprintf("unlimited UDP source %d", i))
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := c.Write(payload); err != nil {
			return err
		}
		b := make([]byte, 2048)
		n, err := c.Read(b)
		if err == nil && !bytes.Equal(b[:n], payload) {
			err = fmt.Errorf("UDP %d corrupt", i)
		}
		return err
	}
	if mixed {
		for i := range 256 {
			c, err := net.DialTimeout("udp", entry, 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			udp = append(udp, c)
			if err = checkUDP(c, i); err != nil {
				t.Fatal(err)
			}
		}
	}
	var heldMu sync.Mutex
	var held []net.Conn
	t.Cleanup(func() {
		for _, c := range held {
			c.Close()
		}
	})
	check := func(c net.Conn, i int) error {
		payload := []byte(fmt.Sprintf("unlimited connection %04d: %s", i, bytes.Repeat([]byte{byte(i%251 + 1)}, 1024)))
		_ = c.SetDeadline(time.Now().Add(20 * time.Second))
		if _, err := c.Write(payload); err != nil {
			return err
		}
		got := make([]byte, len(payload))
		if _, err := io.ReadFull(c, got); err != nil {
			return err
		}
		if !bytes.Equal(got, payload) {
			return fmt.Errorf("connection %d corrupt", i)
		}
		return nil
	}
	var wg sync.WaitGroup
	failures := make(chan error, count)
	barrier := make(chan struct{})
	for i := range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-barrier
			c, err := net.DialTimeout("tcp", entry, 10*time.Second)
			if err == nil {
				err = check(c, i)
			}
			if err != nil {
				if c != nil {
					c.Close()
				}
				failures <- err
				return
			}
			heldMu.Lock()
			held = append(held, c)
			heldMu.Unlock()
		}()
	}
	close(barrier)
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if len(held) != count {
		t.Fatalf("only %d/%d held connections served", len(held), count)
	}
	for _, m := range []*peerauto.Manager{m, p} {
		s := m.Snapshot()
		if !s.Flows.Unlimited || s.Flows.Limit != 0 || s.Flows.Live != count+len(udp) || s.Flows.LiveUDP != len(udp) || s.Flows.Rejected != 0 || s.Generation != gen || s.Data.Code != "available" {
			t.Fatalf("hidden ceiling or load failure: %+v", s)
		}
	}
	for i, c := range held {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := check(c, i); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	for i, c := range udp {
		if err := checkUDP(c, i); err != nil {
			t.Error(err)
		}
	}
	t.Logf("%s %s: %d held application connections verified twice, no enrollment ceiling, generation %d", carrier, mode, count, gen)
}

func logUnlimitedFixtureResources(t *testing.T, phase string) {
	t.Helper()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	var rss uint64
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if info, err := p.MemoryInfo(); err == nil {
			rss = info.RSS
		}
	}
	t.Logf("unlimited fixture %s: RSS=%d KiB heap=%d KiB goroutines=%d GOMAXPROCS=%d; production pressure guards retained", phase, rss/1024, memory.HeapAlloc/1024, runtime.NumGoroutine(), runtime.GOMAXPROCS(0))
}
