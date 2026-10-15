package manage

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/enginectl"
	"github.com/backpack/backpack/internal/tui"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

func askWatDirection(current string) string {
	idx := tui.ChooseOpt("ICMP Packet Direction (Both Ends Must Match)", []tui.Option{{Title: "Normal — Iran sends Echo Requests", Desc: "Kharej sends authenticated Echo Replies"}, {Title: "Reverse — Kharej sends Echo Requests", Desc: "works when Iran-originated ICMP requests are blocked"}})
	if idx == 0 {
		return "normal"
	}
	if idx == 1 {
		return "reverse"
	}
	return current
}
func setupPeerServer(transport string) {
	if transport == "auto" {
		pick := tui.ChooseOpt("Auto Setup", []tui.Option{
			{Title: "Quick TCP/UDP Setup", Desc: "All protocols in both directions; four stable paths with automatic replenishment"},
			{Title: "Advanced Setup", Desc: "Choose any approved carrier, custom endpoint, Wat, managed HTTPS or Layer 3"},
		})
		if pick < 0 {
			return
		}
		if pick == 0 {
			setupQuickAutoServer()
			return
		}
	}
	tui.Info("Enrolled peer forwarding. Auto supports Reverse, Direct initiation and datagram L3; Wat needs Linux CAP_NET_RAW.")
	n := NewTunnel{Role: "server", Transport: transport, Name: uniqueName(tui.PromptDefault("Tunnel Name", "server-"+transport)), Token: tui.PromptDefault("Enrollment Secret (At Least 32 Bytes)", randomToken(64))}
	l3Mode := false
	if transport == "auto" {
		pick := tui.ChooseOpt("Auto Traffic Mode", []tui.Option{{Title: "TCP/UDP Port Forwarding", Desc: "Each enrolled carrier may use Reverse or Direct initiation"}, {Title: "Layer 3 (Stable TUN)", Desc: "Routed IP datagrams; optional forwarded ports"}})
		if pick < 0 {
			return
		}
		l3Mode = pick == 1
	}
	n.Ports = tui.Prompt("Forwarded Ports (Optional For L3): ")
	public := PublicIPv4()
	if public == "-" {
		public = ""
	}
	iran := tui.PromptDefault("Iran Explicit IPv4 / Approved Listener Host", public)
	accept := tui.Confirm("Forward UDP As Well As TCP", false)
	n.Tune = &FineTune{AcceptUDP: accept, sent: map[string]bool{"acceptUDP": true}}
	if transport == "waticmp" {
		direction := askWatDirection("")
		if direction == "" {
			return
		}
		n.WatICMP = &config.WatICMPConfig{Enabled: true, PeerIP: tui.Prompt("Kharej Explicit IPv4: "), TunnelID: randomToken(32), Direction: direction, MaxPayload: 512, MaxStreams: 128, Paths: 4}
	} else {
		a := config.AutoConfig{Enabled: true, ObserveOnly: tui.Confirm("Start In Observe-Only Mode (No Automatic Cross-Carrier Switch)", true), TunnelID: randomToken(32), IranID: randomToken(32), KharejID: randomToken(32), Ports: parsePorts(n.Ports), AcceptUDP: accept}
		limit, err := strconv.Atoi(strings.TrimSpace(tui.PromptDefault("Optional Maximum Connections (0 = No Limit)", "0")))
		if err != nil || limit < 0 {
			tui.Error("Connection limit must be a nonnegative integer; 0 means no limit.")
			return
		}
		a.ConnectionLimit = &limit
		if l3Mode {
			a.Mode = "l3"
			a.L3 = &config.AutoL3Config{IranIP: tui.PromptDefault("Iran TUN Address/Prefix", "10.44.0.1/30"), KharejIP: tui.PromptDefault("Kharej TUN Address/Prefix", "10.44.0.2/30"), MTU: 1400, Interface: tui.Prompt("Local TUN Interface (Blank = Pair-Specific Default): ")}
		}
		carriers := []config.AutoCarrier{}
		for _, v := range config.AutoCarriers() {
			if !l3Mode || v.L3 {
				carriers = append(carriers, v)
			}
		}
		for len(a.Candidates) < 4 {
			opts := []tui.Option{}
			for _, v := range carriers {
				opts = append(opts, tui.Option{Title: v.Label, Desc: "Explicitly enrolled " + v.Socket + " endpoint; authenticated encryption"})
			}
			if len(a.Candidates) > 0 {
				opts = append(opts, tui.Option{Title: "Finish Candidate Enrollment"})
			}
			pick := tui.ChooseOpt("Approved Carrier — All Its Inbound Ports Must Be Allowed", opts)
			if pick < 0 {
				return
			}
			if pick == len(carriers) {
				break
			}
			p := config.AutoCandidate{ID: tui.PromptDefault("Candidate ID (1-16 Safe Characters)", fmt.Sprintf("carrier%d", len(a.Candidates)+1))}
			p.Transport = carriers[pick].Value
			mode := tui.ChooseOpt("Connection Initiator", []tui.Option{{Title: "Reverse — Kharej Connects To Iran"}, {Title: "Direct — Iran Connects To Kharej"}})
			if mode < 0 {
				return
			}
			if mode == 1 {
				p.Mode = "direct"
			}
			if p.Transport == config.WATICMP {
				p.IranIP = iran
				p.KharejIP = tui.Prompt("Kharej Explicit IPv4: ")
				p.Direction = askWatDirection("")
				p.MaxPayload = 512
				if p.Direction == "" {
					return
				}
			} else {
				host := iran
				if p.Mode == "direct" {
					host = tui.Prompt("Approved Kharej Listener Host/IP: ")
				}
				port := tui.Prompt("Approved Data Listener Port: ")
				if !validPort(port) {
					tui.Error("Invalid data listener port.")
					return
				}
				p.Endpoint = net.JoinHostPort(host, port)
				if p.ListenerIsIran() {
					p.LocalBind = net.JoinHostPort("0.0.0.0", port)
				}
				if carriers[pick].Raw && p.Transport != config.XDI {
					p.Packet = &config.AutoPacketConfig{Interface: tui.Prompt("Local Raw Interface (Optional): ")}
					if p.Transport == config.SPOOF {
						p.IranIP = iran
						p.KharejIP = tui.Prompt("Kharej Real IPv4: ")
						p.Packet.SpoofProfile = tui.PromptDefault("Spoof Profile", "udp")
						p.Packet.IranSource = tui.Prompt("Iran Forged Source IPv4: ")
						p.Packet.KharejSource = tui.Prompt("Kharej Forged Source IPv4: ")
					}
					if p.Transport == "sni" {
						p.Packet.SNIDomain = tui.PromptDefault("SNI Domain", "www.microsoft.com")
					}
				}
				if carriers[pick].Helper {
					p.Helper = promptAutoHelper(p)
				}
			}
			a.Candidates = append(a.Candidates, p)
		}
		n.Auto = &a
	}
	s, e := specFromPeerNew(n)
	if e != nil {
		tui.Error(e.Error())
		tui.PressEnter()
		return
	}
	link := pendingReverseLink(s, iran, linkExtras{})
	if link == "" {
		tui.Error("Enrollment cannot produce a valid peer link; check both addresses and maps.")
		tui.PressEnter()
		return
	}
	summarisePeer(s)
	tui.Warn("This Setup Link Contains The Enrollment Secret — Keep It Private.")
	fmt.Println("sudo fullpack link apply '" + link + "'")
	if tui.Confirm("Create This Explicitly Enrolled Tunnel", true) {
		if !finishSetup(s) {
			return
		}
	}
	tui.PressEnter()
}
func setupPeerClient(transport string) {
	raw := tui.Prompt("Trusted Version-3 Peer Setup Link: ")
	l, e := DecodeShareLink(raw)
	if e != nil {
		tui.Error(e.Error())
		tui.PressEnter()
		return
	}
	if l.Tr != transport || l.Kind != "reverse" || l.From != "iran" {
		tui.Error("This link is not the selected Iran peer enrollment.")
		tui.PressEnter()
		return
	}
	f := MirrorForPeer(l)
	n := f.ToNewTunnel()
	n.Name = uniqueName(tui.PromptDefault("Tunnel Name", f.Name))
	if n.Auto != nil {
		if n.Auto.L3 != nil {
			n.Auto.L3.Interface = tui.Prompt("Local TUN Interface (Optional): ")
		}
		for i := range n.Auto.Candidates {
			p := &n.Auto.Candidates[i]
			if !p.ListenerIsIran() && p.Transport != config.WATICMP {
				p.LocalBind = tui.PromptDefault("Local Listener Bind For "+p.ID, net.JoinHostPort("0.0.0.0", addrPort(p.Endpoint)))
			}
			if p.Packet == nil && (p.Transport == config.PCK || p.Transport == "sni") {
				p.Packet = &config.AutoPacketConfig{}
			}
			if p.Packet != nil {
				p.Packet.Interface = tui.Prompt("Local Raw Interface For " + p.ID + " (Optional): ")
			}
			if p.Helper != nil {
				h := p.Helper
				h.Binary = tui.Prompt("Local Helper Executable For " + p.ID + " (Blank = Installed Default): ")
				if !p.ListenerIsIran() {
					if p.Transport == "reality" {
						h.PrivateKey = tui.Prompt("Matching Local REALITY Private Key: ")
					} else {
						h.Certificate = tui.Prompt("Matching Local Certificate Path: ")
						h.Key = tui.Prompt("Local Certificate Private Key Path: ")
					}
				} else if p.Transport != "reality" {
					h.CAFile = tui.Prompt("Local CA File For " + p.ID + " (Optional): ")
				}
			}
		}
	}
	s, e := specFromPeerNew(n)
	if e != nil {
		tui.Error(e.Error())
		tui.PressEnter()
		return
	}
	summarisePeer(s)
	if tui.Confirm("Create This Peer With The Same Approved Manifest", true) {
		if !finishSetup(s) {
			return
		}
	}
	tui.PressEnter()
}
func summarisePeer(s TunnelSpec) {
	tui.Info("Enrolled Forwarding — " + selectedTransport(s))
	if s.Auto.Enabled {
		if limit := s.Auto.ConnectionCapacity(); limit == 0 {
			tui.Info("Application connections: no limit (available server resources still apply).")
		} else {
			tui.Info(fmt.Sprintf("Application connection limit: %d", limit))
		}
		tui.Info("Authority: Iran " + s.Auto.IranID + "; foreign peer: " + s.Auto.KharejID)
		if s.Auto.L3 != nil {
			tui.Info("Stable L3 TUN: " + s.Auto.L3.IranIP + " ↔ " + s.Auto.L3.KharejIP)
		}
		for _, p := range s.Auto.Candidates {
			if p.Transport == config.WATICMP {
				tui.Info(p.ID + ": Wat " + p.Direction + " ICMP, Iran=" + p.IranIP + ", Kharej=" + p.KharejIP)
			} else {
				tui.Info(p.ID + ": " + string(p.Transport) + " " + defaultString(p.Mode, "reverse") + " data endpoint " + p.Endpoint)
			}
		}
		tui.Info("Observe-only: " + onOff(s.Auto.ObserveOnly) + "; probes, commits and recovery use these approved carriers only.")
	} else {
		tui.Info("Wat peer: " + s.WatICMP.PeerIP + "; direction: " + s.WatICMP.Direction + "; no TCP/UDP tunnel port.")
	}
	if s.Role == "server" {
		tui.Info("User ports: " + strings.Join(s.Ports, ", "))
	}
}

