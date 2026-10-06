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

	"github.com/xianxu/pair/cmd/internal/storagegc"
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
	cv := s.converger(t)
	if err := cv.converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dep); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the dependency is still at %s", dep)
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
		if !errors.Is(err, errSavedWorkFull) {
			t.Fatalf("err = %v", err)
		}
		if f := ClassifyConvergeError(PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep)}, err); f.Class != FailureHold {
			t.Fatalf("classified %+v, want a hold", f)
		}
		if entries, _ := os.ReadDir(s.layout.SavedWork()); len(entries) != MaxSavedWork {
			t.Fatalf("%d entries, want the %d old ones and no new one", len(entries), MaxSavedWork)
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
		defer assertNoSavedWork(t, s.layout)
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
		defer assertNoSavedWork(t, s.layout)
		cv := s.converger(t)
		cv.agentNow = func(context.Context) EvidenceAgent { return AgentDetached }
		err := cv.converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep})
		if !errors.Is(err, errAgentAppeared) {
			t.Fatalf("err = %v", err)
		}
		if f := ClassifyConvergeError(PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep)}, err); f.Class != FailureHold || f.Cause != StopReasonAgentLive {
			t.Fatalf("classified %+v, want the live-agent hold the plan gives", f)
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

// assertNoSavedWork: a refused set-aside leaves no entry behind (it would
// count toward the cap).
func assertNoSavedWork(t *testing.T, l SlotLayout) {
	t.Helper()
	if entries, _ := os.ReadDir(l.SavedWork()); len(entries) != 0 {
		t.Errorf("a refused set-aside left %d saved-work entries", len(entries))
	}
}

// TestSetAsideHoldsMatchThePlan (BR-12): the converge step names the same hold
// PlanSlot gives the same condition, through the one SetAsideHold, and a
// checkout that recovered meanwhile is retryable, never a hand-off.
func TestSetAsideHoldsMatchThePlan(t *testing.T) {
	for _, agent := range AllEvidenceAgents() {
		s := newObservedSlot(t)
		dep := brokenDep(t, s)
		cv := s.converger(t)
		cv.agentNow = func(context.Context) EvidenceAgent { return agent }
		err := cv.converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep})
		reason, held := SetAsideHold(agent, false)
		if !held {
			if err != nil {
				t.Errorf("agent %s: %v", agent, err)
			}
			continue
		}
		if f := ClassifyConvergeError(PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep)}, err); f.Class != FailureHold || f.Cause != reason {
			t.Errorf("agent %s: classified %+v, the plan gives hold %s", agent, f, reason)
		}
		assertNoSavedWork(t, s.layout)
	}
	s := newObservedSlot(t)
	dep := s.addDep(t, fakeDep) // healthy: no longer broken when the step runs
	err := s.converger(t).converge(context.Background(), PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep), Path: dep})
	if f := ClassifyConvergeError(PlannedStep{Step: StepSetAside, Resource: DepResource(fakeDep)}, err); !errors.Is(err, errCheckoutRecovered) || f.Class != FailureRetryable {
		t.Fatalf("recovered checkout: %v classified %+v, want retryable", err, f)
	}
}

// TestReconcileResultListsWhatItSetAside: a broken dependency is set aside and
// re-cloned, and the result names the entry and how to restore it.
func TestReconcileResultListsWhatItSetAside(t *testing.T) {
	s := newObservedSlot(t)
	dep := brokenDep(t, s)
	s.f.DepSources = map[string]bool{fakeDep: true}
	r, err := NewWorkspaceProvisioner(s.f).Ensure(context.Background(), ProvisionRequest{Path: s.f.Primary, Slot: 1, Agent: AgentNone})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.SetAside) != 1 || r.SetAside[0].Path != dep || r.SetAside[0].State != "complete" {
		t.Fatalf("set aside %+v", r.SetAside)
	}
	if !strings.Contains(r.Warning, "restore with: mv") {
		t.Fatalf("warning %q does not name the restore", r.Warning)
	}
	if _, err := os.Stat(filepath.Join(dep, "construct", "base.manifest")); err != nil {
		t.Fatal("the dependency was not re-cloned")
	}
}

// TestSavedWorkIsCollectedPastRetention: the reconciler that writes saved work
// collects it after storagegc.RetentionPeriod, by saved_at or, for an entry
// without a readable manifest, by the directory's age; a symlink in saved work
// is never followed or removed.
func TestSavedWorkIsCollectedPastRetention(t *testing.T) {
	s := newObservedSlot(t)
	l := s.layout
	now := time.Now()
	writeEntry := func(name string, savedAt time.Time, manifest bool) string {
		entry := filepath.Join(l.SavedWork(), name)
		os.MkdirAll(filepath.Join(entry, "tree"), 0o700)
		if manifest {
			if err := (ProvisionStore{}).Write(filepath.Join(entry, "manifest.json"), SavedWorkManifest{SchemaVersion: 1, State: "complete", SavedAt: savedAt}); err != nil {
				t.Fatal(err)
			}
		}
		os.Chtimes(entry, savedAt, savedAt)
		return entry
	}
	old := writeEntry("old", now.Add(-storagegc.RetentionPeriod-time.Hour), true)
	young := writeEntry("young", now.Add(-time.Hour), true)
	orphan := writeEntry("orphan", now.Add(-storagegc.RetentionPeriod-time.Hour), false)
	target := t.TempDir()
	link := filepath.Join(l.SavedWork(), "link")
	os.Symlink(target, link)
	removed, err := collectSavedWork(l, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{old, orphan} {
		if _, err := os.Lstat(gone); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s past retention was kept", gone)
		}
	}
	for _, kept := range []string{young, link, target} {
		if _, err := os.Lstat(kept); err != nil {
			t.Errorf("%s was removed: %v", kept, err)
		}
	}
	if len(removed) != 2 {
		t.Fatalf("removed %v", removed)
	}
	// A reconcile run on the slot collects as it starts.
	again := writeEntry("again", now.Add(-storagegc.RetentionPeriod-time.Hour), true)
	if _, err := NewWorkspaceProvisioner(s.f).Ensure(context.Background(), ProvisionRequest{Path: s.f.Primary, Slot: 1, Agent: AgentNone}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(again); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a reconcile run did not collect saved work past retention")
	}
}
