package peerauto

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

func packetProbeCount(size int) int {
	if size == 512 {
		return 1
	}
	return 67
}
func combineProbeDigest(stream, packet string) string {
	h := sha256.Sum256([]byte("peer-auto-datagram-integrity-v1:" + stream + ":" + packet))
	return hex.EncodeToString(h[:])
}

func exchangePacketProbe(ctx context.Context, p *packetPipe, size int) (time.Duration, time.Duration, string, error) {
	count := packetProbeCount(size)
	h := sha256.New()
	var rtt time.Duration
	bulk := time.Now()
	retries := 0
	for i := 0; i < count; i++ {
		payload := make([]byte, 512)
		if _, err := rand.Read(payload); err != nil {
			return 0, 0, "", err
		}
		binary.BigEndian.PutUint16(payload, uint16(i))
		start := time.Now()
		matched := false
		for attempt := 0; attempt < 3 && !matched; attempt++ {
			if err := p.send(packetProbe, 0, payload); err != nil {
				return 0, 0, "", err
			}
			wait, cancel := context.WithTimeout(ctx, time.Second)
			for wait.Err() == nil {
				reply, err := p.receive(wait, func(g uint64) bool { return g == 0 })
				if err != nil {
					break
				}
				if bytes.Equal(reply.data, payload) {
					matched = true
					break
				}
			}
			cancel()
			if !matched {
				retries++
				if retries > 4 {
					return 0, 0, "", errors.New("datagram probe loss budget exceeded")
				}
			}
		}
		if !matched {
			return 0, 0, "", errors.New("datagram probe integrity/echo timed out")
		}
		h.Write(payload)
		if i < 3 {
			rtt += time.Since(start)
		}
		if i == 2 {
			bulk = time.Now()
		}
		// At most 128 KB/s application traffic, including the reverse echo.
		if !pause(ctx, 8*time.Millisecond) {
			return 0, 0, "", ctx.Err()
		}
	}
	return rtt, time.Since(bulk), hex.EncodeToString(h.Sum(nil)), nil
}

func respondPacketProbe(ctx context.Context, p *packetPipe, size int) func() (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	count := packetProbeCount(size)
	var seen [67]bool
	var payloads [67][512]byte
	var resultErr error
	go func() {
		defer close(done)
		for ctx.Err() == nil {
			message, err := p.receive(ctx, func(g uint64) bool { return g == 0 })
			if err != nil {
				if ctx.Err() == nil {
					resultErr = err
				}
				return
			}
			if len(message.data) != 512 {
				continue
			}
			index := int(binary.BigEndian.Uint16(message.data))
			if index >= count {
				continue
			}
			if seen[index] && !bytes.Equal(payloads[index][:], message.data) {
				resultErr = errors.New("datagram probe challenge changed")
				return
			}
			copy(payloads[index][:], message.data)
			seen[index] = true
			if err := p.send(packetProbe, 0, message.data); err != nil {
				resultErr = err
				return
			}
		}
	}()
	var once sync.Once
	return func() (string, error) {
		once.Do(func() { cancel(); <-done })
		if resultErr != nil {
			return "", resultErr
		}
		h := sha256.New()
		for i := 0; i < count; i++ {
			if !seen[i] {
				return "", errors.New("datagram probe incomplete")
			}
			h.Write(payloads[i][:])
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}
}
