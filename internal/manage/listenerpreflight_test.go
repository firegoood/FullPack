package manage

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/backpack/backpack/config"
)

func TestAutoListenerPreflightUsesLocalRoleAndHelperPorts(t *testing.T) {
	s := TunnelSpec{Role: "server", Auto: config.AutoConfig{Enabled: true, Candidates: []config.AutoCandidate{
		{ID: "reverse", Transport: config.TCP, Endpoint: "192.0.2.1:25000", LocalBind: "0.0.0.0:25000"},
		{ID: "direct", Transport: config.WSS, Mode: "direct", Endpoint: "192.0.2.2:25001"},
		{ID: "helper", Transport: "xhttp", Endpoint: "192.0.2.1:25002", Helper: &config.AutoHelperConfig{ReversePort: 25130}},
		{ID: "icmp", Transport: config.WATICMP},
		{ID: "raw", Transport: config.PCK, Endpoint: "192.0.2.1:25003"},
		{ID: "udp", Transport: config.QUIC, Endpoint: "192.0.2.1:25004"},
		{ID: "off", Transport: config.TCP, Endpoint: "192.0.2.1:25005", Disabled: true},
		{ID: "portless", Transport: config.SPOOF, Endpoint: "192.0.2.1:25006", Packet: &config.AutoPacketConfig{SpoofProfile: "icmp"}},
	}}}
	claims := carrierListeners(s)
	if len(claims) != 5 || claims[0].address != "0.0.0.0:25000" || claims[2].address != "127.0.0.1:25130" || claims[3].network != "tcp" || claims[4].network != "udp" {
		t.Fatalf("wrong local listener claims: %+v", claims)
	}
	s.Role = "client"
	claims = carrierListeners(s)
	if len(claims) != 1 || claims[0].address != "192.0.2.2:25001" {
		t.Fatalf("foreign listeners checked locally: %+v", claims)
	}
}

func TestCreateRefusesCarrierConflictBeforeAnyConfigurationWrite(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	n := peerTestNew(true)
	n.Name = "occupied-carrier-preflight"
	n.Auto.Candidates = []config.AutoCandidate{{ID: "held", Transport: config.TCP, Endpoint: l.Addr().String(), LocalBind: l.Addr().String()}}
	n.Ports = "127.0.0.1:62001=2052"
	_, _, err = CreateTunnel(n)
	if err == nil || !strings.Contains(err.Error(), "Auto held") || !strings.Contains(err.Error(), l.Addr().String()) {
		t.Fatalf("creation reached persistence or hid occupied carrier: %v", err)
	}
}

func TestStoppedDirectAndL3ClaimsKeepForeignPortsSeparate(t *testing.T) {
	for _, tc := range []struct {
		cfg     config.Config
		count   int
		network string
	}{
		{config.Config{Direct: config.DirectConfig{Role: "kharej", Addr: "0.0.0.0:25000"}}, 1, "tcp"},
		{config.Config{Direct: config.DirectConfig{Role: "iran", Addr: "192.0.2.2:25000", Ports: []string{"25001"}, AcceptUDP: true}}, 2, "tcp"},
		{config.Config{L3: config.L3Config{Mode: "listen", Addr: "0.0.0.0:25000", Carrier: "udp", Paths: 3}}, 3, "udp"},
		{config.Config{L3: config.L3Config{Mode: "listen", Addr: "0.0.0.0:25000", Carrier: "pck"}}, 1, "tcp"},
		{config.Config{L3: config.L3Config{Mode: "dial", Addr: "192.0.2.2:25000"}}, 0, ""},
	} {
		got := configuredDirectListeners(tc.cfg)
		if len(got) != tc.count || len(got) > 0 && got[0].network != tc.network {
			t.Fatalf("wrong stopped claims: %+v", got)
		}
		for _, c := range got {
			if c.address == "192.0.2.2:25000" {
				t.Fatal("foreign port reserved locally")
			}
		}
	}
}

func TestListenerPreflightRejectsOccupiedTCPUDPAndConfiguredReservations(t *testing.T) {
	for _, network := range []string{"tcp", "udp"} {
		t.Run(network, func(t *testing.T) {
			var address string
			if network == "tcp" {
				l, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer l.Close()
				address = l.Addr().String()
			} else {
				l, err := net.ListenPacket("udp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer l.Close()
				address = l.LocalAddr().String()
			}
			claim := listenerClaim{network, address, "test carrier"}
			if err := checkListenerClaims([]listenerClaim{claim}, nil, bindListenerProbe); err == nil || !strings.Contains(err.Error(), address) {
				t.Fatalf("occupied listener not diagnosed: %v", err)
			}
		})
	}
	claim := listenerClaim{"tcp", "0.0.0.0:25000", "new carrier"}
	reserved := listenerClaim{"tcp", "127.0.0.1:25000", "stopped raw tunnel"}
	called := false
	err := checkListenerClaims([]listenerClaim{claim}, []listenerClaim{reserved}, func(listenerClaim) error { called = true; return nil })
	if err == nil || called || !strings.Contains(err.Error(), "stopped raw tunnel") {
		t.Fatalf("configured reservation missed: %v", err)
	}
	if err := checkListenerClaims([]listenerClaim{claim, claim}, nil, func(listenerClaim) error { return nil }); err == nil {
		t.Fatal("duplicate new listeners accepted")
	}
	if listenersOverlap(claim, listenerClaim{"udp", ":25000", "other protocol"}) {
		t.Fatal("TCP and UDP falsely conflict")
	}
}

func TestQuickAutoSuggestionChecksWholeLocalPoolAndPrivateHelpers(t *testing.T) {
	for _, mode := range []string{"stream", "l3"} {
		var checked []listenerClaim
		p, err := suggestQuickAutoPorts(mode, []listenerClaim{{"tcp", ":24443", "existing tunnel"}, {"udp", ":24443", "existing tunnel"}}, func(c listenerClaim) error { checked = append(checked, c); return nil })
		if err != nil || p.BasePort != 24699 || p.DirectPort != 24731 || !p.PeerCheckRequired {
			t.Fatalf("bad suggestion: %+v %v", p, err)
		}
		for _, c := range checked {
			_, port, err := net.SplitHostPort(c.address)
			if err != nil {
				t.Fatal(err)
			}
			number, _ := strconv.Atoi(port)
			if number >= p.DirectPort && number < p.DirectPort+18 {
				t.Fatal("remote Direct port was probed on Iran")
			}
		}
	}
	p, err := suggestQuickAutoPorts("stream", nil, func(c listenerClaim) error {
		if c.address == "127.0.0.1:24586" {
			return errors.New("helper port owned")
		}
		return nil
	})
	if err != nil || p.BasePort != 24699 {
		t.Fatalf("private helper collision missed: %+v %v", p, err)
	}
	count := 0
	if _, err := suggestQuickAutoPorts("stream", nil, func(listenerClaim) error { count++; return errors.New("busy") }); err == nil || count != 128 {
		t.Fatalf("search not bounded: %d %v", count, err)
	}
}
