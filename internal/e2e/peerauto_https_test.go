//go:build linux

package e2e

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
)

func autoTestHelper(t *testing.T, tool string) string {
	t.Helper()
	version := map[string]string{"naive": "v154.0.8037.49-4", "sing-box": "v1.14.2", "xray": "v26.3.27"}[tool]
	path := filepath.Join("/usr/local/lib/fullpack/helpers", tool, version+"-"+runtime.GOARCH, tool)
	if _, err := os.Stat(path); err != nil {
		t.Skip("checksum-pinned optional helper unavailable: " + tool)
	}
	return path
}
func autoHTTPSIdentity(t *testing.T, tr string) *config.AutoHelperConfig {
	t.Helper()
	h := &config.AutoHelperConfig{ReversePort: freePort(t), ServerName: "localhost", UUID: "00112233-4455-4677-8899-aabbccddeeff", Path: "/auto-private"}
	if tr == "naive" {
		h.UUID, h.Path = "", ""
		h.Username, h.Password = "auto-user", "auto-helper-secret"
		h.Binary = autoTestHelper(t, "sing-box")
	} else {
		h.Binary = autoTestHelper(t, "xray")
	}
	if tr == "reality" {
		h.Path = ""
		key, err := ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		h.PrivateKey = base64.RawURLEncoding.EncodeToString(key.Bytes())
		h.PublicKey = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
		h.ShortID = "0123456789abcdef"
		cover := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
		cover.EnableHTTP2 = true
		cover.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
		cover.StartTLS()
		t.Cleanup(cover.Close)
		_, port, _ := net.SplitHostPort(cover.Listener.Addr().String())
		h.Target = "localhost:" + port
		return h
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	h.CAPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	dir := t.TempDir()
	h.Certificate, h.Key = filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	if err = os.WriteFile(h.Certificate, []byte(h.CAPEM), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(h.Key, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: private}), 0600); err != nil {
		t.Fatal(err)
	}
	return h
}
func TestPeerAutoManagedHTTPSBothDirections(t *testing.T) {
	if os.Getenv("FULLPACK_AUTO_HELPERS") != "1" {
		t.Skip("opt-in installed official helper integration")
	}
	for _, tr := range []string{"naive", "xhttp", "reality"} {
		for _, mode := range []string{"reverse", "direct"} {
			t.Run(fmt.Sprintf("%s/%s", tr, mode), func(t *testing.T) {
				testPeerCoordinatedSwitchKeepsUserListener(t, config.TransportType(tr), true, mode)
			})
		}
	}
}

func TestPeerAutoPoolGeneratedHTTPSBothDirections(t *testing.T) {
	if os.Getenv("FULLPACK_AUTO_HELPERS") != "1" {
		t.Skip("opt-in pinned helper pool identity integration")
	}
	for _, tr := range []string{"naive", "xhttp", "reality"} {
		for _, mode := range []string{"reverse", "direct"} {
			t.Run(tr+"/"+mode, func(t *testing.T) {
				testPeerCoordinatedSwitchKeepsUserListener(t, config.TransportType(tr), true, mode, "generated")
			})
		}
	}
}
