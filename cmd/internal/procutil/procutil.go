// Package procutil holds the tiny cross-runtime process primitives that every
// pair OSRuntime needs: liveness, incarnation identity, and command-line
// inspection. Extracting them here keeps one source of truth as the Go-migration
// ports (#93) each grow an OSRuntime.
package procutil

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func positivePID(pid string) (int, bool) {
	n, err := strconv.Atoi(pid)
	return n, err == nil && n > 0
}

// Alive reports whether pid names a live process (kill -0). An empty pid is
// never alive.
func Alive(pid string) bool {
	if pid == "" {
		return false
	}
	return exec.Command("kill", "-0", pid).Run() == nil
}

// Command returns pid's full command line via `ps -p <pid> -o command=`, trimmed
// of the trailing newline. Empty string on any error (dead pid, no ps, etc.) —
// callers treat "no argv" as "not our process".
func Command(pid string) string {
	if pid == "" {
		return ""
	}
	out, err := exec.Command("ps", "-p", pid, "-o", "command=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\n")
}

func ProcessChildren() map[string][]string {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return nil
	}
	children := make(map[string][]string)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		children[fields[1]] = append(children[fields[1]], fields[0])
	}
	return children
}

func DescendantPIDs(root string, children map[string][]string) []string {
	if root == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{root: true}
	queue := []string{root}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		out = append(out, pid)
		for _, child := range children[pid] {
			if seen[child] {
				continue
			}
			seen[child] = true
			queue = append(queue, child)
		}
	}
	return out
}

func LsofNames(pid string) []string {
	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
	if err != nil {
		return nil
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			names = append(names, line[1:])
		}
	}
	return names
}

// WaitCode waits for cmd and maps the result to an exit code: 0 on success, the
// process's code on a normal non-zero exit, -1 when the failure was not an exit
// at all (a start error, a signal the runtime could not classify).
//
// One source, because it was two. couchcore.ExecRunner and ptychild.Child had
// byte-identical copies plus a one-line errors.As wrapper each -- three packages
// carrying one decision about what "the child's exit code" means. #146 M1's
// boundary review raised it twice before it was consolidated here, which is the
// signal a repeat family is meant to send.
func WaitCode(cmd *exec.Cmd) int {
	err := cmd.Wait()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}
