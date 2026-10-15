package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseToolingKeepsPrereleaseIdentity(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux release tooling")
	}
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make unavailable; release CI explicitly requires this regression")
	}
	recipe, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1.9.4", "v1.9.4-beta.1"} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "internal/app"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Makefile"), recipe, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "internal/app/app.go"), []byte("package app\nconst Version = \""+version+"\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(makePath, "-s", "version")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("version recipe: %s: %v", out, err)
		}
		got, err := os.ReadFile(filepath.Join(dir, "VERSION"))
		if err != nil || strings.TrimSpace(string(got)) != version {
			t.Fatalf("release suffix lost: %q: %v", got, err)
		}
	}
	workflow, err := os.ReadFile("../../.github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`if [ "$tag" != "$declared" ]`, `sudo env BP_RELEASE_TAG="$TAG" bash /tmp/fullpack-install.sh`} {
		if !strings.Contains(string(workflow), required) {
			t.Fatalf("release identity check missing: %s", required)
		}
	}
}
