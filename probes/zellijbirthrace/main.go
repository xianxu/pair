// Repro driver for pair#287: how often does a new zellij session die at birth?
//
// zellij 0.45.1 panics (`zellij-server/src/lib.rs:1462`, an unwrap on
// `session_data` in `ServerInstruction::RemoveClient`) when a connection that
// was accepted BEFORE any real client initialized the session closes. The
// operator saw it as "new threads never start": a blank pane echoing keys, the
// server dead ~20 ms after "Starting Zellij server!". Pair's trigger was its own
// title poller, spawned before `zellij --new-session-with-layout` and probing
// `zellij list-sessions` (which connects to EVERY session socket) at once.
//
// Two modes, one per half of that claim:
//
//	go run ./probes/zellijbirthrace poke   -n 5
//	go run ./probes/zellijbirthrace launch -n 10 [-pair PATH] [-hammer 10ms]
//
// poke is the zellij bug with no Pair in the room: start a bare session, and the
// moment its socket appears, connect and hang up. launch is Pair's cold-create
// path end to end: N `pair resume <new-tag> --layout3` in a scratch git repo,
// each verdict read from Pair's own birth evidence (the agent pane's sidecar)
// and zellij's log. -hammer adds a concurrent `list-sessions` loop at the given
// cadence, which is what Couch's cold-resume registration poll did (10 ms).
//
// Both need the harness sandbox OFF: zellij's sockets and log live under the
// real $TMPDIR, and ptys are refused inside it.
//
// launch cannot run pair from here directly when "here" is a Pair pane: the
// launcher refuses a nested launch by walking the PROCESS ANCESTRY for zellij,
// so scrubbing the environment is not enough. The parent therefore re-execs this
// probe through `sh -c '… &'`, the shell exits, and the child is reparented to
// launchd -- out of zellij's tree. The child writes its report to a file the
// parent prints.
//
// What a launch trial leaves behind, and what removes it (ARCH-FUNERAL): a
// scratch repo, an isolated XDG_DATA_HOME (so no state lands in the operator's
// ~/.local/share/pair), a bin dir holding a fake `claude` that only sleeps, the
// zellij session, and Pair's detached sidecars (title poller, session watcher).
// The session is deleted BY THE NAME PAIR RECORDED for our tag, every process
// whose argv carries the per-trial tag is killed, and the three dirs are
// removed -- on every path, because every path returns through run()'s defers.
//
// Results: see SKILL.md (baseline before the #287 fix, and after).
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"

	"github.com/xianxu/pair/probes/zellijprobe"
)

// childVerb is the detached launch child's argv[1]. Underscored so nobody types
// it by accident; it carries no trial tag, so the per-trial pkill never
// matches the probe itself.
const childVerb = "__launch-child"

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 {
		// `make test-smoke` runs every probe with no arguments. This one is an
		// operator's instrument: poke deliberately kills zellij servers and
		// launch starts real Pair sessions, neither fit for an unattended loop.
		// So a bare run says what it is and succeeds -- safe by construction,
		// like probes/zellijcalls. SKILL.md is the runbook.
		fmt.Println("zellijbirthrace: an operator-driven instrument (pair#287); see probes/zellijbirthrace/SKILL.md")
		fmt.Println("usage: zellijbirthrace poke|launch [flags]   (sandbox off)")
		return 0
	}
	switch args[0] {
	case "poke":
		return runPoke(args[1:])
	case "launch":
		return runLaunchParent(args[1:])
	case childVerb:
		return runLaunchChild(args[1:])
	}
	fmt.Fprintf(os.Stderr, "zellijbirthrace: unknown mode %q\n", args[0])
	return 2
}

// --- shared zellij observations -------------------------------------------

func zellijTmp() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("zellij-%d", os.Getuid()))
}

// panicMark is a read position in zellij's log. Counting panics AFTER a mark
// rather than in the whole file keeps the 200-odd historical panics out of the
// verdict. A log that shrank was rotated, and then the whole file is new.
type panicMark struct {
	path string
	size int64
}

func markPanics(path string) panicMark {
	info, err := os.Stat(path)
	if err != nil {
		return panicMark{path: path}
	}
	return panicMark{path: path, size: info.Size()}
}

func (m panicMark) since() int {
	f, err := os.Open(m.path)
	if err != nil {
		return 0
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0
	}
	if info.Size() >= m.size {
		if _, err := f.Seek(m.size, io.SeekStart); err != nil {
			return 0
		}
	}
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		if strings.Contains(sc.Text(), "Panic occurred") {
			n++
		}
	}
	return n
}

