package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"crypto/rand"
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

func freshSecret() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func freeAddress() string {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().String()
}

type echoFixture struct {
	address string
	tcp     net.Listener
	udp     net.PacketConn
	wg      sync.WaitGroup
	mu      sync.Mutex
	active  map[net.Conn]bool
	closed  bool
}

func startEcho(ctx context.Context) (*echoFixture, error) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	u, err := net.ListenPacket("udp4", l.Addr().String())
	if err != nil {
		l.Close()
		return nil, err
	}
	e := &echoFixture{address: l.Addr().String(), tcp: l, udp: u, active: map[net.Conn]bool{}}
	e.wg.Add(2)
	go func() {
		defer e.wg.Done()
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			e.mu.Lock()
			if e.closed {
				e.mu.Unlock()
				c.Close()
				return
			}
			e.active[c] = true
			e.wg.Add(1)
			e.mu.Unlock()
			go func() {
				defer e.wg.Done()
				defer c.Close()
				defer func() { e.mu.Lock(); delete(e.active, c); e.mu.Unlock() }()
				_, _ = io.Copy(c, c)
			}()
		}
	}()
	go func() {
		defer e.wg.Done()
		b := make([]byte, 16000)
		for {
			n, peer, err := u.ReadFrom(b)
			if err != nil {
				return
			}
			_, _ = u.WriteTo(b[:n], peer)
		}
	}()
	return e, nil
}

func (e *echoFixture) Close() {
	e.tcp.Close()
	e.udp.Close()
	e.mu.Lock()
	e.closed = true
	for c := range e.active {
		c.Close()
	}
	e.mu.Unlock()
	e.wg.Wait()
}

type xrayFixture struct {
	address   string
	passwords []string
	cmd       *exec.Cmd
}

func startXray(ctx context.Context, binary, echoAddress string, users int) (*xrayFixture, error) {
	x := &xrayFixture{address: freeAddress()}
	clients := []map[string]any{}
	for i := range users {
		password := freshSecret()
		x.passwords = append(x.passwords, password)
		clients = append(clients, map[string]any{"email": fmt.Sprintf("synthetic-%d@audit.invalid", i), "method": "aes-256-gcm", "password": password, "iv_check": true})
	}
	_, port, _ := net.SplitHostPort(x.address)
	var p int
	fmt.Sscan(port, &p)
	_, echoPort, _ := net.SplitHostPort(echoAddress)
	// Current Xray deliberately blocks private/loopback outbound destinations.
	// Permit only this generated echo port in this isolated synthetic fixture.
	// Never edit the installed Xray policy or allow an arbitrary destination.
	finalRules := []any{map[string]any{"action": "allow", "network": "tcp,udp", "ip": []string{"127.0.0.1/32"}, "port": echoPort}}
	cfg := map[string]any{"log": map[string]any{"loglevel": "none"}, "inbounds": []any{map[string]any{
		"listen": "127.0.0.1", "port": p, "protocol": "shadowsocks", "tag": "synthetic-only",
		"settings":       map[string]any{"method": "aes-256-gcm", "network": "tcp,udp", "clients": clients},
		"streamSettings": map[string]any{"network": "tcp", "tcpSettings": map[string]any{"header": map[string]any{"type": "none"}}},
	}}, "outbounds": []any{map[string]any{"protocol": "freedom", "settings": map[string]any{"finalRules": finalRules}}}}
	data, _ := json.Marshal(cfg)
	x.cmd = exec.CommandContext(ctx, binary, "run", "-config", "stdin:")
	stdin, err := x.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	// The synthetic account config never enters a file, argv or log.
	x.cmd.Stdout, x.cmd.Stderr = io.Discard, io.Discard
	if err = x.cmd.Start(); err != nil {
		return nil, err
	}
	if _, err = stdin.Write(data); err != nil {
		stdin.Close()
		x.Close()
		return nil, err
	}
	stdin.Close()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		c, err := net.DialTimeout("tcp", x.address, 100*time.Millisecond)
		if err == nil {
			c.Close()
			return x, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	x.Close()
	return nil, fmt.Errorf("isolated Xray did not open its loopback listener")
}

func (x *xrayFixture) Close() {
	if x != nil && x.cmd != nil && x.cmd.Process != nil {
		_ = x.cmd.Process.Kill()
		_ = x.cmd.Wait()
	}
}

type autoFixture struct {
	entry      string
	iran, peer *peerauto.Manager
	cancel     context.CancelFunc
	done       chan error
	dir        string
}

func startAuto(ctx context.Context, backend string, limit int) (*autoFixture, error) {
	child, cancel := context.WithCancel(ctx)
	dir, err := os.MkdirTemp("", "fullpack-synthetic-state-")
	if err != nil {
		cancel()
		return nil, err
	}
	f := &autoFixture{entry: freeAddress(), cancel: cancel, done: make(chan error, 2), dir: dir}
	endpoint := freeAddress()
	maps := []string{f.entry + "=" + backend}
	a := config.AutoConfig{Enabled: true, TunnelID: freshSecret(), IranID: freshSecret(), KharejID: freshSecret(), Ports: maps, MaxConnections: limit, AcceptUDP: true,
		Candidates: []config.AutoCandidate{{ID: "tcp", Transport: config.TCP, Endpoint: endpoint}}}
	token := freshSecret()
	f.iran, err = peerauto.New(child, &config.Config{Auto: a, Server: config.ServerConfig{BindAddr: endpoint, Token: token, Ports: maps, LogLevel: "error"}}, filepath.Join(dir, "iran.json"))
	if err != nil {
		cancel()
		os.RemoveAll(dir)
		return nil, err
	}
	f.peer, err = peerauto.New(child, &config.Config{Auto: a, Client: config.ClientConfig{RemoteAddr: endpoint, Token: token, LogLevel: "error"}}, filepath.Join(dir, "peer.json"))
	if err != nil {
		cancel()
		f.iran.Close()
		os.RemoveAll(dir)
		return nil, err
	}
	go func() { f.done <- f.iran.Start() }()
	go func() { f.done <- f.peer.Start() }()
	until := time.Now().Add(20 * time.Second)
	for time.Now().Before(until) {
		if f.iran.Snapshot().Active == "tcp" {
			return f, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	f.Close()
	return nil, fmt.Errorf("isolated Auto pair failed to bootstrap")
}

func (f *autoFixture) Close() {
	if f == nil {
		return
	}
	f.cancel()
	f.iran.Close()
	f.peer.Close()
	<-f.done
	<-f.done
	os.RemoveAll(f.dir)
}
