package manage

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

// An enrolled node rebuild carries its own local identities from its private
// config, never from a redacted settings response sent back across the network.
func carryAutoNodeLocal(n *NewTunnel, current TunnelSpec) error {
	if n.Auto == nil || current.Auto.TunnelID == "" {
		return nil
	}
	old, incoming := peerauto.Defaults(current.Auto).PeerEnrollment(), peerauto.Defaults(*n.Auto).PeerEnrollment()
	old.ObserveOnly, incoming.ObserveOnly = false, false
	if n.Role != current.Role || strings.TrimSpace(n.Token) != current.Token || !reflect.DeepEqual(old, incoming) {
		return fmt.Errorf("shared Auto enrollment changed: create a new pair instead of replacing this node's working enrollment")
	}
	a := *n.Auto
	a.Candidates = append([]config.AutoCandidate(nil), a.Candidates...)
	a.ObserveOnly, a.MaxMemoryMB = current.Auto.ObserveOnly, current.Auto.MaxMemoryMB
	if a.L3 != nil {
		v := *a.L3
		if v.Interface == "" && current.Auto.L3 != nil {
			v.Interface = current.Auto.L3.Interface
		}
		a.L3 = &v
	}
	for i := range a.Candidates {
		p, prior := &a.Candidates[i], current.Auto.Candidates[i]
		if p.LocalBind == "" {
			p.LocalBind = prior.LocalBind
		}
		if prior.Packet != nil {
			v := config.AutoPacketConfig{}
			if p.Packet != nil {
				v = *p.Packet
			}
			if v.Interface == "" {
				v.Interface = prior.Packet.Interface
			}
			p.Packet = &v
		}
		if p.Helper != nil && prior.Helper != nil {
			v := *p.Helper
			for _, pair := range []struct {
				dst   *string
				value string
			}{
				{&v.Binary, prior.Helper.Binary}, {&v.CAFile, prior.Helper.CAFile},
				{&v.Certificate, prior.Helper.Certificate}, {&v.Key, prior.Helper.Key}, {&v.PrivateKey, prior.Helper.PrivateKey},
			} {
				if *pair.dst == "" {
					*pair.dst = pair.value
				}
			}
			p.Helper = &v
		}
	}
	n.Auto = &a
	return nil
}

// Each wrapper reuses the existing CLI/web HTTPS setup, identity generation,
// certificate validation and authenticated REALITY cover check. The wrapper
// role follows the candidate's listener, independently from Iran authority.
func prepareAutoHelpers(s *TunnelSpec) error {
	for i := range s.Auto.Candidates {
		p := &s.Auto.Candidates[i]
		if p.Helper == nil || p.Disabled {
			continue
		}
		h := *p.Helper
		p.Helper = &h
		role := "client"
		if (s.Role == "server") == p.ListenerIsIran() {
			role = "server"
		}
		m := ManagedHTTPS{InternalPort: strconv.Itoa(h.ReversePort), Binary: h.Binary, Username: h.Username, Password: h.Password, ServerName: h.ServerName, Certificate: h.Certificate, Key: h.Key, CAFile: h.CAFile, UUID: h.UUID, Path: h.Path, Host: h.Host, Target: h.Target, PrivateKey: h.PrivateKey, PublicKey: h.PublicKey, ShortID: h.ShortID}
		// Embedded CA stays in the enrolled config; the runtime helper writes
		// it into a private temporary directory rather than a persistent path.
		tmp := TunnelSpec{Name: s.Name + "-" + p.ID, Role: role, Transport: "tcp", Token: s.Token}
		public := p.Endpoint
		if role == "server" && p.LocalBind != "" {
			public = p.LocalBind
		}
		if err := applyManagedHTTPS(&tmp, string(p.Transport), public, m); err != nil {
			return fmt.Errorf("candidate %s: %w", p.ID, err)
		}
		if p.Transport == "naive" {
			if role == "server" {
				h.Binary, h.Username, h.Password, h.Certificate, h.Key = tmp.NaiveServer.Binary, tmp.NaiveServer.Username, tmp.NaiveServer.Password, tmp.NaiveServer.Certificate, tmp.NaiveServer.Key
			} else {
				h.Binary = tmp.NaiveClient.Binary
			}
		} else if role == "server" {
			v := tmp.XrayServer
			h.Binary, h.UUID, h.Path, h.Certificate, h.Key, h.PrivateKey, h.ShortID = v.Binary, v.UUID, v.Path, v.Certificate, v.Key, v.PrivateKey, v.ShortID
			if p.Transport == "reality" {
				_, pub, err := managedRealityKey(h.PrivateKey)
				if err != nil {
					return err
				}
				if h.PublicKey != "" && h.PublicKey != pub {
					return fmt.Errorf("candidate %s REALITY private key does not match enrolled public key", p.ID)
				}
				h.PublicKey = pub
			}
		} else {
			h.Binary = tmp.XrayClient.Binary
		}
		if role == "client" && p.Transport != "reality" && h.CAPEM == "" && h.CAFile != "" {
			raw, err := os.ReadFile(h.CAFile)
			if err != nil {
				return err
			}
			h.CAPEM = string(raw)
		}
		if role == "server" && p.Transport != "reality" {
			raw, err := os.ReadFile(h.Certificate)
			if err != nil {
				return err
			}
			if h.CAPEM == "" {
				h.CAPEM = string(raw)
			}
			host, _, _ := net.SplitHostPort(p.Endpoint)
			if err := validateNaiveLinkHost(h.Certificate, host); p.Transport == "naive" && err != nil {
				return err
			}
		}
	}
	return nil
}

