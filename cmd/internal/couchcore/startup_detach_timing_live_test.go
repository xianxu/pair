package couchcore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

type startupPhase struct {
	name    string
	elapsed time.Duration
}
type startupPhases struct {
	mu   sync.Mutex
	rows []startupPhase
}

func (p *startupPhases) add(name string, started time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rows = append(p.rows, startupPhase{name, time.Since(started)})
}
func (p *startupPhases) report(t *testing.T, operation string, wall time.Duration) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	var measured time.Duration
	for i, row := range p.rows {
		measured += row.elapsed
		t.Logf("%s phase[%d] %s=%s", operation, i+1, row.name, row.elapsed)
	}
	t.Logf("%s wall=%s exclusive_measured=%s residual=%s", operation, wall, measured, wall-measured)
	p.rows = nil
}

type startupOwnerTiming struct {
	phases      *startupPhases
	diagnostics *testing.T
}

func (p startupOwnerTiming) Probe(ctx context.Context, name, data, scope, tag string) (launcher.SessionOwnerObservation, error) {
	start := time.Now()
	defer p.phases.add("owner.Probe", start)
	observed, err := (launcher.SessionOwnerProbe{}).Probe(ctx, name, data, scope, tag)
	if p.diagnostics != nil {
		info, socketErr := os.Lstat(observed.Server.Socket)
		var socketMode os.FileMode
		if info != nil {
			socketMode = info.Mode()
		}
		p.diagnostics.Logf("owner boundary session=%s state=%d server=%+v socket_mode=%s socket_error=%v probe_error=%v", name, observed.State, observed.Server, socketMode, socketErr, err)
	}
	return observed, err
}
func (p startupOwnerTiming) Revalidate(ctx context.Context, value launcher.SessionOwnerObservation) error {
	start := time.Now()
	defer p.phases.add("owner.Revalidate", start)
	return (launcher.SessionOwnerProbe{}).Revalidate(ctx, value)
}

type startupProcTiming struct {
	OSProcOps
	phases *startupPhases
}

func (p startupProcTiming) SignalGroup(pid int, signal os.Signal) error {
	start := time.Now()
	defer p.phases.add("proc.SignalGroup", start)
	return p.OSProcOps.SignalGroup(pid, signal)
}
func (p startupProcTiming) Exists(pid int) Liveness {
	start := time.Now()
	defer p.phases.add("proc.Exists", start)
	return p.OSProcOps.Exists(pid)
}
func (p startupProcTiming) Identity(pid int) (string, error) {
	start := time.Now()
	defer p.phases.add("proc.Identity", start)
	return p.OSProcOps.Identity(pid)
}

