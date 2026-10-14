package peerauto

import (
	"errors"
	"slices"
	"time"
)

type PoolStatus struct {
	Target   int      `json:"target"`
	RefillAt int      `json:"refillAt"`
	Healthy  int      `json:"healthy"`
	Scanning bool     `json:"scanning"`
	Selected []string `json:"selected"`
	Reason   string   `json:"reason"`
}

type poolRuntime struct {
	selected                     []string // guarded by Manager.mu, also persisted on both peers
	status                       PoolStatus
	scanning                     bool
	force                        bool
	cursor, rounds, healthCursor int
	nextSweep, lastStandbyHealth time.Time
}

func (m *Manager) poolStableLocked(e *candidate, now time.Time) bool {
	return e != nil && e.ctx.Err() == nil && !m.runtimeDisabled[e.spec.ID] &&
		e.stablePasses >= 3 && e.failedProbes == 0 && !now.Before(e.quarantineUntil) &&
		!e.lastDataPass.IsZero() && now.Sub(e.lastDataPass) <= max(2*time.Duration(m.policy.ProbeInterval)*time.Second, time.Minute)
}

func (m *Manager) poolProbePassed(e *candidate, full bool) {
	if m.policy.Selection == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e.lastDataPass = time.Now()
	e.failedProbes = 0
	e.quarantineUntil = time.Time{}
	if full {
		if time.Since(e.lastPoolSample) >= 10*time.Second {
			e.stablePasses = min(3, e.stablePasses+1)
			e.lastPoolSample = time.Now()
		}
		e.probeAt = time.Now()
	}
}

