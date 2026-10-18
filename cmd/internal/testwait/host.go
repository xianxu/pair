package testwait

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// HostState is best-effort failure context, never an assertion or a probe on a
// passing hot path. It does not scan processes or retain their arguments.
func HostState() string { return hostState(hostLoad) }

func hostState(load func() (string, error)) string {
	value, err := load()
	if err != nil {
		value = "unavailable: " + err.Error()
	}
	return fmt.Sprintf("host os=%s cpus=%d loadavg=%s", runtime.GOOS, runtime.NumCPU(), strings.TrimSpace(value))
}

func hostLoad() (string, error) {
	if runtime.GOOS == "linux" {
		raw, err := os.ReadFile("/proc/loadavg")
		return string(raw), err
	}
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("no load probe for %s", runtime.GOOS)
	}
	// A failed diagnostic must not turn into another unbounded wait. Its
	// timeout is printed as unavailable, never a test failure of its own.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	raw, err := exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "vm.loadavg").Output()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return string(raw), err
}