func editPeerMenu(name string, s TunnelSpec) {
	if s.Auto.TunnelID != "" && !s.Auto.Enabled {
		tui.Info("Auto is disabled. Retain the enrolled state; restore the matching Auto config on both peers or create a separate manual tunnel.")
		tui.PressEnter()
		return
	}
	summarisePeer(s)
	if !s.Auto.Enabled {
		if tui.Confirm("Edit Native Peer / Direction (Update Both Ends)", false) {
			s.WatICMP.PeerIP = tui.PromptDefault("Peer IPv4", s.WatICMP.PeerIP)
			s.WatICMP.Direction = askWatDirection(s.WatICMP.Direction)
			if s.Role == "client" {
				s.RemoteAddr = net.JoinHostPort(s.WatICMP.PeerIP, "0")
			}
			report(applySpec(s), "native settings updated")
		}
		return
	}
	var status peerauto.Status
	if e := enginectl.Ask(name, enginectl.OpAutoStatus, &status); e != nil {
		tui.Error(e.Error())
		tui.PressEnter()
		return
	}
	b, _ := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(b))
	choices := []tui.Option{{Title: "Run Protocol Tests Now"}, {Title: "Manual Carrier Override"}, {Title: "Lock To A Carrier"}, {Title: "Disable A Standby Carrier Until Restart"}, {Title: "Remove Runtime Lock"}, {Title: "Observe Only"}, {Title: "Resume Auto Decisions"}, {Title: "Rescan Enrolled Reserve Pool"}}
	pick := tui.ChooseOpt("Auto Operations", choices)
	if pick < 0 {
		return
	}
	op := []string{"test-now", "switch", "lock", "disable-carrier", "unlock", "observe", "resume", "rescan"}[pick]
	a := peerauto.Action{Operation: op}
	if pick < 4 {
		a.Candidate = tui.Prompt("Enrolled Candidate ID: ")
	}
	report(enginectl.AskBody(name, enginectl.OpAutoAction, a, nil), "Auto operation completed")
}
