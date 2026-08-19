//go:build !darwin && !linux

package procutil

import (
	"os/exec"
	"strings"
)

// Identity is the portable fallback for platforms without a native process
// start token in this package.
func Identity(pid string) string {
	if pid == "" {
		return ""
	}
	out, err := exec.Command("ps", "-p", pid, "-o", "lstart=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
