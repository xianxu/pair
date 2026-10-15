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
	identity, _ := nativeIdentity(n)
	return identity
}

func nativeIdentity(n int) (string, error) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(n) + "/stat")
	if err != nil {
		return "", err
	}
	p, err := parseLinuxStat(n, raw)
	if err != nil {
		return "", err
	}
	if p.Zombie {
		// Preserve the public Identity token while snapshots omit terminal rows.
		return p.Identity, os.ErrNotExist
	}
	return p.Identity, nil
}
