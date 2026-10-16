//go:build linux

package e2e

import (
	"bytes"
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

// Slow applications must exhaust only their own finite byte credit. These
// loopback connections are not synthetic identities or a WAN/long-soak claim.
func TestPeerAutoSlowReadersPreserveControlAndOtherApplications(t *testing.T) {
	for _, transport := range []config.TransportType{config.TCP, config.WSS, config.STEALTH} {
		for _, mode := range []string{"reverse", "direct"} {
			t.Run(string(transport)+"/"+mode, func(t *testing.T) {
				backend, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				var backendMu sync.Mutex
				var backendConns []net.Conn
				var backendWorkers sync.WaitGroup
				acceptDone := make(chan struct{})
				go func() {
					defer close(acceptDone)
					for {
						c, err := backend.Accept()
						if err != nil {
							return
						}
						backendMu.Lock()
						backendConns = append(backendConns, c)
						backendMu.Unlock()
						backendWorkers.Add(1)
						go func() {
							defer backendWorkers.Done()
							defer c.Close()
							var marker [1]byte
							if _, err := io.ReadFull(c, marker[:]); err != nil {
								return
							}
							if marker[0] == 'S' {
								buf := make([]byte, 32768)
								for range 512 {
									if _, err := c.Write(buf); err != nil {
										return
									}
								}
							} else {
								io.Copy(c, c)
							}
						}()
					}
				}()
				t.Cleanup(func() {
					backend.Close()
					<-acceptDone
					backendMu.Lock()
					for _, c := range backendConns {
						c.Close()
					}
					backendMu.Unlock()
					backendWorkers.Wait()
				})
				endpoint := fmt.Sprintf("127.0.0.1:%d", freePort(t))
				entry := fmt.Sprintf("127.0.0.1:%d", freePort(t))
				maps := []string{entry + "=" + backend.Addr().String()}
				zero := 0
				a := config.AutoConfig{Enabled: true, TunnelID: "slow-reader-test-tunnel", IranID: "slow-reader-test-iran", KharejID: "slow-reader-test-peer", Ports: maps, ConnectionLimit: &zero, MaxMemoryMB: 2048,
					Candidates: []config.AutoCandidate{{ID: "route", Transport: transport, Mode: mode, Endpoint: endpoint}}}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				const token = "slow-reader-enrolled-test-token-0123456789"
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
							t.Error("slow-reader workers did not join")
						}
					}
				})
				until := time.Now().Add(45 * time.Second)
				for m.Snapshot().Active != "route" && time.Now().Before(until) {
					time.Sleep(25 * time.Millisecond)
				}
				initial := m.Snapshot()
				if initial.Active != "route" {
					t.Fatalf("pair unavailable: %+v", initial)
				}
				for _, peer := range []*peerauto.Manager{m, p} {
					mux := peer.Snapshot().Candidates[0].Mux
					if mux == nil || mux.Version != 2 || !mux.Isolated {
						t.Fatalf("full enrollment did not enable bounded v2 flow isolation: %+v", mux)
					}
				}
				var slow []net.Conn
				t.Cleanup(func() {
					for _, c := range slow {
						c.Close()
					}
				})
				for range 64 {
					c, err := net.DialTimeout("tcp", entry, 3*time.Second)
					if err != nil {
						t.Fatal(err)
					}
					slow = append(slow, c)
					_ = c.(*net.TCPConn).SetReadBuffer(4096)
					_ = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
					if _, err := c.Write([]byte{'S'}); err != nil {
						t.Fatal(err)
					}
				}
				until = time.Now().Add(25 * time.Second)
				for time.Now().Before(until) {
					mux := m.Snapshot().Candidates[0].Mux
					if mux != nil && mux.BufferedReceive >= 5<<20 {
						break
					}
					time.Sleep(25 * time.Millisecond)
				}
				st := m.Snapshot()
				mux := st.Candidates[0].Mux
				if mux == nil || mux.BufferedReceive < 5<<20 {
					t.Fatalf("fixture did not exceed the former 4MiB shared bucket: %+v", mux)
				}
				t.Logf("64 slow readers queued %d bytes; per-stream bound %d", mux.BufferedReceive, mux.MaxBufferedStream)
				fast, err := net.DialTimeout("tcp", entry, 3*time.Second)
				if err != nil {
					t.Fatal(err)
				}
				defer fast.Close()
				_ = fast.SetDeadline(time.Now().Add(18 * time.Second))
				if _, err := fast.Write([]byte{'F'}); err != nil {
					t.Fatal(err)
				}
				// Span multiple 2s heartbeats and the 8s control exchange timeout.
				for i := range 10 {
					payload := bytes.Repeat([]byte{byte(i + 1)}, 1024)
					if _, err := fast.Write(payload); err != nil {
						t.Fatal(err)
					}
					out := make([]byte, len(payload))
					if _, err := io.ReadFull(fast, out); err != nil || !bytes.Equal(out, payload) {
						t.Fatalf("unrelated application stalled/corrupt: %v", err)
					}
					s := m.Snapshot()
					peer := p.Snapshot()
					if s.Active != "route" || s.Generation != initial.Generation || peer.Generation != initial.Generation || s.Flows.Interrupted != initial.Flows.Interrupted || s.Flows.Rejected != initial.Flows.Rejected {
						t.Fatalf("slow reader fenced unrelated users: %+v / %+v", s, peer)
					}
					time.Sleep(time.Second)
				}
			})
		}
	}
}
