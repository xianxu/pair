//go:build linux

package procutil

import (
	"os"
	"strconv"
)

// Identity returns Linux's kernel start-time tick for this PID incarnation.
func Identity(pid string) string {
	n, ok := positivePID(pid)
	if !ok {
		return ""
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(n) + "/stat")
	if err != nil {
		return ""
	}
	p, err := parseLinuxStat(n, raw)
	if err != nil {
		return ""
	}
	return p.Identity
}
