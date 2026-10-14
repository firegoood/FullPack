//go:build linux

package mssclamp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClampWaitsForConcurrentFirewallOwnerAndRemovesBothChains(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nif test \"$1\" != -w || test \"$2\" != 5; then echo 'xtables lock held' >&2; exit 4; fi\nstate='" + dir + "/rule.'${0##*/}'.'$6\ncase \"$5\" in\n-A) touch \"$state\";;\n-D) if test -f \"$state\"; then rm \"$state\"; else exit 1; fi;;\nesac\n"
	for _, name := range []string{"iptables", "ip6tables"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	Apply("l3", "fp-lock-test", 1400, 0, nil)
	installed, _ := filepath.Glob(filepath.Join(dir, "rule.*"))
	if len(installed) != 4 {
		t.Fatalf("lock contention prevented clamp installation: %v", installed)
	}
	Remove("l3", "fp-lock-test", 1400, 0)
	left, _ := filepath.Glob(filepath.Join(dir, "rule.*"))
	if len(left) != 0 {
		t.Fatalf("lock contention left clamp rules: %v", left)
	}
}
