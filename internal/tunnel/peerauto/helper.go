package peerauto

import (
	"context"
	"crypto/ecdh"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tunnel/naive"
	"github.com/backpack/backpack/internal/utils/network"
	"github.com/sirupsen/logrus"
)

func helperConfigs(p config.AutoCandidate) (config.ServerConfig, config.ClientConfig) {
	h := p.Helper
	target := net.JoinHostPort("127.0.0.1", strconv.Itoa(h.ReversePort))
	bind := p.LocalBind
	if bind == "" {
		bind = p.Endpoint
	}
	noUDP := false
	s := config.ServerConfig{Transport: config.TCP, BindAddr: target, AcceptUDP: &noUDP}
	c := config.ClientConfig{Transport: config.TCP, RemoteAddr: target}
	if p.Transport == "naive" {
		s.Naive = config.NaiveServerConfig{Binary: h.Binary, Listen: bind, Username: h.Username, Password: h.Password, Certificate: h.Certificate, Key: h.Key}
		c.Naive = config.NaiveClientConfig{Binary: h.Binary, Server: p.Endpoint, Username: h.Username, Password: h.Password, CAFile: h.CAFile}
	} else {
		s.Xray = config.XrayServerConfig{Binary: h.Binary, Listen: bind, Mode: string(p.Transport), UUID: h.UUID, ServerName: h.ServerName, Path: h.Path, Host: h.Host, Certificate: h.Certificate, Key: h.Key, PrivateKey: h.PrivateKey, Target: h.Target, ShortID: h.ShortID}
		c.Xray = config.XrayClientConfig{Binary: h.Binary, Server: p.Endpoint, Mode: string(p.Transport), UUID: h.UUID, ServerName: h.ServerName, Path: h.Path, Host: h.Host, CAFile: h.CAFile, PublicKey: h.PublicKey, ShortID: h.ShortID}
	}
	return s, c
}
func validateHelper(p config.AutoCandidate, iran, local bool) error {
	h := p.Helper
	if p.Transport == "naive" {
		if h.UUID != "" || h.Path != "" || h.Host != "" || h.PrivateKey != "" || h.PublicKey != "" || h.Target != "" || h.ShortID != "" {
			return errors.New("Naive does not accept Xray identity fields")
		}
	} else if h.Username != "" || h.Password != "" {
		return errors.New("Xray does not accept Naive proxy credentials")
	}
	if h.ReversePort < 1 || h.ReversePort > 65535 {
		return errors.New("HTTPS helper needs an enrolled private loopback port")
	}
	_, port, _ := net.SplitHostPort(p.Endpoint)
	if port == strconv.Itoa(h.ReversePort) {
		return errors.New("helper public and private listener ports must differ")
	}
	if len(h.CAPEM) > 8192 {
		return errors.New("helper CA exceeds its 8 KiB limit")
	}
	if h.CAPEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(h.CAPEM)) {
			return errors.New("helper CA contains no PEM certificates")
		}
		if p.Transport == "reality" {
			return errors.New("REALITY cannot use TLS CA trust")
		}
	}
	s, c := helperConfigs(p)
	if p.Transport == "naive" {
		if h.Username == "" || h.Password == "" || strings.ContainsAny(h.Username, ":\r\n") || strings.ContainsAny(h.Password, "\r\n") {
			return errors.New("valid HTTPS proxy credentials required")
		}
	} else {
		if err := naive.ValidatePublicXrayIdentity(c.Xray); err != nil {
			return err
		}
	}
	if !local {
		return nil
	}
	listen := iran == p.ListenerIsIran()
	if listen {
		if p.Transport == "reality" {
			raw, err := base64.RawURLEncoding.DecodeString(h.PrivateKey)
			if err != nil {
				return err
			}
			key, err := ecdh.X25519().NewPrivateKey(raw)
			if err != nil {
				return err
			}
			if base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()) != h.PublicKey {
				return errors.New("REALITY private key does not match enrolled public key")
			}
			return naive.ValidateXrayServer(&s)
		}
		pair, err := tls.LoadX509KeyPair(h.Certificate, h.Key)
		if err != nil {
			return err
		}
		leaf, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			return err
		}
		host, _, _ := net.SplitHostPort(p.Endpoint)
		if p.Transport != "naive" {
			host = h.ServerName
		}
		if err := leaf.VerifyHostname(host); err != nil {
			return fmt.Errorf("enrolled helper hostname: %w", err)
		}
		if h.CAPEM != "" {
			roots, intermediates := x509.NewCertPool(), x509.NewCertPool()
			roots.AppendCertsFromPEM([]byte(h.CAPEM))
			for _, raw := range pair.Certificate[1:] {
				cert, err := x509.ParseCertificate(raw)
				if err != nil {
					return err
				}
				intermediates.AddCert(cert)
			}
			if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, Intermediates: intermediates, DNSName: host}); err != nil {
				return fmt.Errorf("local certificate is not authenticated by enrolled trust: %w", err)
			}
		}
		if p.Transport == "naive" {
			return naive.ValidateServer(&s)
		}
		return naive.ValidateXrayServer(&s)
	}
	// Embedded public trust is checked above and written privately only when
	// starting the helper. Setup validation need not create a temporary file.
	if h.CAPEM != "" && h.CAFile == "" {
		c.Naive.CAFile = ""
		c.Xray.CAFile = ""
	}
	if p.Transport == "naive" {
		return naive.ValidateClient(&c)
	}
	return naive.ValidateXrayClient(&c)
}

