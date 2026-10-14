// Disposable namespace fixture. Never installs or rewrites a user's enrollment.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/manage"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
)

func main() {
	if len(os.Args) != 2 || os.Getenv("FULLPACK_AUTO_NETNS") != "1" {
		panic("disposable privileged fixture only")
	}
	dir := os.Args[1]
	a := peerauto.Defaults(config.AutoConfig{Enabled: true, ObserveOnly: true, Mode: "l3", TunnelID: "disposable-l3-pool-12345", IranID: "disposable-iran-12345", KharejID: "disposable-kharej-12345", MaxMemoryMB: 384, Selection: &config.AutoSelectionConfig{RetryInterval: 30}, L3: &config.AutoL3Config{IranIP: "172.29.89.1/30", KharejIP: "172.29.89.2/30", MTU: 1400, Interface: "fpauto"}})
	for _, choice := range manage.QuickAutoL3Carriers() {
		host, port := "10.79.0.1", 24443+choice.PortOffset
		if choice.Mode == "direct" {
			host, port = "10.79.0.2", 24475+choice.PortOffset
		}
		p := config.AutoCandidate{ID: choice.ID, Transport: choice.Transport, Mode: choice.Mode, Endpoint: fmt.Sprintf("%s:%d", host, port), IranIP: "10.79.0.1", KharejIP: "10.79.0.2"}
		if choice.Transport == config.PCK || choice.Transport == config.SPOOF || choice.Transport == "sni" {
			p.Packet = &config.AutoPacketConfig{Interface: "eth0"}
			if choice.Transport == config.SPOOF {
				p.Packet.SpoofProfile, p.Packet.IranSource, p.Packet.KharejSource = "udp", "10.79.0.1", "10.79.0.2"
			}
			if choice.Transport == "sni" {
				p.Packet.SNIDomain = "www.apple.com"
			}
		}
		a.Candidates = append(a.Candidates, p)
	}
	const token = "disposable-enrollment-secret-0123456789"
	for _, role := range []string{"iran", "kharej"} {
		c := &config.Config{Auto: a}
		if role == "iran" {
			c.Server = config.ServerConfig{Transport: config.UDP, BindAddr: "0.0.0.0:0", Token: token, SkipOptz: true}
		} else {
			c.Client = config.ClientConfig{Transport: config.UDP, RemoteAddr: "10.79.0.1:24443", Token: token, SkipOptz: true}
		}
		if err := peerauto.Validate(c); err != nil {
			panic(err)
		}
		f, err := os.OpenFile(filepath.Join(dir, role+".toml"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		if err = toml.NewEncoder(f).Encode(c); err != nil {
			panic(err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}
	}
}
