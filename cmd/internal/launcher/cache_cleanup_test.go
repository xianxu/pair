package launcher

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestLauncherCacheCleanupProcess(t *testing.T) {
	if marker := os.Getenv("PAIR_LAUNCHER_CACHE_CLEANUP_MARKER"); marker != "" {
		root := os.Getenv("HOME") // The real TestMain selected this private root.
		module := filepath.Join(root, "go", "pkg", "mod", "example@v1")
		if err := os.MkdirAll(module, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example"), 0444); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(module, 0555); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(marker, []byte(root), 0600); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("PAIR_LAUNCHER_CACHE_CLEANUP_BLOCK_PARENT") == "1" {
			if err := os.Chmod(filepath.Dir(root), 0555); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, blocked := range []string{"0", "1"} {
		t.Run("blocked-parent="+blocked, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "observed-root")
			parent := t.TempDir()
			t.Cleanup(func() {
				if err := os.Chmod(parent, 0700); err != nil {
					t.Error(err)
				}
			})
			cmd := exec.CommandContext(t.Context(), executable, "-test.run=^TestLauncherCacheCleanupProcess$", "-test.count=1")
			cmd.Env = testpair.Environment(t, t.TempDir(), "TMPDIR="+parent, "PAIR_LAUNCHER_CACHE_CLEANUP_MARKER="+marker, "PAIR_LAUNCHER_CACHE_CLEANUP_BLOCK_PARENT="+blocked)
			out, runErr := cmd.CombinedOutput()
			if err := os.Chmod(parent, 0700); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(marker)
			if err != nil {
				t.Fatalf("child marker: %v; run=%v\n%s", err, runErr, out)
			}
			root := string(raw)
			t.Cleanup(func() {
				if err := testpair.RemoveCacheTree(root); err != nil {
					t.Error(err)
				}
			})
			if blocked == "1" {
				if runErr == nil || !strings.Contains(string(out), "permission denied") {
					t.Fatalf("cleanup failure did not fail TestMain: %v\n%s", runErr, out)
				}
			} else {
				if runErr != nil {
					t.Fatalf("launcher child: %v\n%s", runErr, out)
				}
				if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("TestMain retained private HOME/module cache: %s: %v", root, err)
				}
			}
		})
	}
}
