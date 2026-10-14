//go:build linux

package mssclamp

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Apply installs the clamp for one interface, for both address families.
//
// Best effort throughout: a kernel without the TCPMSS target still gives a
// working tunnel, just one that needs its MTU set by hand at the far ends.
func Apply(kind, iface string, mtu, configured int, log *logrus.Logger) {
	// First, before anything uses it. This guard used to sit below the early
	// return for Off, so calling Apply with no logger and clamping turned off
	// dereferenced nil — the one combination that skipped straight past the
	// check meant to prevent exactly that.
	if log == nil {
		log = logrus.StandardLogger()
	}
	if configured == Off {
		log.Debugf("%s: mss clamping is turned off for %s", kind, iface)
		return
	}

	for _, r := range Rules(kind, iface, mtu, configured) {
		// Swept first, so a rule left behind by a process that was killed is
		// replaced rather than duplicated.
		sweep(r)
		if out, err := runClampRule(r, "-A"); err != nil {
			// Loud, not debug. A clamp that fails to install leaves exactly the
			// fault it exists to prevent — a tunnel that passes small packets
			// and stalls large ones — and a silent failure here once meant it
			// was missing everywhere while looking fine.
			log.Warnf("%s: could not clamp mss on %s: %s %s: %v: %s",
				kind, iface, r.Cmd, strings.Join(r.Args("-A"), " "), err, strings.TrimSpace(string(out)))
			continue
		}
		log.Infof("%s: %s segments crossing %s are clamped to %d bytes", kind, r.Label, iface, r.MSS)
	}
}

// Remove takes the rules back out. Called when the tunnel stops.
func Remove(kind, iface string, mtu, configured int) {
	if configured == Off {
		return
	}
	for _, r := range Rules(kind, iface, mtu, configured) {
		sweep(r)
	}
}

// sweep deletes every copy of a rule, not just one.
//
// iptables -D removes a single matching rule, so one call leaves any duplicates
// in place. The loop stops on the first failure, which is what "there are none
// left" looks like. The bound is there so a kernel that somehow always succeeds
// cannot spin forever.
func sweep(r Rule) {
	for i := 0; i < 1024; i++ {
		if _, err := runClampRule(r, "-D"); err != nil {
			return
		}
	}
}

// Parallel tunnel shutdown shares the namespace's xtables lock. Without -w,
// a transient lock conflict looks like an absent rule and leaves it behind.
func runClampRule(r Rule, operation string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, r.Cmd, append([]string{"-w", "5"}, r.Args(operation)...)...).CombinedOutput()
}
