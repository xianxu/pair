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
	"syscall"
)

func positivePID(pid string) (int, bool) {
	n, err := strconv.Atoi(pid)
	return n, err == nil && n > 0 && n <= 1<<31-1
}

// Alive reports whether pid names a live process. An empty, non-numeric or
// non-positive pid is never alive.
//
// A syscall, not `kill -0` in a subprocess (#220). The registry filter calls
// this once per line on an interactive path, and a ~6ms process spawn per line
// is a strange price for a signal that costs nothing.
//
// Routed through positivePID deliberately. Signal 0 is the NULL signal, so the
// immediate bug without the guard is a false positive rather than a delivered
// signal: Kill(0, 0) and Kill(-1, 0) both succeed, so a registry line reading
// `1 0` would report a live pane that does not exist. The guard also keeps the
// pid-selector semantics — 0 means the caller's process group, -1 every process
// the user owns — from ever reaching a call that passes a real signal.
// identity_darwin.go and identity_other.go already gate on it; Alive did not,
// because a subprocess `kill` merely failed on those inputs.
//
// EPERM means the process exists but belongs to another user — alive, and the
// distinction the exit-status check silently got right before.
func Alive(pid string) bool {
	n, ok := positivePID(pid)
	if !ok {
		return false
	}
	err := syscall.Kill(n, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// ProcessChildren groups a single native table by parent PID.
func ProcessChildren() map[string][]string {
	rows, err := Table()
	if err != nil {
		return nil
	}
	children := make(map[string][]string)
	for _, p := range rows {
		parent := strconv.Itoa(p.PPID)
		children[parent] = append(children[parent], strconv.Itoa(p.PID))
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

// Process is one row of Table: a pid, its parent, start identity, niceness,
// and short kernel Name, all from the same read. Name may be truncated.
// Listed PIDs whose metadata cannot be read retain only PID; their identity,
// name and terminal state are unknown. Only confirmed disappearance omits a PID.
type Process struct {
	PID, PPID int
	Identity  string
	Nice      int
	Name      string
	// Zombie means the kernel reports a terminated process, retained until reaped.
	Zombie bool
	// TTY is the controlling terminal's device number in decimal, "" for
	// none or unknown. It comes from the same read as the rest of the row
	// (Darwin e_tdev, Linux tty_nr), so a terminal's processes are one filter
	// of one table read (pair#695: the agent's tty is teardown's witness).
	TTY string
}

// TTYProcesses is every row whose controlling terminal is tty. Pure.
func TTYProcesses(table []Process, tty string) []Process {
	if tty == "" {
		return nil
	}
	var out []Process
	for _, p := range table {
		if p.TTY == tty {
			out = append(out, p)
		}
	}
	return out
}
