package manage

import (
	"crypto/sha256"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

// QuickAutoSetup approves the registry's complete pool in either traffic plane.
// Runtime selection remains a bounded subset of that immutable enrollment.
type QuickAutoSetup struct {
	Mode       string `json:"mode,omitempty"`
	IranIP     string `json:"iranIp"`
	KharejIP   string `json:"kharejIp"`
	BasePort   int    `json:"basePort"`
	DirectPort int    `json:"directPort"`
}

type QuickAutoCarrier struct {
	ID         string               `json:"id"`
	Transport  config.TransportType `json:"transport"`
	Mode       string               `json:"mode"`
	PortOffset int                  `json:"portOffset"`
	Direction  string               `json:"direction,omitempty"`
}

// QuickAutoCarriers is the explicitly reviewed full stream pool. It shares
// the registry with advanced setup and includes both Wat Echo directions.
func QuickAutoCarriers() []QuickAutoCarrier   { return quickAutoPoolChoices(false) }
func QuickAutoL3Carriers() []QuickAutoCarrier { return quickAutoPoolChoices(true) }
func quickAutoPoolChoices(l3 bool) []QuickAutoCarrier {
	out := []QuickAutoCarrier{}
	registry := config.AutoCarriers()
	// Put the built-in low-prerequisite bootstrap paths first.
	ordered := []config.AutoCarrier{}
	for _, t := range []config.TransportType{config.TCP, config.WSS, config.STEALTH} {
		c, _ := config.FindAutoCarrier(t)
		if !l3 {
			ordered = append(ordered, c)
		}
	}
	for _, c := range registry {
		if l3 && c.L3 || !l3 && c.Value != config.TCP && c.Value != config.WSS && c.Value != config.STEALTH {
			ordered = append(ordered, c)
		}
	}
	for i, c := range ordered {
		for _, mode := range []string{"reverse", "direct"} {
			suffix := "r"
			if mode == "direct" {
				suffix = "d"
			}
			if c.Value == config.WATICMP {
				for _, dir := range []string{"normal", "reverse"} {
					out = append(out, QuickAutoCarrier{ID: "wat-" + suffix + "-" + dir[:1], Transport: c.Value, Mode: mode, PortOffset: i, Direction: dir})
				}
			} else {
				out = append(out, QuickAutoCarrier{ID: string(c.Value) + "-" + suffix, Transport: c.Value, Mode: mode, PortOffset: i})
			}
		}
	}
	return out
}

func expandQuickAuto(n NewTunnel) (NewTunnel, error) {
	if n.Transport != "auto" || n.Role != "server" || n.Auto != nil || n.WatICMP != nil {
		return n, fmt.Errorf("quick Auto requires a new Iran/server Auto enrollment; do not mix it with an existing manifest")
	}
	q := *n.QuickAuto
	q.IranIP, q.KharejIP = strings.TrimSpace(q.IranIP), strings.TrimSpace(q.KharejIP)
	for _, address := range []string{q.IranIP, q.KharejIP} {
		ip := net.ParseIP(address)
		if ip == nil || ip.To4() == nil || ip.IsUnspecified() || ip.IsMulticast() || ip.Equal(net.IPv4bcast) {
			return n, fmt.Errorf("quick Auto requires both explicit unicast peer IPv4 addresses")
		}
	}
	q.IranIP = net.ParseIP(q.IranIP).To4().String()
	q.KharejIP = net.ParseIP(q.KharejIP).To4().String()
	if q.BasePort == 0 {
		q.BasePort = 24443
	}
	if q.DirectPort == 0 {
		q.DirectPort = q.BasePort + 32
	}
	if q.BasePort < 1 || q.BasePort > 65390 || q.DirectPort < 1 || q.DirectPort > 65390 {
		return n, fmt.Errorf("quick Auto needs two base ports in 1-65390 with room for its reviewed data/private listener ranges")
	}
	if strings.TrimSpace(n.Token) == "" {
		n.Token = randomToken(64)
	}
	if q.Mode != "" && q.Mode != "stream" && q.Mode != "l3" {
		return n, fmt.Errorf("quick Auto mode must be stream or l3")
	}
	limit := 0 // no application connection ceiling; enroll this choice explicitly
	a := config.AutoConfig{Enabled: true, ObserveOnly: false, ConnectionLimit: &limit, TunnelID: randomToken(32), IranID: randomToken(32), KharejID: randomToken(32), Selection: &config.AutoSelectionConfig{Target: 4, RefillAt: 2, StableSamples: 3, RetryInterval: 60, HelperEpoch: time.Now().Unix()}}
	choices := QuickAutoCarriers()
	if q.Mode == "l3" {
		a.Mode = "l3"
		choices = QuickAutoL3Carriers()
		hash := sha256.Sum256([]byte(a.TunnelID))
		prefix := fmt.Sprintf("10.%d.%d.", 64+int(hash[0])%64, hash[1])
		network := int(hash[2]) & 252
		a.L3 = &config.AutoL3Config{IranIP: fmt.Sprintf("%s%d/30", prefix, network+1), KharejIP: fmt.Sprintf("%s%d/30", prefix, network+2), MTU: 1400}
	}
	if n.Tune != nil {
		a.AcceptUDP = n.Tune.AcceptUDP
	}
	for _, choice := range choices {
		host, port := q.IranIP, q.BasePort+choice.PortOffset
		if choice.Mode == "direct" {
			host, port = q.KharejIP, q.DirectPort+choice.PortOffset
		}
		p := config.AutoCandidate{ID: choice.ID, Transport: choice.Transport, Mode: choice.Mode, Endpoint: net.JoinHostPort(host, strconv.Itoa(port))}
		p.IranIP, p.KharejIP = q.IranIP, q.KharejIP
		if choice.Transport == config.WATICMP {
			p.Endpoint = ""
			p.Direction = choice.Direction
			p.MaxPayload = 512
		} else if choice.Mode == "reverse" {
			p.LocalBind = net.JoinHostPort("0.0.0.0", strconv.Itoa(port))
		}
		if choice.Transport == config.SPOOF {
			p.Packet = &config.AutoPacketConfig{SpoofProfile: "udp", IranSource: q.IranIP, KharejSource: q.KharejIP}
		}
		if choice.Transport == "sni" {
			p.Packet = &config.AutoPacketConfig{SNIDomain: "www.apple.com"}
		}
		carrier, _ := config.FindAutoCarrier(choice.Transport)
		if carrier.Helper {
			var err error
			p.Helper, err = peerauto.PoolHelperIdentity(a, p, n.Token, port+128)
			if err != nil {
				return n, err
			}
		}
		a.Candidates = append(a.Candidates, p)
	}
	n.Auto, n.QuickAuto = &a, nil
	return n, nil
}

func setupQuickAutoServer() {
	public := PublicIPv4()
	if public == "-" {
		public = ""
	}
	n := NewTunnel{Name: uniqueName(tui.PromptDefault("Tunnel Name", "server-auto")), Role: "server", Transport: "auto"}
	mode := tui.ChooseOpt("Quick Auto Traffic", []tui.Option{{Title: "TCP/UDP port forwarding", Desc: "All 18 stream protocols in Reverse and Direct; select four stable paths"}, {Title: "IP network (Layer 3)", Desc: "All six datagram protocols in both directions; one stable TUN"}})
	if mode < 0 {
		return
	}
	q := QuickAutoSetup{IranIP: tui.PromptDefault("Iran IPv4", public), KharejIP: tui.Prompt("Kharej IPv4: ")}
	if mode == 1 {
		q.Mode = "l3"
	}
	n.Ports = tui.Prompt("Forwarded Ports (e.g. 2052,8880; optional for Layer 3): ")
	accept := tui.Confirm("Forward UDP As Well As TCP", false)
	n.Tune = &FineTune{AcceptUDP: accept, sent: map[string]bool{"acceptUDP": true}}
	n.QuickAuto = &q
	s, err := specFromPeerNew(n)
	if err != nil {
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}
	link := pendingReverseLink(s, q.IranIP, linkExtras{})
	if link == "" {
		tui.Error("Could not build the trusted peer setup link.")
		tui.PressEnter()
		return
	}
	summarisePeer(s)
	tui.Info("Quick Auto approves the displayed full protocol pool, selects up to four paths after three data probes, and rescans when only two remain healthy. Missing local prerequisites are SKIPPED; no helper is downloaded implicitly.")
	tui.Warn("This Setup Link Contains The Enrollment Secret — Keep It Private.")
	fmt.Println("sudo fullpack link apply '" + link + "'")
	if tui.Confirm("Approve These Endpoints And Create Quick Auto", true) && !finishSetup(s) {
		return
	}
	tui.PressEnter()
}
