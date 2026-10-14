package naive

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/backpack/backpack/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"strconv"
	"strings"
)

// ProbeRealityTLS and ProbeRealityTransport retain the setup probe's exact
// certificate/TLS/ALPN and authenticated Chrome/Vision data requirements.
func ProbeRealityTLS(ctx context.Context, target, serverName string, roots *x509.CertPool) error {
	d := tls.Dialer{NetDialer: &net.Dialer{}, Config: &tls.Config{
		ServerName: serverName, MinVersion: tls.VersionTLS13, NextProtos: []string{"h2"}, RootCAs: roots,
		CurvePreferences: []tls.CurveID{tls.X25519},
	}}
	conn, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		return err
	}
	defer conn.Close()
	state := conn.(*tls.Conn).ConnectionState()
	if state.Version != tls.VersionTLS13 || state.NegotiatedProtocol != "h2" {
		return errors.New("cover endpoint does not support TLS 1.3 and HTTP/2")
	}
	return nil
}

func ProbeRealityTransport(ctx context.Context, target, host, binary string) error {
	_, _, err := net.SplitHostPort(target)
	if err != nil {
		return err
	}
	echo, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer echo.Close()
	stopEcho := context.AfterFunc(ctx, func() { echo.Close() })
	defer stopEcho()
	done := make(chan struct{})
	go func() {
		defer close(done)
		peer, err := echo.Accept()
		if err != nil {
			return
		}
		defer peer.Close()
		stop := context.AfterFunc(ctx, func() { peer.Close() })
		defer stop()
		_, _ = io.Copy(peer, peer)
	}()
	// Join the echo worker even when helper startup or authentication fails.
	defer func() { echo.Close(); <-done }()
	reserved, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return errors.New("no free port for the REALITY cover probe")
	}
	port := reserved.Addr().(*net.TCPAddr).Port
	reserved.Close()
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	private, public := base64.RawURLEncoding.EncodeToString(privateKey.Bytes()), base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes())
	if err != nil {
		return err
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	token := hex.EncodeToString(nonce[:])
	server := config.ServerConfig{Transport: config.TCP, BindAddr: echo.Addr().String(),
		Xray: config.XrayServerConfig{Binary: binary, Mode: "reality", Listen: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
			UUID: fmt.Sprintf("%s-%s-%s-%s-%s", token[:8], token[8:12], token[12:16], token[16:20], token[20:32]), ServerName: host, PrivateKey: private, ShortID: token[:16], Target: target}}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	sh, err := StartXrayServer(ctx, &server, logger)
	if err != nil {
		return fmt.Errorf("REALITY cover server: %w", err)
	}
	defer sh.Close()
	client := config.ClientConfig{Transport: config.TCP, RemoteAddr: server.BindAddr,
		Xray: config.XrayClientConfig{Binary: binary, Mode: "reality", Server: server.Xray.Listen,
			UUID: server.Xray.UUID, ServerName: host, PublicKey: public, ShortID: server.Xray.ShortID}}
	ch, err := StartXrayClient(ctx, &client, logger)
	if err != nil {
		return fmt.Errorf("REALITY cover client: %w", err)
	}
	defer ch.Close()
	dialer, err := proxy.SOCKS5("tcp", strings.TrimPrefix(ch.ProxyURL(), "socks5://"), nil, &net.Dialer{})
	if err != nil {
		return err
	}
	conn, err := dialer.(proxy.ContextDialer).DialContext(ctx, "tcp", server.BindAddr)
	if err != nil {
		return fmt.Errorf("REALITY cover authentication: %w", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	payload := []byte(token)
	if n, err := conn.Write(payload); err != nil {
		return err
	} else if n != len(payload) {
		return io.ErrShortWrite
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		return fmt.Errorf("REALITY cover did not carry authenticated data: %w", err)
	}
	if !bytes.Equal(got, payload) {
		return errors.New("REALITY cover probe returned corrupt data")
	}
	return ctx.Err()
}
