package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/internal/testbinary"
)

var shimBinaryCache testbinary.Cache

func TestMain(m *testing.M) {
	code := m.Run()
	if err := shimBinaryCache.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	os.Exit(code)
}

func shimExecutable(t *testing.T, destination string) string {
	t.Helper()
	dir, err := shimBinaryCache.Directory(func(dir string) error {
		//lint:allow test-go-build — the suite-owned cache compiles this shim once; all basename/PID controls alias that executable until TestMain cleanup
		cmd := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(dir, "shim"), ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build shim: %w\n%s", err, out)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "shim"), destination); err != nil {
		t.Fatal(err)
	}
	return destination
}
