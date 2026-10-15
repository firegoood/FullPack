package manage

import (
	"github.com/backpack/backpack/config"
	"reflect"
	"strings"
	"testing"
)

func quickTestNew() NewTunnel {
	return NewTunnel{Name: "quick-auto", Role: "server", Transport: "auto", Ports: "2052,8880",
		QuickAuto: &QuickAutoSetup{IranIP: "192.0.2.1", KharejIP: "192.0.2.2"}}
}

func TestQuickAutoBuildsReviewedEndpointsAndTrustedPeerLink(t *testing.T) {
	n := quickTestNew()
	s, err := specFromNew(n)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Token) != 64 || !s.Auto.Enabled || s.Auto.ObserveOnly || len(s.Auto.Candidates) != 38 || s.Auto.Selection == nil || s.Auto.Selection.Target != 4 || s.Auto.Selection.RefillAt != 2 || s.Auto.ConnectionLimit == nil || s.Auto.ConnectionCapacity() != 0 || s.Auto.Cooldown != 300 || s.Auto.MinimumDwell != 600 || s.Auto.MaxSwitchesPerHour != 2 {
		t.Fatalf("quick setup lost opt-in/security/policy bounds: %#v", s.Auto)
	}
	seen := map[string]bool{}
	for _, p := range s.Auto.Candidates {
		seen[string(p.Transport)+":"+p.Mode] = true
	}
	for _, c := range config.AutoCarriers() {
		for _, mode := range []string{"reverse", "direct"} {
			if !seen[string(c.Value)+":"+mode] {
				t.Fatalf("registry transport/direction omitted: %s %s", c.Value, mode)
			}
		}
	}
	if s.Auto.Candidates[0].Endpoint != "192.0.2.1:24443" || s.Auto.Candidates[1].Endpoint != "192.0.2.2:24475" {
		t.Fatal("reviewed direction-specific port bases changed")
	}
	raw := pendingReverseLink(s, n.QuickAuto.IranIP, linkExtras{})
	if raw == "" {
		t.Fatal("quick setup produced no enrollment link")
	}
	peer, err := PeerNewTunnel(raw, "quick-peer", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if peer.Token != s.Token || !reflect.DeepEqual(peer.Auto.PeerEnrollment(), s.Auto.PeerEnrollment()) {
		t.Fatal("import changed credentials or endpoint authorization")
	}
	if strings.Contains(raw, "0.0.0.0") || peer.Auto.Candidates[0].LocalBind != "" {
		t.Fatal("local bind exported to foreign peer")
	}
	other, err := specFromNew(quickTestNew())
	if err != nil || other.Auto.TunnelID == s.Auto.TunnelID || other.Token == s.Token {
		t.Fatal("fresh quick setups reused enrollment identity/secret")
	}
}

func TestQuickAutoL3BuildsAllDatagramsAndKeepsItsStableTunEnrollment(t *testing.T) {
	n := quickTestNew()
	n.QuickAuto.Mode = "l3"
	n.Ports = ""
	s, err := specFromNew(n)
	if err != nil {
		t.Fatal(err)
	}
	if s.Auto.L3 == nil || s.Auto.Mode != "l3" || len(s.Auto.Candidates) != 12 {
		t.Fatal("quick L3 pool incomplete")
	}
	for _, p := range s.Auto.Candidates {
		c, _ := config.FindAutoCarrier(p.Transport)
		if !c.L3 {
			t.Fatal("IP routed over reliable stream")
		}
	}
	if s.Auto.L3.IranIP == s.Auto.L3.KharejIP || s.Auto.L3.Interface != "" {
		t.Fatal("invalid shared/local TUN enrollment")
	}
}

func TestQuickAutoRejectsMissingAddressesConflictingPortsAndMixedEnrollment(t *testing.T) {
	for _, kind := range []string{"missing-iran", "missing-kharej", "wildcard", "multicast", "port-overflow", "user-port-clash", "client", "other-transport", "existing-auto"} {
		t.Run(kind, func(t *testing.T) {
			n := quickTestNew()
			switch kind {
			case "missing-iran":
				n.QuickAuto.IranIP = ""
			case "missing-kharej":
				n.QuickAuto.KharejIP = ""
			case "wildcard":
				n.QuickAuto.KharejIP = "0.0.0.0"
			case "multicast":
				n.QuickAuto.IranIP = "224.0.0.1"
			case "port-overflow":
				n.QuickAuto.BasePort = 65534
			case "user-port-clash":
				n.QuickAuto.BasePort = 2052
			case "client":
				n.Role = "client"
			case "other-transport":
				n.Transport = "tcp"
			case "existing-auto":
				n.Auto = peerTestNew(true).Auto
			}
			if _, err := specFromNew(n); err == nil {
				t.Fatal("unsafe or ambiguous quick enrollment accepted")
			}
		})
	}
}
