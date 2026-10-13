package peerauto

import (
	"bytes"
	"context"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	packetHeader      = 53
	packetChunk       = 384 // also fits the smallest supported raw IPv4 path
	packetIP     byte = 0
	packetProbe  byte = 1
)

// A fresh authenticated control session authorizes exactly one packet pipe.
// Directional AEAD keys, monotonically increasing nonces and a fixed replay
// window are independent from KCP/QUIC and from the production generation.
type packetPipe struct {
	endpoint   packetEndpoint
	id         [16]byte
	tx, rx     cipher.AEAD
	sendMu     sync.Mutex
	sequence   uint64
	high       uint64
	replay     [64]uint64
	assemblies [16]packetAssembly
	probe      bool
}
type packetAssembly struct {
	id, generation uint64
	kind           byte
	count, total   int
	mask           uint32
	at             time.Time
	data           [9000]byte
}
type packetMessage struct {
	kind       byte
	generation uint64
	data       []byte
}

func newPacketPipe(s carrierSession, w *wire, iran bool) (*packetPipe, error) {
	transport, ok := s.(packetTransport)
	if !ok {
		return nil, errors.New("enrolled L3 carrier has no datagram plane")
	}
	p := &packetPipe{probe: w.base.Transaction != ""}
	copy(p.id[:], keyed(w.key, "github.com/backpack/backpack/peer-auto/v1/packet-id:"))
	toIran := keyed(w.key, "github.com/backpack/backpack/peer-auto/v1/packet-key:", "to-iran")
	toKharej := keyed(w.key, "github.com/backpack/backpack/peer-auto/v1/packet-key:", "to-kharej")
	tx, rx := toIran, toKharej
	if iran {
		tx, rx = rx, tx
	}
	var err error
	p.tx, err = chacha20poly1305.New(tx)
	if err != nil {
		return nil, err
	}
	p.rx, err = chacha20poly1305.New(rx)
	if err != nil {
		return nil, err
	}
	p.endpoint, err = transport.packetChannel(p.id)
	return p, err
}
func (p *packetPipe) close() { p.endpoint.close() }
func (p *packetPipe) send(kind byte, generation uint64, data []byte) error {
	if len(data) == 0 || len(data) > 9000 || kind > packetProbe || (kind == packetProbe) != p.probe {
		return errors.New("packet purpose/size rejected")
	}
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	count := (len(data) + packetChunk - 1) / packetChunk
	if p.sequence > math.MaxUint64-uint64(count) {
		return errors.New("packet nonce exhausted; reconnect required")
	}
	id := p.sequence + 1
	for i := 0; i < count; i++ {
		p.sequence++
		b := make([]byte, packetHeader, packetHeader+packetChunk+p.tx.Overhead())
		copy(b, packetMagic[:])
		copy(b[8:24], p.id[:])
		b[24] = kind
		binary.BigEndian.PutUint64(b[25:33], p.sequence)
		binary.BigEndian.PutUint64(b[33:41], generation)
		binary.BigEndian.PutUint64(b[41:49], id)
		b[49], b[50] = byte(i), byte(count)
		binary.BigEndian.PutUint16(b[51:53], uint16(len(data)))
		var nonce [12]byte
		binary.BigEndian.PutUint64(nonce[4:], p.sequence)
		b = p.tx.Seal(b, nonce[:], data[i*packetChunk:min((i+1)*packetChunk, len(data))], b)
		if err := p.endpoint.send(b); err != nil {
			return err
		}
	}
	return nil
}

// receive has one consumer per pipe. No unauthenticated packet can advance
// replay state or allocate a reassembly. Incomplete messages use sixteen fixed
// slots and expire after one second; user datagrams are never retransmitted.
func (p *packetPipe) receive(ctx context.Context, allowed func(uint64) bool) (packetMessage, error) {
	for {
		b, err := p.endpoint.receive(ctx)
		if err != nil {
			return packetMessage{}, err
		}
		if len(b) < packetHeader+p.rx.Overhead() || len(b) > packetHeader+packetChunk+p.rx.Overhead() || !bytes.Equal(b[:8], packetMagic[:]) || !bytes.Equal(b[8:24], p.id[:]) {
			continue
		}
		kind := b[24]
		seq := binary.BigEndian.Uint64(b[25:33])
		gen := binary.BigEndian.Uint64(b[33:41])
		id := binary.BigEndian.Uint64(b[41:49])
		index, count, total := int(b[49]), int(b[50]), int(binary.BigEndian.Uint16(b[51:53]))
		if seq == 0 || id == 0 || seq < id || seq != id+uint64(index) || kind > packetProbe || (kind == packetProbe) != p.probe || total < 1 || total > 9000 || count != (total+packetChunk-1)/packetChunk || index >= count || count > 32 || !allowed(gen) {
			continue
		}
		if seq <= p.high && (p.high-seq >= 4096 || p.replay[(seq%4096)/64]&(1<<(seq%64)) != 0) {
			continue
		}
		var nonce [12]byte
		binary.BigEndian.PutUint64(nonce[4:], seq)
		plain, err := p.rx.Open(nil, nonce[:], b[packetHeader:], b[:packetHeader])
		if err != nil || len(plain) != min(packetChunk, total-index*packetChunk) {
			continue
		}
		if seq > p.high {
			if seq-p.high >= 4096 {
				p.replay = [64]uint64{}
			} else {
				for n := p.high + 1; ; n++ {
					p.replay[(n%4096)/64] &= ^(uint64(1) << (n % 64))
					if n == seq {
						break
					}
				}
			}
			p.high = seq
		}
		p.replay[(seq%4096)/64] |= uint64(1) << (seq % 64)
		if count == 1 {
			return packetMessage{kind, gen, plain}, nil
		}
		now := time.Now()
		var slot *packetAssembly
		for i := range p.assemblies {
			a := &p.assemblies[i]
			if a.id != 0 && now.Sub(a.at) > time.Second {
				a.id = 0
			}
			if a.id == id {
				slot = a
				break
			}
		}
		if slot == nil {
			for i := range p.assemblies {
				if p.assemblies[i].id == 0 {
					slot = &p.assemblies[i]
					*slot = packetAssembly{id: id, generation: gen, kind: kind, count: count, total: total, at: now}
					break
				}
			}
		}
		if slot == nil || slot.generation != gen || slot.kind != kind || slot.total != total || slot.count != count {
			continue
		}
		copy(slot.data[index*packetChunk:], plain)
		slot.mask |= 1 << index
		if slot.mask == (uint32(1)<<count)-1 {
			data := append([]byte(nil), slot.data[:total]...)
			slot.id = 0
			return packetMessage{kind, gen, data}, nil
		}
	}
}
