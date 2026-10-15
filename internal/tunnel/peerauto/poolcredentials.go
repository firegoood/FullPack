package peerauto

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"github.com/backpack/backpack/config"
)

// Chrome/uTLS clients need a supported ECDSA TLS certificate. RFC 6979 signing
// makes its public DER reproducible at the enrolled peer without a global RNG
// override, exported private key or a certificate-verification exception.
type deterministicPoolSigner struct{ *ecdsa.PrivateKey }

func (s deterministicPoolSigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.PrivateKey.Sign(nil, digest, opts)
}

// Fresh-pair identities are purpose-separated from transport keys. Paired
// nodes already share the enrollment secret; neither existing operator keys
// nor receiver-local paths/private material are copied into enrollment links.
func poolCredential(a config.AutoConfig, p config.AutoCandidate, token, purpose string) ([]byte, error) {
	return hkdf.Key(sha256.New, []byte(token), []byte(a.TunnelID), "github.com/backpack/backpack/auto-pool/identity/v1:"+purpose+":"+p.ID+":"+p.Endpoint, 32)
}

// PoolHelperIdentity is pure: setup/export generates only matching public trust
// and paired account credentials. Private keys are materialized at runtime on
// the receiving node, never returned by this API.
func PoolHelperIdentity(a config.AutoConfig, p config.AutoCandidate, token string, port int) (*config.AutoHelperConfig, error) {
	if a.Selection == nil || a.Selection.HelperEpoch <= 0 || len(token) < 32 {
		return nil, errors.New("pool identity needs a fresh paired secret and identity epoch")
	}
	h := &config.AutoHelperConfig{Generated: true, ReversePort: port, ServerName: p.ID + ".fullpack.invalid"}
	seed, err := poolCredential(a, p, token, "account")
	if err != nil {
		return nil, err
	}
	if p.Transport == "naive" {
		h.Username = "fullpack"
		h.Password = hex.EncodeToString(seed)
	} else {
		seed[6] = (seed[6] & 15) | 64
		seed[8] = (seed[8] & 63) | 128
		h.UUID = fmt.Sprintf("%x-%x-%x-%x-%x", seed[:4], seed[4:6], seed[6:8], seed[8:10], seed[10:16])
	}
	if p.Transport == "reality" {
		h.ServerName = "www.apple.com"
		h.Target = "www.apple.com:443"
		h.ShortID = hex.EncodeToString(seed[16:24])
		keySeed, err := poolCredential(a, p, token, "reality-x25519")
		if err != nil {
			return nil, err
		}
		key, err := ecdh.X25519().NewPrivateKey(keySeed)
		if err != nil {
			return nil, err
		}
		h.PublicKey = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	} else {
		if p.Transport == "xhttp" {
			h.Path = "/fullpack/" + p.ID
		}
		p.Helper = h
		cert, _, err := poolTLSMaterial(a, p, token)
		if err != nil {
			return nil, err
		}
		h.CAPEM = string(cert)
	}
	return h, nil
}

func poolTLSMaterial(a config.AutoConfig, p config.AutoCandidate, token string) ([]byte, []byte, error) {
	var key *ecdsa.PrivateKey
	for i := range 256 {
		seed, err := poolCredential(a, p, token, fmt.Sprintf("tls-p256-%d", i))
		if err != nil {
			return nil, nil, err
		}
		key, err = ecdsa.ParseRawPrivateKey(elliptic.P256(), seed)
		if err == nil {
			break
		}
	}
	if key == nil {
		return nil, nil, errors.New("cannot derive a valid pool TLS key")
	}
	host, _, err := net.SplitHostPort(p.Endpoint)
	if err != nil {
		return nil, nil, err
	}
	if net.ParseIP(host) == nil {
		return nil, nil, errors.New("generated pool TLS requires an explicit IP endpoint")
	}
	start := time.Unix(a.Selection.HelperEpoch, 0).UTC()
	public, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return nil, nil, err
	}
	serial := sha256.Sum256(append([]byte("pool certificate serial:"), public...))
	tmpl := &x509.Certificate{SerialNumber: new(big.Int).SetBytes(serial[:16]), Subject: pkix.Name{CommonName: p.Helper.ServerName}, NotBefore: start.Add(-24 * time.Hour), NotAfter: start.AddDate(1, 0, 0), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, DNSNames: []string{p.Helper.ServerName}, IPAddresses: []net.IP{net.ParseIP(host)}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), deterministicPoolSigner{key})
	if err != nil {
		return nil, nil, err
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), nil
}

func validatePoolHelperIdentity(a config.AutoConfig, p config.AutoCandidate, token string) error {
	h := p.Helper
	if h.Certificate != "" || h.Key != "" || h.PrivateKey != "" || h.CAFile != "" {
		return errors.New("generated pool identity must not carry local private material or paths")
	}
	want, err := PoolHelperIdentity(a, p, token, h.ReversePort)
	if err != nil {
		return err
	}
	actual := *h
	actual.Binary = "" // executable is always receiver-local
	if !reflect.DeepEqual(actual, *want) {
		return errors.New("generated pool public trust/account does not match its paired enrollment")
	}
	return nil
}

func generatedHelperLocal(a config.AutoConfig, p config.AutoCandidate, token string, listen bool) (config.AutoCandidate, func(), error) {
	cleanup := func() {}
	if p.Helper == nil || !p.Helper.Generated {
		return p, cleanup, nil
	}
	h := *p.Helper
	p.Helper = &h
	tool, version := "xray", "v26.3.27"
	if p.Transport == "naive" {
		if listen {
			tool, version = "sing-box", "v1.14.2"
		} else {
			tool, version = "naive", "v154.0.8037.49-4"
		}
	}
	if h.Binary == "" {
		h.Binary = filepath.Join("/usr/local/lib/fullpack/helpers", tool, version+"-"+runtime.GOARCH, tool)
	}
	st, err := os.Stat(h.Binary)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0111 == 0 {
		return p, cleanup, fmt.Errorf("%w: install the pinned %s helper on this node", errPoolPrerequisite, tool)
	}
	if !listen {
		return p, cleanup, nil
	}
	if p.Transport == "reality" {
		seed, err := poolCredential(a, p, token, "reality-x25519")
		if err != nil {
			return p, cleanup, err
		}
		h.PrivateKey = base64.RawURLEncoding.EncodeToString(seed)
		return p, cleanup, nil
	}
	cert, key, err := poolTLSMaterial(a, p, token)
	if err != nil {
		return p, cleanup, err
	}
	dir, err := os.MkdirTemp("", "fullpack-pool-identity-*")
	if err != nil {
		return p, cleanup, err
	}
	cleanup = func() {
		os.Remove(filepath.Join(dir, "certificate.pem"))
		os.Remove(filepath.Join(dir, "key.pem"))
		os.Remove(dir)
	}
	h.Certificate, h.Key = filepath.Join(dir, "certificate.pem"), filepath.Join(dir, "key.pem")
	if err = os.WriteFile(h.Certificate, cert, 0600); err == nil {
		err = os.WriteFile(h.Key, key, 0600)
	}
	if err != nil {
		cleanup()
		return p, func() {}, err
	}
	return p, cleanup, nil
}
