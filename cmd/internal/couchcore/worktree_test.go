package couchcore

import "testing"

func TestResolveWalksUpToWorktreeRoot(t *testing.T) {
	// The kbench/competition/arc-agi-3 case: a subdirectory resolves to the
	// tree that contains it, because the collision hazards are tree-scoped.
	sub := "/Users/x/workspace/kbench/competition/arc-agi-3"
	g := NewFakeGit(map[GitCall]string{
		{Dir: sub, Args: "rev-parse --show-toplevel"}: "/Users/x/workspace/kbench\n",
	})
	wt, err := Resolve(sub, g, NewFakePathOps(nil))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if wt != Worktree("/Users/x/workspace/kbench") {
		t.Fatalf("Resolve = %q", wt)
	}
}

func TestResolveAppliesPhysicalToTheInputBeforeCallingGit(t *testing.T) {
	// Canned only for the PHYSICAL dir: if Resolve skips Physical on the way
	// in, git is invoked in /link/repo, finds no canned reply, and errors.
	g := NewFakeGit(map[GitCall]string{
		{Dir: "/real/repo", Args: "rev-parse --show-toplevel"}: "/real/repo",
	})
	p := NewFakePathOps(map[string]string{"/link/repo": "/real/repo"})
	if _, err := Resolve("/link/repo", g, p); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if g.Ops[0] != "/real/repo: rev-parse --show-toplevel" {
		t.Fatalf("git ran in %q; the input must be resolved first", g.Ops[0])
	}
}

func TestResolveAppliesPhysicalToGitsAnswer(t *testing.T) {
	// git can itself report a symlinked toplevel; the stored identity must be
	// the physical one or two spellings become two trees.
	g := NewFakeGit(map[GitCall]string{
		{Dir: "/real/repo/sub", Args: "rev-parse --show-toplevel"}: "/link/root",
	})
	p := NewFakePathOps(map[string]string{
		"/real/repo/sub": "/real/repo/sub",
		"/link/root":     "/real/root",
	})
	wt, err := Resolve("/real/repo/sub", g, p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if wt != Worktree("/real/root") {
		t.Fatalf("Resolve = %q, want the physical root", wt)
	}
}

func TestResolveErrorsOutsideARepo(t *testing.T) {
	if _, err := Resolve("/tmp", NewFakeGit(nil), NewFakePathOps(nil)); err == nil {
		t.Fatal("expected an error outside a git worktree")
	}
}

func TestResolveErrorsWhenPathDoesNotResolve(t *testing.T) {
	p := NewFakePathOps(nil)
	p.Fail("/gone")
	if _, err := Resolve("/gone", NewFakeGit(nil), p); err == nil {
		t.Fatal("an unresolvable path must error, not become its own identity")
	}
}

func TestWorktreeKeyFoldsAndRepoDoesNot(t *testing.T) {
	w := Worktree("/Users/x/KBench")
	if w.Repo() != "KBench" {
		t.Errorf("Repo = %q; display must keep case", w.Repo())
	}
	if w.Key() != foldWith("/Users/x/KBench", caseInsensitiveFS()) {
		t.Errorf("Key = %q", w.Key())
	}
}
