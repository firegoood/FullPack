package peerauto

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/utils/network"
)

// Forged-source carriers always report the same real peer. Auto owns one raw
// socket per candidate and uses authenticated random conversation IDs above it
// so fresh KCP/Noise probes cannot replace the production KCP session. Manual
// spoof framing is untouched. Neither probes nor reconnects open another port.
const spoofHeader = 56 // magic(8), conversation(16), HMAC-SHA256(32)
var spoofMagic = [8]byte{'B', 'P', 'A', 'S', 'P', '0', '0', '1'}
var spoofHubs = struct {
	sync.Mutex
	hubs map[string]*spoofHub
}{hubs: make(map[string]*spoofHub)}

type spoofAddr struct {
	peer net.Addr
	id   [16]byte
}

func (a spoofAddr) Network() string { return "peer-auto-spoof" }
func (a spoofAddr) String() string  { return a.peer.String() + "/" + hex.EncodeToString(a.id[:]) }

type spoofDatagram struct {
	body []byte
	peer net.Addr
}
type spoofHub struct {
	raw       net.PacketConn
	peer      net.Addr
	key       []byte
	poolKey   string
	mu        sync.Mutex
	refs      int
	server    *spoofSocket
	clients   map[[16]byte]*spoofSocket
	admitted  map[[16]byte]bool
	retired   [256][16]byte
	retiredAt int
	done      chan struct{}
	joined    chan struct{}
	writeMu   sync.Mutex
	stopOnce  sync.Once
}
type spoofSocket struct {
	hub      *spoofHub
	id       [16]byte
	server   bool
	queue    chan spoofDatagram
	done     chan struct{}
	once     sync.Once
	mu       sync.Mutex
	deadline time.Time
	changed  chan struct{}
}

