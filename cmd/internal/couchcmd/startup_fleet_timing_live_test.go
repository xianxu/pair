//go:build darwin || linux

package couchcmd

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	pairterminal "github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// The outer test captures actual parent stderr counters without replacing a
// process-global writer while production Console workers are running.
func TestStartupFleetTimingLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" || os.Getenv("PAIR_LIVE_STARTUP_PERF") != "1" || os.Getenv("PAIR_LIVE_STARTUP_FLEET") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1 PAIR_LIVE_STARTUP_FLEET=1")
	}
	if os.Getenv("PAIR_STARTUP_FLEET_CHILD") == "1" {
		runStartupFleetTiming(t)
		return
	}
	evidence := os.Getenv("PAIR_STARTUP_FLEET_EVIDENCE")
	if !filepath.IsAbs(evidence) {
		t.Fatal("PAIR_STARTUP_FLEET_EVIDENCE must be an absolute private evidence directory")
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	output, err := os.OpenFile(filepath.Join(evidence, "fleet-raw.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestStartupFleetTimingLive$", "-test.v", "-test.timeout=20m")
	cmd.Env = testpair.Environment(t, t.TempDir(), "PAIR_STARTUP_FLEET_CHILD=1", "PAIR_LIVE_COUCH=1", "PAIR_LIVE_STARTUP_PERF=1", "PAIR_LIVE_STARTUP_FLEET=1", "PAIR_STARTUP_FLEET_EVIDENCE="+evidence, "PAIR_STARTUP_PERF_BIN_DIR="+os.Getenv("PAIR_STARTUP_PERF_BIN_DIR"), "PAIR_STARTUP_FLEET_SMOKE_LIMIT="+os.Getenv("PAIR_STARTUP_FLEET_SMOKE_LIMIT"))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout, cmd.Stderr = output, output
	var identityMu sync.Mutex
	var helperIdentity string
	cmd.Cancel = func() error {
		identityMu.Lock()
		expected := helperIdentity
		identityMu.Unlock()
		if cmd.Process == nil || expected == "" || procutil.Identity(strconv.Itoa(cmd.Process.Pid)) != expected {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	runErr := cmd.Start()
	if runErr == nil {
		identityMu.Lock()
		helperIdentity = procutil.Identity(strconv.Itoa(cmd.Process.Pid))
		identityMu.Unlock()
		runErr = cmd.Wait()
	}
	closeErr := output.Close()
	cleanupErr := fleetCleanupManifest(evidence)
	if err := fleetSummarizeCounters(evidence); err != nil {
		t.Error(err)
	}
	if err := errors.Join(runErr, closeErr, cleanupErr); err != nil {
		t.Fatalf("fleet failed; every attempt retained at %s: %v", evidence, err)
	}
	t.Logf("all fleet attempts and host samples retained at %s", evidence)
}

type fleetOwnershipManifest struct {
	SocketRoot, Zellij, StoreDir string
	Env                          []string
	Sessions                     []string
}

func fleetCleanupManifest(evidence string) error {
	raw, err := os.ReadFile(filepath.Join(evidence, "owned-fixtures.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var owned fleetOwnershipManifest
	if err := json.Unmarshal(raw, &owned); err != nil {
		return err
	}
	if !filepath.IsAbs(owned.SocketRoot) || !strings.HasPrefix(filepath.Base(owned.SocketRoot), "p218.") || (filepath.Dir(owned.SocketRoot) != "/private/tmp" && filepath.Dir(owned.SocketRoot) != "/tmp") {
		return errors.New("refuse cleanup outside exact private socket namespace")
	}
	if _, err := os.Stat(owned.SocketRoot); os.IsNotExist(err) {
		return nil
	}
	canonical, err := filepath.EvalSymlinks(owned.SocketRoot)
	if err != nil || canonical != owned.SocketRoot {
		return errors.New("refuse noncanonical fixture socket root")
	}
	retainErr := fleetRetainDiagnostics(owned.SocketRoot, evidence)
	for _, session := range owned.Sessions {
		if !strings.HasPrefix(session, "pair-f21-") || strings.ContainsAny(session, "/\x00\r\n") {
			return errors.Join(retainErr, errors.New("refuse nonfixture session cleanup"))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, owned.Zellij, "delete-session", session, "--force")
		cmd.Env = owned.Env
		_ = cmd.Run()
		cancel()
	}
	// Failure/timeout cleanup stays confined to exact process generations in
	// the private fixture store/socket namespace; it never searches by tag alone.
	if strings.HasPrefix(filepath.Clean(owned.StoreDir), filepath.Clean(owned.SocketRoot)+string(os.PathSeparator)) {
		ns, err := couchcore.ResolveCouchNamespace(owned.StoreDir, "/unused")
		if err == nil {
			if snapshot, err := couchcore.NewThreadStore(ns).Snapshot(); err == nil {
				for _, record := range snapshot.Records {
					for _, inc := range record.Incarnations {
						name := ""
						if record.SessionBinding != nil {
							name = record.SessionBinding.Name
						}
						if inc.Start != nil && inc.Start.SessionBinding != nil {
							name = inc.Start.SessionBinding.Name
						}
						for _, ownedName := range owned.Sessions {
							if name == ownedName {
								fleetEndExact(inc.PID, inc.Identity, true)
							}
						}
					}
				}
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	processes, err := procutil.SnapshotNamed(ctx, "zellij")
	cancel()
	if err != nil {
		return errors.Join(retainErr, fmt.Errorf("private server cleanup observation: %w", err))
	}
	for _, process := range processes {
		if process.Err != nil || len(process.Args) != 3 || process.Args[1] != "--server" {
			continue
		}
		socket := process.Args[2]
		if !strings.HasPrefix(filepath.Clean(socket), filepath.Clean(owned.SocketRoot)+string(os.PathSeparator)) {
			continue
		}
		for _, session := range owned.Sessions {
			if filepath.Base(socket) == session {
				fleetEndExact(process.PID, process.Identity, false)
			}
		}
	}
	return errors.Join(retainErr, os.RemoveAll(owned.SocketRoot))
}
func fleetEndExact(pid int, identity string, group bool) {
	if pid <= 0 || identity == "" || procutil.Identity(strconv.Itoa(pid)) != identity {
		return
	}
	target := pid
	if group {
		target = -pid
	}
	_ = syscall.Kill(target, syscall.SIGTERM)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = fleetWait(ctx, func() bool { return procutil.Identity(strconv.Itoa(pid)) != identity })
	if procutil.Identity(strconv.Itoa(pid)) == identity {
		_ = syscall.Kill(target, syscall.SIGKILL)
	}
}

type fleetDiscardHost struct{ *hostty.FakeHost }

func (h fleetDiscardHost) Write(p []byte) (int, error) { return len(p), nil }
func (h fleetDiscardHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return len(p), nil
}

type fleetPane struct {
	address                  couchcore.ThreadAddress
	session, marker, pidFile string
	fixture                  *pairlifecycletest.ControlledZellij
	server                   launcher.SessionServerIdentity
	agent                    couchcore.ProcessIdentity
}
type fleetChild struct {
	handle                            couchcore.Handle
	terminal                          *ptychild.Child
	address                           couchcore.ThreadAddress
	prefix                            []byte
	firstFrame, deliveredFrame, spawn time.Time
}
type fleetTracker struct {
	mu           sync.Mutex
	children     map[string]*fleetChild
	markers      map[couchcore.ThreadAddress]string
	adopted      map[couchcore.ThreadAddress]bool
	active, peak int
	failures     []string
}
type fleetRunner struct {
	inner   *couchcore.PtyRunner
	tracker *fleetTracker
}

func (r fleetRunner) remember(h couchcore.Handle, env []string) {
	if h == nil {
		return
	}
	r.tracker.mu.Lock()
	defer r.tracker.mu.Unlock()
	child := r.tracker.children[h.ID()]
	if child == nil {
		child = &fleetChild{}
		r.tracker.children[h.ID()] = child
	}
	child.handle = h
	child.terminal = h.(couchcore.TerminalHandle).Terminal()
	child.spawn = time.Now()
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		switch key {
		case "COUCH_THREAD_SCOPE":
			child.address.RepoScope = value
		case "COUCH_THREAD_TAG":
			child.address.Tag = couchcore.ThreadTag(value)
		}
	}
}
func (r fleetRunner) Start(dir string, argv, env []string) (couchcore.Handle, error) {
	h, err := r.inner.Start(dir, argv, env)
	r.remember(h, env)
	return h, err
}
func (r fleetRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, bound time.Duration) (couchcore.BlockedHandle, error) {
	h, err := r.inner.StartBlocked(ctx, dir, argv, env, bound)
	r.remember(h, env)
	return h, err
}

type fleetSample struct {
	Index, Limit, Count, PeakBackground                                      int
	RootReturnMS, AllFramesMS, AdoptionMS, LeaveMS, StartupCPUMS, LeaveCPUMS float64
	FrameMS                                                                  map[string]float64
	Children                                                                 map[string]string
	Clients                                                                  map[string][]fleetClientBirth
	Errors                                                                   []string
	ParentStages                                                             []fleetParentStage
	Queries                                                                  []fleetZellijQuery
	AttachedRows, PostLeaveRows                                              []launcher.Session
	ClientsAfterAttachedQueries                                              map[string][]fleetClientObservation
}
type fleetClientBirth struct {
	PID, PPID            int
	Identity, Executable string
	KernelStartUTC       string
	KernelSpawnMS        *float64
}

type fleetZellijQuery struct {
	Phase, Operation, Tag            string
	Args                             []string
	Started, Finished                time.Time
	DurationMS                       float64
	Stdout, Stderr                   []byte
	StdoutBytes, StderrBytes         int
	StdoutTruncated, StderrTruncated bool
	ExitCode                         int
	Error                            string
}

// The nine-sample schedule retains at most 8 MiB of raw query streams and
// 4096 query records per sample. Any source/sample truncation or omitted record
// fails the measurement explicitly; it never changes the production query.
const fleetQueryByteLimit = 8 * 1024 * 1024
const fleetQueryEventLimit = 4096

type fleetQueryCollector struct {
	mu                        sync.Mutex
	events                    []fleetZellijQuery
	bytes, omitted, truncated int
	// Immutable before operations start; Leave's shared context can therefore
	// attribute each named authority query to its actual fixture thread.
	sessionTags map[string]string
}

func (c *fleetQueryCollector) context(ctx context.Context, phase, operation, tag string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return launcher.WithZellijQueryObserver(ctx, c.observer(phase, operation, tag))
}

func (c *fleetQueryCollector) observer(phase, operation, tag string) func(launcher.ZellijQueryObservation) {
	return func(e launcher.ZellijQueryObservation) {
		queryTag := tag
		for i, arg := range e.Args {
			if arg == "--session" && i+1 < len(e.Args) {
				if actual, ok := c.sessionTags[e.Args[i+1]]; ok {
					queryTag = actual
				}
				break
			}
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		if len(c.events) == fleetQueryEventLimit {
			c.omitted++
			return
		}
		stdoutSize := min(len(e.Stdout), fleetQueryByteLimit-c.bytes)
		stderrSize := min(len(e.Stderr), fleetQueryByteLimit-c.bytes-stdoutSize)
		stdoutTruncated := e.StdoutTruncated || stdoutSize < len(e.Stdout)
		stderrTruncated := e.StderrTruncated || stderrSize < len(e.Stderr)
		if stdoutTruncated || stderrTruncated {
			c.truncated++
		}
		errText := ""
		if e.Err != nil {
			errText = e.Err.Error()
		}
		c.events = append(c.events, fleetZellijQuery{
			Phase: phase, Operation: operation, Tag: queryTag,
			Args: append([]string(nil), e.Args...), Started: e.Started, Finished: e.Finished,
			DurationMS: fleetMillis(e.Duration),
			Stdout:     append([]byte(nil), e.Stdout[:stdoutSize]...), Stderr: append([]byte(nil), e.Stderr[:stderrSize]...),
			StdoutBytes: e.StdoutBytes, StderrBytes: e.StderrBytes,
			StdoutTruncated: stdoutTruncated, StderrTruncated: stderrTruncated,
			ExitCode: e.ExitCode, Error: errText,
		})
		c.bytes += stdoutSize + stderrSize
	}
}

func (c *fleetQueryCollector) snapshot() ([]fleetZellijQuery, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	if c.omitted > 0 || c.truncated > 0 {
		err = fmt.Errorf("query evidence incomplete: omitted=%d truncated=%d (limits %d records, %d raw bytes)", c.omitted, c.truncated, fleetQueryEventLimit, fleetQueryByteLimit)
	}
	return append([]fleetZellijQuery(nil), c.events...), err
}

type fleetClientObservation struct {
	PID                                int
	ExpectedIdentity, ObservedIdentity string
	ObservedAt                         time.Time
	State                              string
	Found, SameGeneration              bool
	Error                              string
}

// A fresh native snapshot excludes terminal processes. Matching a live client
// generation is separate evidence from server-reported connectivity; neither
// a prior marker frame nor a persisted identity token proves current attachment.
func fleetPostQueryClients(clients map[string][]fleetClientBirth, processes []procutil.Observation, snapshotErr error) map[string][]fleetClientObservation {
	byPID := make(map[int]procutil.Observation, len(processes))
	for _, process := range processes {
		byPID[process.PID] = process
	}
	observedAt := time.Now()
	out := make(map[string][]fleetClientObservation, len(clients))
	for tag, births := range clients {
		for _, birth := range births {
			e := fleetClientObservation{PID: birth.PID, ExpectedIdentity: birth.Identity, ObservedAt: observedAt, State: "unknown"}
			if snapshotErr != nil {
				e.Error = snapshotErr.Error()
			} else if process, ok := byPID[birth.PID]; ok {
				e.Found, e.ObservedIdentity = true, process.Identity
				if process.Err != nil {
					e.Error = process.Err.Error()
				} else if process.Identity == birth.Identity {
					e.State, e.SameGeneration = "same-generation", true
				} else {
					e.State = "replaced"
				}
			} else {
				e.State = "absent-from-snapshot"
			}
			out[tag] = append(out[tag], e)
		}
	}
	return out
}

// These checks validate experiment completeness; production ownership and
// cleanup continue through their existing paths even when a sample is refused.
func fleetClientBirthErrors(panes []fleetPane, clients map[string][]fleetClientBirth) []string {
	var failures []string
	expected := make(map[string]bool, len(panes))
	for _, pane := range panes {
		tag := string(pane.address.Tag)
		expected[tag] = true
		births := clients[tag]
		if len(births) != 1 {
			failures = append(failures, fmt.Sprintf("native clients %s: observed %d, want exactly 1", tag, len(births)))
			continue
		}
		if births[0].PID <= 0 || births[0].Identity == "" {
			failures = append(failures, "native client generation unknown "+tag)
		}
	}
	for tag := range clients {
		if !expected[tag] {
			failures = append(failures, "unexpected native client tag "+tag)
		}
	}
	return failures
}

func fleetPostQueryClientErrors(clients map[string][]fleetClientObservation) []string {
	var failures []string
	for tag, observations := range clients {
		for _, e := range observations {
			if !e.SameGeneration {
				failures = append(failures, fmt.Sprintf("post-query native client %s pid=%d state=%s: %s", tag, e.PID, e.State, e.Error))
			}
		}
	}
	return failures
}

// Retain regular diagnostic files while the private fixture still exists.
// Sockets, pipes and symlinks are intentionally excluded from this byte copy.
func fleetRetainDiagnostics(root, evidence string) error {
	if !filepath.IsAbs(root) || !filepath.IsAbs(evidence) {
		return errors.New("diagnostic root and evidence must be absolute")
	}
	if rel, err := filepath.Rel(root, evidence); err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return errors.New("diagnostic evidence must be outside the fixture root")
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if os.IsNotExist(walkErr) && path == root {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(evidence, "fixture-diagnostics", rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0700)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return errors.Join(err, in.Close())
		}
		_, copyErr := io.Copy(out, in)
		return errors.Join(copyErr, in.Close(), out.Close())
	})
}

// These are consecutive lifecycle stages, not nested IO timings. In
// particular record+baseline includes store durability; it does not isolate fsync.
type fleetParentStage struct {
	Operation, Tag, Stage string
	DurationMS            float64
}
type fleetParentTimings struct {
	mu     sync.Mutex
	events []fleetParentStage
}

func (p *fleetParentTimings) observer(operation, tag string) func(string, time.Duration) {
	return func(stage string, elapsed time.Duration) {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.events = append(p.events, fleetParentStage{operation, tag, stage, fleetMillis(elapsed)})
	}
}
func (p *fleetParentTimings) context(ctx context.Context, operation, tag string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return couchcore.WithLaunchTimingObserver(ctx, p.observer(operation, tag))
}
func (p *fleetParentTimings) snapshot() []fleetParentStage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]fleetParentStage(nil), p.events...)
}

func fleetMillis(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
func fleetCPUTime() (time.Duration, error) {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0, err
	}
	return time.Duration(usage.Utime.Sec+usage.Stime.Sec)*time.Second + time.Duration(usage.Utime.Usec+usage.Stime.Usec)*time.Microsecond, nil
}
func fleetWait(ctx context.Context, condition func() bool) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if condition() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func runFleetSample(t *testing.T, c *couchcore.Couch, tracker *fleetTracker, inner *couchcore.PtyRunner, current **couchtty.Console, panes []fleetPane, repo string, limit, index int, evidence string) (sample fleetSample) {
	sample = fleetSample{Index: index, Limit: limit, Count: len(panes), FrameMS: map[string]float64{}, Children: map[string]string{}, Clients: map[string][]fleetClientBirth{}}
	var parentTimings fleetParentTimings
	queries := fleetQueryCollector{sessionTags: make(map[string]string, len(panes))}
	for _, pane := range panes {
		queries.sessionTags[pane.session] = string(pane.address.Tag)
	}
	defer func() {
		sample.ParentStages = parentTimings.snapshot()
		var captureErr error
		sample.Queries, captureErr = queries.snapshot()
		if captureErr != nil {
			sample.Errors = append(sample.Errors, captureErr.Error())
		}
	}()
	fmt.Fprintf(os.Stderr, "fleet sample begin index=%d limit=%d\n", index, limit)
	defer func() { fmt.Fprintf(os.Stderr, "fleet sample end index=%d\n", index) }()
	couchcore.LifecycleParallelism = limit
	tracker.mu.Lock()
	tracker.children = map[string]*fleetChild{}
	tracker.adopted = map[couchcore.ThreadAddress]bool{}
	tracker.active, tracker.peak = 0, 0
	tracker.failures = nil
	tracker.mu.Unlock()
	reader, input := io.Pipe()
	console := couchtty.New(fleetDiscardHost{hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100})}, reader)
	*current = console
	console.SetErrorWriter(os.Stderr)
	wireResolver(console, c)
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name == "resume" {
			tracker.mu.Lock()
			tracker.active++
			if tracker.active > tracker.peak {
				tracker.peak = tracker.active
			}
			tracker.mu.Unlock()
			defer func() { tracker.mu.Lock(); tracker.active--; tracker.mu.Unlock() }()
		}
		tag := call.Args["tag"]
		if call.Name != "leave" {
			call.Context = parentTimings.context(call.Context, call.Name, tag)
		} else {
			tag = "fleet"
		}
		call.Context = queries.context(call.Context, "operation", call.Name, tag)
		value, err := actual(call)
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		if err != nil {
			tracker.failures = append(tracker.failures, fmt.Sprintf("%s: %v", call.Name, err))
		}
		if call.Name == "attach" && err == nil {
			address := couchcore.ThreadAddress{RepoScope: call.Args["repo-scope"], Tag: couchcore.ThreadTag(call.Args["tag"])}
			if tracker.adopted[address] {
				tracker.failures = append(tracker.failures, "duplicate attach "+string(address.Tag))
			}
			tracker.adopted[address] = true
		}
		return value, err
	})
	done := make(chan int, 1)
	startedConsole := false
	defer func() {
		// Retain the measured attempt before Stop or client teardown can
		// rewrite records or remove diagnostics, including an early root error.
		if err := fleetRetainDiagnostics(os.Getenv("TMPDIR"), filepath.Join(evidence, fmt.Sprintf("sample-%02d", index))); err != nil {
			sample.Errors = append(sample.Errors, "retain diagnostics: "+err.Error())
		}
		console.Stop()
		_ = input.Close()
		_ = reader.Close()
		if startedConsole {
			select {
			case code := <-done:
				if code != 0 {
					sample.Errors = append(sample.Errors, fmt.Sprintf("Console exit%d", code))
				}
			case <-time.After(15 * time.Second):
				sample.Errors = append(sample.Errors, "Console did not join")
			}
		}
		tracker.mu.Lock()
		children := make([]*fleetChild, 0, len(tracker.children))
		for _, child := range tracker.children {
			sample.Children[string(child.address.Tag)] = string(child.prefix)
			children = append(children, child)
		}
		tracker.mu.Unlock()
		for _, child := range children {
			if child.terminal != nil {
				_ = child.terminal.Close()
			}
		}
		*current = nil
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	stopHost := fleetHostSampler(ctx, evidence, index)
	defer stopHost()
	cpuStart, err := fleetCPUTime()
	if err != nil {
		sample.Errors = append(sample.Errors, err.Error())
		return
	}
	start := time.Now()
	initial, err := c.StartInteractive(queries.context(parentTimings.context(ctx, "root", "root"), "startup", "root", "root"), couchcore.StartArgs{Cwd: repo, Stack: "claude"})
	sample.RootReturnMS = fleetMillis(time.Since(start))
	if err != nil {
		sample.Errors = append(sample.Errors, "initial root: "+err.Error())
		return
	}
	if err := beginConsole(console, initial); err != nil {
		sample.Errors = append(sample.Errors, "begin Console: "+err.Error())
		return
	}
	startedConsole = true
	go func() { done <- console.Run() }()
	err = fleetWait(ctx, func() bool {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		if len(tracker.failures) > 0 {
			return true
		}
		if len(tracker.adopted) != len(panes) || len(tracker.children) != len(panes) {
			return false
		}
		for _, child := range tracker.children {
			if child.firstFrame.IsZero() || child.deliveredFrame.IsZero() {
				select {
				case <-child.handle.Exited():
					tracker.failures = append(tracker.failures, "helper exited without native pane frame "+string(child.address.Tag))
					return true
				default:
					return false
				}
			}
		}
		return true
	})
	cpuEnd, cpuErr := fleetCPUTime()
	sample.StartupCPUMS = fleetMillis(cpuEnd - cpuStart)
	if cpuErr != nil {
		sample.Errors = append(sample.Errors, cpuErr.Error())
	}
	if err != nil {
		sample.Errors = append(sample.Errors, "all first frames: "+err.Error())
	}
	tracker.mu.Lock()
	sample.PeakBackground = tracker.peak
	sample.Errors = append(sample.Errors, tracker.failures...)
	var lastFrame time.Time
	for _, child := range tracker.children {
		tag := string(child.address.Tag)
		if !child.firstFrame.IsZero() {
			sample.FrameMS[tag] = fleetMillis(child.firstFrame.Sub(start))
			if child.firstFrame.After(lastFrame) {
				lastFrame = child.firstFrame
			}
		}
		sample.Children[tag] = string(child.prefix)
	}
	tracker.mu.Unlock()
	if !lastFrame.IsZero() {
		sample.AllFramesMS = fleetMillis(lastFrame.Sub(start))
	}
	sample.AdoptionMS = fleetMillis(time.Since(start))
	fmt.Fprintf(os.Stderr, "fleet phase index=%d verification\n", index)
	for tag, prefix := range sample.Children {
		for _, line := range strings.Split(prefix, "\n") {
			if at := strings.Index(line, "pair startup"); at >= 0 {
				fmt.Fprintf(os.Stderr, "fleet child index=%d tag=%s %s\n", index, tag, strings.TrimSpace(line[at:]))
			}
		}
	}
	if len(sample.Errors) > 0 {
		return
	}
	// Native client PID/start inspection follows the meaningful model frames;
	// its calls are outside startup wall and retained in the actual counters.
	processes, err := procutil.SnapshotNamed(ctx, "zellij")
	if err != nil {
		sample.Errors = append(sample.Errors, "client birth observation: "+err.Error())
		return
	}
	for _, process := range processes {
		if process.Err != nil {
			continue
		}
		for _, pane := range panes {
			matched := false
			for i, arg := range process.Args {
				if arg == "attach" && i+1 < len(process.Args) && process.Args[i+1] == pane.session {
					matched = true
				}
			}
			if !matched {
				continue
			}
			birth := fleetClientBirth{PID: process.PID, PPID: process.PPID, Identity: process.Identity, Executable: process.Executable}
			if runtime.GOOS == "darwin" {
				if seconds, micros, ok := strings.Cut(process.Identity, "."); ok {
					sec, e1 := strconv.ParseInt(seconds, 10, 64)
					usec, e2 := strconv.ParseInt(micros, 10, 64)
					if e1 == nil && e2 == nil {
						when := time.Unix(sec, usec*1000)
						birth.KernelStartUTC = when.UTC().Format(time.RFC3339Nano)
						elapsed := fleetMillis(when.Sub(start))
						birth.KernelSpawnMS = &elapsed
					}
				}
			}
			sample.Clients[string(pane.address.Tag)] = append(sample.Clients[string(pane.address.Tag)], birth)
		}
	}
	sample.Errors = append(sample.Errors, fleetClientBirthErrors(panes, sample.Clients)...)
	// After meaningful model frames, verify all live clients once. No CLI
	// readiness probes run concurrently with the client handoff.
	rows, err := (launcher.ZellijSource{}).SnapshotSessionsContext(queries.context(ctx, "attached", "verify", "fleet"), fleetSessionNames(panes))
	sample.AttachedRows = append([]launcher.Session(nil), rows...)
	// Exactly one fresh verification observation, never a retry of the CLI
	// answer. Preserve absent, replaced and unreadable generations separately.
	afterProcesses, afterErr := procutil.SnapshotNamed(ctx, "zellij")
	sample.ClientsAfterAttachedQueries = fleetPostQueryClients(sample.Clients, afterProcesses, afterErr)
	sample.Errors = append(sample.Errors, fleetPostQueryClientErrors(sample.ClientsAfterAttachedQueries)...)
	if afterErr != nil {
		sample.Errors = append(sample.Errors, "post-query native observation: "+afterErr.Error())
	}
	if err != nil {
		sample.Errors = append(sample.Errors, "attached sessions: "+err.Error())
		return
	}
	if len(rows) != len(panes) {
		sample.Errors = append(sample.Errors, fmt.Sprintf("attached rows%d want%d", len(rows), len(panes)))
		return
	}
	for _, row := range rows {
		if row.State != launcher.SessionAttached {
			sample.Errors = append(sample.Errors, fmt.Sprintf("session not attached %s: state=%s", row.Name, row.State))
		}
	}
	if sample.PeakBackground > min(limit, len(panes)-1) {
		sample.Errors = append(sample.Errors, "background pool exceeded configured limit")
	}
	if err := fleetSurvival(ctx, panes); err != nil {
		sample.Errors = append(sample.Errors, err.Error())
		return
	}
	fmt.Fprintf(os.Stderr, "fleet phase index=%d leave\n", index)
	cpuStart, err = fleetCPUTime()
	if err != nil {
		sample.Errors = append(sample.Errors, err.Error())
		return
	}
	leaving := time.Now()
	value, err := console.Ops()(couchcore.OperationCall{Name: "leave", Implicit: true, Context: parentTimings.context(ctx, "leave", "fleet"), Args: map[string]string{"mode": string(couchcore.LeaveDetach)}})
	sample.LeaveMS = fleetMillis(time.Since(leaving))
	cpuEnd, cpuErr = fleetCPUTime()
	sample.LeaveCPUMS = fleetMillis(cpuEnd - cpuStart)
	if cpuErr != nil {
		sample.Errors = append(sample.Errors, cpuErr.Error())
	}
	if err != nil {
		sample.Errors = append(sample.Errors, "LeaveDetach: "+err.Error())
		return
	}
	left, ok := value.(couchcore.LeaveResult)
	if !ok || len(left.Detached) != len(panes) || len(left.Skipped) != 0 {
		sample.Errors = append(sample.Errors, fmt.Sprintf("Leave result=%+v", value))
		return
	}
	fmt.Fprintf(os.Stderr, "fleet phase index=%d postleave\n", index)
	if err := fleetSurvival(ctx, panes); err != nil {
		sample.Errors = append(sample.Errors, err.Error())
		return
	}
	for _, pane := range panes {
		record, err := c.Threads.GetThread(pane.address)
		if err != nil || len(record.Incarnations) != 0 {
			sample.Errors = append(sample.Errors, fmt.Sprintf("incarnation retirement %s: %+v %v", pane.address.Tag, record.Incarnations, err))
		}
	}
	rows, err = (launcher.ZellijSource{}).SnapshotSessionsContext(queries.context(ctx, "postleave", "verify", "fleet"), fleetSessionNames(panes))
	sample.PostLeaveRows = append([]launcher.Session(nil), rows...)
	if err != nil {
		sample.Errors = append(sample.Errors, err.Error())
		return
	}
	for _, row := range rows {
		if row.State != launcher.SessionDetached {
			sample.Errors = append(sample.Errors, "server not detached "+row.Name)
		}
	}
	return
}

