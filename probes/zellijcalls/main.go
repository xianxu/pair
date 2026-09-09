// probes/zellijcalls is a `zellij` that records every invocation, then runs the
// real one. Put its build directory first in PATH and every zellij subprocess a
// pair/couch startup spawns lands in a TSV.
//
// WHY A SHIM AND NOT INSTRUMENTED CALL SITES. pair spawns zellij from eight
// distinct places (osruntime.go x6, session_quiescence.go x2), so instrumenting
// "the seam" means instrumenting eight of them and silently missing the ninth
// someone adds. A shim in PATH cannot be bypassed by a call site it has never
// heard of -- the same argument as #199's paneWriter not being an io.Writer.
//
// WHY GO AND NOT A SHELL SCRIPT. The first cut was bash and read the clock with
// `python3 -c` twice per call: ~25ms per interpreter start, so ~50ms injected
// into every zellij call, against calls that take ~45ms. The instrument would
// have roughly doubled the thing it exists to measure, and worse, the injected
// cost lands OUTSIDE each recorded duration and INSIDE the wall span -- which is
// exactly the ratio the report's conclusion turns on. macOS ships bash 3.2, so
// EPOCHREALTIME is not available to fix it in shell. Measured overhead now: see
// the self-report `zellijcalls --overhead`.
//
// A probe must not perturb what it observes (pair#199).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() { os.Exit(run()) }

func run() int {
	if len(os.Args) > 1 && os.Args[1] == "--overhead" {
		return reportOverhead()
	}
	// ONLY act as a shim when invoked AS zellij. `make test-smoke` runs `go run`
	// over every probes/*/ holding a package main, which would otherwise run this
	// with no arguments -- and a bare `zellij` STARTS A SESSION, so the smoke
	// suite would leak a server on every run. The argv[0] check makes that
	// impossible rather than remembered, which is the same reason the smoke loop
	// exists instead of a hand-maintained list.
	if filepath.Base(os.Args[0]) != "zellij" {
		fmt.Printf("zellijcalls: a `zellij` shim; it only acts when invoked under that name.\n"+
			"  built as %q, so doing nothing.\n"+
			"  usage: eval \"$(probes/zellijcalls/trace.sh arm)\" then run couch\n",
			filepath.Base(os.Args[0]))
		return 0
	}
	real, err := realZellij()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zellijcalls: %v\n", err)
		return 127
	}

	start := time.Now()
	cmd := exec.Command(real, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	runErr := cmd.Run()
	elapsed := time.Since(start)

	code := 0
	if runErr != nil {
		if exit, ok := runErr.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			code = 127
		}
	}
	record(start, elapsed, code, os.Args[1:])
	return code
}

// record appends one row. A failure to log must never change what the caller
// sees: the shim is transparent or it is not a shim.
func record(start time.Time, elapsed time.Duration, code int, args []string) {
	path := os.Getenv("PAIR_ZELLIJ_TRACE")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	// %q on each arg: `action write-chars "a b"` must not re-split on the space
	// when the report groups by subcommand.
	quoted := make([]string, 0, len(args))
	for _, a := range args {
		quoted = append(quoted, fmt.Sprintf("%q", a))
	}
	fmt.Fprintf(f, "%d\t%d\t%d\t%s\n",
		start.UnixNano(), elapsed.Nanoseconds(), code, strings.Join(quoted, " "))
}

// realZellij is the first zellij in PATH that is not this shim.
func realZellij() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate self: %w", err)
	}
	selfDir, _ := filepath.EvalSymlinks(filepath.Dir(self))
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved == selfDir {
			continue
		}
		candidate := filepath.Join(dir, "zellij")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no real zellij in PATH (excluding %s)", selfDir)
}

// reportOverhead measures what this shim adds, so the report is read with the
// residual in view rather than assumed to be zero.
func reportOverhead() int {
	real, err := realZellij()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zellijcalls: %v\n", err)
		return 1
	}
	const n = 10
	var direct, shimmed time.Duration
	for i := 0; i < n; i++ {
		s := time.Now()
		_ = exec.Command(real, "--version").Run()
		direct += time.Since(s)

		s = time.Now()
		c := exec.Command(real, "--version")
		c.Stdin, c.Stdout, c.Stderr = nil, nil, nil
		_ = c.Run()
		record(s, time.Since(s), 0, []string{"--version"})
		shimmed += time.Since(s)
	}
	fmt.Printf("direct  %v/call\nshimmed %v/call\noverhead %v/call\n",
		direct/n, shimmed/n, (shimmed-direct)/n)
	return 0
}