// AutoPeerLocal contains only destination-local settings. It cannot change a
// candidate endpoint, account, generation policy or paired public identity.
type AutoPeerLocal struct {
	ID          string `json:"id"`
	LocalBind   string `json:"localBind,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Binary      string `json:"binary,omitempty"`
	CAFile      string `json:"caFile,omitempty"`
	Certificate string `json:"certificate,omitempty"`
	Key         string `json:"key,omitempty"`
	PrivateKey  string `json:"privateKey,omitempty"`
}

func applyAutoPeerLocal(a *config.AutoConfig, locals []AutoPeerLocal) error {
	seen := map[string]bool{}
	for _, v := range locals {
		if v.ID == "" {
			if a.L3 == nil || v.LocalBind != "" || v.Binary != "" || v.CAFile != "" || v.Certificate != "" || v.Key != "" || v.PrivateKey != "" || seen[""] {
				return fmt.Errorf("invalid local TUN override")
			}
			local := *a.L3
			local.Interface = v.Interface
			a.L3 = &local
			seen[""] = true
			continue
		}
		found := false
		if seen[v.ID] {
			return fmt.Errorf("duplicate local candidate override")
		}
		seen[v.ID] = true
		for i := range a.Candidates {
			p := &a.Candidates[i]
			if p.ID != v.ID {
				continue
			}
			found = true
			if v.LocalBind != "" && p.ListenerIsIran() {
				return fmt.Errorf("dialing peer cannot override a listener bind")
			}
			p.LocalBind = v.LocalBind
			if v.Interface != "" {
				if p.Packet == nil {
					if p.Transport != config.PCK && p.Transport != "sni" {
						return fmt.Errorf("candidate has no raw packet settings")
					}
					p.Packet = &config.AutoPacketConfig{}
				}
				local := *p.Packet
				local.Interface = v.Interface
				p.Packet = &local
			}
			if p.Helper != nil {
				h := *p.Helper
				h.Binary, h.CAFile, h.Certificate, h.Key, h.PrivateKey = v.Binary, v.CAFile, v.Certificate, v.Key, v.PrivateKey
				p.Helper = &h
			} else if v.Binary != "" || v.CAFile != "" || v.Certificate != "" || v.Key != "" || v.PrivateKey != "" {
				return fmt.Errorf("local helper fields require an enrolled helper")
			}
		}
		if !found {
			return fmt.Errorf("unknown local candidate ID")
		}
	}
	return nil
}

type PeerLocalRequirement struct {
	ID          string `json:"id"`
	Transport   string `json:"transport"`
	Helper      bool   `json:"helper"`
	Raw         bool   `json:"raw"`
	Listener    bool   `json:"listener"`
	DefaultBind string `json:"defaultBind,omitempty"`
}

func PeerLocalRequirements(raw, transport string) (any, error) {
	l, err := DecodeShareLink(raw)
	if err != nil {
		return nil, err
	}
	if l.V != 3 || l.From != "iran" || l.Tr != transport || l.Kind != "reverse" {
		return nil, fmt.Errorf("use the matching trusted Iran enrollment")
	}
	f := MirrorForPeer(l)
	out := struct {
		L3         bool                   `json:"l3"`
		Candidates []PeerLocalRequirement `json:"candidates"`
	}{Candidates: []PeerLocalRequirement{}}
	if f.Auto != nil {
		out.L3 = f.Auto.L3 != nil
		for _, p := range f.Auto.Candidates {
			v, _ := config.FindAutoCarrier(p.Transport)
			r := PeerLocalRequirement{ID: p.ID, Transport: string(p.Transport), Helper: v.Helper, Raw: p.Transport == config.PCK || p.Transport == config.SPOOF || p.Transport == "sni", Listener: !p.ListenerIsIran()}
			if r.Listener {
				r.DefaultBind = net.JoinHostPort("0.0.0.0", addrPort(p.Endpoint))
			}
			out.Candidates = append(out.Candidates, r)
		}
	}
	return out, nil
}
