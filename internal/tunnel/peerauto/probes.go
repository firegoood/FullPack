package peerauto

import (
	"context"
	"errors"
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/probe"
	"github.com/backpack/backpack/internal/tunnel/waticmp"
	"io"
	"net"
	"time"
)

type probeConnection struct {
	w       *wire
	session carrierSession
	done    chan struct{}
}
type pendingProbe struct {
	ctx                        context.Context
	candidate, peerIncarnation string
	ch                         chan probeConnection
}

func (m *Manager) deliverProbe(ctx context.Context, w *wire, session carrierSession) {
	m.mu.Lock()
	p := m.pendingProbes[w.base.Transaction]
	m.mu.Unlock()
	if p == nil {
		return
	}
	pc := probeConnection{w: w, session: session, done: make(chan struct{})}
	select {
	case p.ch <- pc:
	case <-ctx.Done():
		return
	case <-p.ctx.Done():
		return
	}
	select {
	case <-pc.done:
	case <-ctx.Done():
	case <-p.ctx.Done():
	}
}

func (m *Manager) openProbeControl(ctx context.Context, s carrierSession, p config.AutoCandidate, tx string) (*wire, error) {
	var c net.Conn
	var err error
	initiator := m.iran != p.ListenerIsIran()
	if initiator {
		c, err = s.Open(ctx)
	} else {
		c, err = s.Accept(ctx)
	}
	if err != nil {
		return nil, err
	}
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	if initiator {
		_, err = c.Write([]byte{0})
	} else {
		var b [1]byte
		_, err = io.ReadFull(c, b[:])
		if err == nil && b[0] != 0 {
			err = errors.New("expected probe control stream")
		}
	}
	if err != nil {
		c.Close()
		return nil, err
	}
	probeTx := tx
	if !initiator {
		probeTx = ""
	}
	w, _, err := m.authenticate(c, s, p.ID, probeTx)
	if err != nil {
		c.Close()
		return nil, err
	}
	if w.base.Transaction != tx {
		if w.packet != nil {
			w.packet.close()
		}
		c.Close()
		return nil, errors.New("probe transaction mismatch")
	}
	return w, nil
}
func probeWatConnect(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token, tx string, iran bool) (carrierSession, error) {
	a.TunnelID += ":probe:" + tx
	s := watSettings(a, p, token, iran)
	s.Token = string(keyed([]byte(token), "ephemeral-probe:", tx))
	s.MaxStreams = 2
	s.MaxBps = 256000
	s.HandshakeTimeout = 10 * time.Second
	w, e := waticmp.Connect(ctx, iran, s)
	if e != nil {
		return nil, e
	}
	return watCarrier{w}, nil
}
func (m *Manager) respondIsolatedProbe(ctx context.Context, e *candidate, tx string, probeBytes int) (digest string, release func(), err error) {
	var s carrierSession
	var w *wire
	var cleanup func()
	if !e.spec.ListenerIsIran() && e.spec.Transport != config.WATICMP {
		m.mu.Lock()
		pending := m.pendingProbes[tx]
		m.mu.Unlock()
		if pending == nil {
			return "", nil, errors.New("unapproved direct probe")
		}
		select {
		case pc := <-pending.ch:
			w, s = pc.w, pc.session
			cleanup = func() { close(pc.done) }
		case <-ctx.Done():
			return "", nil, ctx.Err()
		}
	} else if e.spec.Transport == config.WATICMP {
		s, err = probeWatConnect(ctx, m.policy, e.spec, m.token, tx, false)
	} else {
		s, err = dialCarrier(ctx, m.policy, e.spec, m.token)
	}
	if err != nil {
		return "", nil, err
	}
	if w == nil {
		w, err = m.openProbeControl(ctx, s, e.spec, tx)
		if err != nil {
			s.Close()
			return "", nil, err
		}
		stop := afterFuncJoined(ctx, func() { w.c.Close(); s.Close() })
		cleanup = func() {
			stop()
			if w.packet != nil {
				w.packet.close()
			}
			w.c.Close()
			s.Close()
		}
	}
	release = cleanup
	keep := false
	defer func() {
		if !keep {
			cleanup()
		}
	}()
	var finishPacket func() (string, error)
	if w.packet != nil {
		finishPacket = respondPacketProbe(ctx, w.packet, probeBytes)
		defer func() { _, _ = finishPacket() }()
	}
	for _, size := range probeSizes(probeBytes) {
		digest, err = probe.RespondPaced(w.c, size)
		if err != nil {
			return "", nil, err
		}
	}
	r, err := w.receive()
	if finishPacket != nil {
		packetDigest, packetErr := finishPacket()
		if packetErr != nil {
			return "", nil, packetErr
		}
		digest = combineProbeDigest(digest, packetDigest)
	}
	if err != nil || r.Op != "PROBE_RESULT" || r.Transaction != tx || r.Digest != digest || r.Bytes != probeBytes {
		return "", nil, errors.New("isolated session integrity result rejected")
	}
	if err = w.send(message{Op: "PROBE_RESULT", Transaction: tx, Digest: digest, Bytes: probeBytes}); err != nil {
		return "", nil, err
	}
	// Keep this session alive until the main authenticated control exchange
	// confirms the authority received the final result. Wat Write is buffered:
	// closing the entire session immediately would discard its last response.
	keep = true
	return digest, release, nil
}
func probeSizes(n int) []int {
	if n == 512 {
		return []int{512}
	}
	return []int{512, 512, 512, 32768}
}
