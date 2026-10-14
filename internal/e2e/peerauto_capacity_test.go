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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

// Connections are application sockets through one enrolled pair. They are not
// independent users, a WAN bandwidth estimate or a 1000-user capacity claim.
func TestPeerAutoBoundedApplicationConnectionCapacity(t *testing.T) {
	if os.Getenv("FULLPACK_AUTO_LOAD") != "1" {
		t.Skip("opt-in bounded 100/192-connection load check")
	}
	backend := startEchoBackend(t)
	endpoint := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	entry := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	maps := []string{entry + "=" + backend.addr}
	a := config.AutoConfig{Enabled: true, TunnelID: "bounded-connection-load", IranID: "bounded-load-iran", KharejID: "bounded-load-kharej", Ports: maps, MaxConnections: 192,
		Candidates: []config.AutoCandidate{{ID: "tcp", Transport: config.TCP, Endpoint: endpoint}}}
	token := "bounded-connection-load-secret-0123456789"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := peerauto.New(ctx, &config.Config{Auto: a, Server: config.ServerConfig{BindAddr: endpoint, Token: token, Ports: maps}}, filepath.Join(t.TempDir(), "iran.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := peerauto.New(ctx, &config.Config{Auto: a, Client: config.ClientConfig{RemoteAddr: endpoint, Token: token}}, filepath.Join(t.TempDir(), "kharej.json"))
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
	deadline := time.Now().Add(30 * time.Second)
	for m.Snapshot().Active != "tcp" && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if m.Snapshot().Active != "tcp" {
		t.Fatal("load pair did not bootstrap")
	}
	var held []net.Conn
	t.Cleanup(func() {
		for _, c := range held {
			c.Close()
		}
	})
	payload := randomPayload(t, 4096)
	echo := func(c net.Conn) error {
		c.SetDeadline(time.Now().Add(20 * time.Second))
		if _, err := c.Write(payload); err != nil {
			return err
		}
		got := make([]byte, len(payload))
		if _, err := io.ReadFull(c, got); err != nil {
			return err
		}
		if !bytes.Equal(got, payload) {
			return fmt.Errorf("load payload corrupted")
		}
		return nil
	}
	for _, count := range []int{100, 192} {
		for len(held) < count {
			c, err := net.DialTimeout("tcp", entry, 5*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			held = append(held, c)
			if err := echo(c); err != nil {
				t.Fatal(err)
			}
		}
		if m.Snapshot().Flows.Live != count {
			t.Fatalf("expected %d admitted live sockets: %+v", count, m.Snapshot().Flows)
		}
		before := m.Snapshot().Generation
		probeCtx, stop := context.WithTimeout(ctx, 10*time.Second)
		err := m.Action(probeCtx, peerauto.Action{Operation: "test-now", Candidate: "tcp"})
		stop()
		if err == nil || !strings.Contains(err.Error(), "SKIPPED") {
			t.Fatalf("load did not reserve application capacity: %v", err)
		}
		start := time.Now()
		failures := make(chan error, count)
		var wg sync.WaitGroup
		for _, c := range held {
			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				for range 32 {
					if err := echo(c); err != nil {
						failures <- err
						return
					}
				}
			}(c)
		}
		wg.Wait()
		close(failures)
		for err := range failures {
			t.Error(err)
		}
		if m.Snapshot().Generation != before || m.Snapshot().Active != "tcp" {
			t.Fatal("skipped load probe changed the active generation")
		}
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		t.Logf("%d simultaneous loopback application connections: %d integrity bytes echoed in %s; process heap=%d goroutines=%d", count, count*32*len(payload), time.Since(start), mem.HeapAlloc, runtime.NumGoroutine())
	}
	extra, err := net.DialTimeout("tcp", entry, 5*time.Second)
	if err == nil {
		defer extra.Close()
		extra.SetDeadline(time.Now().Add(3 * time.Second))
		_, err = extra.Write(payload)
		if err == nil {
			var b [1]byte
			_, err = extra.Read(b[:])
		}
	}
	if err == nil || m.Snapshot().Flows.Live > 192 {
		t.Fatal("193rd connection escaped the admission ceiling")
	}
}
