package peerauto

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
)

func TestMuxIsolationRequiresAcceptedProductionEnrollment(t *testing.T) {
	for _, mode := range []string{"accepted", "manifest-rejected", "locally-disabled"} {
		t.Run(mode, func(t *testing.T) {
			cfg := stateConfig()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			m, err := New(ctx, cfg, filepath.Join(t.TempDir(), "iran.json"))
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			peerCfg := stateConfig()
			peerCfg.Client = config.ClientConfig{RemoteAddr: cfg.Server.BindAddr, Token: cfg.Server.Token}
			peerCfg.Server = config.ServerConfig{}
			p, err := New(ctx, peerCfg, filepath.Join(t.TempDir(), "peer.json"))
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			if mode == "manifest-rejected" {
				p.manifest = "different-manifest"
			}
			if mode == "locally-disabled" {
				m.runtimeDisabled["noise"] = true
			}
			a, b := net.Pipe()
			local, err := multiplex(a, true, 2)
			if err != nil {
				t.Fatal(err)
			}
			remote, err := multiplex(b, false, 2)
			if err != nil {
				local.Close()
				t.Fatal(err)
			}
			defer local.Close()
			defer remote.Close()
			// Wrappers may not lose the underlying flow policy or bypass checks.
			wrapped := &rawPacketCarrier{carrierSession: &helperCarrier{carrierSession: boundCarrier{carrierSession: local, binding: "authenticated-Noise-NNpsk0"}}}
			if muxSession(wrapped) != muxSession(local) {
				t.Fatal("carrier wrapper lost its bounded mux")
			}
			done := make(chan struct{})
			go func() { defer close(done); m.runSession(cfg.Auto.Candidates[0], local) }()
			control, err := remote.Open(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer control.Close()
			_ = control.SetDeadline(time.Now().Add(2 * time.Second))
			if _, err = control.Write([]byte{0}); err != nil {
				t.Fatal(err)
			}
			_, _, authErr := p.authenticate(control, remote, "noise", "")
			if mode == "manifest-rejected" {
				// The listener rejects HELLO before answering; no complete
				// enrollment can authorize the post-enrollment budget here.
				if authErr == nil {
					t.Fatal("foreign manifest was accepted")
				}
			} else if authErr != nil {
				t.Fatal(authErr)
			}
			if mode == "accepted" {
				until := time.Now().Add(time.Second)
				for (!muxFlowStatus(local).Isolated || !muxFlowStatus(local).ControlPriority) && time.Now().Before(until) {
					time.Sleep(time.Millisecond)
				}
				if !muxFlowStatus(local).Isolated || !muxFlowStatus(local).ControlPriority {
					t.Fatal("accepted production session kept a shared deadlock budget")
				}
			} else {
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("rejected session did not stop")
				}
				if muxFlowStatus(local).Isolated || muxFlowStatus(local).ControlPriority {
					t.Fatal("unaccepted production session bypassed its receive budget")
				}
			}
			cancel()
			local.Close()
			remote.Close()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("enrollment test retained a session worker")
			}
		})
	}
}
