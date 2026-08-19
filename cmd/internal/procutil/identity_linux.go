//go:build linux

package procutil

import (
	"os"
	"strings"
)

// Identity returns Linux's kernel start-time tick for this PID incarnation.
func Identity(pid string) string {
	if pid == "" || strings.ContainsAny(pid, "/\\") {
		return ""
	}
	raw, err := os.ReadFile("/proc/" + pid + "/stat")
	if err != nil {
		return ""
	}
	// The parenthesized command can contain spaces; field 3 begins after its
	// final ')', and starttime is field 22 (index 19 in the remainder).
	end := strings.LastIndexByte(string(raw), ')')
	if end < 0 {
		return ""
	}
	fields := strings.Fields(string(raw[end+1:]))
	if len(fields) <= 19 {
		return ""
	}
	return fields[19]
}
