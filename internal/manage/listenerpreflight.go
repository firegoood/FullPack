package manage

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/portmap"
)

type listenerClaim struct{ network, address, label string }

// Only sockets owned by this side belong here. A foreign backend/dial address
// is never tested as a local listener. Raw TCP carriers reserve a bound socket
// even though ss -l does not show a listening socket for them.
func carrierListeners(s TunnelSpec) []listenerClaim {
	var out []listenerClaim
	if s.Auto.Enabled {
		for _, p := range s.Auto.Candidates {
			if p.Disabled || (s.Role == "server") != p.ListenerIsIran() {
				continue
			}
			c, ok := config.FindAutoCarrier(p.Transport)
			if !ok || c.Socket == "icmp" {
				continue
			}
			address := p.LocalBind
			if address == "" {
				address = p.Endpoint
			}
			network := c.Socket
			if network == "raw" {
				profile := ""
				if p.Packet != nil {
					profile = p.Packet.SpoofProfile
				}
				switch profile {
				case "", "udp":
					network = "udp"
				case "tcp":
					network = "tcp"
				default:
					continue // ICMP identifiers / portless raw protocols are not sockets
				}
			}
			out = append(out, listenerClaim{network, address, "Auto " + p.ID})
			if p.Helper != nil && p.Helper.ReversePort > 0 {
				out = append(out, listenerClaim{"tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(p.Helper.ReversePort)), "Auto " + p.ID + " private helper"})
			}
		}
	} else if s.Role == "server" && !s.WatICMP.Enabled && s.Transport != "xdi" {
		tr := selectedTransport(s)
		if managedTransport(tr) {
			out = append(out, listenerClaim{"tcp", managedEndpoint(s), "HTTPS public listener"})
		}
		_, port, err := net.SplitHostPort(s.BindAddr)
		if err == nil && port != "0" {
			network := "tcp"
			if isDatagram(s.Transport) && s.Transport != "pck" {
				network = "udp"
			}
			out = append(out, listenerClaim{network, s.BindAddr, "tunnel listener"})
		}
	}
	return out
}

func bindListenerProbe(c listenerClaim) error {
	if c.network == "udp" {
		l, err := net.ListenPacket("udp", c.address)
		if err != nil {
			return err
		}
		return l.Close()
	}
	l, err := net.Listen("tcp", c.address)
	if err != nil {
		return err
	}
	return l.Close()
}

func listenersOverlap(a, b listenerClaim) bool {
	if a.network != b.network {
		return false
	}
	ah, ap, ae := net.SplitHostPort(a.address)
	bh, bp, be := net.SplitHostPort(b.address)
	wild := func(h string) bool { return h == "" || h == "0.0.0.0" || h == "::" }
	return ae == nil && be == nil && ap == bp && (wild(ah) || wild(bh) || ah == bh)
}

func configuredListeners(except string) []listenerClaim {
	var out []listenerClaim
	for _, t := range List() {
		if t.Name == except {
			continue
		}
		s, err := LoadSpec(t.Name)
		if err != nil {
			if cfg, loadErr := LoadTunnelConfig(t.Name); loadErr == nil {
				claims := configuredDirectListeners(cfg)
				for i := range claims {
					claims[i].label = "tunnel " + t.Name + " / " + claims[i].label
				}
				out = append(out, claims...)
			}
			continue
		}
		claims := carrierListeners(s)
		if s.Role == "server" {
			if mappings, err := portmap.Expand(s.Ports, "127.0.0.1"); err == nil {
				for _, m := range mappings {
					claims = append(claims, listenerClaim{"tcp", m.Listen, "forwarded listener"})
					if s.AcceptUDP {
						claims = append(claims, listenerClaim{"udp", m.Listen, "forwarded listener"})
					}
				}
			}
		}
		for i := range claims {
			claims[i].label = "tunnel " + t.Name + " / " + claims[i].label
		}
		out = append(out, claims...)
	}
	return out
}

func configuredDirectListeners(cfg config.Config) []listenerClaim {
	var out []listenerClaim
	var ports []string
	acceptUDP := false
	if cfg.Direct.Enabled() {
		if cfg.Direct.ResolvedRole() == "origin" {
			out = append(out, listenerClaim{"tcp", cfg.Direct.Addr, "Direct listener"})
		} else {
			ports, acceptUDP = cfg.Direct.Ports, cfg.Direct.AcceptUDP
		}
	} else if cfg.L3.Enabled() {
		if strings.EqualFold(cfg.L3.Mode, "listen") {
			tr := config.TransportType(cfg.L3.Carrier)
			if tr == "" {
				tr = config.UDP
			}
			p := config.AutoCandidate{ID: "L3", Transport: tr, Endpoint: cfg.L3.Addr, Packet: &config.AutoPacketConfig{SpoofProfile: cfg.L3.SpoofProfile}}
			out = carrierListeners(TunnelSpec{Role: "server", Auto: config.AutoConfig{Enabled: true, Candidates: []config.AutoCandidate{p}}})
			if tr == config.UDP && len(out) == 1 {
				host, text, err := net.SplitHostPort(cfg.L3.Addr)
				base, _ := strconv.Atoi(text)
				if err == nil && base > 0 {
					for i := 1; i < min(max(1, cfg.L3.Paths), 65536-base); i++ {
						out = append(out, listenerClaim{"udp", net.JoinHostPort(host, strconv.Itoa(base+i)), "L3 UDP path"})
					}
				}
			}
		} else {
			ports, acceptUDP = cfg.L3.Ports, cfg.L3.AcceptUDP
		}
	}
	if mappings, err := portmap.Expand(ports, "127.0.0.1"); err == nil {
		for _, m := range mappings {
			out = append(out, listenerClaim{"tcp", m.Listen, "forwarded listener"})
			if acceptUDP {
				out = append(out, listenerClaim{"udp", m.Listen, "forwarded listener"})
			}
		}
	}
	return out
}

func checkListenerClaims(claims, existing []listenerClaim, probe func(listenerClaim) error) error {
	for i, c := range claims {
		for _, pool := range [][]listenerClaim{existing, claims[:i]} {
			for _, other := range pool {
				if listenersOverlap(c, other) {
					return fmt.Errorf("%s: %s %s is already reserved by %s; choose another local port", c.label, c.network, c.address, other.label)
				}
			}
		}
		if err := probe(c); err != nil {
			return fmt.Errorf("%s: cannot bind local %s %s: %w; choose a free local port (no service was stopped)", c.label, c.network, c.address, err)
		}
	}
	return nil
}

func refuseBusyCarrierListeners(s TunnelSpec) error {
	claims := carrierListeners(s)
	if s.Role == "server" {
		mappings, err := portmap.Expand(s.Ports, "127.0.0.1")
		if err != nil {
			return err
		}
		for _, m := range mappings {
			claims = append(claims, listenerClaim{"tcp", m.Listen, "forwarded listener"})
			if s.AcceptUDP {
				claims = append(claims, listenerClaim{"udp", m.Listen, "forwarded listener"})
			}
		}
	}
	err := checkListenerClaims(claims, configuredListeners(s.Name), bindListenerProbe)
	if err != nil && s.Auto.Enabled && s.Role == "client" {
		return fmt.Errorf("%w; change the approved endpoint on Iran and import a fresh peer setup link", err)
	}
	return err
}

type QuickAutoPorts struct {
	BasePort          int  `json:"basePort"`
	DirectPort        int  `json:"directPort"`
	PeerCheckRequired bool `json:"peerCheckRequired"`
}

// Suggestions inspect this host only. Kharej rechecks its own approved Direct
// listeners on import; we never scan a supplied remote IP or rewrite that link.
func SuggestQuickAutoPorts(mode, forwards string) (QuickAutoPorts, error) {
	if mode != "" && mode != "stream" && mode != "l3" {
		return QuickAutoPorts{}, fmt.Errorf("invalid Auto traffic mode")
	}
	existing := configuredListeners("")
	if mode != "l3" {
		mappings, err := portmap.Expand(parsePorts(forwards), "127.0.0.1")
		if err != nil {
			return QuickAutoPorts{}, err
		}
		for _, m := range mappings {
			existing = append(existing, listenerClaim{"tcp", m.Listen, "requested forwarded listener"}, listenerClaim{"udp", m.Listen, "requested forwarded listener"})
		}
	}
	return suggestQuickAutoPorts(mode, existing, bindListenerProbe)
}

func suggestQuickAutoPorts(mode string, existing []listenerClaim, probe func(listenerClaim) error) (QuickAutoPorts, error) {
	choices := QuickAutoCarriers()
	if mode == "l3" {
		choices = QuickAutoL3Carriers()
	}
	for i := range 128 {
		base := 24443 + 256*i
		var claims []listenerClaim
		for _, c := range choices {
			if c.Mode != "reverse" {
				continue
			}
			r, _ := config.FindAutoCarrier(c.Transport)
			if r.Socket == "icmp" {
				continue
			}
			network := r.Socket
			if network == "raw" {
				network = "udp"
			}
			claims = append(claims, listenerClaim{network, net.JoinHostPort("0.0.0.0", strconv.Itoa(base+c.PortOffset)), c.ID})
			if r.Helper {
				claims = append(claims, listenerClaim{"tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(base+c.PortOffset+128)), c.ID + " private helper"})
			}
		}
		if checkListenerClaims(claims, existing, probe) == nil {
			return QuickAutoPorts{base, base + 32, true}, nil
		}
	}
	return QuickAutoPorts{}, fmt.Errorf("no free Auto listener range found; choose and review explicit ports")
}
