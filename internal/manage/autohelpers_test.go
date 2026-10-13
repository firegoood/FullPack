package manage

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
)

func TestAutoHTTPSRolesLinksAndRedactedEdits(t *testing.T) {
	usePanelRealityProbe(t, func(context.Context, string, string, string) error { return nil })
	for _, tr := range []string{"naive", "xhttp", "reality"} {
		for _, mode := range []string{"reverse", "direct"} {
			t.Run(tr+"/"+mode, func(t *testing.T) {
				binary, cert, key := managedWizardFixture(t)
				h := &config.AutoHelperConfig{Binary: binary, ReversePort: 3086, Username: "auto-user", Password: "auto-password", ServerName: "localhost", UUID: "00112233-4455-4677-8899-aabbccddeeff"}
				if tr == "reality" {
					h.Username, h.Password = "", ""
					private, public, err := managedRealityKey("")
					if err != nil {
						t.Fatal(err)
					}
					h.PrivateKey, h.PublicKey, h.ShortID, h.Target = private, public, "0123456789abcdef", "cover.example:443"
				} else {
					if tr == "xhttp" {
						h.Username, h.Password = "", ""
					} else {
						h.UUID = ""
					}
					raw, err := os.ReadFile(cert)
					if err != nil {
						t.Fatal(err)
					}
					h.CAPEM = string(raw)
					if tr == "xhttp" {
						h.Path = "/auto-private"
					}
					if mode == "reverse" {
						h.Certificate, h.Key = cert, key
					}
				}
				n := peerTestNew(true)
				n.Auto.Candidates = []config.AutoCandidate{{ID: "https", Transport: config.TransportType(tr), Mode: mode, Endpoint: "127.0.0.1:24443", Helper: h}}
				s, err := specFromPeerNew(n)
				if err != nil {
					t.Fatal(err)
				}
				var cfg config.Config
				if _, err = toml.Decode(s.Render(), &cfg); err != nil {
					t.Fatal(err)
				}
				raw, err := sharePeerLink(s.Name, "127.0.0.1", cfg)
				if err != nil {
					t.Fatal(err)
				}
				link, err := DecodeShareLink(raw)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(link.AutoEnrollment, binary) || strings.Contains(link.AutoEnrollment, key) || h.PrivateKey != "" && strings.Contains(link.AutoEnrollment, h.PrivateKey) {
					t.Fatal("local/private helper material exported")
				}
				var enrolled config.AutoConfig
				if err = json.Unmarshal([]byte(link.AutoEnrollment), &enrolled); err != nil {
					t.Fatal(err)
				}
				if enrolled.Candidates[0].Helper.Password != h.Password || enrolled.Candidates[0].Mode != mode {
					t.Fatal("paired account/direction changed")
				}
				mirrored := reverseClientFromLink(link, "127.0.0.1")
				if mirrored.Role != "client" || mirrored.Auto.Candidates[0].Helper.PublicKey != enrolled.Candidates[0].Helper.PublicKey || mirrored.Transport != tr {
					t.Fatal("generic link mirror tried to prepare local helper identity before receiver overrides")
				}
				locals := []AutoPeerLocal{{ID: "https", Binary: binary}}
				if mode == "direct" {
					if tr == "reality" {
						locals[0].PrivateKey = h.PrivateKey
					} else {
						locals[0].Certificate, locals[0].Key = cert, key
					}
				}
				peer, err := PeerNewTunnel(raw, "auto-helper-peer", "auto", locals...)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = specFromPeerNew(peer); err != nil {
					t.Fatal(err)
				}
				settings := tunnelSettingsFrom(s)
				public := settings.Auto.Candidates[0].Helper
				if public.Password != "" || public.PrivateKey != "" {
					t.Fatal("settings returned helper secrets")
				}
				settings.Auto.ObserveOnly = !s.Auto.ObserveOnly
				edited, changed, err := editedPeerSpec(s, TunnelEdit{Auto: &settings.Auto})
				if err != nil || !changed {
					t.Fatalf("redacted edit failed: %v", err)
				}
				if edited.Auto.Candidates[0].Helper.Password != s.Auto.Candidates[0].Helper.Password || edited.Auto.Candidates[0].Helper.PrivateKey != s.Auto.Candidates[0].Helper.PrivateKey {
					t.Fatal("redacted edit erased secret")
				}
			})
		}
	}
}

