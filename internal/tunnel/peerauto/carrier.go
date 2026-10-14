package peerauto

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/waticmp"
	"github.com/backpack/backpack/internal/utils/network"
	"github.com/gorilla/websocket"
	"github.com/quic-go/quic-go"
	"github.com/xtaci/smux"
)

// Session preserves native Wat multiplexing. Only byte-stream carriers receive
// smux; QUIC and Wat never acquire a second stream multiplexer.
type carrierSession interface {
	Open(context.Context) (net.Conn, error)
	Accept(context.Context) (net.Conn, error)
	Binding(string) (string, error)
	Close() error
}
type watCarrier struct{ c *waticmp.Conn }

func (w watCarrier) Open(ctx context.Context) (net.Conn, error)   { return w.c.OpenStreamSync(ctx) }
func (w watCarrier) Accept(ctx context.Context) (net.Conn, error) { return w.c.AcceptStream(ctx) }
func (w watCarrier) Binding(_ string) (string, error)             { return w.c.Proof("peer-auto-v1"), nil }
func (w watCarrier) Close() error                                 { return w.c.CloseWithError(0, "peer endpoint stopped") }

type quicCarrier struct{ c *quic.Conn }

func (q quicCarrier) Open(ctx context.Context) (net.Conn, error) {
	s, e := q.c.OpenStreamSync(ctx)
	if e != nil {
		return nil, e
	}
	return network.NewQUICStreamConn(s, q.c), nil
}
func (q quicCarrier) Accept(ctx context.Context) (net.Conn, error) {
	s, e := q.c.AcceptStream(ctx)
	if e != nil {
		return nil, e
	}
	return network.NewQUICStreamConn(s, q.c), nil
}
func (q quicCarrier) Binding(token string) (string, error) {
	return network.QUICClientProof(q.c, token)
}
func (q quicCarrier) Close() error { return q.c.CloseWithError(0, "peer endpoint stopped") }
func (q quicCarrier) packetChannel(_ [16]byte) (packetEndpoint, error) {
	return packetEndpoint{send: q.c.SendDatagram, receive: q.c.ReceiveDatagram, close: func() {}}, nil
}

type noiseCarrier struct{ s *smux.Session }

func (s noiseCarrier) Open(ctx context.Context) (net.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.s.OpenStream()
}
func (s noiseCarrier) Accept(ctx context.Context) (net.Conn, error) {
	stop := afterFuncJoined(ctx, func() { s.s.Close() })
	defer stop()
	return s.s.AcceptStream()
}
func (s noiseCarrier) Binding(_ string) (string, error) { return "authenticated-Noise-NNpsk0", nil }
func (s noiseCarrier) Close() error                     { return s.s.Close() }
func multiplex(c net.Conn, iran bool) (carrierSession, error) {
	cfg := smux.DefaultConfig()
	cfg.MaxReceiveBuffer = 4 << 20
	cfg.MaxStreamBuffer = 128 << 10
	cfg.KeepAliveInterval = 5 * time.Second
	cfg.KeepAliveTimeout = 15 * time.Second
	var s *smux.Session
	var err error
	if iran {
		s, err = smux.Server(c, cfg)
	} else {
		s, err = smux.Client(c, cfg)
	}
	if err != nil {
		c.Close()
		return nil, err
	}
	return noiseCarrier{s}, nil
}

type acceptCarrier func(context.Context) (carrierSession, error)

