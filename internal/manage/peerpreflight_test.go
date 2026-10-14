package manage

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
)

// An occupied listener may belong to an unrelated tunnel/proxy. Refuse both
// Auto initiation directions and native Wat before any host mutation; do not
// stop that listener or treat the foreign backend as a local Iran listener.
func TestPeerSetupRefusesOccupiedForwardsBeforeHostChanges(t *testing.T) {
	first, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	_, port, _ := net.SplitHostPort(first.Addr().String())
	_, other, _ := net.SplitHostPort(second.Addr().String())
	control, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, controlPort, _ := net.SplitHostPort(control.Addr().String())
	control.Close()
	for _, mode := range []string{"reverse", "direct", "waticmp"} {
		t.Run(mode, func(t *testing.T) {
			n := peerTestNew(mode != "waticmp")
			n.Ports = fmt.Sprintf("127.0.0.1:%s=127.0.0.1:2052,127.0.0.1:%s=127.0.0.1:8880", port, other)
			if n.Auto != nil {
				n.Auto.Candidates = []config.AutoCandidate{{ID: "tcp", Transport: config.TCP, Endpoint: net.JoinHostPort("192.0.2.1", controlPort), Mode: mode}}
			}
			s, err := specFromPeerNew(n)
			if err != nil {
				t.Fatal(err)
			}
			restore := tui.SetInput(strings.NewReader("\n"))
			defer restore()
			output, err := os.CreateTemp(t.TempDir(), "setup-output")
			if err != nil {
				t.Fatal(err)
			}
			defer output.Close()
			previous := os.Stdout
			os.Stdout = output
			defer func() { os.Stdout = previous }()
			created := finishSetup(s)
			os.Stdout = previous
			body, err := os.ReadFile(output.Name())
			if err != nil {
				t.Fatal(err)
			}
			if created || !strings.Contains(string(body), "Nothing Written") || strings.Contains(string(body), "Applying Network Optimizations") {
				t.Fatalf("occupied listener reached host mutation: created=%v output=%s", created, body)
			}
			for _, p := range []string{port, other} {
				if !strings.Contains(string(body), p) {
					t.Fatalf("occupied port %s missing from diagnostic: %s", p, body)
				}
			}
			s.Role = "client"
			if err := refuseBusyForwards(s); err != nil {
				t.Fatalf("foreign backend listener was incorrectly refused: %v", err)
			}
		})
	}
}

func TestPeerSetupRejectsMissingAddressesBeforeCreation(t *testing.T) {
	for _, field := range []string{"iran-wat", "kharej-wat", "reverse-host", "direct-host"} {
		t.Run(field, func(t *testing.T) {
			n := peerTestNew(true)
			switch field {
			case "iran-wat":
				n.Auto.Candidates[0].IranIP = ""
			case "kharej-wat":
				n.Auto.Candidates[0].KharejIP = ""
			case "reverse-host", "direct-host":
				n.Auto.Candidates = []config.AutoCandidate{{ID: "tcp", Transport: config.TCP, Endpoint: ":24443"}}
				if field == "direct-host" {
					n.Auto.Candidates[0].Mode = "direct"
				}
			}
			if _, err := specFromPeerNew(n); err == nil {
				t.Fatal("missing peer address accepted before creation")
			}
		})
	}
}
