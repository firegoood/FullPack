// Package peerauto owns an opted-in enrolled tunnel's listeners, carrier
// sessions and switching decisions. Manual engines and legacy chain stay separate.
package peerauto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/snispoof"
	"github.com/backpack/backpack/internal/tunnel/portmap"
	"github.com/backpack/backpack/internal/tunnel/waticmp"
	"github.com/backpack/backpack/internal/utils/network"
)

var identity = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,64}$`)
var candidateID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,16}$`)
var localInterface = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,15}$`)

func Defaults(a config.AutoConfig) config.AutoConfig {
	if a.L3 != nil {
		local := *a.L3
		if local.MTU == 0 {
			local.MTU = 1400
		}
		a.L3 = &local
	}
	if a.ProbeInterval == 0 {
		a.ProbeInterval = 600
	}
	if a.Cooldown == 0 {
		a.Cooldown = 300
	}
	if a.MinimumDwell == 0 {
		a.MinimumDwell = 600
	}
	if a.MaxSwitchesPerHour == 0 {
		a.MaxSwitchesPerHour = 2
	}
	if a.FailureThreshold == 0 {
		a.FailureThreshold = 3
	}
	if a.MaxConnections == 0 {
		a.MaxConnections = 64
	}
	if a.MaxMemoryMB == 0 {
		a.MaxMemoryMB = 256
	}
	return a
}

func Validate(c *config.Config) error {
	return validate(c, true)
}

// ValidateEnrollment checks shared policy without consuming receiver-local
// helper keys or paths. Runtime startup always uses the stricter Validate.
func ValidateEnrollment(c *config.Config) error { return validate(c, false) }
func validate(c *config.Config, local bool) error {
	if !c.Auto.Enabled {
		return nil
	}
	if runtime.GOOS != "linux" || os.Getenv("BACKPACK_DISABLE_AUTO") == "1" {
		return fmt.Errorf("peer Auto is disabled on this host")
	}
	a := Defaults(c.Auto)
	if a.Mode != "" && a.Mode != "stream" && a.Mode != "l3" {
		return fmt.Errorf("Auto mode must be stream or l3")
	}
	if (a.Mode == "l3") != (a.L3 != nil) {
		return fmt.Errorf("Auto l3 mode requires its enrolled interface settings")
	}
	if a.AllowPlaintextFallback {
		return fmt.Errorf("plaintext Auto adapters are not verified in this version; allow_plaintext_fallback must remain false")
	}
	if a.MaxMemoryMB < 64 || a.MaxMemoryMB > 4096 {
		return fmt.Errorf("Auto memory pressure guard must be 64-4096 MB")
	}
	if c.L3.Enabled() || c.Direct.Enabled() {
		return fmt.Errorf("Auto owns its enrolled mode; remove legacy global Direct/L3 settings")
	}
	if (c.Server.BindAddr != "") == (c.Client.RemoteAddr != "") {
		return fmt.Errorf("Auto needs exactly one Iran/server or Kharej/client role")
	}
	if !identity.MatchString(a.TunnelID) || !identity.MatchString(a.IranID) || !identity.MatchString(a.KharejID) || a.IranID == a.KharejID {
		return fmt.Errorf("Auto requires distinct stable enrolled Iran/Kharej identities and a tunnel ID (16-64 safe characters)")
	}
	token := c.Client.Token
	if c.Server.BindAddr != "" {
		token = c.Server.Token
	}
	if len(token) < 32 {
		return fmt.Errorf("Auto enrollment secret must have at least 32 bytes")
	}
	if a.L3 != nil {
		if err := validateAutoL3(a, token); err != nil {
			return err
		}
	}
	if len(c.Server.FallbackTransports)+len(c.Client.FallbackTransports)+len(c.Client.FallbackAddrs) != 0 || c.Client.HealthFailover || c.Client.LoadBalance {
		return fmt.Errorf("Auto is the sole switching owner; remove legacy carrier/address fallback for this opted-in tunnel")
	}
	if c.Server.Naive.Enabled() || c.Client.Naive.Enabled() || c.Server.Xray.Enabled() || c.Client.Xray.Enabled() || c.Server.PPROF || c.Client.PPROF {
		return fmt.Errorf("managed helpers/profiling are not supported by peer Auto; keep this tunnel in manual mode")
	}
	if c.Server.ProxyProtocol || c.Server.ZeroCopy || c.Client.ZeroCopy || c.Server.Sniffer || c.Client.Sniffer || c.Server.WebPort != 0 || c.Client.WebPort != 0 || c.Server.BandwidthMbps != 0 {
		return fmt.Errorf("Auto does not implement Proxy Protocol, zero-copy, per-port sniffer/monitor or the legacy bandwidth limit; disable these explicitly")
	}
	if c.Client.Proxy != "" || c.Client.LocalAddr != "" || c.Client.Interface != "" || c.Client.EdgeIP != "" || c.Client.MSS != 0 || c.Server.MSS != 0 || c.Client.SOMark != 0 {
		return fmt.Errorf("Auto does not implement legacy proxy, source/interface binding, CDN edge override or MSS settings; remove them explicitly")
	}
	if c.Server.BindAddr != "" && c.Server.AcceptUDP != nil && c.Server.ForwardsUDP() != a.AcceptUDP {
		return fmt.Errorf("Auto UDP policy must match the Iran server setting")
	}
	backend := "127.0.0.1"
	if a.L3 != nil {
		backend, _, _ = strings.Cut(a.L3.KharejIP, "/")
	}
	maps, err := portmap.Expand(a.Ports, backend)
	if c.Server.SimpleAuth || c.Client.SimpleAuth {
		return fmt.Errorf("Auto cannot use raw-token/simple-auth TLS proxy mode")
	}
	if err != nil || len(maps) == 0 && a.Mode != "l3" || len(maps) > 32 {
		return fmt.Errorf("Auto requires valid enrolled mappings (1-32 stream, 0-32 L3): %v", err)
	}
	if c.Server.BindAddr != "" {
		x, err := portmap.Expand(c.Server.Ports, backend)
		b, _ := json.Marshal(x)
		d, _ := json.Marshal(maps)
		if err != nil || string(b) != string(d) {
			return fmt.Errorf("Iran forwarded maps must match the enrolled Auto manifest")
		}
	}
	if a.MaxConnections < 1 || a.MaxConnections > 192 || a.ProbeInterval < 30 || a.ProbeInterval > 86400 || a.Cooldown < 5 || a.MinimumDwell < 5 || a.MaxSwitchesPerHour < 1 || a.MaxSwitchesPerHour > 12 || a.FailureThreshold < 3 || a.FailureThreshold > 20 {
		return fmt.Errorf("Auto resource/timing policy outside its conservative bounds")
	}
	if len(a.Candidates) < 1 || len(a.Candidates) > 4 {
		return fmt.Errorf("Auto needs 1-4 explicitly enrolled candidates")
	}
	seen, endpoints, enabled := map[string]bool{}, map[string]bool{}, 0
	lockFound := a.ManualLock == ""
	for _, p := range a.Candidates {
		if !candidateID.MatchString(p.ID) || seen[p.ID] {
			return fmt.Errorf("invalid/duplicate Auto candidate ID")
		}
		seen[p.ID] = true
		adapter, supported := config.FindAutoCarrier(p.Transport)
		if !supported {
			return fmt.Errorf("candidate %s uses an unverified Auto adapter %s", p.ID, p.Transport)
		}
		if p.Mode != "" && p.Mode != "reverse" && p.Mode != "direct" {
			return fmt.Errorf("candidate %s mode must be reverse or direct", p.ID)
		}
		if a.Mode == "l3" && !adapter.L3 {
			return fmt.Errorf("candidate %s is a reliable stream carrier, unsuitable for routed IP packets", p.ID)
		}
		if adapter.Helper != (p.Helper != nil) {
			return fmt.Errorf("candidate %s needs matching HTTPS helper settings", p.ID)
		}
		if p.Helper != nil {
			if err := validateHelper(p, c.Server.BindAddr != "", local && !p.Disabled); err != nil {
				return fmt.Errorf("candidate %s: %w", p.ID, err)
			}
		}
		if p.Packet != nil {
			if p.Transport != config.PCK && p.Transport != config.SPOOF && p.Transport != "sni" {
				return fmt.Errorf("candidate %s has inapplicable raw packet settings", p.ID)
			}
			if p.Packet.Interface != "" && !localInterface.MatchString(p.Packet.Interface) {
				return fmt.Errorf("invalid raw interface name")
			}
			if p.Transport != config.SPOOF && (p.Packet.IranSource != "" || p.Packet.KharejSource != "" || p.Packet.SpoofProfile != "") {
				return fmt.Errorf("spoof settings require the spoof carrier")
			}
			if p.Transport != "sni" && p.Packet.SNIDomain != "" {
				return fmt.Errorf("SNI domain requires the SNI carrier")
			}
		}
		if p.Transport == config.SPOOF {
			if p.Packet == nil {
				return fmt.Errorf("spoof requires explicit paired packet settings")
			}
			if _, err := network.ParseSpoofProfile(p.Packet.SpoofProfile); err != nil {
				return err
			}
			for _, raw := range []string{p.IranIP, p.KharejIP, p.Packet.IranSource, p.Packet.KharejSource} {
				ip, err := netip.ParseAddr(raw)
				if err != nil || !ip.Is4() || ip.IsUnspecified() || ip.IsMulticast() || ip.String() == "255.255.255.255" {
					return fmt.Errorf("spoof requires explicit unicast IPv4 peers and sources")
				}
			}
			host, _, _ := net.SplitHostPort(p.Endpoint)
			want := p.IranIP
			if !p.ListenerIsIran() {
				want = p.KharejIP
			}
			if host != want {
				return fmt.Errorf("spoof endpoint must match its explicit real listener IP")
			}
		}
		if p.Transport == "sni" {
			domain := ""
			if p.Packet != nil {
				domain = p.Packet.SNIDomain
			}
			if _, err := snispoof.BuildHello(snispoof.DomainOr(domain)); err != nil {
				return err
			}
		}
		if p.Transport == config.SPOOF || p.Transport == "sni" || p.Transport == config.PCK || a.Mode == "l3" && p.Transport != config.QUIC {
			for _, listen := range []bool{true, false} {
				settings := packetCarrierConfig(a, p, carrierSecret(a, p, token), listen)
				if err := settings.Validate(); err != nil {
					return fmt.Errorf("candidate %s: %w", p.ID, err)
				}
			}
		}
		if p.Disabled {
			continue
		}
		enabled++
		if p.ID == a.ManualLock {
			lockFound = true
		}
		switch p.Transport {
		case config.WATICMP:
			if err := watSettings(a, p, token, true).Validate(); err != nil {
				return err
			}
			if err := watSettings(a, p, token, false).Validate(); err != nil {
				return err
			}
		default:
			host, port, err := net.SplitHostPort(p.Endpoint)
			n, _ := strconv.Atoi(port)
			ip := net.ParseIP(host)
			if err != nil || host == "" || strings.TrimSpace(host) != host || ip != nil && (ip.IsUnspecified() || ip.IsMulticast()) || n < 0 || n > 65535 || n == 0 && adapter.Socket != "icmp" {
				return fmt.Errorf("candidate %s needs an explicitly permitted peer listener host:port", p.ID)
			}
			// A wildcard/NAT local bind can collide even when public hostnames
			// differ. Require a distinct enrolled port within each socket family.
			key := fmt.Sprintf("%t:%s:%d", p.ListenerIsIran(), adapter.Socket, n)
			spoofUDP := p.Transport == config.SPOOF && strings.EqualFold(strings.TrimSpace(p.Packet.SpoofProfile), "udp")
			if spoofUDP {
				key = fmt.Sprintf("%t:udp:%d", p.ListenerIsIran(), n)
			}
			if adapter.Socket == "icmp" {
				key += ":" + p.ID
			} // authenticated pair/candidate-derived ICMP tags
			if endpoints[key] {
				return fmt.Errorf("candidate listeners conflict: %s", key)
			}
			for _, mapping := range maps {
				_, userPort, _ := net.SplitHostPort(mapping.Listen)
				userN, _ := strconv.Atoi(userPort)
				if userN == n && (p.ListenerIsIran() && (adapter.Socket == "tcp" || adapter.Socket == "udp" && a.AcceptUDP) || spoofUDP && a.AcceptUDP) {
					return fmt.Errorf("candidate %s port conflicts with a forwarded user listener", p.ID)
				}
			}
			endpoints[key] = true
			if spoofUDP {
				other := fmt.Sprintf("%t:udp:%d", !p.ListenerIsIran(), n)
				if endpoints[other] {
					return fmt.Errorf("candidate listeners conflict with the paired spoof receive port")
				}
				endpoints[other] = true
			}
			if p.Helper != nil {
				private := fmt.Sprintf("%t:tcp:%d", p.ListenerIsIran(), p.Helper.ReversePort)
				if endpoints[private] {
					return fmt.Errorf("candidate listeners conflict at private helper target")
				}
				endpoints[private] = true
				for _, mapping := range maps {
					_, port, _ := net.SplitHostPort(mapping.Listen)
					number, _ := strconv.Atoi(port)
					if p.ListenerIsIran() && number == p.Helper.ReversePort {
						return fmt.Errorf("private helper target conflicts with an Iran user port")
					}
				}
			}
			if p.LocalBind != "" {
				_, bindPort, err := net.SplitHostPort(p.LocalBind)
				if err != nil || bindPort != port {
					return fmt.Errorf("candidate local bind must use the enrolled endpoint port")
				}
			}
		}
	}
	if enabled == 0 || !lockFound {
		return fmt.Errorf("Auto has no enabled candidate or manual lock names a disabled candidate")
	}
	return nil
}

func manifestHash(a config.AutoConfig) string {
	// Local runtime state/observe choice never changes the enrolled authority,
	// endpoints, security floor or forwarding authorization.
	a = a.PeerEnrollment()
	a.ObserveOnly = false
	b, _ := json.Marshal(Defaults(a))
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func watSettings(a config.AutoConfig, p config.AutoCandidate, token string, iran bool) waticmp.Settings {
	ip := p.IranIP
	if iran {
		ip = p.KharejIP
	}
	h := sha256.Sum256([]byte(a.TunnelID + ":" + p.ID))
	return waticmp.Settings{Enabled: true, BidirectionalStreams: p.Mode == "direct", PeerIP: ip, TunnelID: hex.EncodeToString(h[:]), Direction: p.Direction, Token: token,
		MaxStreams: min(256, Defaults(a).MaxConnections+16), MaxPayload: p.MaxPayload, MaxBps: 1_000_000, Paths: 4}
}

func tlsSettings(c *config.Config) network.TLSSettings {
	return network.TLSSettings{CertFile: c.Server.TLSCertFile, KeyFile: c.Server.TLSKeyFile, ACMEDomain: c.Server.ACMEDomain, ACMEEmail: c.Server.ACMEEmail}
}
