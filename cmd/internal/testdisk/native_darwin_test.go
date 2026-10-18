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
