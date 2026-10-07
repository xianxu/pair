//go:build linux

package procutil

import (
	"os"
	"strconv"
	"strings"
)

// Table reads each process's parent and start identity from ONE read of its
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
		end := strings.LastIndexByte(string(raw), ')')
		if end < 0 {
			continue
		}
		fields := strings.Fields(string(raw[end+1:]))
		if len(fields) <= 19 {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		out = append(out, Process{PID: pid, PPID: ppid, Identity: fields[19]})
	}
	return out, nil
}
