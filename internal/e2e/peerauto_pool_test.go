package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

// Six actual encrypted carriers in both initiation directions, with network
// faults on two chosen standbys. Replenishment must not restart the pair, move
// user listeners or spend a route-generation/switch budget.
func TestPeerAutoPoolReplenishesFourAfterTwoStandbyFailures(t *testing.T) {
	backend := startEchoBackend(t)
	udpBackend := startUDPEcho(t)
	entry, udpEntry := freePort(t), freePort(t)
	maps := []string{fmt.Sprintf("127.0.0.1:%d=%s", entry, backend.addr), fmt.Sprintf("127.0.0.1:%d=%s", udpEntry, udpBackend)}
	proxies := map[string]*autoFaultProxy{}
	candidates := []config.AutoCandidate{}
	for _, tr := range []config.TransportType{config.TCP, config.WSS, config.STEALTH} {
		for _, mode := range []string{"reverse", "direct"} {
			id := string(tr) + "-" + mode[:1]
			p := newAutoFaultProxy(t)
			proxies[id] = p
			candidates = append(candidates, config.AutoCandidate{ID: id, Transport: tr, Mode: mode, Endpoint: p.listener.Addr().String(), LocalBind: p.target})
		}
	}
	a := config.AutoConfig{Enabled: true, TunnelID: "pool-replenish-test", IranID: "pool-enrolled-iran", KharejID: "pool-enrolled-kharej", Ports: maps, AcceptUDP: true, Candidates: candidates, Selection: &config.AutoSelectionConfig{RetryInterval: 30}}
	token := "pool-replenishment-test-secret-0123456789"
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
	waitFour := func() {
		t.Helper()
		until := time.Now().Add(3 * time.Minute)
		for time.Now().Before(until) {
			l, r := m.Snapshot(), p.Snapshot()
			if l.Selection != nil && r.Selection != nil && !l.Selection.Scanning && l.Selection.Healthy == 4 && r.Selection.Healthy == 4 && len(l.Selection.Selected) == 4 && slices.Equal(l.Selection.Selected, r.Selection.Selected) && l.Active != "" && l.Active == r.Active && l.Generation == r.Generation {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		logAutoFixtureResources(t, "pool bootstrap failure")
		t.Logf("pool details: Iran=%+v Kharej=%+v", m.Snapshot().Selection, p.Snapshot().Selection)
		t.Fatalf("four measured paths never agreed: Iran=%+v Kharej=%+v", m.Snapshot(), p.Snapshot())
	}
	waitFour()
	before := m.Snapshot()
	held, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", entry), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	checkHeld := func() {
		t.Helper()
		payload := randomPayload(t, 4096)
		_ = held.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := held.Write(payload); err != nil {
			t.Fatal(err)
		}
		got := make([]byte, len(payload))
		if _, err := io.ReadFull(held, got); err != nil || !bytes.Equal(got, payload) {
			t.Fatalf("pool refill interrupted production: %v", err)
		}
	}
	checkHeld()
	failed := []string{}
	for _, id := range before.Selection.Selected {
		if id != before.Active && len(failed) < 2 {
			proxies[id].fail()
			failed = append(failed, id)
		}
	}
	if len(failed) != 2 {
		t.Fatal("test did not fault two selected standbys")
	}
	// Observe the threshold-triggered rescan instead of counting an old or
	// merely authenticated reserve as stable.
	until := time.Now().Add(25 * time.Second)
	sawRescan := false
	for time.Now().Before(until) {
		s := m.Snapshot()
		if s.Selection.Healthy <= 2 && s.Selection.Scanning {
			sawRescan = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !sawRescan {
		t.Fatalf("two-path threshold did not trigger scanning: %+v", m.Snapshot())
	}
	checkHeld()
	waitFour()
	checkHeld()
	after := m.Snapshot()
	if after.Generation != before.Generation || after.Active != before.Active {
		t.Fatal("reserve replenishment changed the production generation")
	}
	for _, id := range failed {
		if slices.Contains(after.Selection.Selected, id) {
			t.Fatal("failed path retained as selected stable")
		}
	}
	if err := udpRoundTrip(fmt.Sprintf("127.0.0.1:%d", udpEntry), randomPayload(t, 8000)); err != nil {
		t.Fatal(err)
	}
	t.Logf("refilled 4 -> 2 -> 4 at unchanged generation %d; active=%s; selected=%v", after.Generation, after.Active, after.Selection.Selected)
	// Only two routes now physically remain. An active-path failure must use
	// ordinary fenced recovery and report the shortfall rather than inventing
	// extra stable paths or treating user sockets as migration proof.
	held.Close()
	proxies[after.Active].fail()
	for _, id := range after.Selection.Selected {
		if id != after.Active {
			proxies[id].fail()
			break
		}
	}
	until = time.Now().Add(90 * time.Second)
	recovered := false
	for time.Now().Before(until) {
		l, r := m.Snapshot(), p.Snapshot()
		if l.Active != "" && l.Active != after.Active && l.Active == r.Active && l.Generation == r.Generation && l.Generation > after.Generation && l.Selection.Healthy == 2 {
			recovered = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !recovered {
		t.Fatalf("active loss/two-route shortfall did not recover truthfully: %+v %+v", m.Snapshot(), p.Snapshot())
	}
	if err := (&tunnel{Entry: fmt.Sprintf("127.0.0.1:%d", entry)}).roundTrip(randomPayload(t, 32768)); err != nil {
		t.Fatal(err)
	}
	if err := udpRoundTrip(fmt.Sprintf("127.0.0.1:%d", udpEntry), randomPayload(t, 8000)); err != nil {
		t.Fatal(err)
	}
	if m.Snapshot().Selection.Healthy > 2 {
		t.Fatal("unavailable routes falsely counted as stable")
	}
}
