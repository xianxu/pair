package couchcore

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The catalog pin also advances layergraph: a linked slot may read an absent
// local substrate from its primary checkout, but may never adopt it as slot-owned.
func TestDeclaredDepsLinkedWorktreePrimaryAnchor(t *testing.T) {
	w := newDepsWorld(t)
	primary := w.checkout("primary/pair", "substrate ../base\n", true)
	base := w.checkout("primary/base", "", true)
	runGit := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	for _, dir := range []string{primary, base} {
		runGit(dir, "init", "-b", "main")
		runGit(dir, "add", "construct")
		runGit(dir, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-m", "fixture")
	}
	slot := filepath.Join(w.env, "slot")
	if err := os.Mkdir(slot, 0700); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(slot, "pair")
	runGit(primary, "worktree", "add", "-b", "main-slot1", linked)
	got, err := DeclaredDepsOf(slot, linked)
	if err != nil || len(got.Deps) != 1 {
		t.Fatalf("dependencies=%+v err=%v", got, err)
	}
	d := got.Deps[0]
	if d.Path != base || !d.Present || !d.Outside || d.Rel != "" || d.Owner != linked {
		t.Fatalf("primary substrate lost identity/outside ownership: %+v", d)
	}
	commits := substrateCommits(linked, func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		out, err := cmd.Output()
		return strings.TrimSpace(string(out)), err
	})
	want := runGit(base, "rev-parse", "HEAD")
	if len(commits) != 1 || commits[0].Path != base || commits[0].Commit != want {
		t.Fatalf("prepare substrate report=%+v; want %s at %s", commits, base, want)
	}
}
