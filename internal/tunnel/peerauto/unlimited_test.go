package peerauto

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestUnlimitedEnrollmentPreservesLegacyAndFencesDifferentCapacity(t *testing.T) {
	c := stateConfig()
	legacy := manifestHash(c.Auto)
	if Defaults(c.Auto).ConnectionCapacity() != 64 {
		t.Fatal("legacy default changed")
	}
	for _, limit := range []int{0, 1000, 4096, math.MaxInt} {
		c.Auto.ConnectionLimit = &limit
		if err := Validate(c); err != nil {
			t.Fatal(err)
		}
		if manifestHash(c.Auto) == legacy {
			t.Fatal("capacity change escaped enrollment fencing")
		}
		copy := Defaults(c.Auto)
		if copy.ConnectionLimit == c.Auto.ConnectionLimit || copy.ConnectionCapacity() != limit {
			t.Fatal("policy aliases mutable caller capacity")
		}
		w := watSettings(c.Auto, c.Auto.Candidates[0], "unlimited-test-secret-at-least-32bytes", true)
		if w.StreamLimit == nil || limit == 0 && *w.StreamLimit != 0 || limit > 0 && *w.StreamLimit < limit {
			t.Fatal("Wat kept a hidden stream ceiling")
		}
		if limit == 0 && incomingStreams(c.Auto) != 0 {
			t.Fatal("unlimited Auto did not use manual QUIC credit semantics")
		}
	}
	bad := -1
	c.Auto.ConnectionLimit = &bad
	if Validate(c) == nil {
		t.Fatal("negative connection limit accepted")
	}
	unset := c.Auto
	unset.ConnectionLimit = nil
	if manifestHash(unset) != legacy {
		t.Fatal("optional field changed legacy manifest")
	}
}

func TestUnlimitedFlowCountDoesNotSkipProbesOrReportCapacity(t *testing.T) {
	m, _, _ := followerForState(t)
	zero := 0
	m.policy.ConnectionLimit = &zero
	m.admission = &admissionGate{live: 2000, udp: 1000}
	resources.Lock()
	oldAt, oldRSS, oldCPU := resources.at, resources.rss, resources.cpu
	resources.at, resources.rss, resources.cpu = time.Now(), 0, 0
	resources.Unlock()
	t.Cleanup(func() {
		resources.Lock()
		resources.at, resources.rss, resources.cpu = oldAt, oldRSS, oldCPU
		resources.Unlock()
	})
	if m.pressure() {
		t.Fatal("application connection count became an implicit ceiling")
	}
	s := m.Snapshot()
	if !s.Flows.Unlimited || s.Flows.Limit != 0 || s.Flows.Live != 2000 || s.Data.Code != "available" {
		t.Fatalf("unlimited status: %+v", s)
	}
	resources.Lock()
	resources.at, resources.rss = time.Now(), uint64(m.policy.MaxMemoryMB+1)<<20
	resources.Unlock()
	if !m.pressure() || m.ctx.Err() != nil {
		t.Fatal("RSS guard disabled or production cancelled")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.admission.begin(ctx, true); err == nil {
		t.Fatal("shutdown admitted a new flow")
	}
}