// sessionState reads `zellij list-sessions --no-formatting` for one name. Only
// ever called once the session is born or dead -- never inside its birth
// window, which is the whole point of this issue.
func sessionState(name string) (listed, exited bool) {
	out, _ := exec.Command("zellij", "list-sessions", "--no-formatting").Output()
	for _, line := range strings.Split(string(out), "\n") {
		head, _, _ := strings.Cut(line, " [Created")
		if strings.TrimSpace(head) == name {
			return true, strings.Contains(line, "EXITED")
		}
	}
	return false, false
}

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// --- poke: the zellij bug, no Pair ----------------------------------------

func runPoke(args []string) int {
	fs := flag.NewFlagSet("poke", flag.ContinueOnError)
	n := fs.Int("n", 5, "trials")
	sockDir := fs.String("sockdir", filepath.Join(zellijTmp(), "contract_version_1"), "zellij socket dir")
	logPath := fs.String("zellij-log", filepath.Join(zellijTmp(), "zellij-log", "zellij.log"), "zellij log")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	layout, err := os.CreateTemp("", "zellijbirthrace-*.kdl")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.Remove(layout.Name())
	if _, err := layout.WriteString("layout {\n    pane command=\"sleep\" {\n        args \"600\"\n    }\n}\n"); err != nil {
		layout.Close()
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	layout.Close()

	counts := map[string]int{}
	for i := 0; i < *n; i++ {
		verdict := pokeTrial(i, *sockDir, *logPath, layout.Name())
		counts[strings.Fields(verdict)[0]]++
		fmt.Printf("trial %d: %s\n", i, verdict)
	}
	fmt.Printf("PROBE-RESULT mode=poke n=%d survived=%d died=%d inconclusive=%d\n",
		*n, counts["survived"], counts["died"], counts["inconclusive"])
	return 0
}

func pokeTrial(i int, sockDir, logPath, layout string) string {
	mark := markPanics(logPath)
	prefix := fmt.Sprintf("birthrace%d", i)
	name := fmt.Sprintf("%s-%d", prefix, os.Getpid())
	sock := filepath.Join(sockDir, name)

	poked := make(chan bool, 1)
	go func() {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(sock); err == nil {
				if conn, err := net.Dial("unix", sock); err == nil {
					conn.Close()
					poked <- true
					return
				}
			}
			time.Sleep(time.Millisecond)
		}
		poked <- false
	}()

	s, err := zellijprobe.Start(zellijprobe.Options{
		ConfigFile: filepath.Join(repoRoot(), "zellij", "config.kdl"),
		Layout:     layout,
		NamePrefix: prefix,
		Env:        zellijprobe.Scrub([]string{"ZELLIJ", "PAIR_", "COUCH_", "CMUX_"}, "TERM=xterm-256color"),
	})
	if err != nil {
		return "inconclusive (start: " + err.Error() + ")"
	}
	defer s.Close()
	if !<-poked {
		return "inconclusive (socket never appeared)"
	}
	time.Sleep(2 * time.Second)
	if mark.since() > 0 {
		return "died"
	}
	if listed, exited := sessionState(name); !listed || exited {
		return "died"
	}
	return "survived"
}

// --- launch: Pair's cold create, end to end --------------------------------

type launchFlags struct {
	n       int
	pair    string
	hammer  time.Duration
	timeout time.Duration
	logPath string
	report  string
	parent  int // the launch parent's pid; the child stops when it is gone
}

func parseLaunchFlags(mode string, args []string) (launchFlags, error) {
	var lf launchFlags
	fs := flag.NewFlagSet(mode, flag.ContinueOnError)
	fs.IntVar(&lf.n, "n", 10, "trials")
	fs.StringVar(&lf.pair, "pair", "pair", "the pair binary to launch")
	fs.DurationVar(&lf.hammer, "hammer", 0, "run `zellij list-sessions` at this cadence during each trial (0 = off)")
	fs.DurationVar(&lf.timeout, "timeout", 15*time.Second, "per-trial wait for birth or death")
	fs.StringVar(&lf.logPath, "zellij-log", filepath.Join(zellijTmp(), "zellij-log", "zellij.log"), "zellij log")
	fs.StringVar(&lf.report, "report", "", "(child) where to write the report")
	fs.IntVar(&lf.parent, "parent", 0, "(child) the parent's pid")
	err := fs.Parse(args)
	return lf, err
}

func (lf launchFlags) argv() []string {
	return []string{
		"-n", fmt.Sprint(lf.n), "-pair", lf.pair, "-hammer", lf.hammer.String(),
		"-timeout", lf.timeout.String(), "-zellij-log", lf.logPath, "-report", lf.report,
		"-parent", fmt.Sprint(lf.parent),
	}
}

