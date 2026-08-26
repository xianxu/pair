package couchcore

import (
	"crypto/rand"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// tempRepo creates a real one-commit git repo and returns its canonical
// worktree. Hermetic: no test may resolve against the ambient checkout, or it
// asserts on the developer's directory layout rather than on couch.
func tempRepo(t *testing.T) Worktree {
	t.Helper()
	dir := t.TempDir()
	run := func(d string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = d
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run(dir, "init", "-q", "-b", "main", ".")
	run(dir, "config", "user.email", "t@example.com")
	run(dir, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(dir, "add", ".")
	run(dir, "commit", "-qm", "init")

	tree, err := Resolve(dir, ExecGit{}, OSPathOps{})
	if err != nil {
		t.Fatalf("resolve temp repo: %v", err)
	}
	return tree
}

// TestGuardRefusesAgainstARealLiveProcess pins BR-1 against the real probes.
//
// The unit tests use FakeProcOps, so they prove the logic but not that
// OSProcOps can actually answer. That gap is exactly where the fail-open bug
// lived: a probe that forked `kill -0` returned "dead" whenever forking was
// restricted, and PruneDead then deleted a LIVE actor's record and admitted a
// second agent onto its tree.
//
// This registers a real long-lived child, then attempts a second spawn on the
// same tree. The refusal must happen BEFORE anything is forked, so `pair` is
// never launched and the test needs no zellij.
func TestGuardRefusesAgainstARealLiveProcess(t *testing.T) {
	// Deliberately NOT gated. A pin that only runs under PAIR_LIVE_COUCH is not
	// a pin: nothing in the default suite sets it, so the behaviour it protects
	// can regress silently. It is hermetic instead -- its own temp repo, never
	// the ambient checkout -- so it can run everywhere.
	tree := tempRepo(t)

	runner := ExecRunner{}
	child, err := runner.Start(t.TempDir(), []string{"sh", "-c", "sleep 30"}, nil)
	if err != nil {
		t.Fatalf("start a real child: %v", err)
	}
	defer func() { _ = child.Signal(os.Kill); _ = child.Wait() }()

	proc := OSProcOps{}
	rec := ActorRecord{
		ID:        "couch-livecheck",
		Args:      StartArgs{Worktree: tree},
		PID:       child.PID(),
		Identity:  child.Identity(),
		StartedAt: SystemClock{}.Now(),
	}
	if rec.Identity == "" {
		t.Fatal("ExecRunner recorded no identity token for a real child")
	}
	if got := proc.Exists(rec.PID); got != Live {
		t.Fatalf("real running child probes as %v -- the guard cannot work", got)
	}

	ns, err := ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatalf("ResolveCouchNamespace: %v", err)
	}
	store := NewStore(ns.Dir())
	if err := store.Save(NewRegistry().Insert(rec), NewNamingTable()); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	resolver := NewFakePolicyResolver()
	resolver.SetDefault(PolicyResult{
		PolicyVersion: 1, PolicyDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RepoIdentity: "live-repo", AdmissionKey: string(tree),
		Capacity: PolicyCapacity{Kind: CapacityBounded, Limit: 1}, OnCapacity: CapacityReject,
	}, nil)
	c, err := New(ns, runner, OSPathOps{}, ExecGit{}, proc, store, SystemClock{}, NewRandomIDGen(), resolver, rand.Reader)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := c.Liveness(rec); got != Live {
		t.Fatalf("Liveness = %v against a real running process, want Live", got)
	}

	_, _, err = c.Spawn(StartArgs{Worktree: tree, Cwd: string(tree)})
	var occ *CapacityExceededError
	if !errors.As(err, &occ) {
		t.Fatalf("second spawn err = %v, want *CapacityExceededError -- a live actor "+
			"must refuse its tree when probed for real", err)
	}
	if len(occ.Incumbents) != 1 {
		t.Fatalf("incumbents = %+v", occ.Incumbents)
	}
}
