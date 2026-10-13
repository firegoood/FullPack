package config

// AutoCarrier describes actual enrolled engine adapters. Both setup interfaces
// consume this registry; availability/privilege failures remain explicit.
type AutoCarrier struct {
	Value  TransportType `json:"value"`
	Label  string        `json:"label"`
	Socket string        `json:"socket"`
	Raw    bool          `json:"raw"`
	Helper bool          `json:"helper"`
	L3     bool          `json:"l3"`
}

func AutoCarriers() []AutoCarrier {
	return []AutoCarrier{
		{WATICMP, "Wat ICMP", "icmp", true, false, false},
		{STEALTH, "TCP + Stealth (Noise)", "tcp", false, false, false},
		{TCP, "TCP", "tcp", false, false, false},
		{TCPMUX, "TCP Mux", "tcp", false, false, false},
		{WS, "WebSocket", "tcp", false, false, false},
		{WSMUX, "WebSocket Mux", "tcp", false, false, false},
		{WSS, "Secure WebSocket", "tcp", false, false, false},
		{WSSMUX, "Secure WebSocket Mux", "tcp", false, false, false},
		{KCP, "KCP", "udp", false, false, false},
		{UDP, "UDP", "udp", false, false, true},
		{XDI, "xDi / ICMP", "icmp", true, false, true},
		{PCK, "PCK", "tcp", true, false, true},
		{QUIC, "QUIC", "udp", false, false, true},
		{SPOOF, "IP Spoofing", "raw", true, false, true},
		{"sni", "SNI Spoofing", "tcp", true, false, true},
		{"naive", "Naive / HTTP2", "tcp", false, true, false},
		{"xhttp", "XHTTP / TLS", "tcp", false, true, false},
		{"reality", "REALITY / Vision", "tcp", false, true, false},
	}
}

func FindAutoCarrier(t TransportType) (AutoCarrier, bool) {
	for _, c := range AutoCarriers() {
		if c.Value == t {
			return c, true
		}
	}
	return AutoCarrier{}, false
}

func (c AutoCandidate) ListenerIsIran() bool { return c.Mode != "direct" }