// This baseline runs the production parent ResumeContextWith and Detach and
// production Pair executable in a private PTY. The agent pane is a shell
// stand-in; it witnesses actual scoped owner argv, but starts no real agent.
// All fixture/build/readiness work is outside the timed operation windows.
func TestStartupManagedAttachDetachTimingLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" || os.Getenv("PAIR_LIVE_STARTUP_PERF") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1")
	}
	t.Setenv("TMPDIR", "/tmp")
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Keep build caches outside the private fixture HOME. They carry compiled
	// code only; runtime files and configuration remain private.
	buildEnv := os.Environ()
	if os.Getenv("PAIR_STARTUP_PERF_BIN_DIR") == "" {
		bin := filepath.Join(base, "bin")
		if err := os.MkdirAll(bin, 0700); err != nil {
			t.Fatal(err)
		}
		buildCtx, cancelBuild := context.WithTimeout(t.Context(), 3*time.Minute)
		testpair.AtContext(t, buildCtx, filepath.Join(bin, "pair"))
		cancelBuild()
	}
	var explicit []string
	for _, key := range []string{"PAIR_STARTUP_PERF_BIN_DIR", "PAIR_STARTUP_PERF_ISOLATE_SOCKET", "PAIR_STARTUP_PERF_DIAGNOSTICS", "PAIR_STARTUP_PERF_DIAG_DIR"} {
		if value, ok := os.LookupEnv(key); ok {
			explicit = append(explicit, key+"="+value)
		}
	}
	home := filepath.Join(base, "HOME")
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		dir := filepath.Join(base, key)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		explicit = append(explicit, key+"="+dir)
	}
	explicit = append(explicit, "SHELL=/bin/sh", "TMPDIR=/tmp", "PAIR_STARTUP_TIMING=1")
	testpair.ApplyEnvironment(t, home, explicit...)
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(base, "bin")
	if supplied := os.Getenv("PAIR_STARTUP_PERF_BIN_DIR"); supplied != "" {
		if !filepath.IsAbs(supplied) {
			t.Fatal("PAIR_STARTUP_PERF_BIN_DIR must be absolute")
		}
		bin = supplied
	} else {
		helper := startupHelperExecutable(t, &startupHelperCache, root, buildEnv)
		if err := os.Symlink(helper, filepath.Join(bin, "pair-launch-helper")); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PAIR_HOME", root)
	if os.Getenv("PAIR_STARTUP_PERF_ISOLATE_SOCKET") == "1" {
		socketRoot, err := os.MkdirTemp("/private/tmp", "p218.")
		if err != nil {
			t.Fatal(err)
		}
		socketRoot, err = filepath.EvalSymlinks(socketRoot)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(socketRoot); err != nil {
				t.Errorf("remove private socket root: %v", err)
			}
		})
		t.Setenv("TMPDIR", socketRoot)
		t.Logf("isolated socket TMPDIR=%s; socket workload excludes co-tenant sessions", socketRoot)
	}
	if os.Getenv("PAIR_STARTUP_PERF_DIAGNOSTICS") == "1" {
		realZellij, err := exec.LookPath("zellij")
		if err != nil {
			t.Fatal(err)
		}
		realZellij, err = filepath.EvalSymlinks(realZellij)
		if err != nil {
			t.Fatal(err)
		}
		diagnosticRoot := os.Getenv("PAIR_STARTUP_PERF_DIAG_DIR")
		if !filepath.IsAbs(diagnosticRoot) {
			t.Fatal("PAIR_STARTUP_PERF_DIAG_DIR must be an absolute private directory")
		}
		diagnosticDir := filepath.Join(diagnosticRoot, fmt.Sprintf("run-%d-%d", os.Getpid(), time.Now().UnixNano()))
		if err := os.MkdirAll(diagnosticDir, 0700); err != nil {
			t.Fatal(err)
		}
		diagnosticBin := filepath.Join(base, "zellij-diagnostic-bin")
		if err := os.Mkdir(diagnosticBin, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PAIR_STARTUP_PERF_BOUNDARY_TRACE", filepath.Join(diagnosticDir, "zellij-boundaries.txt"))
		// exec preserves the native server generation. Runtime/config evidence
		// belongs only to this explicitly requested private diagnostic run.
		quotedReal := "'" + strings.ReplaceAll(realZellij, "'", "'\\''") + "'"
		script := "#!/bin/sh\n{ printf 'pid=%s ppid=%s HOME=%s TMPDIR=%s XDG_RUNTIME_DIR=%s XDG_CONFIG_HOME=%s XDG_DATA_HOME=%s ZELLIJ_SOCKET_DIR=%s ZELLIJ_CONFIG_DIR=%s PAIR_HOME=%s PATH=%s' \"$$\" \"$PPID\" \"$HOME\" \"$TMPDIR\" \"$XDG_RUNTIME_DIR\" \"$XDG_CONFIG_HOME\" \"$XDG_DATA_HOME\" \"$ZELLIJ_SOCKET_DIR\" \"$ZELLIJ_CONFIG_DIR\" \"$PAIR_HOME\" \"$PATH\"; for arg do printf ' arg=%s' \"$arg\"; done; printf '\\n'; } >> \"$PAIR_STARTUP_PERF_BOUNDARY_TRACE\"\n" +
			"capture=0\nfor arg do case \"$arg\" in list-panes) capture=1;; esac; done\n" +
			"if [ \"$capture\" = 1 ]; then\n" + quotedReal + " \"$@\" > \"$PAIR_STARTUP_PERF_BOUNDARY_TRACE.stdout.$$\" 2> \"$PAIR_STARTUP_PERF_BOUNDARY_TRACE.stderr.$$\"\nstatus=$?\ncat \"$PAIR_STARTUP_PERF_BOUNDARY_TRACE.stdout.$$\"\ncat \"$PAIR_STARTUP_PERF_BOUNDARY_TRACE.stderr.$$\" >&2\nexit \"$status\"\nfi\nexec " + quotedReal + " \"$@\"\n"
		if err := os.WriteFile(filepath.Join(diagnosticBin, "zellij"), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", diagnosticBin+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Logf("diagnostic real_zellij=%s pair_bin=%s config=%s evidence=%s", realZellij, bin, filepath.Join(root, "zellij"), diagnosticDir)
	}
	repo := filepath.Join(base, "repo")
	if out, err := exec.Command("git", "init", "-q", "-b", "main", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	scope, err := launcher.ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(base, "data")
	t.Setenv("COUCH_PAIR_DATA_DIR", data)
	ns, err := ResolveCouchNamespace(filepath.Join(base, "couch"), "/unused")
	if err != nil {
		t.Fatal(err)
	}
	phases := &startupPhases{}
	checker := NewScopedThreadArtifactCollisionChecker(data)
	ownerTiming := startupOwnerTiming{phases: phases}
	if os.Getenv("PAIR_STARTUP_PERF_DIAGNOSTICS") == "1" {
		ownerTiming.diagnostics = t
	}
	checker.OwnerProbe = ownerTiming
	// The regular failure tail can lose startup lines once Zellij renders its
	// screen. Preserve a bounded prefix at the real PTY sink before first byte.
	var childOutputMu sync.Mutex
	var childOutputPrefix []byte
	runner := &PtyRunner{LaunchHelper: filepath.Join(bin, "pair-launch-helper"), Sink: func(_ context.Context, _ string, batch ptychild.OutputBatch) error {
		childOutputMu.Lock()
		defer childOutputMu.Unlock()
		remaining := 64*1024 - len(childOutputPrefix)
		if remaining > len(batch.Raw) {
			remaining = len(batch.Raw)
		}
		childOutputPrefix = append(childOutputPrefix, batch.Raw[:remaining]...)
		return nil
	}}
	c, err := New(ns, runner, OSPathOps{}, ExecGit{}, startupProcTiming{phases: phases}, NewStore(ns.Dir()), SystemClock{}, NewFixedIDGen("baseline"), rand.Reader, checker)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	var entropy [8]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	session := "pair-perf-" + hex.EncodeToString(entropy[:])
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Errorf("controlled zellij cleanup: %v", err)
		}
	})
	record := warmDetachedRecord(t)
	record.Address.RepoScope = scope.Key
	record.StartingPath, record.WorkingPath = repo, repo
	record.Reservation = false
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: scope.Key, Tag: string(record.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := launcher.EnsureThreadAddressForPair(data, scope, string(record.Address.Tag), false); err != nil {
		t.Fatal(err)
	}
	indexSessionForConformance(t, data, record.Address, session)
	// A normal Pair-created session already has its saved layout. This stand-in
	// fixture omits editors, so seed that durable metadata before measurement.
	if err := os.WriteFile(paths.WorkbenchLayout(), []byte(string(launcher.Layout2)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := fixture.AddOwnerWrapperPane(ctx, paths.ScrollbackRaw("claude")); err != nil {
		t.Fatal(err)
	}
	if err := fixture.KillClient(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Threads.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	diagnoseFailure := func() {
		if os.Getenv("PAIR_STARTUP_PERF_DIAGNOSTICS") != "1" {
			return
		}
		diagnosticContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = ownerTiming.Probe(diagnosticContext, session, data, scope.Key, string(record.Address.Tag))
		out, err := exec.CommandContext(diagnosticContext, "zellij", "--config-dir", filepath.Join(root, "zellij"), "list-sessions", "--short").CombinedOutput()
		var selected []string
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, session) {
				selected = append(selected, line)
			}
		}
		t.Logf("failed handoff selected config session rows=%q query_error=%v", selected, err)
	}
	sentinel := exec.Command("sleep", "60")
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sentinel.Process.Kill(); _ = sentinel.Wait() })
	sentinelID, err := (OSProcOps{}).Identity(sentinel.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	stagePhases := &startupPhases{}
	ctx = context.WithValue(ctx, launchTimingObserverKey{}, launchTimingObserver(func(name string, elapsed time.Duration) {
		stagePhases.mu.Lock()
		stagePhases.rows = append(stagePhases.rows, startupPhase{name, elapsed})
		stagePhases.mu.Unlock()
	}))
	start := time.Now()
	_, handle, err := c.ResumeContextWith(ctx, record.Address, ResumeOptions{WarmOnly: true})
	resumeWall := time.Since(start)
	phases.report(t, "managed_resume", resumeWall)
	stagePhases.report(t, "managed_resume_stages", resumeWall)
	t.Log("owner/proc phases are nested inside launch stages; do not add the two measured totals")
	if handle != nil {
		t.Cleanup(func() { _ = handle.Signal(TermSignal); handle.Wait() })
	}
	if err != nil {
		t.Fatal(err)
	}
	// Registration proves owner presence, not that the child has finished its
	// own checks and attached. Wait for exact selected-session client evidence
	// before detach, and report that separately from parent startup return.
	ready := time.Now()
	for {
		rows, queryErr := (launcher.ZellijSource{}).SnapshotSessionsContext(ctx, []string{session})
		if queryErr != nil {
			diagnoseFailure()
			_, output := handle.(interface{ LaunchOutput() (string, []byte) }).LaunchOutput()
			t.Fatalf("readiness query: %v child_output=%q", queryErr, output)
		}
		if len(rows) == 1 && rows[0].State == launcher.SessionAttached {
			break
		}
		select {
		case <-handle.Exited():
			diagnoseFailure()
			_, output := handle.(interface{ LaunchOutput() (string, []byte) }).LaunchOutput()
			t.Fatalf("Pair exited before attach readiness: code=%d output=%q", handle.Wait(), output)
		case <-ctx.Done():
			diagnoseFailure()
			_, output := handle.(interface{ LaunchOutput() (string, []byte) }).LaunchOutput()
			t.Fatalf("attach readiness: %v child_output=%q", ctx.Err(), output)
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Logf("managed_attach_ready after_parent_return=%s startup_total=%s", time.Since(ready), time.Since(start))
	childOutputMu.Lock()
	childStartupOutput := string(childOutputPrefix)
	childOutputMu.Unlock()
	for _, line := range strings.Split(childStartupOutput, "\n") {
		if at := strings.Index(line, "pair startup:"); at >= 0 {
			t.Logf("child %s", strings.TrimSpace(line[at:]))
		}
	}
	t.Log("child deltas partition startup through blocking handoff entry; elapsed_ms is cumulative; parent registration overlaps child startup")
	start = time.Now()
	detached, err := c.Detach(ctx, record.Address)
	detachWall := time.Since(start)
	phases.report(t, "detach", detachWall)
	stagePhases.report(t, "detach_stages", detachWall)
	if err != nil {
		t.Fatal(err)
	}
	if len(detached.Incarnations) != 0 {
		t.Fatalf("incarnation not retired: %+v", detached.Incarnations)
	}
	if got, err := (OSProcOps{}).Identity(sentinel.Process.Pid); err != nil || got != sentinelID {
		t.Fatalf("unrelated sentinel changed: %q %v", got, err)
	}
	rows, err := (launcher.ZellijSource{}).SnapshotSessionsContext(ctx, []string{session})
	if err != nil || len(rows) != 1 || rows[0].State != launcher.SessionDetached {
		t.Fatalf("session survival: %+v %v", rows, err)
	}
	t.Log(fmt.Sprintf("sentinel_survived=true owned_session=%s retired=true; parent phases exclude child's owner reads; use zellij shim trace and launcher owner timing companion", session))
}
