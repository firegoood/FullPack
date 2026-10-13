package manage

import (
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
)

func promptAutoHelper(p config.AutoCandidate) *config.AutoHelperConfig {
	h := &config.AutoHelperConfig{Binary: tui.Prompt("Local Helper Executable (Blank = Installed Default): ")}
	// The port is paired because the helper permits exactly that loopback
	// target. It is never used as a public listener or management endpoint.
	h.ReversePort = tui.PromptInt("Private Loopback Port (Unique On Listener)", 3080)
	h.ServerName = tui.Prompt("TLS/REALITY Server Name: ")
	if p.Transport == "naive" {
		h.Username = tui.PromptDefault("HTTPS Proxy Username", "fullpack")
		h.Password = tui.Prompt("HTTPS Proxy Password (Blank On Iran Listener = Generate): ")
	} else {
		h.UUID = tui.Prompt("Shared UUID (Blank On Iran Listener = Generate): ")
		if p.Transport == "xhttp" {
			h.Path = tui.Prompt("XHTTP Private Path (Blank On Iran Listener = Generate): ")
			h.Host = tui.Prompt("XHTTP Host Override (Optional): ")
		} else {
			h.Target = tui.Prompt("REALITY TLS Cover Host:Port: ")
			h.ShortID = tui.Prompt("REALITY Short ID (Blank On Iran Listener = Generate): ")
			if p.ListenerIsIran() {
				h.PrivateKey = tui.Prompt("Local REALITY Private Key (Optional): ")
			} else {
				h.PublicKey = tui.Prompt("Kharej REALITY Public Key: ")
			}
		}
	}
	if p.Transport != "reality" {
		if p.ListenerIsIran() {
			h.Certificate = tui.Prompt("Local Certificate Path (Optional): ")
			h.Key = tui.Prompt("Local Certificate Key Path (Optional): ")
		} else {
			h.CAFile = tui.Prompt("Local Trusted CA File (Optional): ")
		}
	}
	return h
}