func fleetSessionNames(panes []fleetPane) []string {
	names := make([]string, 0, len(panes))
	for _, pane := range panes {
		names = append(names, pane.session)
	}
	return names
}
func fleetSurvival(ctx context.Context, panes []fleetPane) error {
	states, err := (launcher.OSServerStates{}).ServerStates(ctx)
	if err != nil {
		return err
	}
	for _, pane := range panes {
		state, ok := states[pane.session]
		if !ok || state.Verdict != launcher.ServerReachable || state.Server != pane.server {
			return fmt.Errorf("owned server generation changed %s: %+v", pane.session, state)
		}
		identity, err := (couchcore.OSProcOps{}).Identity(pane.agent.PID)
		if err != nil || identity != pane.agent.Identity {
			return fmt.Errorf("stand-in agent generation changed %s: %q %v", pane.session, identity, err)
		}
	}
	return nil
}

func fleetHostSampler(parent context.Context, evidence string, index int) func() {
	ctx, cancel := context.WithCancel(parent)
	file, err := os.OpenFile(filepath.Join(evidence, fmt.Sprintf("sample-%02d-host.log", index)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return func() { cancel() }
	}
	fmt.Fprintf(file, "sample=%d CouchPID=%d start=%s\n", index, os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
	args := []string{"-l", "0", "-s", "1", "-pid", strconv.Itoa(os.Getpid()), "-stats", "pid,cpu,mem,threads"}
	if runtime.GOOS == "linux" {
		args = []string{"-b", "-d", "1", "-p", strconv.Itoa(os.Getpid())}
	}
	cmd := exec.CommandContext(ctx, "top", args...)
	cmd.Stdout, cmd.Stderr = file, file
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(file, "sampler_error=%v\n", err)
		_ = file.Close()
		return cancel
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		fmt.Fprintf(file, "end=%s\n", time.Now().UTC().Format(time.RFC3339Nano))
		_ = file.Close()
		close(done)
	}()
	return func() { cancel(); <-done }
}

func fleetSummarizeCounters(evidence string) error {
	file, err := os.Open(filepath.Join(evidence, "fleet-raw.log"))
	if err != nil {
		return err
	}
	defer file.Close()
	bySample := map[int]map[int]fleetNativeCounts{}
	current := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "fleet sample begin ") {
			for _, field := range strings.Fields(line) {
				if strings.HasPrefix(field, "index=") {
					current, _ = strconv.Atoi(strings.TrimPrefix(field, "index="))
				}
			}
		}
		if strings.HasPrefix(line, "fleet sample end ") {
			current = 0
			continue
		}
		pid, counts, ok, err := fleetParseSnapshot(line)
		if err != nil {
			return err
		}
		if !ok || current == 0 {
			continue
		}
		if bySample[current] == nil {
			bySample[current] = map[int]fleetNativeCounts{}
		}
		sum := bySample[current][pid]
		sum.Snapshots += counts.Snapshots
		sum.Full += counts.Full
		sum.Named += counts.Named
		sum.Tables += counts.Tables
		sum.Commands += counts.Commands
		sum.Identities += counts.Identities
		bySample[current][pid] = sum
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(bySample, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(evidence, "native-counters-by-sample-pid.json"), data, 0600)
}