func runLaunchParent(args []string) int {
	lf, err := parseLaunchFlags("launch", args)
	if err != nil {
		return 2
	}
	if lf.pair, err = exec.LookPath(lf.pair); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if lf.pair, err = filepath.Abs(lf.pair); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dir, err := os.MkdirTemp("", "zellijbirthrace-report-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)
	lf.report = filepath.Join(dir, "report")
	lf.parent = os.Getpid()
	done := lf.report + ".done"

	// `&` inside sh, and sh exits: the child's parent becomes launchd.
	script := `"$0" "$@" </dev/null >"$BIRTHRACE_CHILD_LOG" 2>&1 &`
	cmd := exec.Command("/bin/sh", append([]string{"-c", script, self, childVerb}, lf.argv()...)...)
	cmd.Env = append(os.Environ(), "BIRTHRACE_CHILD_LOG="+filepath.Join(dir, "child.log"))
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "detach child: %v\n", err)
		return 1
	}

	budget := time.Duration(lf.n)*(lf.timeout+15*time.Second) + time.Minute
	deadline := time.Now().Add(budget)
	printed := 0
	for {
		raw, _ := os.ReadFile(lf.report)
		lines := strings.SplitAfter(string(raw), "\n")
		for ; printed < len(lines) && strings.HasSuffix(lines[printed], "\n"); printed++ {
			fmt.Print(lines[printed])
		}
		if code, err := os.ReadFile(done); err == nil {
			raw, _ := os.ReadFile(lf.report)
			fmt.Print(strings.Join(strings.SplitAfter(string(raw), "\n")[printed:], ""))
			if strings.TrimSpace(string(code)) != "0" {
				childLog, _ := os.ReadFile(filepath.Join(dir, "child.log"))
				fmt.Fprintf(os.Stderr, "child failed:\n%s", zellijprobe.TailOf(string(childLog), 2000))
				return 1
			}
			return 0
		}
		if time.Now().After(deadline) {
			fmt.Fprintf(os.Stderr, "zellijbirthrace: child did not finish within %s\n", budget)
			return 1
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func runLaunchChild(args []string) int {
	lf, err := parseLaunchFlags(childVerb, args)
	if err != nil || lf.report == "" {
		return 2
	}
	report, err := os.Create(lf.report)
	if err != nil {
		return 1
	}
	code := launchTrials(lf, report)
	report.Close()
	if !parentAlive(lf.parent) {
		// Nobody is left to read the report, and the parent's deferred
		// RemoveAll died with it, so the report dir is ours to remove.
		_ = os.RemoveAll(filepath.Dir(lf.report))
		return code
	}
	_ = os.WriteFile(lf.report+".done", []byte(fmt.Sprintln(code)), 0o600)
	return code
}

// parentAlive reports whether the launch parent still runs. The child is
// detached and reparented to launchd, so it cannot learn this from its ppid.
func parentAlive(pid int) bool {
	return pid > 0 && syscall.Kill(pid, 0) == nil
}

func launchTrials(lf launchFlags, out io.Writer) int {
	scratch, err := os.MkdirTemp("", "birthrace-scratch-*")
	if err != nil {
		fmt.Fprintln(out, "ERROR", err)
		return 1
	}
	defer os.RemoveAll(scratch)
	repo := filepath.Join(scratch, "birthrace")
	xdg := filepath.Join(scratch, "xdg")
	bin := filepath.Join(scratch, "bin")
	for _, d := range []string{repo, xdg, bin} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			fmt.Fprintln(out, "ERROR", err)
			return 1
		}
	}
	if msg, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		fmt.Fprintf(out, "ERROR git init: %v %s\n", err, msg)
		return 1
	}
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\nexec sleep 3600\n"), 0o755); err != nil {
		fmt.Fprintln(out, "ERROR", err)
		return 1
	}
	env := zellijprobe.Scrub(
		[]string{"ZELLIJ", "PAIR_", "COUCH_", "CMUX_", "XDG_DATA_HOME=", "PATH="},
		"XDG_DATA_HOME="+xdg, "PATH="+bin+":"+os.Getenv("PATH"), "TERM=xterm-256color",
	)

	counts := map[string]int{}
	for i := 0; i < lf.n; i++ {
		if !parentAlive(lf.parent) {
			// The parent was killed or gave up. Its budget is n trials, so
			// running the rest would only start sessions nobody is counting.
			fmt.Fprintf(out, "ABORTED after %d trials: the parent is gone\n", i)
			break
		}
		tag := fmt.Sprintf("br%dt%dx", os.Getpid(), i)
		t := launchTrial(lf, env, repo, xdg, tag)
		counts[t.verdict]++
		fmt.Fprintf(out, "trial %d tag=%s verdict=%s birth=%s%s\n", i, tag, t.verdict, t.birth.Round(time.Millisecond), t.note)
	}
	fmt.Fprintf(out, "PROBE-RESULT mode=launch n=%d hammer=%s born=%d died=%d inconclusive=%d\n",
		lf.n, lf.hammer, counts["born"], counts["died"], counts["inconclusive"])
	return 0
}

