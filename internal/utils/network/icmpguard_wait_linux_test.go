//go:build linux

package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestICMPGuardWaitsForConcurrentFirewallOwner(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "installed")
	script := "#!/bin/sh\nif test \"$1\" != -w || test \"$2\" != 5; then echo 'xtables lock held' >&2; exit 4; fi\ncase \"$3\" in\n-C) test -f '" + state + "';;\n-I) touch '" + state + "';;\n-D) rm -f '" + state + "';;\nesac\n"
	if err := os.WriteFile(filepath.Join(dir, "iptables"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	g := installXdiEchoGuard([xdiTagLen]byte{1, 2, 3, 4})
	if !g.Installed() {
		t.Fatal("concurrent firewall owner prevented scoped guard installation")
	}
	g.remove()
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("scoped rule remained after cleanup")
	}
}
