package couchcore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
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

type startupOwnerTiming struct{ phases *startupPhases }

func (p startupOwnerTiming) Probe(ctx context.Context, name, data, scope, tag string) (launcher.SessionOwnerObservation, error) {
	start := time.Now()
	defer p.phases.add("owner.Probe", start)
	return (launcher.SessionOwnerProbe{}).Probe(ctx, name, data, scope, tag)
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
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		dir := filepath.Join(base, key)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv(key, dir)
	}
	t.Setenv("SHELL", "/bin/sh")
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME", "PAIR_DATA_DIR", "COUCH_TREE", "COUCH_STORE_DIR", "COUCH_THREAD_TAG", "COUCH_THREAD_SCOPE", "PAIR_COUCH_LAUNCH_PROFILE"} {
		t.Setenv(key, "")
	}
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
		if err := os.MkdirAll(bin, 0700); err != nil {
			t.Fatal(err)
		}
		for _, target := range []struct{ name, pkg string }{{"pair", "./cmd/pair-go"}, {"pair-launch-helper", "./cmd/pair-launch-helper"}} {
			buildCtx, cancelBuild := context.WithTimeout(t.Context(), 3*time.Minute)
			cmd := exec.CommandContext(buildCtx, "go", "build", "-o", filepath.Join(bin, target.name), target.pkg)
			cmd.Env = buildEnv
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			cancelBuild()
			if err != nil {
				t.Fatalf("build %s: %v %s", target.name, err, out)
			}
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PAIR_HOME", root)
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
	checker.OwnerProbe = startupOwnerTiming{phases}
	runner := &PtyRunner{LaunchHelper: filepath.Join(bin, "pair-launch-helper")}
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
	t.Cleanup(func() { _ = fixture.Close() })
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
	sentinel := exec.Command("sleep", "60")
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sentinel.Process.Kill(); _ = sentinel.Wait() })
	sentinelID, err := (OSProcOps{}).Identity(sentinel.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, handle, err := c.ResumeContextWith(ctx, record.Address, ResumeOptions{WarmOnly: true})
	phases.report(t, "managed_resume", time.Since(start))
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
			_, output := handle.(interface{ LaunchOutput() (string, []byte) }).LaunchOutput()
			t.Fatalf("readiness query: %v child_output=%q", queryErr, output)
		}
		if len(rows) == 1 && rows[0].State == launcher.SessionAttached {
			break
		}
		select {
		case <-handle.Exited():
			_, output := handle.(interface{ LaunchOutput() (string, []byte) }).LaunchOutput()
			t.Fatalf("Pair exited before attach readiness: code=%d output=%q", handle.Wait(), output)
		case <-ctx.Done():
			_, output := handle.(interface{ LaunchOutput() (string, []byte) }).LaunchOutput()
			t.Fatalf("attach readiness: %v child_output=%q", ctx.Err(), output)
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Logf("managed_attach_ready after_parent_return=%s startup_total=%s", time.Since(ready), time.Since(start))
	start = time.Now()
	detached, err := c.Detach(ctx, record.Address)
	phases.report(t, "detach", time.Since(start))
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
