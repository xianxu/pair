package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/internal/testbinary"
)

var installedBinaryCache testbinary.Cache

func installedExecutables(t *testing.T) (string, string) {
	t.Helper()
	root := repoRoot(t)
	dir, err := installedBinaryCache.Directory(func(dir string) error {
		//lint:allow test-go-build — the suite-owned cache compiles Couch once and retains it until TestMain cleanup
		couch := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(dir, "couch"), "./cmd/couch")
		couch.Dir = root
		if out, err := couch.CombinedOutput(); err != nil {
			return fmt.Errorf("build couch: %w\n%s", err, out)
		}
		fixture := exec.CommandContext(t.Context(), "go", "test", "-c", "-o", filepath.Join(dir, "terminal-fixture"), "./cmd/internal/couchcore")
		fixture.Dir = root
		if out, err := fixture.CombinedOutput(); err != nil {
			return fmt.Errorf("build terminal fixture: %w\n%s", err, out)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "couch"), filepath.Join(dir, "terminal-fixture")
}
