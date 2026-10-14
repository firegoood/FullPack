package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerPinsTheApplicationReleaseKey(t *testing.T) {
	src, err := os.ReadFile("../../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if got := shellValue(string(src), "RELEASE_PUBLIC_KEY"); got != ReleasePublicKey {
		t.Fatalf("installer and application release keys differ")
	}
	if pub, err := base64.StdEncoding.DecodeString(ReleasePublicKey); err != nil || len(pub) != ed25519.PublicKeySize {
		t.Fatal("invalid pinned release key")
	}
}

// Run the actual offline/online installer functions, including OpenSSL, before
// a binary can be extracted or run. Forged metadata must not trigger fallback.
func TestInstallerRequiresTagBoundSignaturesAndChecksums(t *testing.T) {
	for _, program := range []string{"bash", "openssl"} {
		if _, err := exec.LookPath(program); err != nil {
			t.Skipf("%s unavailable: %v", program, err)
		}
	}
	src, err := os.ReadFile("../../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(src), "# verify_asset <file>")
	end := strings.Index(string(src), "install_binary_from_tar()")
	if start < 0 || end < start {
		t.Fatal("installer verification functions missing")
	}
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"valid-offline", "valid-online", "missing-signature", "tampered-signature", "replayed-tag", "tampered-checksums", "tampered-archive", "missing-checksums", "unsupported-openssl"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			fixtures, destination, source := filepath.Join(root, "fixtures"), filepath.Join(root, "installed"), filepath.Join(root, "source")
			for _, dir := range []string{fixtures, destination, source} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			asset, tag := "fullpack_linux_amd64.tar.gz", "v1.9.1"
			archive := []byte("a fixture archive; extraction must wait for verification")
			sums := []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), asset))
			sig := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(key, ReleaseSignedMessage(tag, sums))) + "\n")
			switch name {
			case "tampered-signature":
				sig[0] ^= 1
			case "replayed-tag":
				tag = "v1.9.2"
			case "tampered-checksums":
				sums[0] ^= 1
			case "tampered-archive":
				archive[0] ^= 1
			}
			for filename, data := range map[string][]byte{asset: archive, "SHA256SUMS": sums, "SHA256SUMS.sig": sig, "RELEASE_TAG": []byte(tag + "\n")} {
				if name == "missing-signature" && filename == "SHA256SUMS.sig" || name == "missing-checksums" && filename == "SHA256SUMS" {
					continue
				}
				if err := os.WriteFile(filepath.Join(fixtures, filename), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			block := "set -eu\ninfo() { :; }; warn() { :; }; err() { echo \"$*\" >&2; }\n" + string(src[start:end])
			block += "\nfetch() { echo \"$1\" >> \"$DEST/requests\"; cp \"$FIXTURES/${1##*/}\" \"$2\"; }\n"
			block += "resolve_release_tag() { printf 'v1.9.1\\n'; }\nINSTALL_DIR=$DEST\nSCRIPT_DIR=$FIXTURES\n"
			if name == "valid-online" {
				block += "SCRIPT_DIR=$SOURCE\n"
			}
			if name == "unsupported-openssl" {
				block += "openssl() { return 1; }\n"
			}
			block += "if install_release; then echo verified > \"$DEST/accepted\"; else exit 1; fi\n"
			path := filepath.Join(root, "verify.sh")
			if err := os.WriteFile(path, []byte(block), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", filepath.ToSlash(path))
			cmd.Env = append(os.Environ(), "FIXTURES="+filepath.ToSlash(fixtures), "DEST="+filepath.ToSlash(destination), "SOURCE="+filepath.ToSlash(source), "ASSET="+asset, "REPO=firegoood/FullPack", "RELEASE_PUBLIC_KEY="+base64.StdEncoding.EncodeToString(pub), "BP_RELEASE_TAG=")
			out, err := cmd.CombinedOutput()
			good := strings.HasPrefix(name, "valid-")
			if good && err != nil || !good && err == nil {
				t.Fatalf("unexpected installation result: %v\n%s", err, out)
			}
			if !good && !strings.Contains(string(out), "Offline installation needs") {
				t.Fatalf("refusal did not come from verification: %s", out)
			}
			_, accepted := os.Stat(filepath.Join(destination, "accepted"))
			if good != (accepted == nil) {
				t.Fatal("invalid release was accepted or valid release refused")
			}
			if name == "valid-online" {
				requests, err := os.ReadFile(filepath.Join(destination, "requests"))
				if err != nil || strings.Count(string(requests), "/releases/download/v1.9.1/") != 3 || strings.Contains(string(requests), "/latest/") {
					t.Fatalf("downloads did not share one immutable release tag: %s (%v)", requests, err)
				}
			}
		})
	}
}
