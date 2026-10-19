// Package execalarm replaces the current process with a command under a kernel
// alarm, retaining inherited descriptors (including the scratch lease on fd 3).
package execalarm

import (
	"fmt"
	"strconv"
)

func parse(args []string) (int, []string, error) {
	if len(args) < 3 || args[1] != "--" || args[2] == "" {
		return 0, nil, fmt.Errorf("exec-alarm: expected SECONDS -- PROGRAM [ARGS...]")
	}
	seconds, err := strconv.Atoi(args[0])
	if err != nil || seconds < 1 || seconds > 600 {
		return 0, nil, fmt.Errorf("exec-alarm: seconds must be an integer from 1 through 600")
	}
	return seconds, args[2:], nil
}