type trialResult struct {
	verdict string
	birth   time.Duration
	note    string
}

func launchTrial(lf launchFlags, env []string, repo, xdg, tag string) (result trialResult) {
	mark := markPanics(lf.logPath)
	start := time.Now()
	cmd := exec.Command(lf.pair, "resume", tag, "--layout3")
	cmd.Dir = repo
	cmd.Env = env
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 140})
	if err != nil {
		return trialResult{verdict: "inconclusive", note: " (start: " + err.Error() + ")"}
	}
	seen := &zellijprobe.SyncBuffer{}
	go func() { _, _ = io.Copy(seen, f) }()
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()

	stopHammer := hammer(lf.hammer)
	defer func() {
		stopHammer()
		teardown(xdg, tag, f, cmd, exited)
	}()

	pane := filepath.Join(xdg, "pair", "repos", "*", "pane-"+tag+"-claude.json")
	deadline := start.Add(lf.timeout)
	for time.Now().Before(deadline) {
		if matches, _ := filepath.Glob(pane); len(matches) > 0 {
			result.birth = time.Since(start)
			stopHammer()
			// Born means the pane ran, so the session had initialized; listing it
			// now cannot land in its birth window.
			time.Sleep(500 * time.Millisecond)
			name := sessionName(xdg, tag)
			listed, exitedState := sessionState(name)
			switch {
			case mark.since() > 0:
				return trialResult{verdict: "died", birth: result.birth, note: " (panic after the pane was born)"}
			case !listed || exitedState:
				return trialResult{verdict: "died", birth: result.birth, note: fmt.Sprintf(" (session %q not live after birth)", name)}
			}
			return trialResult{verdict: "born", birth: result.birth}
		}
		if mark.since() > 0 {
			return trialResult{verdict: "died", note: " (zellij panic before the pane was born)"}
		}
		select {
		case <-exited:
			return trialResult{verdict: "inconclusive", note: " (pair exited: " + strconv.Quote(zellijprobe.TailOf(seen.String(), 300)) + ")"}
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	return trialResult{verdict: "inconclusive", note: " (neither birth nor panic within " + lf.timeout.String() + ")"}
}

// hammer runs `zellij list-sessions --short` back to back with cadence between
// runs until the returned stop is called (idempotent).
func hammer(cadence time.Duration) func() {
	if cadence <= 0 {
		return func() {}
	}
	stop := make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = exec.Command("zellij", "list-sessions", "--short").Run()
			select {
			case <-stop:
				return
			case <-time.After(cadence):
			}
		}
	}()
	return func() {
		once.Do(func() { close(stop) })
		wg.Wait()
	}
}

// sessionName is the zellij session Pair bound our tag to, from Pair's own
// index. Never guessed from a pattern: teardown deletes by this name only.
func sessionName(xdg, tag string) string {
	indexes, _ := filepath.Glob(filepath.Join(xdg, "pair", "repos", "*", "session-names.jsonl"))
	for _, path := range indexes {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range bytes.Split(raw, []byte("\n")) {
			var entry struct {
				SessionName string `json:"session_name"`
				Tag         string `json:"tag"`
			}
			if json.Unmarshal(line, &entry) == nil && entry.Tag == tag && entry.SessionName != "" {
				return entry.SessionName
			}
		}
	}
	return ""
}

func teardown(xdg, tag string, f *os.File, cmd *exec.Cmd, exited <-chan struct{}) {
	if name := sessionName(xdg, tag); name != "" {
		_ = exec.Command("zellij", "kill-session", name).Run()
		_ = exec.Command("zellij", "delete-session", "--force", name).Run()
	}
	_ = f.Close()
	if cmd.Process != nil {
		// pty.Start made the launcher a session leader, so its pid is its group.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
	}
	// The detached sidecars (title poller, session watcher) and a dead server's
	// orphaned client are outside the launcher's group; each carries the tag in
	// its argv, and the tag is unique to this trial (the trailing x keeps t1x
	// from matching t10x).
	err := exec.Command("pkill", "-KILL", "-f", tag).Run()
	var exit *exec.ExitError
	if err != nil && !(errors.As(err, &exit) && exit.ExitCode() == 1) { // 1 = nothing matched
		fmt.Fprintf(os.Stderr, "pkill %s: %v\n", tag, err)
	}
}
