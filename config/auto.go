package config

// AutoConfig is an enrolled pair's policy. It never infers public endpoints
// from fallback addresses or creates an unapproved management listener.
type AutoConfig struct {
	// Mode is empty/stream for stable port forwarding, or l3 for an owned TUN.
	Mode string `toml:"mode" json:"mode,omitempty"`
	// L3 enrolls the shared addresses and MTU. Interface is a local setting.
	L3 *AutoL3Config `toml:"l3" json:"l3,omitempty"`
	// Enabled opts into the enrolled coordinator; false/absent keeps the manual engine and legacy fallback.
	Enabled bool `toml:"enabled" json:"enabled"`
	// ObserveOnly bootstraps the primary/last-good route and measures candidates without automatic cross-carrier changes.
	ObserveOnly bool `toml:"observe_only" json:"observeOnly"`
	// TunnelID is the immutable 16-64 safe-character pair identity.
	TunnelID string `toml:"tunnel_id" json:"tunnelId"`
	// IranID identifies the sole switching authority, independent of packet direction.
	IranID string `toml:"iran_id" json:"iranId"`
	// KharejID identifies the distinct enrolled follower; changing identities requires a new pair.
	KharejID string `toml:"kharej_id" json:"kharejId"`
	// These exact maps are authenticated as part of the manifest. The follower
	// dials only this enrolled backend allowlist, indexed by mapping number.
	Ports []string `toml:"ports" json:"ports"`
	// AcceptUDP permits framed UDP forwarding on the enrolled Iran user ports.
	AcceptUDP bool `toml:"accept_udp" json:"acceptUdp"`
	// Candidates lists 1-4 explicitly approved endpoints, independent of the active carrier.
	Candidates []AutoCandidate `toml:"candidates" json:"candidates"`
	// AllowPlaintextFallback must remain false: plaintext Auto adapters are unsupported in this version.
	AllowPlaintextFallback bool `toml:"allow_plaintext_fallback" json:"allowPlaintextFallback"`
	// AllowDegradationSwitch optionally permits changing a reachable path after sustained severe measured degradation.
	AllowDegradationSwitch bool `toml:"allow_degradation_switch" json:"allowDegradationSwitch"`
	// ProbeInterval is seconds between paced bulk tests, default 600, minimum 30; schedules include jitter.
	ProbeInterval int `toml:"probe_interval" json:"probeInterval"` // seconds, default 600
	// Cooldown is seconds between optional cross-carrier changes, default 300, minimum 5.
	// Debounced no-route recovery uses a separate 5-second gate; all commits still extend this cooldown.
	Cooldown int `toml:"cooldown" json:"cooldown"` // seconds, default 300
	// MinimumDwell is seconds before optional degradation switching, default 600, minimum 5.
	MinimumDwell int `toml:"minimum_dwell" json:"minimumDwell"` // seconds, default 600
	// MaxSwitchesPerHour limits optional changes using all persisted cross-carrier commits, default 2, range 1-12.
	// Debounced no-route recovery retains the absolute persisted ceiling of 12 changes per hour.
	MaxSwitchesPerHour int `toml:"max_switches_per_hour" json:"maxSwitchesPerHour"` // default 2
	// FailureThreshold debounces carrier failure, default 3, range 3-20; skipped probes do not count.
	FailureThreshold int `toml:"failure_threshold" json:"failureThreshold"` // consecutive actual path failures, default 3
	// MaxConnections bounds simultaneous user connections, default 64, maximum 192; probes reserve separate streams.
	MaxConnections int `toml:"max_connections" json:"maxConnections"` // default 64, maximum 192
	// ManualLock fixes the allowed active candidate ID; runtime unlock cannot weaken an enrolled lock.
	ManualLock string `toml:"manual_lock" json:"manualLock"`
	// MaxMemoryMB skips probes above this local process RSS pressure threshold, default 256 MB; it is not a hard memory limit.
	MaxMemoryMB int `toml:"max_memory_mb" json:"maxMemoryMb"` // default 256
}

type AutoCandidate struct {
	// ID identifies an approved candidate, 1-16 safe characters, unique within this pair.
	ID string `toml:"id" json:"id"`
	// Transport selects a registered carrier. Auto adds authenticated encryption
	// to legacy carriers that do not provide it; manual transport wire is unchanged.
	Transport TransportType `toml:"transport" json:"transport"`
	// Mode is empty/reverse (Kharej dials Iran) or direct (Iran dials Kharej).
	// Geography and the sole switching authority do not change with this field.
	Mode string `toml:"mode" json:"mode,omitempty"`
	// Helper enrolls an HTTPS wrapper's identity and restricted loopback target.
	Helper *AutoHelperConfig `toml:"helper" json:"helper,omitempty"`
	// Packet tunes the existing raw packet carriers. Interface is local-only.
	Packet *AutoPacketConfig `toml:"packet" json:"packet,omitempty"`
	// Disabled prevents this candidate from opening a listener or dialing after startup.
	Disabled bool `toml:"disabled" json:"disabled"`
	// Endpoint is the explicitly permitted listener on the side Mode selects.
	Endpoint string `toml:"endpoint" json:"endpoint"`
	// LocalBind is an optional listener-local address with the enrolled port;
	// never exported to the peer or interpreted as a dial destination.
	LocalBind string `toml:"local_bind" json:"localBind,omitempty"`
	// IranIP is Iran's explicit unicast IPv4 for a Wat candidate.
	IranIP string `toml:"iran_ip" json:"iranIp"`
	// KharejIP is Kharej's explicit unicast IPv4 for a Wat candidate.
	KharejIP string `toml:"kharej_ip" json:"kharejIp"`
	// Direction is normal or reverse for Wat Echo Requests; application users still enter on Iran.
	Direction string `toml:"direction" json:"direction"`
	// MaxPayload caps Wat datagrams at 256-1400 bytes, default 512, subject to authenticated PMTU discovery.
	MaxPayload int `toml:"max_payload" json:"maxPayload"`
}