func openAutoSpoof(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string, listen bool) (net.PacketConn, net.Addr, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	var conversation [16]byte
	if !listen {
		if _, err := io.ReadFull(rand.Reader, conversation[:]); err != nil {
			return nil, nil, err
		}
	}
	c := packetCarrierConfig(a, p, token, listen)
	data, _ := json.Marshal(c)
	id := sha256.Sum256(data)
	poolKey := hex.EncodeToString(id[:])
	spoofHubs.Lock()
	defer spoofHubs.Unlock()
	h := spoofHubs.hubs[poolKey]
	if h == nil {
		profile, err := network.ParseSpoofProfile(p.Packet.SpoofProfile)
		if err != nil {
			return nil, nil, err
		}
		_, portText, _ := net.SplitHostPort(p.Endpoint)
		port, _ := strconv.Atoi(portText)
		peer := net.ParseIP(c.Spoof.SpoofPeerIP)
		raw, err := network.NewSpoofPacketConn(listen, token, network.SpoofCarrier{Port: uint16(port), Uplink: profile, Downlink: profile, SrcIP: c.Spoof.SpoofSrcIP, PeerSrcIP: c.Spoof.SpoofPeerSrcIP, Interface: c.Spoof.SpoofInterface, SockBuf: 1 << 20, MTU: 576}, peer)
		if err != nil {
			return nil, nil, err
		}
		h = &spoofHub{raw: raw, peer: &net.IPAddr{IP: peer}, key: keyed([]byte(token), "peer-auto-spoof-conversations/v1"), poolKey: poolKey, clients: make(map[[16]byte]*spoofSocket), admitted: make(map[[16]byte]bool), done: make(chan struct{}), joined: make(chan struct{})}
		spoofHubs.hubs[poolKey] = h
		go h.read()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	select {
	case <-h.done:
		return nil, nil, net.ErrClosed
	default:
	}
	if listen && h.server != nil || !listen && len(h.clients) >= 8 {
		return nil, nil, errors.New("spoof conversation resource limit")
	}
	s := &spoofSocket{hub: h, server: listen, queue: make(chan spoofDatagram, 64), done: make(chan struct{}), changed: make(chan struct{})}
	if listen {
		h.server = s
	} else {
		s.id = conversation
		h.clients[s.id] = s
	}
	h.refs++
	return s, spoofAddr{peer: h.peer, id: s.id}, nil
}

func (h *spoofHub) read() {
	defer close(h.joined)
	defer h.stopOnce.Do(func() { close(h.done) })
	b := make([]byte, 2048)
	for {
		n, peer, err := h.raw.ReadFrom(b)
		if err != nil {
			return
		}
		if n < spoofHeader || n > 1200 || string(b[:8]) != string(spoofMagic[:]) {
			continue
		}
		mac := hmac.New(sha256.New, h.key)
		mac.Write(b[:24])
		mac.Write(b[spoofHeader:n])
		if !hmac.Equal(mac.Sum(nil), b[24:spoofHeader]) {
			continue
		}
		var id [16]byte
		copy(id[:], b[8:24])
		h.mu.Lock()
		s := h.clients[id]
		if h.server != nil {
			if !h.admitted[id] {
				retired := false
				for _, old := range h.retired {
					if old == id {
						retired = true
						break
					}
				}
				if retired || len(h.admitted) >= 8 {
					h.mu.Unlock()
					continue
				}
				h.admitted[id] = true
			}
			s = h.server
		}
		if s != nil && len(s.queue) < cap(s.queue) {
			select {
			case <-s.done:
			default:
				select {
				case s.queue <- spoofDatagram{body: append([]byte(nil), b[spoofHeader:n]...), peer: spoofAddr{peer: peer, id: id}}:
				default:
				}
			}
		}
		h.mu.Unlock()
	}
}
func (s *spoofSocket) retire(peer net.Addr) {
	a, ok := peer.(spoofAddr)
	if !ok {
		return
	}
	h := s.hub
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.admitted, a.id)
	h.retired[h.retiredAt%len(h.retired)] = a.id
	h.retiredAt++
}
func (s *spoofSocket) ReadFrom(b []byte) (int, net.Addr, error) {
	for {
		s.mu.Lock()
		deadline, changed := s.deadline, s.changed
		s.mu.Unlock()
		var timeout <-chan time.Time
		var timer *time.Timer
		if !deadline.IsZero() {
			timer = time.NewTimer(time.Until(deadline))
			timeout = timer.C
		}
		var d spoofDatagram
		var err error
		select {
		case d = <-s.queue:
		case <-s.done:
			err = net.ErrClosed
		case <-s.hub.done:
			err = net.ErrClosed
		case <-timeout:
			err = os.ErrDeadlineExceeded
		case <-changed:
			if timer != nil {
				timer.Stop()
			}
			continue
		}
		if timer != nil {
			timer.Stop()
		}
		if err != nil {
			return 0, nil, err
		}
		if len(d.body) > len(b) {
			return 0, nil, io.ErrShortBuffer
		}
		return copy(b, d.body), d.peer, nil
	}
}
func (s *spoofSocket) WriteTo(b []byte, peer net.Addr) (int, error) {
	select {
	case <-s.done:
		return 0, net.ErrClosed
	default:
	}
	a, ok := peer.(spoofAddr)
	if !ok || len(b) > 1200-spoofHeader {
		return 0, errors.New("invalid spoof conversation datagram")
	}
	frame := make([]byte, spoofHeader+len(b))
	copy(frame, spoofMagic[:])
	copy(frame[8:24], a.id[:])
	copy(frame[spoofHeader:], b)
	mac := hmac.New(sha256.New, s.hub.key)
	mac.Write(frame[:24])
	mac.Write(b)
	copy(frame[24:spoofHeader], mac.Sum(nil))
	s.hub.writeMu.Lock()
	n, err := s.hub.raw.WriteTo(frame, a.peer)
	s.hub.writeMu.Unlock()
	if err == nil && n != len(frame) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return 0, err
	}
	return len(b), nil
}
func (s *spoofSocket) Close() error {
	var err error
	s.once.Do(func() {
		spoofHubs.Lock()
		h := s.hub
		h.mu.Lock()
		close(s.done)
		if s.server {
			h.server = nil
		} else {
			delete(h.clients, s.id)
		}
		h.refs--
		last := h.refs == 0
		if last {
			delete(spoofHubs.hubs, h.poolKey)
			h.stopOnce.Do(func() { close(h.done) })
		}
		h.mu.Unlock()
		if last {
			err = h.raw.Close()
			<-h.joined
		}
		spoofHubs.Unlock()
	})
	return err
}
func (s *spoofSocket) LocalAddr() net.Addr { return s.hub.raw.LocalAddr() }
func (s *spoofSocket) SetReadDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deadline = t
	close(s.changed)
	s.changed = make(chan struct{})
	return nil
}
func (s *spoofSocket) SetWriteDeadline(time.Time) error { return nil } // raw writes are bounded datagrams
func (s *spoofSocket) SetDeadline(t time.Time) error    { return s.SetReadDeadline(t) }

type retiredSpoofConn struct {
	net.Conn
	socket *spoofSocket
	once   sync.Once
}

func (c *retiredSpoofConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.socket.retire(c.RemoteAddr()) })
	return err
}