func listenHelperCarrier(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string) (acceptCarrier, func(), error) {
	s, _ := helperConfigs(p)
	accept, closeListener, err := listenNoiseTCP(ctx, s.BindAddr, carrierSecret(a, p, token))
	if err != nil {
		return nil, nil, err
	}
	logger := logrus.New()
	var h *naive.Helper
	if p.Transport == "naive" {
		h, err = naive.StartServer(ctx, &s, logger)
	} else {
		h, err = naive.StartXrayServer(ctx, &s, logger)
	}
	if err != nil {
		closeListener()
		return nil, nil, err
	}
	return accept, func() { h.Close(); closeListener() }, nil
}

type helperCarrier struct {
	carrierSession
	helper  *naive.Helper
	cleanup func()
	once    sync.Once
}

func (h *helperCarrier) Close() error {
	err := h.carrierSession.Close()
	h.once.Do(func() { h.helper.Close(); h.cleanup() })
	return err
}
func dialHelperCarrier(ctx context.Context, a config.AutoConfig, p config.AutoCandidate, token string) (carrierSession, error) {
	_, c := helperConfigs(p)
	cleanup := func() {}
	if p.Helper.CAPEM != "" {
		dir, err := os.MkdirTemp("", "fullpack-auto-trust-*")
		if err != nil {
			return nil, err
		}
		path := filepath.Join(dir, "ca.pem")
		cleanup = func() { os.Remove(path); os.Remove(dir) }
		if err := os.WriteFile(path, []byte(p.Helper.CAPEM), 0600); err != nil {
			cleanup()
			return nil, err
		}
		if p.Transport == "naive" {
			c.Naive.CAFile = path
		} else {
			c.Xray.CAFile = path
		}
	}
	logger := logrus.New()
	var h *naive.Helper
	var err error
	if p.Transport == "naive" {
		h, err = naive.StartClient(ctx, &c, logger)
	} else {
		h, err = naive.StartXrayClient(ctx, &c, logger)
	}
	if err != nil {
		cleanup()
		return nil, err
	}
	proxy, err := network.ParseProxy(h.ProxyURL())
	if err != nil {
		h.Close()
		cleanup()
		return nil, err
	}
	raw, err := network.TcpDialerVia(ctx, &network.Outbound{Proxy: proxy}, c.RemoteAddr, 10*time.Second, 5*time.Second, true, 1, 0, 0, 0)
	if err != nil {
		h.Close()
		cleanup()
		return nil, err
	}
	session, err := secureStream(ctx, raw, carrierSecret(a, p, token), false)
	if err != nil {
		h.Close()
		cleanup()
		return nil, err
	}
	return &helperCarrier{carrierSession: session, helper: h, cleanup: cleanup}, nil
}
