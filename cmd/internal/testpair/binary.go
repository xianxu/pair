// Package testpair owns a real Pair executable for a suite or test process.
package testpair

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

var once sync.Once
var candidate, owned string
var buildErr error

// Run keeps a standalone fallback alive until all tests and cleanups finish.
func Run(m *testing.M) int {
	code := m.Run()
	if owned != "" {
		if err := os.RemoveAll(owned); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return code
}

func Binary(t testing.TB) string { t.Helper(); return BinaryContext(t, t.Context()) }
func At(t testing.TB, destination string) string {
	t.Helper()
	return AtContext(t, t.Context(), destination)
}

func BinaryContext(t testing.TB, ctx context.Context) string {
	t.Helper()
	if err := ctx.Err(); err != nil {
		t.Fatal(err)
	}
	if supplied := os.Getenv("PAIR_TEST_BINARY"); supplied != "" {
		info, err := os.Stat(supplied)
		if err != nil || !filepath.IsAbs(supplied) || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			t.Fatalf("PAIR_TEST_BINARY must be an absolute executable file: %s (%v)", supplied, err)
		}
		return supplied
	}
	once.Do(func() {
		owned, buildErr = os.MkdirTemp("", "pair-test-binary-")
		if buildErr != nil {
			return
		}
		candidate = filepath.Join(owned, "pair")
		buildErr = place(ctx, candidate, "")
	})
	if buildErr != nil {
		t.Fatalf("build shared Pair: %v", buildErr)
	}
	if err := ctx.Err(); err != nil {
		t.Fatal(err)
	}
	return candidate
}

func AtContext(t testing.TB, ctx context.Context, destination string) string {
	t.Helper()
	source := BinaryContext(t, ctx)
	if err := place(ctx, destination, source); err != nil {
		t.Fatalf("place shared Pair: %v", err)
	}
	return destination
}

func place(ctx context.Context, destination, source string) error {
	root, err := reporoot.Current()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "tests/lib/pair-test-binary.sh"), destination)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PAIR_TEST_BINARY="+source)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}