func listenCarrier(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string, tlsSettings network.TLSSettings) (acceptCarrier, func(), error) {
	if p.Helper != nil {
		return listenHelperCarrier(ctx, a, p, token)
	}
	if a.Mode == "l3" && p.Transport != config.QUIC || p.Transport == config.SPOOF || p.Transport == config.TransportType("sni") {
		return listenPacketCarrier(ctx, a, p, carrierSecret(a, p, token))
	}
	bind := p.LocalBind
	if bind == "" {
		bind = p.Endpoint
	}
	switch p.Transport {
	case config.WATICMP:
		return func(c context.Context) (carrierSession, error) {
			w, e := waticmp.Connect(c, p.ListenerIsIran(), watSettings(a, p, token, p.ListenerIsIran()))
			if e != nil {
				return nil, e
			}
			return watCarrier{w}, nil
		}, func() {}, nil
	case config.QUIC:
		l, e := network.QUICListen(bind, network.QUICSettings{DatagramMode: a.Mode == "l3", IncomingStreamLimit: int64(Defaults(a).MaxConnections + 16), KeepAlivePeriod: 5 * time.Second, MaxIdleTimeout: 15 * time.Second, SO_RCVBUF: 1 << 20, SO_SNDBUF: 1 << 20})
		if e != nil {
			return nil, nil, e
		}
		return func(c context.Context) (carrierSession, error) {
			q, e := l.Accept(c)
			if e != nil {
				return nil, e
			}
			return quicCarrier{q}, nil
		}, func() { l.Close() }, nil
	case config.WSS, config.WSSMUX:
		return listenWSS(ctx, p, token, tlsSettings)
	case config.WS, config.WSMUX:
		return listenWSS(ctx, p, carrierSecret(a, p, token), tlsSettings)
	case config.TCP, config.TCPMUX:
		return listenNoiseTCP(ctx, bind, carrierSecret(a, p, token))
	case config.KCP, config.UDP, config.XDI, config.PCK:
		return listenKCP(ctx, a, p, carrierSecret(a, p, token))
	case config.STEALTH:
		return listenNoiseTCP(ctx, bind, token)
	}
	return nil, nil, fmt.Errorf("no authorized carrier adapter for %s", p.Transport)
}
func dialCarrier(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string) (carrierSession, error) {
	if p.Helper != nil {
		return dialHelperCarrier(ctx, a, p, token)
	}
	if a.Mode == "l3" && p.Transport != config.QUIC || p.Transport == config.SPOOF || p.Transport == config.TransportType("sni") {
		return dialPacketCarrier(ctx, a, p, carrierSecret(a, p, token))
	}
	switch p.Transport {
	case config.WATICMP:
		w, e := waticmp.Connect(ctx, !p.ListenerIsIran(), watSettings(a, p, token, !p.ListenerIsIran()))
		if e != nil {
			return nil, e
		}
		return watCarrier{w}, nil
	case config.QUIC:
		q, e := network.QUICDial(ctx, p.Endpoint, network.QUICSettings{DatagramMode: a.Mode == "l3", IncomingStreamLimit: int64(Defaults(a).MaxConnections + 16), DialTimeout: 10 * time.Second, KeepAlivePeriod: 5 * time.Second, MaxIdleTimeout: 15 * time.Second, SO_RCVBUF: 1 << 20, SO_SNDBUF: 1 << 20})
		if e != nil {
			return nil, e
		}
		return quicCarrier{q}, nil
	case config.WSS, config.WSSMUX:
		w, e := network.WebSocketDialer(ctx, nil, p.Endpoint, "", "/peer-auto/v1", 10*time.Second, 5*time.Second, true, token, config.WSS, false, 1, 0, 0, 0)
		if e != nil {
			return nil, e
		}
		binding, e := network.WSSConnProof(w.UnderlyingConn(), token)
		if e != nil {
			w.Close()
			return nil, e
		}
		c, e := multiplex(newWSConn(w), false)
		if e != nil {
			return nil, e
		}
		return boundCarrier{carrierSession: c, binding: binding}, nil
	case config.WS, config.WSMUX:
		u := url.URL{Scheme: "ws", Host: p.Endpoint, Path: "/peer-auto/v1/"}
		d := websocket.Dialer{HandshakeTimeout: 10 * time.Second, ReadBufferSize: 4096, WriteBufferSize: 32768}
		ws, response, e := d.DialContext(ctx, u.String(), nil)
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		if e != nil {
			return nil, e
		}
		return secureStream(ctx, newWSConn(ws), carrierSecret(a, p, token), false)
	case config.TCP, config.TCPMUX:
		r, e := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", p.Endpoint)
		if e != nil {
			return nil, e
		}
		return secureStream(ctx, r, carrierSecret(a, p, token), false)
	case config.KCP, config.UDP, config.XDI, config.PCK:
		r, e := network.KCPDialContext(ctx, p.Endpoint, carrierSecret(a, p, token), candidateKCP(p))
		if e != nil {
			return nil, e
		}
		return secureStream(ctx, r, carrierSecret(a, p, token), false)
	case config.STEALTH:
		r, e := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", p.Endpoint)
		if e != nil {
			return nil, e
		}
		stop := afterFuncJoined(ctx, func() { r.Close() })
		defer stop()
		n, e := network.NoiseClientConn(r, token, 10*time.Second)
		if e != nil {
			r.Close()
			return nil, e
		}
		return multiplex(n, false)
	}
	return nil, fmt.Errorf("no authorized carrier adapter for %s", p.Transport)
}

