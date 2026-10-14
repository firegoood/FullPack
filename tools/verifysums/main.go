// Command verifysums verifies the release's tag-bound checksum signature using
// the same pinned key and message as the updater, before assets are published.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/backpack/backpack/internal/app"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: verifysums <SHA256SUMS> <tag>")
		os.Exit(2)
	}
	sums, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail(err)
	}
	signature, err := os.ReadFile(os.Args[1] + ".sig")
	if err != nil {
		fail(err)
	}
	pub, err := base64.StdEncoding.DecodeString(app.ReleasePublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		fail(fmt.Errorf("invalid pinned FullPack release key"))
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil || !ed25519.Verify(pub, app.ReleaseSignedMessage(os.Args[2], sums), sig) {
		fail(fmt.Errorf("release signature does not match the pinned key and tag"))
	}
	fmt.Println("FullPack release signature verified for", os.Args[2])
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
