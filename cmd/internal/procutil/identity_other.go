//go:build !darwin && !linux

package procutil

import (
	"os/exec"
	"strconv"
	"strings"
)

// Identity is the portable fallback for platforms without a native process
// start token in this package.
func Identity(pid string) string {
	n, ok := positivePID(pid)
	if !ok {
		return ""
	}
	out, err := exec.Command("ps", "-p", strconv.Itoa(n), "-o", "lstart=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
