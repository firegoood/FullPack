package peerauto

import (
	"sync"
	"time"
)

// Control liveness must progress even while the single switching authority is
// waiting for an isolated reserve probe. Several bounded probe/abort exchanges
// can together outlast a follower's 90-second idle deadline. The independent
// loop owns no policy or durable transitions; candidate.mu still serializes
// every authenticated transaction on each control stream.
func (m *Manager) controlLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.controlTick()
		}
	}
}

func (m *Manager) controlTick() {
	// Skip, never queue behind a commit. The next tick observes its completed
	// route; an existing heartbeat batch must finish before COMMIT can start.
	if !m.controlCommit.TryRLock() {
		return
	}
	defer m.controlCommit.RUnlock()
	m.mu.Lock()
	active := m.active
	eps := make([]*candidate, 0, len(m.endpoints))
	for _, e := range m.endpoints {
		eps = append(eps, e)
	}
	m.mu.Unlock()
	var wg sync.WaitGroup
	for _, e := range m.poolControlBatch(eps, active) {
		// A probe/commit already owns this stream. Never let that wait delay
		// another candidate's heartbeat, or queue duplicate control requests.
		if e.ctx.Err() != nil || !e.mu.TryLock() {
			continue
		}
		wg.Add(1)
		go func(e *candidate) {
			defer wg.Done()
			defer e.mu.Unlock()
			r, err := e.w.ask(message{Op: "STATUS"}, "STATUS")
			if err != nil {
				if m.ctx.Err() != nil {
					return
				}
				code, detail := classifyCandidateError(err)
				m.event(e.spec.ID, "CONTROL_LOST", code+": "+detail)
				m.log.WithField("candidate", e.spec.ID).WithField("errorCode", code).Warn("peer control heartbeat failed")
				e.cancel()
				return
			}
			e.lastSeen.Store(time.Now().UnixNano())
			// Read the current route while still holding candidate.mu. A
			// concurrent successful commit must not be compared to an old
			// generation captured before this heartbeat acquired the stream.
			m.mu.Lock()
			current := m.active
			mismatch := current != nil && current.endpoint == e && (r.Active != e.spec.ID || r.Generation != current.generation)
			m.mu.Unlock()
			if mismatch {
				m.event(e.spec.ID, "CONTROL_LOST", "active generation disagrees with authenticated peer status")
				e.cancel()
			}
		}(e)
	}
	wg.Wait()
}