func fleetSchedule(raw string) ([]int, error) {
	if raw == "" {
		return []int{6, 20, 40, 20, 40, 6, 40, 6, 20}, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || (limit != 6 && limit != 20 && limit != 40) {
		return nil, fmt.Errorf("PAIR_STARTUP_FLEET_SMOKE_LIMIT must be empty, 6, 20, or 40")
	}
	return []int{limit}, nil
}

func runStartupFleetTiming(t *testing.T) {
	const fleetSize = 21
	schedule, err := fleetSchedule(os.Getenv("PAIR_STARTUP_FLEET_SMOKE_LIMIT"))
	if err != nil {
		t.Fatal(err)
	}
	evidence := os.Getenv("PAIR_STARTUP_FLEET_EVIDENCE")
	bin := os.Getenv("PAIR_STARTUP_PERF_BIN_DIR")
	if !filepath.IsAbs(bin) {
		t.Fatal("PAIR_STARTUP_PERF_BIN_DIR must contain candidate pair and pair-launch-helper")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	socketBase := "/tmp"
	if runtime.GOOS == "darwin" {
		socketBase = "/private/tmp"
	}
	socketRoot, err := os.MkdirTemp(socketBase, "p218.")
	if err != nil {
		t.Fatal(err)
	}
	var panes []fleetPane
	defer func() {
		if err := fleetRetainDiagnostics(socketRoot, evidence); err != nil {
			t.Error(err)
		}
		for _, pane := range panes {
			if err := pane.fixture.Close(); err != nil {
				t.Errorf("controlled zellij cleanup: %v", err)
			}
		}
		if err := os.RemoveAll(socketRoot); err != nil {
			t.Error(err)
		}
	}()
	socketRoot, err = filepath.EvalSymlinks(socketRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", socketRoot)
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fleetFixtureEnvironment(t, socketRoot, base)
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME", "PAIR_DATA_DIR", "COUCH_TREE", "COUCH_STORE_DIR", "COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "PAIR_COUCH_LAUNCH_PROFILE"} {
		t.Setenv(key, "")
	}
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("PAIR_STARTUP_TIMING", "1")
	t.Setenv("PAIR_HOME", root)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	data := filepath.Join(base, "data")
	t.Setenv("COUCH_PAIR_DATA_DIR", data)
	repo := filepath.Join(base, "repo")
	if out, err := exec.Command("git", "init", "-q", "-b", "main", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	scope, err := launcher.ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	ns, err := couchcore.ResolveCouchNamespace(filepath.Join(base, "couch"), repo)
	if err != nil {
		t.Fatal(err)
	}
	checker := couchcore.NewScopedThreadArtifactCollisionChecker(data)
	var currentConsole *couchtty.Console
	tracker := &fleetTracker{children: map[string]*fleetChild{}, markers: map[couchcore.ThreadAddress]string{}, adopted: map[couchcore.ThreadAddress]bool{}}
	inner := &couchcore.PtyRunner{LaunchHelper: filepath.Join(bin, "pair-launch-helper")}
	terminalProfile, err := runtimebundle.TerminalEnvironment(data)
	if err != nil {
		t.Fatal(err)
	}
	inner.Environment = func() ([]string, error) { return terminalProfile, nil }
	inner.Size = func() ptychild.Size {
		if currentConsole != nil {
			return currentConsole.ChildSize()
		}
		return ptychild.Size{Rows: 23, Cols: 100}
	}
	inner.Sink = func(ctx context.Context, id string, batch ptychild.OutputBatch) error {
		now := time.Now()
		tracker.mu.Lock()
		child := tracker.children[id]
		if child == nil {
			child = &fleetChild{}
			tracker.children[id] = child
		}
		room := 64*1024 - len(child.prefix)
		if room > len(batch.Raw) {
			room = len(batch.Raw)
		}
		child.prefix = append(child.prefix, batch.Raw[:room]...)
		endpoint := child.terminal
		address := child.address
		tracker.mu.Unlock()
		if endpoint != nil {
			frame, err := endpoint.Endpoint().Snapshot(now)
			if err == nil && fleetFrameReady(frame, tracker.markers[address]) {
				tracker.mu.Lock()
				if child.firstFrame.IsZero() {
					child.firstFrame = now
				}
				tracker.mu.Unlock()
			}
		}
		if currentConsole == nil {
			return errors.New("fleet output arrived without Console")
		}
		err := currentConsole.Deliver(ctx, id, batch)
		tracker.mu.Lock()
		if !child.firstFrame.IsZero() && child.deliveredFrame.IsZero() && err == nil {
			child.deliveredFrame = time.Now()
		}
		tracker.mu.Unlock()
		return err
	}
	c, err := couchcore.New(ns, fleetRunner{inner, tracker}, couchcore.OSPathOps{}, couchcore.ExecGit{}, couchcore.OSProcOps{}, couchcore.NewStore(ns.Dir()), couchcore.SystemClock{}, couchcore.NewRandomIDGen(), rand.Reader, checker)
	if err != nil {
		t.Fatal(err)
	}
	c.Layout = couchcore.Layout2
	setupCtx, cancelSetup := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancelSetup()
	var seed [6]byte
	if _, err := rand.Read(seed[:]); err != nil {
		t.Fatal(err)
	}
	slug := hex.EncodeToString(seed[:])
	agentScript := filepath.Join(base, "stand-in-agent")
	if err := os.WriteFile(agentScript, []byte("#!/bin/sh\nprintf '%s' \"$$\" > \"$1\"\nprintf '%s\\n' \"$2\"\nwhile IFS= read -r line; do :; done\n"), 0700); err != nil {
		t.Fatal(err)
	}
	panes = make([]fleetPane, 0, fleetSize)
	cliPath, err := exec.LookPath("zellij")
	if err != nil {
		t.Fatal(err)
	}
	manifest := fleetOwnershipManifest{SocketRoot: socketRoot, Zellij: cliPath, StoreDir: ns.Dir()}
	for _, key := range []string{"PATH", "HOME", "TMPDIR", "SHELL", "TERM", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR", "ZELLIJ_SOCKET_DIR", "ZELLIJ_CONFIG_DIR"} {
		manifest.Env = append(manifest.Env, key+"="+os.Getenv(key))
	}
	for i := 0; i < fleetSize; i++ {
		address := couchcore.ThreadAddress{RepoScope: scope.Key, Tag: couchcore.ThreadTag(fmt.Sprintf("couch-%016x", i+1))}
		working := filepath.Join(repo, fmt.Sprintf("thread-%02d", i))
		if err := os.Mkdir(working, 0700); err != nil {
			t.Fatal(err)
		}
		session := fmt.Sprintf("pair-f21-%s-%02d", slug, i)
		manifest.Sessions = append(manifest.Sessions, session)
		ownedJSON, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(evidence, "owned-fixtures.json"), ownedJSON, 0600); err != nil {
			t.Fatal(err)
		}
		fixture, err := pairlifecycletest.StartControlledZellij(setupCtx, session)
		if err != nil {
			t.Fatal(err)
		}
		pane := fleetPane{address: address, session: fixture.Session, marker: fmt.Sprintf("F21_%s_%02d", slug, i), pidFile: filepath.Join(base, fmt.Sprintf("agent-%02d.pid", i)), fixture: fixture}
		panes = append(panes, pane)
		tracker.markers[address] = pane.marker
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: scope.Key, Tag: string(address.Tag)})
		if err != nil {
			t.Fatal(err)
		}
		if err := launcher.EnsureThreadAddressForPair(data, scope, string(address.Tag), false); err != nil {
			t.Fatal(err)
		}
		scoped := launcher.NewScopedPaths(data, scope, string(address.Tag))
		line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{SessionName: pane.session, ScopeKey: scope.Key, RepoRoot: repo, RepoName: "fleet", Tag: string(address.Tag)})
		if err != nil {
			t.Fatal(err)
		}
		index, err := os.OpenFile(scoped.SessionBindings(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := io.WriteString(index, line+"\n")
		if err := errors.Join(writeErr, index.Close()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(paths.WorkbenchLayout(), []byte(string(launcher.Layout2)+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := fixture.AddOwnerWrapperPane(setupCtx, paths.ScrollbackRaw("claude")); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.CommandContext(setupCtx, "zellij", "--session", pane.session, "action", "new-pane", "--", agentScript, pane.pidFile, pane.marker).CombinedOutput(); err != nil {
			t.Fatalf("stand-in pane: %v %s", err, out)
		}
		if err := fleetWait(setupCtx, func() bool { raw, err := os.ReadFile(pane.pidFile); return err == nil && len(raw) > 0 }); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(pane.pidFile)
		if err != nil {
			t.Fatal(err)
		}
		pid, err := strconv.Atoi(string(raw))
		if err != nil {
			t.Fatal(err)
		}
		identity, err := (couchcore.OSProcOps{}).Identity(pid)
		if err != nil {
			t.Fatal(err)
		}
		panes[i].agent = couchcore.ProcessIdentity{PID: pid, Identity: identity}
		owned, err := (launcher.SessionOwnerProbe{}).Probe(setupCtx, pane.session, data, scope.Key, string(address.Tag))
		if err != nil || owned.State != launcher.SessionOwnerOwned {
			t.Fatalf("fixture owner: %+v %v", owned, err)
		}
		panes[i].server = owned.Server
		if err := fixture.KillClient(); err != nil {
			t.Fatal(err)
		}
		_, err = c.Threads.CreateThread(fleetThreadRecord(address, working, i))
		if err != nil {
			t.Fatal(err)
		}
	}
	sentinel := exec.Command("sleep", "1800")
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sentinel.Process.Kill(); _ = sentinel.Wait() }()
	sentinelID, err := (couchcore.OSProcOps{}).Identity(sentinel.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	previousLimit := couchcore.LifecycleParallelism
	defer func() { couchcore.LifecycleParallelism = previousLimit }()
	samples, err := os.OpenFile(filepath.Join(evidence, "samples.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer samples.Close()
	for sampleIndex, limit := range schedule {
		sample := runFleetSample(t, c, tracker, inner, &currentConsole, panes, repo, limit, sampleIndex+1, evidence)
		if err := json.NewEncoder(samples).Encode(sample); err != nil {
			t.Fatal(err)
		}
		if err := samples.Sync(); err != nil {
			t.Fatal(err)
		}
		if got, err := (couchcore.OSProcOps{}).Identity(sentinel.Process.Pid); err != nil || got != sentinelID {
			t.Fatalf("unrelated sentinel changed: %q %v", got, err)
		}
		if len(sample.Errors) > 0 {
			t.Fatalf("sample%d limit%d failed; retained failed attempt, no retry: %v", sample.Index, sample.Limit, sample.Errors)
		}
	}
}

type fleetNativeCounts struct{ Snapshots, Full, Named, Tables, Commands, Identities int }

func fleetThreadRecord(address couchcore.ThreadAddress, working string, index int) couchcore.ThreadRecord {
	return couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Revision: 1, Layout: couchcore.Layout2, Address: address, StartingPath: working, WorkingPath: working, CreatedAt: time.Now(), LastActiveAt: time.Now().Add(-time.Duration(index) * time.Minute), LatestLaunchProfile: &couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}}
}

func TestFleetSeededThreadRecordIsValid(t *testing.T) {
	record := fleetThreadRecord(couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}, t.TempDir(), 0)
	if err := couchcore.ValidateThreadRecord(record); err != nil {
		t.Fatal(err)
	}
	if record.Layout != couchcore.Layout2 {
		t.Fatalf("fixture record layout %q differs from its retained layout2 session", record.Layout)
	}
}

func fleetParseSnapshot(line string) (int, fleetNativeCounts, bool, error) {
	start := strings.Index(line, "pair startup snapshot:")
	if start < 0 {
		return 0, fleetNativeCounts{}, false, nil
	}
	fields := map[string]string{}
	for _, word := range strings.Fields(line[start+len("pair startup snapshot:"):]) {
		key, value, ok := strings.Cut(word, "=")
		if ok {
			fields[key] = value
		}
	}
	read := func(key string) (int, error) {
		n, err := strconv.Atoi(fields[key])
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid snapshot %s=%q", key, fields[key])
		}
		return n, nil
	}
	pid, err := read("pid")
	if err != nil || pid == 0 {
		return 0, fleetNativeCounts{}, true, fmt.Errorf("invalid snapshot PID")
	}
	counts := fleetNativeCounts{Snapshots: 1}
	switch fields["kind"] {
	case "full":
		counts.Full = 1
	case "named":
		counts.Named = 1
	default:
		return pid, counts, true, fmt.Errorf("invalid snapshot kind=%q", fields["kind"])
	}
	if counts.Tables, err = read("table_calls"); err != nil {
		return pid, counts, true, err
	}
	if counts.Commands, err = read("command_calls"); err != nil {
		return pid, counts, true, err
	}
	if counts.Identities, err = read("identity_calls"); err != nil {
		return pid, counts, true, err
	}
	return pid, counts, true, nil
}
func fleetFrameReady(frame pairterminal.Frame, marker string) bool {
	if !frame.AltScreen || marker == "" {
		return false
	}
	var text strings.Builder
	for _, cell := range frame.Cells {
		text.WriteString(cell.Content)
	}
	return strings.Contains(text.String(), marker)
}

func TestFleetReadinessRequiresAgentMarkerInActualAlternateFrame(t *testing.T) {
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	endpoint, err := pairterminal.NewEndpoint("fleet-test", pairterminal.Geometry{Rows: 24, Cols: 80}, host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(endpoint.Close)
	marker := "F21_real_agent"
	if _, err := endpoint.Feed([]byte("pair startup: phase=attach-handoff\r\n"+marker), time.Now()); err != nil {
		t.Fatal(err)
	}
	frame, err := endpoint.Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if fleetFrameReady(frame, marker) {
		t.Fatal("ordinary startup text counted as a Zellij frame")
	}
	if _, err := endpoint.Feed([]byte("\x1b[?1049h\x1b[2J\x1b[1;1H"+marker), time.Now()); err != nil {
		t.Fatal(err)
	}
	frame, err = endpoint.Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !fleetFrameReady(frame, marker) {
		t.Fatal("meaningful native pane frame not recognized")
	}
}

func TestFleetNativeCountsUseActualReportedCalls(t *testing.T) {
	pid, counts, ok, err := fleetParseSnapshot("child pair startup snapshot: pid=123 kind=named table_calls=1 command_calls=3 identity_calls=2 duration_ms=1.125")
	if err != nil || !ok || pid != 123 || counts.Tables != 1 || counts.Commands != 3 || counts.Identities != 2 || counts.Snapshots != 1 {
		t.Fatalf("parsed=%d %+v %v %v", pid, counts, ok, err)
	}
	if _, _, ok, err := fleetParseSnapshot("pair startup: phase=prelude delta_ms=1 elapsed_ms=2"); ok || err != nil {
		t.Fatalf("unrelated line parsed: %v %v", ok, err)
	}
	if _, _, _, err := fleetParseSnapshot("pair startup snapshot: pid=123 kind=named table_calls=1 command_calls=broken identity_calls=2 duration_ms=1"); err == nil {
		t.Fatal("malformed evidence silently became zero counts")
	}
}

func TestFleetScheduleOverrideIsBoundedAndExplicit(t *testing.T) {
	for _, raw := range []string{"0", "-1", "41", "6,20", "garbage"} {
		if _, err := fleetSchedule(raw); err == nil {
			t.Fatalf("accepted unsafe smoke schedule %q", raw)
		}
	}
	for _, raw := range []string{"6", "20", "40"} {
		schedule, err := fleetSchedule(raw)
		want, _ := strconv.Atoi(raw)
		if err != nil || len(schedule) != 1 || schedule[0] != want {
			t.Fatalf("smoke %q: %v %v", raw, schedule, err)
		}
	}
	schedule, err := fleetSchedule("")
	if err != nil || fmt.Sprint(schedule) != "[6 20 40 20 40 6 40 6 20]" {
		t.Fatalf("balanced schedule %v %v", schedule, err)
	}
}

func TestFleetDiagnosticsSurviveFixtureCleanup(t *testing.T) {
	for _, phase := range []string{"setup", "startup", "complete"} {
		t.Run(phase, func(t *testing.T) {
			root, evidence := t.TempDir(), t.TempDir()
			logDir := filepath.Join(root, "data", "logs")
			if err := os.MkdirAll(logDir, 0700); err != nil {
				t.Fatal(err)
			}
			log := filepath.Join(logDir, "pair-launch.log")
			if err := os.WriteFile(log, []byte("retained "+phase), 0600); err != nil {
				t.Fatal(err)
			}
			// Real roots contain sockets and links too. Never follow a link into
			// the operator's tree or block opening a special file.
			if err := os.Symlink("/unrelated/operator/path", filepath.Join(root, "external")); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(filepath.Join(root, "fifo"), 0600); err != nil {
				t.Fatal(err)
			}
			func() {
				defer os.RemoveAll(root)
				defer func() {
					if err := fleetRetainDiagnostics(root, evidence); err != nil {
						t.Error(err)
					}
				}()
				// The same defer must run on every early-return path.
				if phase != "complete" {
					return
				}
			}()
			raw, err := os.ReadFile(filepath.Join(evidence, "fixture-diagnostics", "data", "logs", "pair-launch.log"))
			if err != nil || string(raw) != "retained "+phase {
				t.Fatalf("diagnostics=%q err=%v", raw, err)
			}
			for _, special := range []string{"external", "fifo"} {
				if _, err := os.Lstat(filepath.Join(evidence, "fixture-diagnostics", special)); !os.IsNotExist(err) {
					t.Fatalf("special file copied: %s %v", special, err)
				}
			}
		})
	}
}

func TestFleetParentTimingContextsKeepConcurrentAttribution(t *testing.T) {
	var timings fleetParentTimings
	var wg sync.WaitGroup
	for _, tag := range []string{"root", "background-a", "background-b"} {
		wg.Go(func() {
			ctx := timings.context(t.Context(), "resume", tag)
			// Exercise the public observer through Detach is covered in
			// couchcore; this tests attribution at the collector boundary.
			observe := timings.observer("resume", tag)
			if ctx == t.Context() {
				t.Error("timing context was not scoped")
			}
			observe("record+baseline", 3*time.Millisecond)
		})
	}
	wg.Wait()
	got := timings.snapshot()
	if len(got) != 3 {
		t.Fatalf("events=%+v", got)
	}
	seen := map[string]bool{}
	for _, event := range got {
		if event.Operation != "resume" || event.Stage != "record+baseline" || event.DurationMS != 3 {
			t.Fatalf("wrong stage attribution: %+v", event)
		}
		seen[event.Tag] = true
	}
	if len(seen) != 3 {
		t.Fatalf("lost attribution: %+v", got)
	}
}

func TestFleetDiagnosticsRejectEmptyRootAndReportCopyFailure(t *testing.T) {
	if err := fleetRetainDiagnostics("", t.TempDir()); err == nil {
		t.Fatal("empty diagnostic root silently lost all evidence")
	}
	root, evidence := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(evidence, "fixture-diagnostics"), []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := fleetRetainDiagnostics(root, evidence); err == nil {
		t.Fatal("retention failure was silently discarded")
	}
}

func TestFleetParentTimingAcceptsInitialAttachWithoutContext(t *testing.T) {
	var timings fleetParentTimings
	// dispatchInitialAttach deliberately leaves OperationCall.Context nil.
	ctx := timings.context(nil, "attach", "root")
	if ctx == nil {
		t.Fatal("missing initial attach context")
	}
}

// Keep setup and spawned Pair clients in the same private fixture environment.
func fleetFixtureEnvironment(t *testing.T, socketRoot, base string) {
	t.Helper()
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	testpair.ApplyEnvironment(t, base, "TMPDIR="+socketRoot)
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		path := filepath.Join(base, key)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv(key, path)
	}
	// Socket addresses have a small Unix-domain path budget. Keep these
	// directly beneath the short private root, independent of inherited Couch
	// settings and testing.TempDir's longer per-test directory names.
	for key, leaf := range map[string]string{
		"ZELLIJ_SOCKET_DIR": "s", "XDG_RUNTIME_DIR": "r", "ZELLIJ_CONFIG_DIR": "c",
	} {
		path := filepath.Join(socketRoot, leaf)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv(key, path)
	}
}

func TestFleetFixtureEnvironmentIsolatesInheritedRuntimePaths(t *testing.T) {
	keys := []string{"ZELLIJ_SOCKET_DIR", "XDG_RUNTIME_DIR", "ZELLIJ_CONFIG_DIR"}
	for _, key := range keys {
		t.Setenv(key, "/inherited/operator/"+key)
	}
	root := t.TempDir()
	fleetFixtureEnvironment(t, root, filepath.Join(root, "fixture"))
	for _, key := range keys {
		path := os.Getenv(key)
		if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
			t.Fatalf("%s escaped private root: %q", key, path)
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
			t.Fatalf("%s directory is not private: %v %v", key, info, err)
		}
	}
	profile, err := runtimebundle.TerminalEnvironment(filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "child-environment")
	runner := &couchcore.PtyRunner{Environment: func() ([]string, error) { return profile, nil }}
	handle, err := runner.Start(root, []string{"/bin/sh", "-c", `printf '%s\n' "$ZELLIJ_SOCKET_DIR" "$XDG_RUNTIME_DIR" "$ZELLIJ_CONFIG_DIR" "$TERM" > "$1"`, "fixture", output}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.(couchcore.TerminalHandle).Terminal().Close()
	select {
	case <-handle.Exited():
	case <-time.After(5 * time.Second):
		_ = handle.Signal(os.Kill)
		t.Fatal("environment child did not exit")
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	want := os.Getenv(keys[0]) + "\n" + os.Getenv(keys[1]) + "\n" + os.Getenv(keys[2]) + "\npair-vt-256color\n"
	if string(raw) != want {
		t.Fatalf("child runtime environment=%q want %q", raw, want)
	}
}

func TestFleetPostQueryClientEvidencePreservesExactGenerationAndUncertainty(t *testing.T) {
	clients := map[string][]fleetClientBirth{"same": {{PID: 11, Identity: "birth-a"}}, "reused": {{PID: 12, Identity: "birth-b"}}, "missing": {{PID: 13, Identity: "birth-c"}}, "unreadable": {{PID: 14, Identity: "birth-d"}}}
	processes := []procutil.Observation{{Process: procutil.Process{PID: 11, Identity: "birth-a"}}, {Process: procutil.Process{PID: 12, Identity: "replacement"}}, {Process: procutil.Process{PID: 14, Identity: "birth-d"}, Err: errors.New("unreadable argv")}}
	got := fleetPostQueryClients(clients, processes, nil)
	if !got["same"][0].SameGeneration || got["reused"][0].SameGeneration || got["missing"][0].Found || got["unreadable"][0].SameGeneration || got["unreadable"][0].Error == "" {
		t.Fatalf("client evidence lost uncertainty: %+v", got)
	}
	failed := fleetPostQueryClients(clients, nil, errors.New("snapshot failed"))
	if failed["same"][0].Error != "snapshot failed" || failed["same"][0].SameGeneration {
		t.Fatalf("snapshot failure became liveness: %+v", failed)
	}
	if got["same"][0].ObservedAt.IsZero() {
		t.Fatal("missing post-query observation time")
	}
}

func TestFleetClientBirthValidationRequiresExactlyOneKnownClientPerExpectedTag(t *testing.T) {
	panes := []fleetPane{{address: couchcore.ThreadAddress{Tag: "a"}}, {address: couchcore.ThreadAddress{Tag: "b"}}}
	known := fleetClientBirth{PID: 11, Identity: "generation"}
	for _, tc := range []struct {
		name      string
		births    map[string][]fleetClientBirth
		wantError bool
	}{
		{"complete", map[string][]fleetClientBirth{"a": {known}, "b": {known}}, false},
		{"missing", map[string][]fleetClientBirth{"a": {known}}, true},
		{"duplicate", map[string][]fleetClientBirth{"a": {known, known}, "b": {known}}, true},
		{"unknown", map[string][]fleetClientBirth{"a": {{PID: 11}}, "b": {known}}, true},
		{"unexpected", map[string][]fleetClientBirth{"a": {known}, "b": {known}, "outside": {known}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := fleetClientBirthErrors(panes, tc.births)
			if (len(got) > 0) != tc.wantError {
				t.Fatalf("validation=%v wantError=%v", got, tc.wantError)
			}
		})
	}
}

func TestFleetPostQueryClientValidationRejectsContradictoryOrUnknownEvidence(t *testing.T) {
	for _, state := range []string{"replaced", "absent-from-snapshot", "unknown"} {
		got := fleetPostQueryClientErrors(map[string][]fleetClientObservation{"tag": {{PID: 11, State: state, Error: "evidence", SameGeneration: false}}})
		if len(got) != 1 || !strings.Contains(got[0], state) || !strings.Contains(got[0], "tag") {
			t.Fatalf("state=%s errors=%v", state, got)
		}
	}
	if got := fleetPostQueryClientErrors(map[string][]fleetClientObservation{"tag": {{PID: 11, State: "same-generation", SameGeneration: true}}}); len(got) != 0 {
		t.Fatalf("known live generation refused: %v", got)
	}
}

func TestFleetQueryCollectorScopesConcurrentParentOperations(t *testing.T) {
	dir := t.TempDir()
	// The reply fixture has no native endpoints; keep admission inside its root.
	t.Setenv("HOME", dir)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(dir, "native"))
	path := filepath.Join(dir, "zellij")
	script := "#!/bin/sh\ncase \"$*\" in\n" +
		" 'list-sessions --short') " + zellijtest.ShellCase(t, "couch.fleet.short") + " ;;\n" +
		" 'list-sessions --no-formatting') " + zellijtest.ShellCase(t, "couch.fleet.full") + " ;;\n" +
		" '--session pair-fixture action list-clients') " + zellijtest.ShellCase(t, "couch.fleet.clients") + " ;;\n *) exit 2 ;;\nesac\n"

	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	var collector, unrelated fleetQueryCollector
	var timings fleetParentTimings
	type inheritedKey struct{}
	parent := context.WithValue(t.Context(), inheritedKey{}, "retained")
	var wg sync.WaitGroup
	cases := []struct{ phase, operation, tag string }{{"startup", "root", "root"}, {"operation", "resume", "background"}, {"operation", "leave", "fleet"}, {"attached", "verify", "fleet"}, {"postleave", "verify", "fleet"}}
	for _, tc := range cases {
		wg.Go(func() {
			ctx := collector.context(timings.context(parent, tc.operation, tc.tag), tc.phase, tc.operation, tc.tag)
			if ctx.Value(inheritedKey{}) != "retained" {
				t.Error("composed observer lost parent context")
			}
			rows, err := (launcher.ZellijSource{Path: path}).SnapshotSessionsContext(ctx, []string{"pair-fixture"})
			if err != nil || len(rows) != 1 || rows[0].State != launcher.SessionAttached {
				t.Errorf("parser changed: %+v %v", rows, err)
			}
		})
	}
	wg.Wait()
	got, err := collector.snapshot()
	if err != nil || len(got) != 2*len(cases) {
		t.Fatalf("events=%d err=%v", len(got), err)
	}
	counts := map[string]int{}
	for _, e := range got {
		counts[e.Phase+"/"+e.Operation+"/"+e.Tag]++
		if e.ExitCode != 0 || len(e.Stdout) == 0 {
			t.Fatalf("lost query data: %+v", e)
		}
	}
	for _, tc := range cases {
		if counts[tc.phase+"/"+tc.operation+"/"+tc.tag] != 2 {
			t.Fatalf("lost attribution: %v", counts)
		}
	}
	if other, err := unrelated.snapshot(); err != nil || len(other) != 0 {
		t.Fatalf("collector leaked across samples: %d %v", len(other), err)
	}
}

func TestFleetQueryCollectorReportsByteAndEventBounds(t *testing.T) {
	var bytesBound fleetQueryCollector
	observe := bytesBound.observer("operation", "leave", "fleet")
	raw := make([]byte, 64*1024)
	for i := 0; i <= fleetQueryByteLimit/len(raw); i++ {
		observe(launcher.ZellijQueryObservation{Stdout: raw, StdoutBytes: len(raw)})
	}
	events, err := bytesBound.snapshot()
	total := 0
	for _, e := range events {
		total += len(e.Stdout) + len(e.Stderr)
	}
	if err == nil || total > fleetQueryByteLimit || !events[len(events)-1].StdoutTruncated {
		t.Fatalf("byte bound=%d err=%v", total, err)
	}
	var eventBound fleetQueryCollector
	for i := 0; i <= fleetQueryEventLimit; i++ {
		eventBound.observer("operation", "resume", "tag")(launcher.ZellijQueryObservation{})
	}
	events, err = eventBound.snapshot()
	if err == nil || len(events) != fleetQueryEventLimit || !strings.Contains(err.Error(), "omitted=1") {
		t.Fatalf("event bound=%d err=%v", len(events), err)
	}
	var sourceTruncated fleetQueryCollector
	sourceTruncated.observer("operation", "resume", "tag")(launcher.ZellijQueryObservation{StdoutTruncated: true})
	if _, err := sourceTruncated.snapshot(); err == nil {
		t.Fatal("source truncation silently accepted as complete evidence")
	}
}

func TestFleetQueryCollectorAttributesLeaveFanoutBySession(t *testing.T) {
	collector := fleetQueryCollector{sessionTags: map[string]string{"pair-first": "first-tag", "pair-second": "second-tag"}}
	observe := collector.observer("operation", "leave", "fleet")
	observe(launcher.ZellijQueryObservation{Args: []string{"--session", "pair-first", "action", "list-panes"}})
	observe(launcher.ZellijQueryObservation{Args: []string{"--session", "pair-second", "action", "list-panes"}})
	events, err := collector.snapshot()
	if err != nil || len(events) != 2 || events[0].Tag != "first-tag" || events[1].Tag != "second-tag" || events[1].Operation != "leave" {
		t.Fatalf("fanout attribution=%+v err=%v", events, err)
	}
}
