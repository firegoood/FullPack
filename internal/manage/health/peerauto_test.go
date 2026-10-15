package health

import (
	"github.com/backpack/backpack/internal/manage/core"
	"github.com/backpack/backpack/internal/tunnel/peerauto"
	"testing"
	"time"
)

func TestAutoRecoveryIsNeverDrivenByLegacyWatchdog(t *testing.T) {
	tunnel := core.Tunnel{Name: "auto-test", Transport: "auto"}
	if !coordinatorOwnsRecovery(tunnel) {
		t.Fatal("Auto has competing recovery owners")
	}
	if checkThroughput(tunnel, newFlowWatch(), map[string]time.Time{}) {
		t.Fatal("legacy throughput recovery operated Auto")
	}
	if coordinatorOwnsRecovery(core.Tunnel{Transport: "stealth"}) {
		t.Fatal("manual watchdog behavior changed")
	}
}

func TestAutoCapacityWarningDoesNotChangeCarrierLiveness(t *testing.T) {
	for _, code := range []string{"capacity", "queueing", "recent_errors"} {
		h := Health{}
		applyAutoStatus(&h, peerauto.Status{State: "HEALTHY", Active: "tcp", Data: &peerauto.DataStatus{Code: code, Detail: "data admission needs attention"}})
		if h.State != "online" || !h.Connected || h.DataCode != code || h.DataWarning == "" {
			t.Fatal("data failure hidden or carrier marked down", h)
		}
	}
	h := Health{}
	applyAutoStatus(&h, peerauto.Status{State: "HEALTHY", Active: "tcp", Data: &peerauto.DataStatus{Code: "available"}})
	if h.DataWarning != "" {
		t.Fatal("healthy admission warned", h)
	}
}
