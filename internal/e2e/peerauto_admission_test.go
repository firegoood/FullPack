package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

func TestPeerAutoUDPShareAndQueuedTCPPreserveData(t *testing.T) {
	tcpBackend, udpBackend := startEchoBackend(t), startUDPEcho(t)
	entry := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	udpEntry := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	endpoint := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	maps := []string{entry + "=" + tcpBackend.addr, udpEntry + "=" + udpBackend}
	a := config.AutoConfig{Enabled: true, TunnelID: "admission-regression", IranID: "admission-regression-iran", KharejID: "admission-regression-peer", Ports: maps, AcceptUDP: true, MaxConnections: 64,
		Candidates: []config.AutoCandidate{{ID: "tcp", Transport: config.TCP, Endpoint: endpoint}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	token := "admission-regression-test-secret-0123456789"
	m, err := peerauto.New(ctx, &config.Config{Auto: a, Server: config.ServerConfig{BindAddr: endpoint, Token: token, Ports: maps}}, filepath.Join(t.TempDir(), "iran.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := peerauto.New(ctx, &config.Config{Auto: a, Client: config.ClientConfig{RemoteAddr: endpoint, Token: token}}, filepath.Join(t.TempDir(), "peer.json"))
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
		for _, owner := range []*peerauto.Manager{m, p} {
			if flows := owner.Snapshot().Flows; flows.Live != 0 || flows.QueuedTCP != 0 {
				t.Error("shutdown retained admission resources", flows)
			}
		}
		for range 2 {
			if err := <-done; err != nil {
				t.Error(err)
			}
		}
	})
	until := time.Now().Add(25 * time.Second)
	for m.Snapshot().Active != "tcp" && time.Now().Before(until) {
		time.Sleep(20 * time.Millisecond)
	}
	if m.Snapshot().Active != "tcp" {
		t.Fatal("pair did not bootstrap")
	}
	generation := m.Snapshot().Generation
	payload := []byte("verified mixed UDP/TCP admission")
	echo := func(c net.Conn) error {
		_ = c.SetDeadline(time.Now().Add(4 * time.Second))
		if _, err := c.Write(payload); err != nil {
			return err
		}
		got := make([]byte, len(payload))
		if _, err := io.ReadFull(c, got); err != nil {
			return err
		}
		if !bytes.Equal(got, payload) {
			return fmt.Errorf("payload mismatch")
		}
		return nil
	}
	var udp, tcp []net.Conn
	t.Cleanup(func() {
		for _, c := range append(udp, tcp...) {
			c.Close()
		}
	})
	for range 16 {
		c, err := net.Dial("udp", udpEntry)
		if err != nil {
			t.Fatal(err)
		}
		udp = append(udp, c)
		if err := echo(c); err != nil {
			t.Fatal(err)
		}
	}
	extraUDP, err := net.Dial("udp", udpEntry)
	if err != nil {
		t.Fatal(err)
	}
	_ = extraUDP.SetDeadline(time.Now().Add(200 * time.Millisecond))
	_, _ = extraUDP.Write(payload)
	var b [100]byte
	_, err = extraUDP.Read(b[:])
	extraUDP.Close()
	if err == nil {
		t.Fatal("UDP escaped its bounded share")
	}
	for range 48 {
		c, err := net.Dial("tcp", entry)
		if err != nil {
			t.Fatal(err)
		}
		tcp = append(tcp, c)
		if err := echo(c); err != nil {
			t.Fatal("idle UDP starved TCP", err)
		}
	}
	if s := m.Snapshot(); s.Data.Code != "capacity" || s.Flows.Live != 64 || s.Flows.LiveUDP != 16 {
		t.Fatalf("capacity hidden: %+v", s)
	}
	queued, err := net.Dial("tcp", entry)
	if err != nil {
		t.Fatal(err)
	}
	defer queued.Close()
	result := make(chan error, 1)
	go func() { result <- echo(queued) }()
	until = time.Now().Add(time.Second)
	for m.Snapshot().Flows.QueuedTCP != 1 && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	if m.Snapshot().Flows.QueuedTCP != 1 {
		t.Fatal("extra TCP was dropped instead of queued")
	}
	tcp[0].Close()
	if err := <-result; err != nil {
		t.Fatal("queued TCP did not recover when capacity freed", err)
	}
	for _, c := range udp {
		if err := echo(c); err != nil {
			t.Fatal("TCP pressure interrupted active UDP", err)
		}
	}
	if s := m.Snapshot(); s.Generation != generation || s.Active != "tcp" || s.Data.Errors["udp_capacity"] == 0 {
		t.Fatal("pressure changed carrier or lost diagnostic", s)
	}
	// Cancellation must join pending admissions and long-lived UDP owners.
	pending, err := net.Dial("tcp", entry)
	if err != nil {
		t.Fatal(err)
	}
	defer pending.Close()
	until = time.Now().Add(time.Second)
	for m.Snapshot().Flows.QueuedTCP != 1 && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	if m.Snapshot().Flows.QueuedTCP != 1 {
		t.Fatal("shutdown fixture did not retain a queued TCP socket")
	}
	cancel()
}
