package smux

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	upstream "github.com/xtaci/smux"
)

func TestOriginalSmuxWireCompatibility(t *testing.T) {
	for _, version := range []int{1, 2} {
		for _, localClient := range []bool{false, true} {
			t.Run(fmt.Sprintf("version=%d/client=%t", version, localClient), func(t *testing.T) {
				a, b := net.Pipe()
				cfg := DefaultConfig()
				cfg.Version = version
				oldCfg := upstream.DefaultConfig()
				oldCfg.Version = version
				var local *Session
				var old *upstream.Session
				var err error
				if localClient {
					local, err = Client(a, cfg)
				} else {
					local, err = Server(a, cfg)
				}
				if err != nil {
					t.Fatal(err)
				}
				if localClient {
					old, err = upstream.Server(b, oldCfg)
				} else {
					old, err = upstream.Client(b, oldCfg)
				}
				if err != nil {
					local.Close()
					t.Fatal(err)
				}
				t.Cleanup(func() { local.Close(); old.Close() })
				if version == 2 {
					if err := local.EnableAuthenticatedFlowControl(); err != nil {
						t.Fatal(err)
					}
				}
				x, err := local.OpenStream()
				if err != nil {
					t.Fatal(err)
				}
				y, err := old.AcceptStream()
				if err != nil {
					t.Fatal(err)
				}
				defer x.Close()
				defer y.Close()
				_ = x.SetDeadline(time.Now().Add(10 * time.Second))
				_ = y.SetDeadline(time.Now().Add(10 * time.Second))
				for _, pair := range [][2]net.Conn{{x, y}, {y, x}} {
					payload := bytes.Repeat([]byte("original-smux-compatible"), 200000)
					done := make(chan error, 1)
					go func() { _, e := pair[0].Write(payload); done <- e }()
					out := make([]byte, len(payload))
					if _, err := io.ReadFull(pair[1], out); err != nil || !bytes.Equal(out, payload) {
						t.Fatalf("wire corrupt: %v", err)
					}
					if err := <-done; err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestReceiveCreditRejectsOverflowBeforeAndAfterEnrollment(t *testing.T) {
	for _, enrolled := range []bool{false, true} {
		t.Run(fmt.Sprint(enrolled), func(t *testing.T) {
			a, b := net.Pipe()
			cfg := DefaultConfig()
			cfg.Version = 2
			cfg.KeepAliveDisabled = true
			s, err := Server(b, cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { s.Close(); a.Close() })
			if enrolled {
				if err := s.EnableAuthenticatedFlowControl(); err != nil {
					t.Fatal(err)
				}
			}
			write := func(cmd byte, p []byte) {
				t.Helper()
				var header [8]byte
				header[0] = 2
				header[1] = cmd
				binary.LittleEndian.PutUint16(header[2:], uint16(len(p)))
				binary.LittleEndian.PutUint32(header[4:], 3)
				_ = a.SetWriteDeadline(time.Now().Add(time.Second))
				if _, err := a.Write(append(header[:], p...)); err != nil {
					t.Fatal(err)
				}
			}
			write(cmdSYN, nil)
			for range 8 {
				write(cmdPSH, make([]byte, 32<<10))
			}
			write(cmdPSH, []byte{1}) // exactly one byte beyond the initial advertised credit
			select {
			case <-s.chProtoError:
				if !errors.Is(s.protoError.Load().(error), ErrFlowControlWindow) {
					t.Fatal(s.protoError.Load())
				}
			case <-time.After(time.Second):
				t.Fatal("peer exceeded a bounded stream window")
			}
		})
	}
}

func TestReceiveCreditCountersWrapWithoutGrowingWindow(t *testing.T) {
	a, _ := flowPair(t, 2)
	st := newStream(3, 32768, a)
	st.numRead = ^uint32(0) - 100
	st.received = st.numRead
	buf := defaultAllocator.Get(32768)
	if !st.pushBytes(buf) || st.received-st.numRead != 32768 {
		t.Fatal("valid modular credit was rejected")
	}
	st.recycleTokens()
	st.received = st.numRead + initialPeerWindow
	buf = defaultAllocator.Get(1)
	if st.pushBytes(buf) {
		t.Fatal("wrapped counters enlarged receive credit")
	}
	defaultAllocator.Put(buf)
}
