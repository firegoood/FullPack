package peerauto

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/l3"
	"github.com/backpack/backpack/internal/utils/network"
)

var packetMagic = [8]byte{'B', 'P', 'A', 'I', 'P', '0', '0', '1'}

type packetEndpoint struct {
	send    func([]byte) error
	receive func(context.Context) ([]byte, error)
	close   func()
}
type packetTransport interface {
	packetChannel([16]byte) (packetEndpoint, error)
}

// A registered channel exists only after fresh enrolled peer authentication.
// Unknown IDs allocate nothing. KCP carries control; user IP skips KCP entirely.
type packetMux struct {
	net.PacketConn
	mu       sync.Mutex
	channels map[[16]byte]*packetInbox
	done     chan struct{}
	once     sync.Once
}
type packetInbox struct {
	queue chan []byte
	done  chan struct{}
}

func newPacketMux(c net.PacketConn) *packetMux {
	return &packetMux{PacketConn: c, channels: make(map[[16]byte]*packetInbox), done: make(chan struct{})}
}
func (m *packetMux) ReadFrom(b []byte) (int, net.Addr, error) {
	for {
		n, peer, err := m.PacketConn.ReadFrom(b)
		if err != nil {
			return n, peer, err
		}
		if n < 8 || !bytes.Equal(b[:8], packetMagic[:]) {
			return n, peer, nil
		}
		if n < 24 || n > 1024 {
			continue
		}
		var id [16]byte
		copy(id[:], b[8:24])
		m.mu.Lock()
		inbox := m.channels[id]
		if inbox != nil && len(inbox.queue) < cap(inbox.queue) {
			select {
			case <-inbox.done:
			case inbox.queue <- append([]byte(nil), b[:n]...):
			default:
			}
		}
		m.mu.Unlock()
	}
}
func (m *packetMux) Close() error {
	var err error
	m.once.Do(func() { close(m.done); err = m.PacketConn.Close() })
	return err
}
func (m *packetMux) channel(id [16]byte, peer net.Addr) (packetEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.channels) >= 4 || m.channels[id] != nil {
		return packetEndpoint{}, errors.New("packet channel resource/identity limit")
	}
	select {
	case <-m.done:
		return packetEndpoint{}, net.ErrClosed
	default:
	}
	inbox := &packetInbox{queue: make(chan []byte, 32), done: make(chan struct{})}
	m.channels[id] = inbox
	var once sync.Once
	return packetEndpoint{
		send: func(b []byte) error {
			n, err := m.PacketConn.WriteTo(b, peer)
			if err == nil && n != len(b) {
				err = io.ErrShortWrite
			}
			return err
		},
		receive: func(ctx context.Context) ([]byte, error) {
			select {
			case b := <-inbox.queue:
				return b, nil
			case <-inbox.done:
				return nil, net.ErrClosed
			case <-m.done:
				return nil, net.ErrClosed
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
		close: func() { once.Do(func() { m.mu.Lock(); delete(m.channels, id); close(inbox.done); m.mu.Unlock() }) },
	}, nil
}

type rawPacketCarrier struct {
	carrierSession
	mux      *packetMux
	peer     net.Addr
	mu       sync.Mutex
	channels []func()
	closed   bool
}

func (c *rawPacketCarrier) packetChannel(id [16]byte) (packetEndpoint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return packetEndpoint{}, net.ErrClosed
	}
	p, err := c.mux.channel(id, c.peer)
	if err == nil {
		c.channels = append(c.channels, p.close)
	}
	return p, err
}
func (c *rawPacketCarrier) Close() error {
	c.mu.Lock()
	c.closed = true
	for _, close := range c.channels {
		close()
	}
	c.channels = nil
	c.mu.Unlock()
	return c.carrierSession.Close()
}

func packetCarrierConfig(a config.AutoConfig, p config.AutoCandidate, token string, listen bool) l3.Config {
	c := l3.Config{Mode: l3.ModeDial, Addr: p.Endpoint, Token: token, Carrier: string(p.Transport), LocalIP: "192.0.2.1/30", PeerIP: "192.0.2.2", SockBuf: 1 << 20, MTU: 1280}
	if listen {
		c.Mode = l3.ModeListen
		if p.LocalBind != "" {
			c.Addr = p.LocalBind
		}
	}
	if p.Packet != nil {
		c.Pck.Interface, c.SNIDomain = p.Packet.Interface, p.Packet.SNIDomain
		iran := listen == p.ListenerIsIran()
		source, peerSource, peer := p.Packet.IranSource, p.Packet.KharejSource, p.KharejIP
		if !iran {
			source, peerSource, peer = p.Packet.KharejSource, p.Packet.IranSource, p.IranIP
		}
		c.Spoof = config.SpoofConfig{SpoofProfile: p.Packet.SpoofProfile, SpoofSrcIP: source, SpoofPeerSrcIP: peerSource, SpoofPeerIP: peer, SpoofInterface: p.Packet.Interface, SpoofSockBuf: 1 << 20}
	}
	return c
}

func listenPacketCarrier(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string) (acceptCarrier, func(), error) {
	raw, _, err := openAutoPacket(ctx, a, p, token, true)
	if err != nil {
		return nil, nil, err
	}
	mux := newPacketMux(raw)
	s := candidateKCP(p)
	s.MTU = 512 // control also fits a small path; user IP stays outside KCP
	if p.Transport == config.SPOOF {
		s.MTU = 480
	}
	s.EphemeralKey = strings.Contains(a.TunnelID, ":probe:")
	s.Pck = nil
	s.UseICMP = false
	l, err := network.KCPListenOnPacketConn(mux, token, s)
	if err != nil {
		mux.Close()
		return nil, nil, err
	}
	accept, closeAll := securedAcceptor(ctx, func() (net.Conn, error) {
		r, err := l.AcceptKCP()
		if err == nil {
			network.ApplyKCPSettings(r, s)
		}
		return r, err
	}, func() { l.Close(); mux.Close() }, func(c context.Context, r net.Conn) (carrierSession, error) {
		if socket, ok := raw.(*spoofSocket); ok {
			r = &retiredSpoofConn{Conn: r, socket: socket}
		}
		secured, err := secureStream(c, r, token, true)
		if err != nil {
			return nil, err
		}
		return &rawPacketCarrier{carrierSession: secured, mux: mux, peer: r.RemoteAddr()}, nil
	})
	return accept, closeAll, nil
}
func dialPacketCarrier(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string) (carrierSession, error) {
	raw, peer, err := openAutoPacket(ctx, a, p, token, false)
	if err != nil {
		return nil, err
	}
	mux := newPacketMux(raw)
	s := candidateKCP(p)
	s.MTU = 512
	if p.Transport == config.SPOOF {
		s.MTU = 480
	}
	s.EphemeralKey = strings.Contains(a.TunnelID, ":probe:")
	s.Pck = nil
	s.UseICMP = false
	r, err := network.KCPDialOnPacketConn(peer, mux, token, s)
	if err != nil {
		mux.Close()
		return nil, err
	}
	secured, err := secureStream(ctx, r, token, false)
	if err != nil {
		mux.Close()
		return nil, err
	}
	return &rawPacketCarrier{carrierSession: secured, mux: mux, peer: peer}, nil
}

func openAutoPacket(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string, listen bool) (net.PacketConn, net.Addr, error) {
	if p.Transport == config.SPOOF {
		return openAutoSpoof(ctx, a, p, token, listen)
	}
	return l3.OpenDatagramCarrier(ctx, packetCarrierConfig(a, p, token, listen))
}
