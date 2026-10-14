package peerauto

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
)

func TestPoolControlChecksDoNotStarveLargeEnrollment(t *testing.T) {
	m := &Manager{policy: Defaults(config.AutoConfig{Selection: &config.AutoSelectionConfig{}})}
	eps := []*candidate{}
	for i := range 48 {
		e := &candidate{spec: config.AutoCandidate{ID: fmt.Sprintf("path-%d", i)}}
		e.lastSeen.Store(time.Now().Add(-45 * time.Second).UnixNano())
		eps = append(eps, e)
	}
	m.pool.selected = []string{eps[0].spec.ID}
	checks := m.poolControlBatch(eps, nil)
	if len(checks) != 48 {
		t.Fatal("aged reserves can be starved past the follower timeout")
	}
	for _, e := range eps {
		e.lastSeen.Store(time.Now().UnixNano())
	}
	checks = m.poolControlBatch(eps, nil)
	if len(checks) != 1 || checks[0] != eps[0] {
		t.Fatal("fresh reserves polled as frequently as selected routes")
	}
}

func TestPoolGeneratedIdentityTrustAndPrivateRedaction(t *testing.T) {
	a := Defaults(config.AutoConfig{TunnelID: "test-pool-identity", Selection: &config.AutoSelectionConfig{HelperEpoch: time.Now().Unix()}})
	for _, tr := range []config.TransportType{"naive", "xhttp", "reality"} {
		for _, mode := range []string{"reverse", "direct"} {
			t.Run(string(tr)+"/"+mode, func(t *testing.T) {
				p := config.AutoCandidate{ID: string(tr) + "-" + mode[:1], Transport: tr, Mode: mode, Endpoint: "192.0.2.1:24443"}
				token := "test-secret-with-at-least-thirty-two-bytes"
				h, err := PoolHelperIdentity(a, p, token, 24580)
				if err != nil {
					t.Fatal(err)
				}
				p.Helper = h
				if h.PrivateKey != "" || h.Key != "" || h.Certificate != "" || h.Binary != "" {
					t.Fatal("public identity returned local/private material")
				}
				if err := validatePoolHelperIdentity(a, p, token); err != nil {
					t.Fatal(err)
				}
				if err := validatePoolHelperIdentity(a, p, token+"wrong"); err == nil {
					t.Fatal("foreign shared secret accepted")
				}
				if tr != "reality" {
					cert, key, err := poolTLSMaterial(a, p, token)
					if err != nil {
						t.Fatal(err)
					}
					pair, err := tls.X509KeyPair(cert, key)
					if err != nil {
						t.Fatal(err)
					}
					leaf, err := x509.ParseCertificate(pair.Certificate[0])
					if err != nil {
						t.Fatal(err)
					}
					roots := x509.NewCertPool()
					roots.AppendCertsFromPEM([]byte(h.CAPEM))
					if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: h.ServerName}); err != nil {
						t.Fatal(err)
					}
					if err := leaf.VerifyHostname("192.0.2.1"); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestPoolSelectionCannotBypassMeasuredPathsOrGenerationFencing(t *testing.T) {
	m, e, _ := followerForState(t)
	m.policy.Selection = Defaults(config.AutoConfig{Selection: &config.AutoSelectionConfig{}}).Selection
	e.stablePasses = 3
	e.lastDataPass = time.Now()
	st := m.store.snapshot()
	for _, ids := range [][]string{{"unknown"}, {e.spec.ID, e.spec.ID}, {"a", "b", "c", "d", "e"}} {
		if m.transition(e, message{Op: "POOL_SELECT", Transaction: fresh(), Generation: st.Generation, Selected: ids}).Op != "REJECT" {
			t.Fatal("invalid pool selection accepted")
		}
	}
	if m.transition(e, message{Op: "POOL_SELECT", Transaction: fresh(), Generation: st.Generation - 1, Selected: []string{e.spec.ID}}).Op != "REJECT" {
		t.Fatal("stale generation selected reserves")
	}
	e.stablePasses = 2
	if m.transition(e, message{Op: "POOL_SELECT", Transaction: fresh(), Generation: st.Generation, Selected: []string{e.spec.ID}}).Op != "REJECT" {
		t.Fatal("handshake/fewer than three probes qualified")
	}
	e.stablePasses = 3
	r := m.transition(e, message{Op: "POOL_SELECT", Transaction: fresh(), Generation: st.Generation + 1, Selected: []string{e.spec.ID}})
	if r.Op != "POOL_ACK" || len(r.Selected) != 1 || m.store.snapshot().Generation != st.Generation {
		t.Fatal("pool agreement changed route fencing or failed after unequal restarts")
	}
	if len(m.store.snapshot().Selected) != 1 {
		t.Fatal("selected subset not persisted")
	}
}

func TestPoolStableSkipsDoNotEraseConfidenceAndFailuresCannotPass(t *testing.T) {
	m := &Manager{policy: Defaults(config.AutoConfig{Selection: &config.AutoSelectionConfig{}}), runtimeDisabled: map[string]bool{}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := &candidate{ctx: ctx, stablePasses: 3, lastDataPass: time.Now()}
	m.probeFailure(e, errProbePressure)
	m.mu.Lock()
	stable := m.poolStableLocked(e, time.Now())
	m.mu.Unlock()
	if !stable {
		t.Fatal("pressure skip erased valid data confidence")
	}
	m.probeFailure(e, context.DeadlineExceeded)
	m.mu.Lock()
	stable = m.poolStableLocked(e, time.Now())
	m.mu.Unlock()
	if stable {
		t.Fatal("failed data probe still reported stable")
	}
	cancel()
	m.mu.Lock()
	stable = m.poolStableLocked(e, time.Now())
	m.mu.Unlock()
	if stable {
		t.Fatal("closed session reported stable")
	}
}
