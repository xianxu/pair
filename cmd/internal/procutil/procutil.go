// Package procutil holds the tiny cross-runtime process primitives that every
// pair OSRuntime needs: liveness, incarnation identity, and command-line
// inspection. Extracting them here keeps one source of truth as the Go-migration
// ports (#93) each grow an OSRuntime.
package procutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
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

// LsofWritableNames reads regular files held writable by pid. Failure or partial
// inspection retains uncertainty as an error rather than reporting no writers.
// Each pipe is capped at 2 MiB; the caller's budget is further capped at 2s.
func LsofWritableNames(ctx context.Context, pid string) ([]string, error) {
	n, ok := positivePID(pid)
	if !ok {
		return nil, fmt.Errorf("lsof: invalid PID %q", pid)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pid = strconv.Itoa(n)
	cmd := exec.CommandContext(ctx, "lsof", "-nP", "-a", "-p", pid, "-Fpfatn")
	// A descendant retaining the pipes must not extend the inspection budget.
	cmd.WaitDelay = 50 * time.Millisecond
	var stdout, stderr lsofOutput
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("lsof: %w", ctx.Err())
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, errors.New("lsof: output limit exceeded")
	}
	if err != nil {
		return nil, fmt.Errorf("lsof: %w: %.4096s", err, stderr.buffer.String())
	}
	if stderr.buffer.Len() != 0 {
		return nil, fmt.Errorf("lsof: incomplete inspection: %.4096s", stderr.buffer.String())
	}
	return parseLsofWritableNames(stdout.buffer.Bytes(), pid)
}

type lsofOutput struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (b *lsofOutput) Write(p []byte) (int, error) {
	remaining := maxCommandBytes - b.buffer.Len()
	if len(p) > remaining {
		b.exceeded = true
		n, _ := b.buffer.Write(p[:remaining])
		return n, errors.New("lsof: output limit exceeded")
	}
	return b.buffer.Write(p)
}

func parseLsofWritableNames(raw []byte, pid string) ([]string, error) {
	malformed := errors.New("lsof: malformed or partial file inspection")
	if len(raw) == 0 || len(raw) > maxCommandBytes || raw[len(raw)-1] != '\n' || bytes.IndexByte(raw, 0) >= 0 {
		return nil, malformed
	}
	lines := strings.Split(string(raw[:len(raw)-1]), "\n")
	if lines[0] != "p"+pid {
		return nil, malformed
	}
	var names []string
	var fd, access, typ, name, seen string
	flush := func() error {
		if fd == "" {
			return nil
		}
		if typ == "" {
			return malformed
		}
		numeric := true
		for _, c := range fd {
			if c < '0' || c > '9' {
				numeric = false
			}
		}
		if !numeric {
			switch fd {
			case "cwd", "rtd", "txt", "mem", "DEL", "ltx", "pd":
				return nil
			default:
				return malformed
			}
		}
		if typ == "REG" {
			if !filepath.IsAbs(name) || (access != "r" && access != "w" && access != "u") {
				return malformed
			}
			if access == "w" || access == "u" {
				names = append(names, name)
			}
		}
		return nil
	}
	for _, line := range lines[1:] {
		// Darwin's non-file descriptors (for example NPOLICY) can emit
		// an empty name field. REG names remain mandatory in flush.
		if len(line) == 0 || (len(line) == 1 && line != "n") {
			return nil, malformed
		}
		key, value := line[:1], line[1:]
		if key == "f" {
			if err := flush(); err != nil {
				return nil, err
			}
			fd, access, typ, name, seen = value, "", "", "", ""
			continue
		}
		if fd == "" || strings.Contains(seen, key) {
			return nil, malformed
		}
		seen += key
		switch key {
		case "a":
			if value != "r" && value != "w" && value != "u" && value != " " {
				return nil, malformed
			}
			access = value
		case "t":
			typ = value
		case "n":
			name = value
		default:
			return nil, malformed
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return names, nil
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
type Process struct {
	PID, PPID int
	Identity  string
	Nice      int
	Name      string
	// Zombie means the kernel reports a terminated process, retained until reaped.
	Zombie bool
}
