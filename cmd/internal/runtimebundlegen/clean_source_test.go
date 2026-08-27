package runtimebundlegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateCommandBootstrapsFromCleanTrackedSource(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	outRoot := filepath.Join(t.TempDir(), "runtime")

	cmd := exec.Command("go", "run", "./cmd/internal/runtimebundle/generatecmd",
		"--repo", repoRoot, "--out", outRoot)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "gocache"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run generatecmd from clean tracked source failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(outRoot, "manifest.json")); err != nil {
		t.Fatalf("generated manifest missing: %v", err)
	}
}
