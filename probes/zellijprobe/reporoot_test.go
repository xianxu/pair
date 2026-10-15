package zellijprobe_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/probes/zellijprobe"
)

func probeCheckout(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/xianxu/pair\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestRepoRootUsesCurrentCheckout(t *testing.T) {
	root := probeCheckout(t)
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	t.Setenv("PAIR_PROBE_REPO", "")
	got, err := zellijprobe.RepoRoot()
	if err != nil || got != root {
		t.Fatalf("root=%q err=%v want %q", got, err, root)
	}
}

func TestRepoRootExplicitCheckout(t *testing.T) {
	root := probeCheckout(t)
	for _, location := range []string{t.TempDir(), probeCheckout(t)} {
		t.Run(filepath.Base(location), func(t *testing.T) {
			t.Chdir(location)
			t.Setenv("PAIR_PROBE_REPO", root)
			got, err := zellijprobe.RepoRoot()
			if err != nil || got != root {
				t.Fatalf("root=%q err=%v want explicit %q", got, err, root)
			}
		})
	}
}

func TestRepoRootInvalidOverrideDoesNotFallBack(t *testing.T) {
	t.Chdir(probeCheckout(t))
	t.Setenv("PAIR_PROBE_REPO", filepath.Join(t.TempDir(), "missing"))
	got, err := zellijprobe.RepoRoot()
	if err == nil || got != "" || !strings.Contains(err.Error(), "PAIR_PROBE_REPO") {
		t.Fatalf("root=%q err=%v", got, err)
	}
}

func TestRepoRootOutsideCheckoutExplainsOverride(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PAIR_PROBE_REPO", "")
	got, err := zellijprobe.RepoRoot()
	if err == nil || got != "" || !strings.Contains(err.Error(), "PAIR_PROBE_REPO") {
		t.Fatalf("root=%q err=%v", got, err)
	}
}

func TestRepoRootRelativeOverride(t *testing.T) {
	root := probeCheckout(t)
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	t.Setenv("PAIR_PROBE_REPO", "..")
	got, err := zellijprobe.RepoRoot()
	if err != nil || got != root {
		t.Fatalf("root=%q err=%v want %q", got, err, root)
	}
}

func TestRepoRootWrongModuleOverride(t *testing.T) {
	t.Chdir(probeCheckout(t))
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "go.mod"), []byte("module example.com/other\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_PROBE_REPO", other)
	got, err := zellijprobe.RepoRoot()
	if err == nil || got != "" || !strings.Contains(err.Error(), "PAIR_PROBE_REPO") {
		t.Fatalf("root=%q err=%v", got, err)
	}
}
