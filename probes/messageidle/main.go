// probes/messageidle counts the zellij and ps processes Couch itself spawns, so
// an idle Couch's messaging cost is a number rather than an impression (#365).
//
// Built twice into one directory, as `zellij` and as `ps`, and put first in the
// PATH of the shell that launches couch. Every pair/couch call site resolves
// both through PATH, so none can bypass it. Unlike probes/zellijcalls it records
// only calls whose PARENT is couch: agents in the slots inherit the PATH and run
// ps themselves, which is not Couch's cost.
//
// It records the parent check, the start time and the argv, then execs the
// real binary. No timing: #365 counts spawns; probes/zellijcalls times them.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const parentComm = "couch"

func main() { os.Exit(run()) }

func run() int {
	name := filepath.Base(os.Args[0])
	// `make test-smoke` runs every probe with `go run` and no arguments; a bare
	// `zellij` would start a session. Only act under one of the shimmed names.
	if name != "zellij" && name != "ps" {
		fmt.Printf("messageidle: a zellij/ps shim; built as %q, so doing nothing.\n"+
			"  usage: eval \"$(probes/messageidle/run.sh arm)\" then launch couch\n", name)
		return 0
	}
	real, err := realBinary(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "messageidle: %v\n", err)
		return 127
	}
	if parentName(os.Getppid()) == parentComm {
		record(name, os.Args[1:])
	}
	cmd := exec.Command(real, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		return 127
	}
	return 0
}

// record appends one row; a failure to log never changes what the caller sees.
func record(name string, args []string) {
	path := os.Getenv("PAIR_MESSAGEIDLE_TRACE")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%d\t%s\t%s\n", time.Now().UnixNano(), name, strings.Join(args, " "))
}

// realBinary is the first name in PATH that is not this shim's directory.
func realBinary(name string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	selfDir, _ := filepath.EvalSymlinks(filepath.Dir(self))
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved == selfDir {
			continue
		}
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no real %s in PATH (excluding %s)", name, selfDir)
}
