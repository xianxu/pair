package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// brokenDep is a declared dependency with a local-only commit, a dirty file,
// an untracked file and an ignored file, made unreadable by losing .git/HEAD.
func brokenDep(t *testing.T, s *observedSlot) string {
	t.Helper()
	dep := s.addDep(t, fakeDep)
	write(t, filepath.Join(dep, ".gitignore"), "ignored.txt\n")
	s.f.git(dep, "add", ".gitignore")
	s.f.git(dep, "commit", "-q", "-m", "local only")
	write(t, filepath.Join(dep, "construct", "base.manifest"), "# dirty\n")
	write(t, filepath.Join(dep, "untracked.txt"), "keep me")
	write(t, filepath.Join(dep, "ignored.txt"), "keep me too")
	os.Remove(filepath.Join(dep, ".git", "HEAD"))
	return dep
}

func readManifests(t *testing.T, l SlotLayout) []SavedWorkManifest {
	t.Helper()
	entries, _ := os.ReadDir(l.SavedWork())
	var out []SavedWorkManifest
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(l.SavedWork(), e.Name(), "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		var m SavedWorkManifest
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

func TestSetAsideMovesTheWholeCheckoutAndCanBeRestored(t *testing.T) {
	s := newObservedSlot(t)
	dep := brokenDep(t, s)
	registered := ""
	cv := s.converger(t)
	cv.registerStore = func(_ context.Context, path string) error { registered = path; return nil }
	if err := cv.converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dep); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the dependency is still at %s", dep)
	}
	if registered != s.layout.Store() {
		t.Fatalf("store registered as %q before the move, want %q", registered, s.layout.Store())
	}
	ms := readManifests(t, s.layout)
	if len(ms) != 1 || ms[0].State != "complete" || ms[0].Kind != "dep" || ms[0].Path != dep {
		t.Fatalf("manifests %+v", ms)
	}
	entries, _ := os.ReadDir(s.layout.SavedWork())
	tree := filepath.Join(s.layout.SavedWork(), entries[0].Name(), "tree")
	for _, name := range []string{"untracked.txt", "ignored.txt"} {
		if raw, err := os.ReadFile(filepath.Join(tree, name)); err != nil || string(raw) != "keep me"+map[string]string{"untracked.txt": "", "ignored.txt": " too"}[name] {
			t.Fatalf("%s not preserved: %q %v", name, raw, err)
		}
	}
	// Restore as the manifest says, repair HEAD: the local-only commit is there.
	if err := os.Rename(tree, dep); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dep, ".git", "HEAD"), "ref: refs/heads/main\n")
	if got := s.f.git(dep, "log", "-1", "--format=%s"); got != "local only" {
		t.Fatalf("restored HEAD commit %q", got)
	}
}

func TestSetAsideRefusals(t *testing.T) {
	t.Run("entry limit", func(t *testing.T) {
		s := newObservedSlot(t)
		dep := brokenDep(t, s)
		for i := 0; i < MaxSavedWork; i++ {
			os.MkdirAll(s.layout.SavedWorkEntry("old", time.Unix(int64(i), 0)), 0o700)
		}
		err := s.converger(t).converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep})
		if err == nil || !strings.Contains(err.Error(), StopReasonSavedWorkFull) {
			t.Fatalf("err = %v", err)
		}
		if _, err := os.Lstat(dep); err != nil {
			t.Fatal("moved despite the limit")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		s := newObservedSlot(t)
		target := t.TempDir()
		link := filepath.Join(s.layout.Env(), fakeDep)
		os.Symlink(target, link)
		err := s.converger(t).converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: link})
		if err == nil {
			t.Fatal("set aside a symlink")
		}
	})
	t.Run("weave lock held", func(t *testing.T) {
		s := newObservedSlot(t)
		dep := brokenDep(t, s)
		holdLock(t, s.layout.SetupLock())
		err := s.converger(t).converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep})
		if !errors.Is(err, errSetupRunning) {
			t.Fatalf("err = %v, want setup running", err)
		}
		if _, err := os.Lstat(dep); err != nil {
			t.Fatal("moved while weave was running")
		}
	})
	t.Run("an agent appears", func(t *testing.T) {
		s := newObservedSlot(t)
		dep := brokenDep(t, s)
		cv := s.converger(t)
		cv.agentNow = func(context.Context) EvidenceAgent { return AgentDetached }
		err := cv.converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep})
		if err == nil || !strings.Contains(err.Error(), StopReasonAgentLive) {
			t.Fatalf("err = %v", err)
		}
		if _, err := os.Lstat(dep); err != nil {
			t.Fatal("moved under a live agent")
		}
	})
}

func TestSetAsideHostRecordsItsBranch(t *testing.T) {
	s := newObservedSlot(t)
	s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
	write(t, filepath.Join(s.layout.Host(), ".git"), "gitdir: /nonexistent/admin\n")
	err := s.converger(t).converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: ResourceHost, Path: s.layout.Host(), Branch: "issue-work"})
	if err != nil {
		t.Fatal(err)
	}
	if ms := readManifests(t, s.layout); len(ms) != 1 || ms[0].Kind != "host" || ms[0].Branch != "issue-work" {
		t.Fatalf("manifests %+v", ms)
	}
}

// TestSetAsideCrashAtTheRenameLeavesNoPartialTree: the rename seam is where a
// crash can land; the manifest is pending, the checkout is whole in one place.
func TestSetAsideCrashAtTheRenameLeavesNoPartialTree(t *testing.T) {
	s := newObservedSlot(t)
	dep := brokenDep(t, s)
	cv := s.converger(t)
	cv.rename = func(oldpath, newpath string) error { return errors.New("crash at the rename seam") }
	if err := cv.converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep}); err == nil {
		t.Fatal("no error from the injected crash")
	}
	if _, err := os.Stat(filepath.Join(dep, "untracked.txt")); err != nil {
		t.Fatal("the checkout is no longer whole at its path")
	}
	ms := readManifests(t, s.layout)
	if len(ms) != 1 || ms[0].State != "pending" || ms[0].Path != dep {
		t.Fatalf("manifests %+v, want one pending record naming the checkout", ms)
	}
	entries, _ := os.ReadDir(s.layout.SavedWork())
	if _, err := os.Lstat(filepath.Join(s.layout.SavedWork(), entries[0].Name(), "tree")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a partial tree exists in saved work")
	}
}
