package peerauto

import (
	"context"
	"testing"
	"time"
)

func TestRecoveryBudgetDoesNotRemoveQualityCooldownOrRestartFencing(t *testing.T) {
	m, e, path := followerForState(t)
	now := time.Now().Truncate(time.Second)
	st := m.store.snapshot()
	st = recordCommit(st, "tcp", now.Add(-time.Minute))
	st = recordCommit(st, "wss", now.Add(-10*time.Second))
	st = recordCommit(st, "wssmux", now)
	if err := m.store.save(st); err != nil {
		t.Fatal(err)
	}
	if got := nextRecoverySwitch(st, now); !got.Equal(now.Add(5 * time.Second)) {
		t.Fatalf("second failure inherited the optional cooldown: %v", got)
	}
	if got := m.nextSwitch(st, now, false); !got.Equal(now.Add(time.Hour - 10*time.Second)) {
		t.Fatalf("emergency commits did not retain the optional hourly limit: %v", got)
	}
	e.ctx, e.cancel = context.WithCancel(m.ctx)
	m.mu.Lock()
	m.installRouteLocked(e, st.Generation, fresh())
	m.mu.Unlock()
	if got := m.Snapshot().NextSwitch; !got.Equal(m.nextSwitch(st, now, false)) {
		t.Fatal("healthy route exposed the emergency gate")
	}
	m.mu.Lock()
	m.fenceLocked()
	m.mu.Unlock()
	if got := m.Snapshot().NextSwitch; !got.Equal(nextRecoverySwitch(st, now)) {
		t.Fatal("no-route status hid the recovery eligibility")
	}
	m.Close()
	s, err := openStore(path, m.manifest)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got := nextRecoverySwitch(s.snapshot(), now); !got.Equal(nextRecoverySwitch(st, now)) {
		t.Fatal("restart replenished the recovery budget")
	}
}

func TestRecoveryStillBoundsFlappingAndExpiresTheOldestCommit(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	st := durableState{LastGood: "first"}
	for i := range 12 {
		st = recordCommit(st, string(rune('a'+i)), now.Add(time.Duration(i-12)*time.Minute))
	}
	want := now.Add(48 * time.Minute)
	if got := nextRecoverySwitch(st, now); !got.Equal(want) {
		t.Fatalf("twelve rapid failures escaped the durable ceiling: got %v want %v", got, want)
	}
	if nextRecoverySwitch(st, want).After(want) {
		t.Fatal("expired oldest commit did not release one recovery slot")
	}
	before := len(st.Switches)
	st = recordCommit(st, st.LastGood, now)
	if len(st.Switches) != before {
		t.Fatal("same-carrier reconnection consumed a switch slot")
	}
}
