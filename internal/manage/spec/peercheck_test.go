package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPeerConfigRoleMarkersDoNotRequireSocketPorts(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		ok         bool
	}{
		{"wat-server", "[server]\nbind_addr='0.0.0.0:0'\ntransport='waticmp'\ntoken='enrolled-secret'\nports=['2052','8880']\n[server.waticmp]\nenabled=true\n", true},
		{"wat-client", "[client]\nremote_addr='192.0.2.1:0'\ntransport='waticmp'\ntoken='enrolled-secret'\n[client.waticmp]\nenabled=true\n", true},
		{"auto-server", "[server]\nbind_addr='0.0.0.0:0'\ntransport='waticmp'\ntoken='enrolled-secret'\nports=['2052','8880']\n[auto]\nenabled=true\n", true},
		{"auto-client", "[client]\nremote_addr='192.0.2.1:0'\ntransport='waticmp'\ntoken='enrolled-secret'\n[auto]\nenabled=true\n", true},
		{"auto-l3-no-forwards", "[server]\nbind_addr='0.0.0.0:24443'\ntransport='udp'\ntoken='enrolled-secret'\n[auto]\nenabled=true\nmode='l3'\n", true},
		{"legacy-server-zero", "[server]\nbind_addr='0.0.0.0:0'\ntransport='tcp'\ntoken='legacy-secret'\nports=['2052']\n", false},
		{"legacy-client-zero", "[client]\nremote_addr='192.0.2.1:0'\ntransport='tcp'\ntoken='legacy-secret'\n", false},
		{"disabled-wat-zero", "[client]\nremote_addr='192.0.2.1:0'\ntransport='waticmp'\ntoken='enrolled-secret'\n[client.waticmp]\nenabled=false\n", false},
		{"auto-empty-host", "[client]\nremote_addr=':0'\ntransport='waticmp'\ntoken='enrolled-secret'\n[auto]\nenabled=true\n", false},
		{"auto-wildcard-peer", "[client]\nremote_addr='0.0.0.0:0'\ntransport='waticmp'\ntoken='enrolled-secret'\n[auto]\nenabled=true\n", false},
		{"auto-invalid-marker", "[server]\nbind_addr='broken:0'\ntransport='waticmp'\ntoken='enrolled-secret'\nports=['2052']\n[auto]\nenabled=true\n", false},
		{"auto-stream-no-forwards", "[server]\nbind_addr='0.0.0.0:0'\ntransport='waticmp'\ntoken='enrolled-secret'\n[auto]\nenabled=true\n", false},
		{"auto-user-port-zero", "[server]\nbind_addr='0.0.0.0:0'\ntransport='waticmp'\ntoken='enrolled-secret'\nports=['0']\n[auto]\nenabled=true\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "peer.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			problems := ValidateConfigFile(path)
			if (len(problems) == 0) != tc.ok {
				t.Fatalf("static role-marker check: problems=%v wantValid=%v", problems, tc.ok)
			}
		})
	}
}
