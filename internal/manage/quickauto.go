package manage

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
)

// QuickAutoSetup chooses a small, explicitly reviewed stream enrollment.
// Full carrier selection, helper identities and L3 remain in advanced setup.
type QuickAutoSetup struct {
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
}

// QuickAutoCarriers is shared by the builder and the panel's endpoint review.
// No raw privilege, third-party helper or imported TLS private key is needed.
func QuickAutoCarriers() []QuickAutoCarrier {
	return []QuickAutoCarrier{
		{"tcp-rev", config.TCP, "reverse", 0},
		{"wss-rev", config.WSS, "reverse", 1},
		{"noise-rev", config.STEALTH, "reverse", 2},
		{"noise-dir", config.STEALTH, "direct", 0},
	}
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
	if q.BasePort == 0 {
		q.BasePort = 24443
	}
	if q.DirectPort == 0 {
		q.DirectPort = q.BasePort + 2
	}
	if q.BasePort < 1 || q.BasePort > 65533 || q.DirectPort < 1 || q.DirectPort > 65535 {
		return n, fmt.Errorf("quick Auto needs three consecutive Reverse ports and one Direct port within 1-65535")
	}
	if strings.TrimSpace(n.Token) == "" {
		n.Token = randomToken(64)
	}
	a := config.AutoConfig{Enabled: true, ObserveOnly: false, TunnelID: randomToken(32), IranID: randomToken(32), KharejID: randomToken(32)}
	if n.Tune != nil {
		a.AcceptUDP = n.Tune.AcceptUDP
	}
	for _, choice := range QuickAutoCarriers() {
		host, port := q.IranIP, q.BasePort+choice.PortOffset
		if choice.Mode == "direct" {
			host, port = q.KharejIP, q.DirectPort
		}
		p := config.AutoCandidate{ID: choice.ID, Transport: choice.Transport, Mode: choice.Mode, Endpoint: net.JoinHostPort(host, strconv.Itoa(port))}
		if choice.Mode == "reverse" {
			p.LocalBind = net.JoinHostPort("0.0.0.0", strconv.Itoa(port))
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
	q := QuickAutoSetup{IranIP: tui.PromptDefault("Iran IPv4", public), KharejIP: tui.Prompt("Kharej IPv4: ")}
	n.Ports = tui.Prompt("Forwarded Ports (e.g. 2052,8880): ")
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
	tui.Info("Quick Auto enrolls these four endpoints with automatic failover enabled. Advanced setup offers custom ports, Wat, HTTPS helpers and L3.")
	tui.Warn("This Setup Link Contains The Enrollment Secret — Keep It Private.")
	fmt.Println("sudo fullpack link apply '" + link + "'")
	if tui.Confirm("Approve These Endpoints And Create Quick Auto", true) && !finishSetup(s) {
		return
	}
	tui.PressEnter()
}