func TestAutoNodeRebuildPreservesOnlyCompatibleLocalIdentity(t *testing.T) {
	a := peerTestNew(true).Auto
	a.Mode = "l3"
	a.Ports = nil
	a.L3 = &config.AutoL3Config{IranIP: "10.44.0.1/30", KharejIP: "10.44.0.2/30", MTU: 1400, Interface: "fp-local"}
	a.MaxMemoryMB = 384
	a.Candidates = []config.AutoCandidate{{ID: "tls", Transport: "reality", Mode: "direct", Endpoint: "peer.example:443", LocalBind: "0.0.0.0:443", Helper: &config.AutoHelperConfig{Binary: "/custom/xray", ReversePort: 3088, UUID: "paired-account", PublicKey: "paired-public", PrivateKey: "local-private"}}, {ID: "pck", Transport: config.PCK, Endpoint: "192.0.2.1:8443", Packet: &config.AutoPacketConfig{Interface: "eth1"}}}
	current := TunnelSpec{Role: "client", Token: strings.Repeat("e", 64), Auto: *a}
	enrolled := a.PeerEnrollment()
	n := NewTunnel{Role: current.Role, Token: current.Token, Auto: &enrolled}
	if err := carryAutoNodeLocal(&n, current); err != nil {
		t.Fatal(err)
	}
	if n.Auto.Candidates[0].Helper.PrivateKey != "local-private" || n.Auto.Candidates[0].Helper.Binary != "/custom/xray" || n.Auto.Candidates[0].LocalBind != "0.0.0.0:443" || n.Auto.Candidates[1].Packet.Interface != "eth1" || n.Auto.L3.Interface != "fp-local" || n.Auto.MaxMemoryMB != 384 {
		t.Fatal("local identity lost on enrolled rebuild")
	}
	if !reflect.DeepEqual(n.Auto.PeerEnrollment(), a.PeerEnrollment()) || enrolled.Candidates[0].Helper.PrivateKey != "" || enrolled.L3.Interface != "" {
		t.Fatal("rebuild changed enrollment or aliased its input")
	}
	for _, change := range []func(*NewTunnel){func(n *NewTunnel) { n.Token = "foreign" }, func(n *NewTunnel) { n.Role = "server" }, func(n *NewTunnel) { n.Auto.Candidates[0].Endpoint = "foreign.example:443" }, func(n *NewTunnel) { n.Auto.Candidates[0].Helper.PublicKey = "foreign" }} {
		v := a.PeerEnrollment()
		n := NewTunnel{Role: current.Role, Token: current.Token, Auto: &v}
		change(&n)
		if carryAutoNodeLocal(&n, current) == nil {
			t.Fatal("incompatible rebuild inherited local private identity")
		}
	}
}

func TestAutoL3EnrollmentHasNoLocalInterfaceOrRequiredPorts(t *testing.T) {
	n := peerTestNew(true)
	n.Ports = ""
	n.Auto.Mode = "l3"
	n.Auto.L3 = &config.AutoL3Config{IranIP: "10.44.0.1/30", KharejIP: "10.44.0.2/30", Interface: "fp-iran", MTU: 1400}
	n.Auto.Candidates = []config.AutoCandidate{{ID: "udp", Transport: config.UDP, Mode: "direct", Endpoint: "192.0.2.2:24443"}}
	s, err := specFromPeerNew(n)
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.Config
	if _, err = toml.Decode(s.Render(), &cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := sharePeerLink(s.Name, "192.0.2.1", cfg)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := PeerNewTunnel(raw, "l3-peer", "auto", AutoPeerLocal{Interface: "fp-kharej"})
	if err != nil {
		t.Fatal(err)
	}
	if peer.Auto.L3.Interface != "fp-kharej" || peer.Auto.Mode != "l3" || len(peer.Auto.Ports) != 0 {
		t.Fatal("L3 enrollment was lost")
	}
	if _, err := PeerNewTunnel(raw, "l3-peer", "auto", AutoPeerLocal{ID: "foreign", LocalBind: "0.0.0.0:24443"}); err == nil {
		t.Fatal("foreign local override accepted")
	}
}