func carrierSecret(a config.AutoConfig, p config.AutoCandidate, token string) string {
	return fmt.Sprintf("%x", keyed([]byte(token), "github.com/backpack/backpack/peer-auto/v1/carrier:", a.TunnelID, p.ID, string(p.Transport), p.Mode))
}

func secureStream(ctx context.Context, raw net.Conn, token string, listen bool) (carrierSession, error) {
	stop := afterFuncJoined(ctx, func() { raw.Close() })
	defer stop()
	var secured net.Conn
	var err error
	if listen {
		secured, err = network.NoiseServerConn(raw, token, 10*time.Second)
	} else {
		secured, err = network.NoiseClientConn(raw, token, 10*time.Second)
	}
	if err != nil {
		raw.Close()
		return nil, err
	}
	return multiplex(secured, listen)
}

func listenNoiseTCP(ctx context.Context, bind, token string) (acceptCarrier, func(), error) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, nil, err
	}
	accept, closeAll := securedAcceptor(ctx, l.Accept, func() { l.Close() }, func(c context.Context, r net.Conn) (carrierSession, error) { return secureStream(c, r, token, true) })
	return accept, closeAll, nil
}

func candidateKCP(p config.AutoCandidate) network.KCPSettings {
	s := network.KCPSettings{MTU: 1000, Interval: 20, Resend: 2, NoDelay: 1, SndWnd: 128, RcvWnd: 128, AckNoDelay: true, SO_RCVBUF: 1 << 20, SO_SNDBUF: 1 << 20, UseICMP: p.Transport == config.XDI}
	if p.Transport == config.PCK {
		s.MTU = 600 // includes the existing raw TCP framing budget
		s.Pck = &network.PcapCarrier{}
		if p.Packet != nil {
			s.Pck.Interface = p.Packet.Interface
		}
	}
	if p.Transport == config.XDI {
		s.MTU = 576
	}
	return s
}

func listenKCP(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string) (acceptCarrier, func(), error) {
	bind := p.LocalBind
	if bind == "" {
		bind = p.Endpoint
	}
	s := candidateKCP(p)
	l, socket, err := network.KCPListen(bind, token, s)
	if err != nil {
		return nil, nil, err
	}
	accept, closeAll := securedAcceptor(ctx, func() (net.Conn, error) {
		r, err := l.AcceptKCP()
		if err == nil {
			network.ApplyKCPSettings(r, s)
		}
		return r, err
	}, func() { l.Close(); socket.Close() }, func(c context.Context, r net.Conn) (carrierSession, error) { return secureStream(c, r, token, true) })
	return accept, closeAll, nil
}

// Raw accepts must not wait behind a silent handshake. In particular, late
// KCP packets from a closed probe can create a ghost conversation. Eight
// bounded handshake workers allow the next approved probe to reach the peer,
// while joined cancellation closes every raw connection and queued session.
func securedAcceptor(parent context.Context, acceptRaw func() (net.Conn, error), closeRaw func(), secure func(context.Context, net.Conn) (carrierSession, error)) (acceptCarrier, func()) {
	ctx, cancel := context.WithCancel(parent)
	ready := make(chan carrierSession, 2)
	permits := make(chan struct{}, 8)
	var wg sync.WaitGroup
	stop := afterFuncJoined(ctx, closeRaw)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ready)
		var handshakes sync.WaitGroup
		defer handshakes.Wait()
		for ctx.Err() == nil {
			select {
			case permits <- struct{}{}:
			case <-ctx.Done():
				return
			}
			r, err := acceptRaw()
			if err != nil {
				<-permits
				cancel()
				return
			}
			handshakes.Add(1)
			go func() {
				defer handshakes.Done()
				defer func() { <-permits }()
				s, err := secure(ctx, r)
				if err != nil {
					r.Close()
					return
				}
				select {
				case ready <- s:
				case <-ctx.Done():
					s.Close()
				}
			}()
		}
	}()
	return func(c context.Context) (carrierSession, error) {
			select {
			case s, ok := <-ready:
				if !ok {
					return nil, net.ErrClosed
				}
				return s, nil
			case <-c.Done():
				return nil, c.Err()
			}
		}, func() {
			cancel()
			closeRaw()
			wg.Wait()
			stop()
			for s := range ready {
				s.Close()
			}
		}
}
