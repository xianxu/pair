//go:build darwin && cgo

package testdisk

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv("PAIR_TEST_DISK_HELPER"); mode != "" {
		if mode == "trace" {
			os.Exit(RunWithWriteBudget("trace-reader", 1<<20, m.Run))
		}
		capBytes, status := uint64(4096), 0
		if mode == "under" {
			capBytes = 1 << 20
		}
		if mode == "failed" {
			status = 7
		}
		os.Exit(RunWithWriteBudget("owned-child", capBytes, func() int {
			file, err := os.Create(filepath.Join(os.Getenv("HOME"), "payload"))
			if err != nil {
				panic(err)
			}
			_, writeErr := file.Write(make([]byte, 256<<10))
			syncErr, closeErr := file.Sync(), file.Close()
			if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
				panic(err)
			}
			return status
		}))
	}
	os.Exit(m.Run())
}

func TestNativeWriteBudgetRefusesActualExcess(t *testing.T) {
	for _, tt := range []struct {
		mode   string
		status int
	}{
		{"under", 0}, {"over", 1}, {"failed", 7},
	} {
		t.Run(tt.mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^$")
			cmd.Env = testpair.Environment(t, t.TempDir(), "PAIR_TEST_DISK_HELPER="+tt.mode)
			out, err := cmd.CombinedOutput()
			status := 0
			if err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				status = exit.ExitCode()
			}
			if status != tt.status || !strings.Contains(string(out), "write budget owned-child") {
				t.Fatalf("status=%d want=%d output=%s", status, tt.status, out)
			}
			if tt.mode != "under" && !strings.Contains(string(out), "exceeded") {
				t.Fatalf("excess was not reported: %s", out)
			}
		})
	}
}

// The workload only reads; cmd/go's action trace must not consume its budget.
func TestCacheTraceReadsHelper(t *testing.T) {
	if os.Getenv("PAIR_TEST_DISK_HELPER") != "trace" {
		return
	}
	for i := 0; i < 50000; i++ {
		if _, err := os.Stat(os.Getenv("HOME")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNativeWriteBudgetExcludesCacheTrace(t *testing.T) {
	root := t.TempDir()
	trace := filepath.Join(root, "cache-action-log")
	cmd := exec.Command(os.Args[0], "-test.run=^TestCacheTraceReadsHelper$", "-test.testlogfile="+trace)
	cmd.Env = testpair.Environment(t, root, "PAIR_TEST_DISK_HELPER=trace")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("read-only workload failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(trace); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("budgeted run created cache trace: %v", err)
	}
	if !strings.Contains(string(out), "test-result cache disabled") {
		t.Fatalf("missing explicit invocation policy: %s", out)
	}
}
