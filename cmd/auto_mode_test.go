package cmd

import (
	"runtime"
	"testing"

	"github.com/backpack/backpack/config"
)

func TestAutoPrimaryUsesEnrolledValidator(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux Auto engine")
	}
	for _, tr := range []config.TransportType{config.TCP, config.KCP, config.PCK, config.TransportType("sni"), config.SPOOF} {
		t.Run(string(tr), func(t *testing.T) {
			p := config.AutoCandidate{ID: "primary", Transport: tr, Endpoint: "127.0.0.1:24001"}
			if tr == "sni" {
				p.Packet = &config.AutoPacketConfig{SNIDomain: "example.com"}
			}
			if tr == config.SPOOF {
				p.IranIP = "127.0.0.1"
				p.KharejIP = "127.0.0.2"
				p.Packet = &config.AutoPacketConfig{SpoofProfile: "udp", IranSource: "127.0.0.3", KharejSource: "127.0.0.4"}
			}
			ports := []string{"24002=127.0.0.1:24003"}
			c := &config.Config{Server: config.ServerConfig{BindAddr: "0.0.0.0:0", Transport: tr, Token: "0123456789abcdef0123456789abcdef", Ports: ports}, Auto: config.AutoConfig{Enabled: true, TunnelID: "validator-tunnel", IranID: "validator-iran-id", KharejID: "validator-kharej-id", Ports: ports, Candidates: []config.AutoCandidate{p}}}
			if err := validateConfig(c); err != nil {
				t.Fatal(err)
			}
			c.Client.SOMark = 1
			if validateConfig(c) == nil {
				t.Fatal("ignored legacy socket mark accepted")
			}
		})
	}
}
