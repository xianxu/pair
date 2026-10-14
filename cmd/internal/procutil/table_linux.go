//go:build linux

package procutil

import (
	"os"
	"strconv"
)

// Table reads each process's parent, start identity and niceness from ONE read of its
// /proc/<pid>/stat (#399 close review BR-15). Identity matches Identity's format.
func Table() ([]Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []Process
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		raw, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue // gone between the listing and the read
		}
		p, err := parseLinuxStat(pid, raw)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}
