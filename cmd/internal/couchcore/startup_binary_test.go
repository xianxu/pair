package couchcore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/internal/testbinary"
)

var startupHelperCache testbinary.Cache

func startupHelperExecutable(t *testing.T, cache *testbinary.Cache, root string, environment []string) string {
	t.Helper()
	dir, err := cache.Directory(func(dir string) error {
		buildCtx, cancelBuild := context.WithTimeout(t.Context(), 3*time.Minute)
		defer cancelBuild()
		//lint:allow test-go-build — the suite-owned cache compiles this distinct launch helper once, outside timed operations, until TestMain cleanup
		cmd := exec.CommandContext(buildCtx, "go", "build", "-o", filepath.Join(dir, "pair-launch-helper"), "./cmd/pair-launch-helper")
		cmd.Env = environment
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build launch helper: %w\n%s", err, out)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "pair-launch-helper")
}

func TestStartupHelperExecutableBuildsOnceWithoutNativeRuntime(t *testing.T) {
	var cache testbinary.Cache
	t.Cleanup(func() {
		if err := cache.Close(); err != nil {
			t.Error(err)
		}
	})
	shimDir := t.TempDir()
	calls := filepath.Join(shimDir, "builds")
	shim := "#!/bin/sh\n[ \"$1\" = build ] && [ \"$2\" = -o ] || exit 97\nprintf 'build\\n' >> \"$PAIR_TEST_HELPER_BUILD_LOG\"\nprintf '#!/bin/sh\\n' > \"$3\"\nchmod 700 \"$3\"\n"
	if err := os.WriteFile(filepath.Join(shimDir, "go"), []byte(shim), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PAIR_TEST_HELPER_BUILD_LOG", calls)
	first := startupHelperExecutable(t, &cache, shimDir, os.Environ())
	second := startupHelperExecutable(t, &cache, shimDir, os.Environ())
	if first != second || filepath.Base(first) != "pair-launch-helper" {
		t.Fatalf("different helper artifacts: %q %q", first, second)
	}
	raw, err := os.ReadFile(calls)
	if err != nil || strings.Count(string(raw), "build\n") != 1 {
		t.Fatalf("helper builds=%q err=%v", raw, err)
	}
	info, err := os.Stat(first)
	if err != nil || info.Mode().Perm()&0111 == 0 {
		t.Fatalf("helper artifact not executable: %v", err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatalf("helper outlived suite cleanup: %v", err)
	}
}
