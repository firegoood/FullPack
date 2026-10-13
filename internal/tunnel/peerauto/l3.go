package peerauto

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync/atomic"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/l3"
	"github.com/sirupsen/logrus"
)

type PacketStatus struct {
	Sent      uint64 `json:"sent"`
	Received  uint64 `json:"received"`
	Dropped   uint64 `json:"dropped"`
	Interface string `json:"interface,omitempty"`
}
type packetCounters struct{ sent, received, dropped atomic.Uint64 }

func l3Settings(a config.AutoConfig, token string, iran bool) l3.Config {
	local, peer := a.L3.IranIP, a.L3.KharejIP
	if !iran {
		local, peer = peer, local
	}
	peer, _, _ = strings.Cut(peer, "/")
	iface := a.L3.Interface
	if iface == "" {
		digest := sha256.Sum256([]byte(a.TunnelID))
		iface = fmt.Sprintf("fp%x", digest[:5])
	}
	return l3.Config{Mode: l3.ModeDial, Addr: "127.0.0.1:1", Token: token, Carrier: "udp", LocalIP: local, PeerIP: peer, MTU: a.L3.MTU, Iface: iface, Ports: a.Ports, AcceptUDP: a.AcceptUDP, MaxConnections: a.MaxConnections}
}
func validateAutoL3(a config.AutoConfig, token string) error {
	if a.L3.Interface != "" && !localInterface.MatchString(a.L3.Interface) {
		return errors.New("invalid local TUN interface name")
	}
	ip1, err := netip.ParsePrefix(a.L3.IranIP)
	if err != nil {
		return errors.New("Iran TUN address needs an IP/prefix")
	}
	ip2, err := netip.ParsePrefix(a.L3.KharejIP)
	if err != nil {
		return errors.New("Kharej TUN address needs an IP/prefix")
	}
	if ip1.Addr() == ip2.Addr() || ip1.Masked() != ip2.Masked() || ip1.Addr().IsUnspecified() || ip2.Addr().IsUnspecified() || ip1.Addr().IsMulticast() || ip2.Addr().IsMulticast() {
		return errors.New("TUN peers must be distinct unicast addresses in the same enrolled subnet")
	}
	if ip1.Addr().Is6() && a.L3.MTU < 1280 {
		return errors.New("IPv6 TUN needs an MTU of at least 1280")
	}
	for _, iran := range []bool{true, false} {
		cfg := l3Settings(a, token, iran)
		if err := cfg.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) startL3() error {
	cfg := l3Settings(m.policy, m.token, m.iran)
	dev, err := m.openPacketDevice(cfg, m.log)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.device = dev
	m.mu.Unlock()
	if n := dev.BatchSize(); n < 1 || n > 256 {
		dev.Close()
		return errors.New("unsupported TUN batch size")
	}
	m.launch(func() { <-m.ctx.Done(); dev.Close() })
	m.launch(m.readPackets)
	if m.iran {
		forwarder, err := l3.NewForwarder(cfg, m.log)
		if err != nil {
			return err
		}
		if forwarder != nil {
			forwarder.SetTunnelState(func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.active != nil })
			m.launch(func() {
				if err := forwarder.Run(m.ctx); err != nil {
					m.log.WithError(err).Error("Auto L3 forwarder stopped")
					m.cancel()
				}
			})
		}
	}
	return nil
}
func validIPPacket(b []byte, mtu int) bool {
	if len(b) > mtu || len(b) < 20 {
		return false
	}
	switch b[0] >> 4 {
	case 4:
		h := int(b[0]&15) * 4
		return h >= 20 && h <= len(b) && int(binary.BigEndian.Uint16(b[2:4])) == len(b)
	case 6:
		return len(b) >= 40 && int(binary.BigEndian.Uint16(b[4:6]))+40 == len(b)
	}
	return false
}
func (m *Manager) readPackets() {
	n := m.device.BatchSize()
	bufs := make([][]byte, n)
	sizes := make([]int, n)
	for i := range bufs {
		bufs[i] = make([]byte, 65535)
	} // TUN segmentation/offload contract
	for m.ctx.Err() == nil {
		count, err := m.device.Read(bufs, sizes)
		if err != nil {
			if m.ctx.Err() == nil {
				m.log.WithError(err).Error("Auto TUN read stopped")
				m.cancel()
			}
			return
		}
		if count < 0 || count > n {
			m.cancel()
			return
		}
		for i := 0; i < count; i++ {
			if sizes[i] < 0 || sizes[i] > len(bufs[i]) || !validIPPacket(bufs[i][:sizes[i]], m.policy.L3.MTU) {
				m.packets.dropped.Add(1)
				continue
			}
			m.mu.Lock()
			r := m.active
			m.mu.Unlock()
			if r == nil || r.endpoint.w.packet == nil || r.ctx.Err() != nil {
				m.packets.dropped.Add(1)
				continue
			}
			if err := r.endpoint.w.packet.send(packetIP, r.generation, bufs[i][:sizes[i]]); err != nil {
				m.packets.dropped.Add(1)
			} else {
				m.packets.sent.Add(1)
			}
		}
	}
}
func (m *Manager) receivePackets(e *candidate) {
	allowed := func(g uint64) bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.active != nil && m.active.endpoint == e && m.active.generation == g && m.active.ctx.Err() == nil
	}
	for e.ctx.Err() == nil {
		packet, err := e.w.packet.receive(e.ctx, allowed)
		if err != nil {
			return
		}
		if !validIPPacket(packet.data, m.policy.L3.MTU) {
			m.packets.dropped.Add(1)
			continue
		}
		m.mu.Lock()
		r := m.active
		if r == nil || r.endpoint != e || r.generation != packet.generation || r.ctx.Err() != nil {
			m.packets.dropped.Add(1)
			m.mu.Unlock()
			continue
		}
		// Admit the write under the route lock so a committed replacement
		// cannot inject an old generation between this check and TUN delivery.
		n, err := m.device.Write([][]byte{packet.data})
		m.mu.Unlock()
		if err != nil || n != 1 {
			m.packets.dropped.Add(1)
			if err != nil && m.ctx.Err() == nil {
				m.log.WithError(fmt.Errorf("TUN write: %w", err)).Warn("Auto packet dropped")
			}
		} else {
			m.packets.received.Add(1)
		}
	}
}

// NewWithPacketDevice uses the same production controller with a caller-owned
// device factory. It supports isolated packet integration tests without granting
// a test process access to the host TUN; New uses the native Linux factory.
func NewWithPacketDevice(ctx context.Context, cfg *config.Config, path string, open func(l3.Config, *logrus.Logger) (l3.PacketDevice, error)) (*Manager, error) {
	m, err := New(ctx, cfg, path)
	if err != nil {
		return nil, err
	}
	if open == nil {
		m.Close()
		return nil, errors.New("nil packet device factory")
	}
	m.openPacketDevice = open
	return m, nil
}
