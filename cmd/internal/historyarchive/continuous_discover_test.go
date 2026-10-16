package historyarchive

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func candidatePaths(cs []Candidate) []string {
	out := []string{}
	for _, c := range cs {
		out = append(out, c.Path)
	}
	sort.Strings(out)
	return out
}

func TestDiscoverWalksMatchedDirectoriesAndSkipsLinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sessions/2026/10/01/rollout-a.jsonl"), "a")
	writeFile(t, filepath.Join(root, "history.jsonl"), "h")
	writeFile(t, filepath.Join(root, "other.txt"), "x")
	if err := os.Symlink(filepath.Join(root, "other.txt"), filepath.Join(root, "sessions/link.jsonl")); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(context.Background(), root, []string{"sessions/*", "history.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"history.jsonl", "sessions/2026/10/01/rollout-a.jsonl"}
	if paths := candidatePaths(got); len(paths) != 2 || paths[0] != want[0] || paths[1] != want[1] {
		t.Fatalf("discovered %v, want %v", paths, want)
	}
	for _, c := range got {
		if !c.Regular || c.Size == 0 || c.Mtime.IsZero() {
			t.Fatalf("candidate lacks stat facts: %+v", c)
		}
	}
}

func TestContinuousRootsOpsTeamsAndSharedGitCommonDirs(t *testing.T) {
	home := t.TempDir()
	for _, agent := range []string{".claude", ".codex", ".qoder"} {
		if err := os.Mkdir(filepath.Join(home, agent), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	ops := filepath.Join(t.TempDir(), "ops")
	writeFile(t, filepath.Join(ops, "teams/devinfra/qa/x.md"), "q")
	writeFile(t, filepath.Join(ops, "teams/devinfra/tl-status-1.side"), "s")
	writeFile(t, filepath.Join(ops, "teams/devinfra/notes.md"), "not archived")
	workspace := t.TempDir()
	for _, repo := range []string{"alpha", "beta"} {
		dir := filepath.Join(workspace, repo)
		if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v %s", err, out)
		}
		writeFile(t, filepath.Join(dir, ".git/sdlc/local-checks/x.log"), repo)
	}
	inv, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	cfg := ContinuousConfig{OpsRoot: ops, Workspace: workspace}
	roots, err := ContinuousRoots(context.Background(), inv, cfg, "host", home, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	byKind := map[string][]SourceRoot{}
	for _, r := range roots {
		byKind[r.Kind] = append(byKind[r.Kind], r)
	}
	if got := byKind["ops-documents"]; len(got) != 1 || got[0].Identity.Root != filepath.Join(ops, "teams/devinfra") {
		t.Fatalf("ops roots %+v", got)
	}
	opsFiles, err := Discover(context.Background(), byKind["ops-documents"][0].Identity.Root, byKind["ops-documents"][0].Patterns)
	if err != nil {
		t.Fatal(err)
	}
	if paths := candidatePaths(opsFiles); len(paths) != 2 || paths[0] != "qa/x.md" || paths[1] != "tl-status-1.side" {
		t.Fatalf("ops files %v", paths)
	}
	if got := byKind["sdlc-runtime"]; len(got) != 2 {
		t.Fatalf("sdlc roots %+v, want one per repository", got)
	}
	// A worktree of alpha is a sibling with the same common dir: still one
	// root per repository, keyed without a worktree.
	alpha := filepath.Join(workspace, "alpha")
	if out, err := exec.Command("git", "-C", alpha, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init").CombinedOutput(); err != nil {
		t.Fatalf("commit: %v %s", err, out)
	}
	if out, err := exec.Command("git", "-C", alpha, "worktree", "add", "-q", filepath.Join(workspace, "alpha-slot1")).CombinedOutput(); err != nil {
		t.Fatalf("worktree add: %v %s", err, out)
	}
	roots, err = ContinuousRoots(context.Background(), inv, cfg, "host", home, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	var sdlc []SourceRoot
	for _, r := range roots {
		if r.Kind == "sdlc-runtime" {
			sdlc = append(sdlc, r)
		}
	}
	if len(sdlc) != 2 {
		t.Fatalf("sdlc roots with a worktree sibling %+v, want one per repository", sdlc)
	}
	for _, r := range sdlc {
		if r.Identity.Worktree != "" {
			t.Fatalf("sdlc identity %+v carries a worktree; it must be stable across siblings", r.Identity)
		}
	}
	for _, kind := range []string{"claude-transcripts", "codex-transcripts", "qoder-transcripts", "codex-database"} {
		if len(byKind[kind]) != 1 {
			t.Fatalf("%s roots %+v", kind, byKind[kind])
		}
	}
}