type AutoL3Config struct {
	// IranIP is Iran's enrolled unicast TUN address with CIDR prefix.
	IranIP string `toml:"iran_ip" json:"iranIp"`
	// KharejIP is the distinct peer TUN address in the same enrolled subnet.
	KharejIP string `toml:"kharej_ip" json:"kharejIp"`
	// MTU bounds inner IP packets, up to 9000 bytes; IPv6 requires at least 1280.
	MTU int `toml:"mtu" json:"mtu"`
	// Interface is a local TUN name; blank selects a stable pair-specific name.
	Interface string `toml:"interface" json:"interface,omitempty"`
}

// Local helper executables, certificate/key paths and private keys are never
// exported. CAPEM, public key and credentials are the peer's approved identity.
type AutoHelperConfig struct {
	// Binary is the local helper executable; blank uses the managed installation.
	Binary string `toml:"binary" json:"binary,omitempty"`
	// ReversePort is the restricted loopback engine port, distinct from public and user ports.
	ReversePort int `toml:"reverse_port" json:"reversePort"`
	// Username is the enrolled Naive account name.
	Username string `toml:"username" json:"username,omitempty"`
	// Password is the paired Naive account secret; settings responses redact it.
	Password string `toml:"password" json:"password,omitempty"`
	// CAPEM is the public CA certificate enrolled for HTTPS peer verification.
	CAPEM string `toml:"ca_pem" json:"caPem,omitempty"`
	// CAFile optionally supplies a dialer's local CA bundle without disabling TLS verification.
	CAFile string `toml:"ca_file" json:"caFile,omitempty"`
	// Certificate is the listener's local TLS certificate path matching its enrolled hostname and CA.
	Certificate string `toml:"certificate" json:"certificate,omitempty"`
	// Key is the listener's local TLS private-key path; never exported in enrollment.
	Key string `toml:"key" json:"key,omitempty"`
	// UUID is the enrolled XHTTP or REALITY account identity.
	UUID string `toml:"uuid" json:"uuid,omitempty"`
	// ServerName is the verified HTTPS hostname or authenticated REALITY cover name.
	ServerName string `toml:"server_name" json:"serverName,omitempty"`
	// Path is the enrolled XHTTP request path.
	Path string `toml:"path" json:"path,omitempty"`
	// Host optionally overrides the enrolled XHTTP HTTP Host header.
	Host string `toml:"host" json:"host,omitempty"`
	// PrivateKey is the listener-local REALITY key matching PublicKey; never exported.
	PrivateKey string `toml:"private_key" json:"privateKey,omitempty"`
	// PublicKey is the enrolled REALITY X25519 public key.
	PublicKey string `toml:"public_key" json:"publicKey,omitempty"`
	// Target is the REALITY cover host:port checked with the authenticated cover probe.
	Target string `toml:"target" json:"target,omitempty"`
	// ShortID is the enrolled REALITY short ID.
	ShortID string `toml:"short_id" json:"shortId,omitempty"`
}

type AutoPacketConfig struct {
	// Interface optionally selects the local raw-packet interface; never exported.
	Interface string `toml:"interface" json:"interface,omitempty"`
	// SpoofProfile selects an existing SPOOF packet profile; Auto adds independent authentication.
	SpoofProfile string `toml:"spoof_profile" json:"spoofProfile,omitempty"`
	// IranSource is Iran's enrolled forged unicast source address for SPOOF.
	IranSource string `toml:"iran_source" json:"iranSource,omitempty"`
	// KharejSource is Kharej's enrolled forged unicast source address for SPOOF.
	KharejSource string `toml:"kharej_source" json:"kharejSource,omitempty"`
	// SNIDomain is the enrolled hostname embedded by the direct SNI carrier.
	SNIDomain string `toml:"sni_domain" json:"sniDomain,omitempty"`
}

// PeerEnrollment copies the paired policy without receiver-local paths,
// interface overrides or helper private keys. Credentials remain paired secrets.
func (a AutoConfig) PeerEnrollment() AutoConfig {
	a.Candidates = append([]AutoCandidate(nil), a.Candidates...)
	a.MaxMemoryMB = 0
	if a.L3 != nil {
		local := *a.L3
		local.Interface = ""
		a.L3 = &local
	}
	for i := range a.Candidates {
		p := &a.Candidates[i]
		p.LocalBind = ""
		if p.Packet != nil {
			v := *p.Packet
			v.Interface = ""
			p.Packet = &v
			if v == (AutoPacketConfig{}) {
				p.Packet = nil
			}
		}
		if p.Helper != nil {
			v := *p.Helper
			v.Binary, v.CAFile, v.Certificate, v.Key, v.PrivateKey = "", "", "", "", ""
			p.Helper = &v
		}
	}
	return a
}

func (a AutoConfig) RedactSecrets() AutoConfig {
	a.Candidates = append([]AutoCandidate(nil), a.Candidates...)
	for i := range a.Candidates {
		if a.Candidates[i].Helper != nil {
			v := *a.Candidates[i].Helper
			v.Password, v.PrivateKey = "", ""
			a.Candidates[i].Helper = &v
		}
	}
	return a
}