// Selected paths have frequent control checks; idle reserves get a small
// authenticated status check after 30 seconds. Checking every due reserve in
// the bounded enrollment avoids starvation under slow full-pool data probes.
func (m *Manager) poolControlBatch(eps []*candidate, active *route) []*candidate {
	if m.policy.Selection == nil {
		return eps
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []*candidate{}
	for _, e := range eps {
		if slices.Contains(m.pool.selected, e.spec.ID) || active != nil && active.endpoint == e || time.Since(time.Unix(0, e.lastSeen.Load())) >= 30*time.Second {
			out = append(out, e)
		}
	}
	return out
}

func (m *Manager) poolRecoveryOrder(eps []*candidate) []*candidate {
	if m.policy.Selection == nil {
		return eps
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []*candidate{}
	for _, id := range m.pool.selected {
		for _, e := range eps {
			if e.spec.ID == id && m.poolStableLocked(e, time.Now()) {
				out = append(out, e)
			}
		}
	}
	return out
}

func (m *Manager) refreshPool(active *route) {
	if m.policy.Selection == nil {
		return
	}
	now := time.Now()
	m.mu.Lock()
	healthy := 0
	for i := range m.status.Candidates {
		s := &m.status.Candidates[i]
		s.Selected = slices.Contains(m.pool.selected, s.ID)
		s.Stable = m.poolStableLocked(m.endpoints[s.ID], now)
		if s.Selected && s.Stable {
			healthy++
		}
	}
	fellToRefill := m.pool.status.Healthy > 2 && healthy <= 2
	if fellToRefill {
		m.pool.scanning = false
		m.pool.nextSweep = now
	}
	m.pool.status.Target, m.pool.status.RefillAt, m.pool.status.Healthy = 4, 2, healthy
	if !m.pool.scanning && (healthy <= 2 || m.pool.force) && !now.Before(m.pool.nextSweep) {
		m.pool.force = false
		m.pool.scanning = true
		m.pool.cursor = 0
		m.pool.rounds = 0
		for id, e := range m.endpoints {
			if !slices.Contains(m.pool.selected, id) && (fellToRefill || e.stablePasses >= 3 || now.Sub(e.lastDataPass) > time.Duration(m.policy.ProbeInterval)*time.Second) {
				e.stablePasses = 0
			}
		}
	}
	m.pool.status.Scanning = m.pool.scanning
	m.pool.status.Reason = "selected paths are measured; active traffic retains its agreed generation"
	if healthy <= 2 {
		m.pool.status.Reason = "insufficient confirmed stable paths; retrying only the explicitly enrolled reserve pool"
	}
	// Probe a selected standby every ten seconds, independent of application
	// backends. Pressure/budget skips never lower its confidence.
	var standby *candidate
	if now.Sub(m.pool.lastStandbyHealth) >= 10*time.Second && len(m.pool.selected) > 0 {
		for range len(m.pool.selected) {
			id := m.pool.selected[m.pool.healthCursor%len(m.pool.selected)]
			m.pool.healthCursor++
			e := m.endpoints[id]
			if e != nil && (active == nil || active.endpoint != e) && e.ctx.Err() == nil {
				standby = e
				break
			}
		}
		m.pool.lastStandbyHealth = now
	}
	m.mu.Unlock()
	if standby != nil {
		if err := m.exchangeProbe(standby, 512); err != nil {
			m.probeFailure(standby, err)
		}
	}
	if m.pressure() {
		return
	}
	// One bounded full probe per leader iteration. Three rounds separate the
	// samples; missing prerequisites remain visible, never a fabricated pass.
	m.mu.Lock()
	var sample *candidate
	wasScanning := m.pool.scanning
	activeNeedsAgreement := active != nil && active.ctx.Err() == nil && !slices.Contains(m.pool.selected, active.endpoint.spec.ID)
	if m.pool.scanning {
		for range len(m.policy.Candidates) {
			p := m.policy.Candidates[m.pool.cursor]
			e := m.endpoints[p.ID]
			if e != nil && e.ctx.Err() == nil && !m.runtimeDisabled[p.ID] && !now.Before(e.quarantineUntil) && e.stablePasses < 3 && now.Sub(e.lastPoolSample) < 10*time.Second {
				break // do not count a round before its separated sample is due
			}
			m.pool.cursor++
			if m.pool.cursor == len(m.policy.Candidates) {
				m.pool.cursor = 0
				m.pool.rounds++
			}
			if e != nil && e.ctx.Err() == nil && !m.runtimeDisabled[p.ID] && !now.Before(e.quarantineUntil) && now.Sub(e.lastPoolSample) >= 10*time.Second {
				sample = e
			}
			if m.pool.rounds >= 3 {
				m.pool.scanning = false
				m.pool.nextSweep = now.Add(time.Duration(m.policy.Selection.RetryInterval) * time.Second)
			}
			if sample != nil || !m.pool.scanning {
				break
			}
		}
	}
	m.mu.Unlock()
	if sample != nil {
		if err := m.probe(sample); err != nil {
			m.probeFailure(sample, err)
		}
	}
	if wasScanning || activeNeedsAgreement {
		m.reconcilePool(active)
	}
}

func (m *Manager) reconcilePool(active *route) {
	m.mu.Lock()
	stable := []*candidate{}
	for _, p := range m.policy.Candidates {
		if e := m.endpoints[p.ID]; m.poolStableLocked(e, time.Now()) {
			stable = append(stable, e)
		}
	}
	slices.SortStableFunc(stable, func(a, b *candidate) int {
		if median(a.samples) > median(b.samples) {
			return -1
		}
		if median(a.samples) < median(b.samples) {
			return 1
		}
		return 0
	})
	chosen := []string{}
	// Never remove an agreed production path just for a benchmark advantage.
	if active != nil && active.ctx.Err() == nil && active.endpoint.ctx.Err() == nil {
		chosen = append(chosen, active.endpoint.spec.ID)
	}
	for _, e := range stable {
		if len(chosen) == 4 {
			break
		}
		if !slices.Contains(chosen, e.spec.ID) {
			chosen = append(chosen, e.spec.ID)
		}
	}
	if slices.Equal(chosen, m.pool.selected) || len(chosen) == 0 {
		m.mu.Unlock()
		return
	}
	var control *candidate
	if active != nil && active.endpoint.ctx.Err() == nil {
		control = active.endpoint
	} else if len(stable) > 0 {
		control = stable[0]
	}
	m.mu.Unlock()
	if control == nil {
		return
	}
	st := m.store.snapshot()
	control.mu.Lock()
	proposalGeneration := max(st.Generation, control.peerGeneration)
	reply, err := control.w.ask(message{Op: "POOL_SELECT", Transaction: fresh(), Generation: proposalGeneration, Selected: chosen}, "POOL_ACK")
	control.mu.Unlock()
	if err == nil && (!slices.Equal(reply.Selected, chosen) || reply.Generation > proposalGeneration) {
		err = errControlRejected
	}
	if err != nil {
		m.event(control.spec.ID, "SKIPPED", "reserve selection awaiting authenticated peer agreement")
		return
	}
	st.Selected = append([]string(nil), chosen...)
	if err = m.store.save(st); err != nil {
		m.event(control.spec.ID, "SKIPPED", "reserve selection persistence failed; production route unchanged")
		return
	}
	m.mu.Lock()
	m.pool.selected = chosen
	m.mu.Unlock()
	m.event(control.spec.ID, "POOL_SELECTED", "both peers agreed on the measured subset of the original enrolled pool")
}

func (m *Manager) acceptPoolSelectionLocked(e *candidate, r message, st durableState) message {
	out := message{Op: "REJECT", Transaction: r.Transaction, Generation: st.Generation, Reason: "invalid pool selection"}
	if m.policy.Selection == nil || len(r.Transaction) != 64 || r.Generation < st.Generation || r.Generation > st.Generation+1024 || len(r.Selected) > 4 || len(r.Selected) == 0 {
		return out
	}
	seen := map[string]bool{}
	for _, id := range r.Selected {
		p := m.endpoints[id]
		active := m.active != nil && m.active.endpoint == p
		if seen[id] || p == nil || p.ctx.Err() != nil || m.runtimeDisabled[id] || !active && (p.stablePasses < 3 || time.Since(p.probeAt) > 2*time.Duration(m.policy.ProbeInterval)*time.Second) {
			return out
		}
		seen[id] = true
	}
	if m.active != nil && m.active.endpoint.ctx.Err() == nil && m.poolStableLocked(m.active.endpoint, time.Now()) && !seen[m.active.endpoint.spec.ID] {
		return out
	}
	st.Selected = append([]string(nil), r.Selected...)
	if err := m.store.save(st); err != nil {
		out.Reason = "durable pool selection failed"
		return out
	}
	m.pool.selected = append([]string(nil), r.Selected...)
	m.pool.status.Target, m.pool.status.RefillAt = 4, 2
	out.Op = "POOL_ACK"
	out.Selected = append([]string(nil), r.Selected...)
	return out
}

var errPoolPrerequisite = errors.New("pool carrier prerequisite unavailable")
