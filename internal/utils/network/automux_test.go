package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/gorilla/websocket"
)

func TestAutoNoiseMuxNegotiatesOnlyWhenBothPeersOffer(t *testing.T) {
	for _, clientAuto := range []bool{false, true} {
		for _, serverAuto := range []bool{false, true} {
			t.Run(fmt.Sprintf("client=%t/server=%t", clientAuto, serverAuto), func(t *testing.T) {
				a, b := net.Pipe()
				t.Cleanup(func() { a.Close(); b.Close() })
				type result struct {
					c   net.Conn
					err error
				}
				peer := make(chan result, 1)
				go func() {
					fn := NoiseServerConn
					if serverAuto {
						fn = NoiseAutoServerConn
					}
					c, err := fn(b, "auto-mux-negotiation-secret", time.Second)
					peer <- result{c, err}
				}()
				fn := NoiseClientConn
				if clientAuto {
					fn = NoiseAutoClientConn
				}
				client, err := fn(a, "auto-mux-negotiation-secret", time.Second)
				server := <-peer
				if err != nil || server.err != nil {
					t.Fatalf("handshake: %v / %v", err, server.err)
				}
				want := 1
				if clientAuto && serverAuto {
					want = 2
				}
				for _, c := range []net.Conn{client, server.c} {
					n := c.(*noiseConn)
					if n.NegotiatedMuxVersion() != want || !n.pad {
						t.Fatalf("mux/padding disagreement: %+v", n)
					}
					_ = c.SetDeadline(time.Now().Add(time.Second))
				}
				done := make(chan error, 1)
				go func() { _, err := server.c.Write([]byte("still encrypted")); done <- err }()
				buf := make([]byte, 15)
				if _, err := io.ReadFull(client, buf); err != nil || string(buf) != "still encrypted" {
					t.Fatal("negotiation changed records", err)
				}
				if err := <-done; err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestAutoNoiseMuxDoesNotBypassTokenAuthentication(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	peer := make(chan error, 1)
	go func() { _, err := NoiseAutoServerConn(b, "right-secret", time.Second); b.Close(); peer <- err }()
	c, err := NoiseAutoClientConn(a, "wrong-secret", time.Second)
	if c != nil || err == nil || <-peer == nil {
		t.Fatal("mux capability bypassed Noise authentication")
	}
}

func TestAutoWSSMuxRequiresAuthenticatedMutualChoice(t *testing.T) {
	for _, tc := range []struct {
		name, choice, proof string
		offer, fail         bool
		want                int
	}{
		{"new", "2", "right", true, false, 2},
		{"old", "", "", true, false, 1},
		{"old-with-proof", "", "right", true, false, 1},
		{"manual", "", "right", false, false, 1},
		{"forged", "2", "wrong", true, true, 1},
		{"unproved", "2", "", true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const token = "auto-wss-mux-private-test-token"
			up := websocket.Upgrader{}
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if (r.Header.Get(AutoMuxHeader) == "2") != tc.offer {
					t.Error("manual/Auto offer changed")
				}
				proof, err := WSSServerProof(r.TLS, token)
				if err != nil || r.Header.Get("Authorization") != "Bearer "+proof {
					t.Error("client proof invalid")
					http.NotFound(w, r)
					return
				}
				h := http.Header{}
				if tc.proof == "right" {
					answer, _ := WSSServerAnswer(r.TLS, token)
					h.Set(WSSServerProofHeader, answer)
				}
				if tc.proof == "wrong" {
					h.Set(WSSServerProofHeader, strings.Repeat("0", 64))
				}
				if tc.choice != "" {
					h.Set(AutoMuxHeader, tc.choice)
				}
				c, err := up.Upgrade(w, r, h)
				if err == nil {
					c.Close()
				}
			}))
			srv.StartTLS()
			defer srv.Close()
			version := 1
			var choice *int
			if tc.offer {
				choice = &version
			}
			c, err := WebSocketDialer(context.Background(), nil, srv.Listener.Addr().String(), "", "/channel", 2*time.Second, time.Second, true, token, config.WSS, false, 1, 0, 0, 0, choice)
			if c != nil {
				c.Close()
			}
			if (err != nil) != tc.fail || version != tc.want {
				t.Fatalf("choice=%d error=%v", version, err)
			}
		})
	}
}
