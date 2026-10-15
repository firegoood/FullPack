package main

// Test-only AES-256-GCM client for the operator's installed Xray fixture.
// SIP004 framing: https://shadowsocks.org/doc/aead.html. No production transport
// or account configuration is read or changed by this tool.
import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/md5" // Shadowsocks' legacy password KDF, not a new production choice.
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

func ssMaster(password string) []byte {
	var key, previous []byte
	for len(key) < 32 {
		h := md5.New()
		h.Write(previous)
		h.Write([]byte(password))
		previous = h.Sum(nil)
		key = append(key, previous...)
	}
	return key[:32]
}

func ssCipher(master, salt []byte) (cipher.AEAD, error) {
	key, err := hkdf.Key(sha1.New, master, salt, "ss-subkey", 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func advanceNonce(n *[12]byte) {
	for i := range n {
		n[i]++
		if n[i] != 0 {
			break
		}
	}
}

type ssConn struct {
	net.Conn
	master       []byte
	reader       cipher.AEAD
	writer       cipher.AEAD
	readNonce    [12]byte
	writeNonce   [12]byte
	readPending  []byte
	writePending []byte
}

func (c *ssConn) Write(b []byte) (int, error) {
	if len(b)+len(c.writePending) > 0x3fff {
		return 0, fmt.Errorf("audit payload exceeds one AEAD chunk")
	}
	var out []byte
	if c.writer == nil {
		salt := make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return 0, err
		}
		var err error
		c.writer, err = ssCipher(c.master, salt)
		if err != nil {
			return 0, err
		}
		out = append(out, salt...)
	}
	payload := append(c.writePending, b...)
	var size [2]byte
	binary.BigEndian.PutUint16(size[:], uint16(len(payload)))
	out = c.writer.Seal(out, c.writeNonce[:], size[:], nil)
	advanceNonce(&c.writeNonce)
	out = c.writer.Seal(out, c.writeNonce[:], payload, nil)
	advanceNonce(&c.writeNonce)
	for len(out) > 0 {
		n, err := c.Conn.Write(out)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, io.ErrShortWrite
		}
		out = out[n:]
	}
	c.writePending = nil
	return len(b), nil
}

func (c *ssConn) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if c.reader == nil {
		salt := make([]byte, 32)
		if _, err := io.ReadFull(c.Conn, salt); err != nil {
			return 0, err
		}
		var err error
		c.reader, err = ssCipher(c.master, salt)
		if err != nil {
			return 0, err
		}
	}
	for len(c.readPending) == 0 {
		length := make([]byte, 2+c.reader.Overhead())
		if _, err := io.ReadFull(c.Conn, length); err != nil {
			return 0, err
		}
		plain, err := c.reader.Open(nil, c.readNonce[:], length, nil)
		if err != nil {
			return 0, err
		}
		advanceNonce(&c.readNonce)
		n := int(binary.BigEndian.Uint16(plain))
		if n > 0x3fff {
			return 0, fmt.Errorf("invalid AEAD frame size")
		}
		body := make([]byte, n+c.reader.Overhead())
		if _, err := io.ReadFull(c.Conn, body); err != nil {
			return 0, err
		}
		c.readPending, err = c.reader.Open(nil, c.readNonce[:], body, nil)
		if err != nil {
			return 0, err
		}
		advanceNonce(&c.readNonce)
	}
	n := copy(b, c.readPending)
	c.readPending = c.readPending[n:]
	return n, nil
}

func ssDestination(address string) []byte {
	a := net.TCPAddr{}
	resolved, err := net.ResolveTCPAddr("tcp4", address)
	if err == nil {
		a = *resolved
	}
	if a.IP.To4() == nil || !a.IP.IsLoopback() || a.Port < 1 {
		panic("audit destination must be IPv4 loopback")
	}
	b := append([]byte{1}, a.IP.To4()...)
	return binary.BigEndian.AppendUint16(b, uint16(a.Port))
}

func ssUDPPacket(password string, plain []byte) ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	a, err := ssCipher(ssMaster(password), salt)
	if err != nil {
		return nil, err
	}
	return a.Seal(salt, make([]byte, 12), plain, nil), nil
}

func ssUDPPlain(password string, packet []byte) ([]byte, error) {
	if len(packet) < 48 {
		return nil, io.ErrUnexpectedEOF
	}
	a, err := ssCipher(ssMaster(password), packet[:32])
	if err != nil {
		return nil, err
	}
	return a.Open(nil, make([]byte, 12), packet[32:], nil)
}
