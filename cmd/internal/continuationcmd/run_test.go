package continuationcmd

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadBodyRejectsOversizedInput(t *testing.T) {
	input := strings.NewReader(strings.Repeat("x", checkpoint.MaxBytes*2))
	if _, err := readBody("-", input); err == nil {
		t.Fatal("accepted oversized input")
	}
	if consumed := checkpoint.MaxBytes*2 - input.Len(); consumed > checkpoint.MaxBytes+1 {
		t.Fatalf("read unbounded source: %d", consumed)
	}
}

// run() does real, non-injected git IO (it builds gitRunner{root} internally),
// so these are integration tests over a real temp repo — not pure units. The
// harness inits a git repo, sets repoRoot (skips rev-parse), and tolerates the
// non-fatal `git push origin HEAD` failure (no origin remote). #105.

func initTempRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "test"},
		{"commit", "--allow-empty", "-m", "root"},
	} {
		c := exec.Command("git", args...)
		c.Dir = repo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return repo
}

func readWrittenContinuation(t *testing.T, repo string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(repo, ContinuationDir, "*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected exactly one continuation, got %v (err %v)", matches, err)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC) }
}

func baseArgs(repo string) runArgs {
	return runArgs{repoRoot: repo, slug: "resume-parser", agent: "claude", issuesCSV: "105", bodyFile: "-"}
}

func TestRun_FoldsDraftWhenInCompaction(t *testing.T) {
	repo := initTempRepo(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "draft-mytag.md"),
		[]byte("=== sticky ===\nfinish the parser\n=== end ===\n"), 0o644)
	env := runEnv{pairTag: "mytag", dataDir: dataDir, zellijSession: "pair-mytag"}

	var b strings.Builder
	err := run(baseArgs(repo), env, fixedClock(),
		strings.NewReader("## NEXT ACTION\n\nreview PR\n"), &b,
		func(string, string) error { return nil })
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	out := readWrittenContinuation(t, repo)
	if !strings.Contains(out, "finish the parser") {
		t.Errorf("draft WIP not folded:\n%s", out)
	}
	if strings.Contains(out, "=== sticky ===") {
		t.Errorf("comments should have been stripped:\n%s", out)
	}
	if !strings.Contains(out, "review PR") {
		t.Errorf("original NEXT ACTION lost:\n%s", out)
	}
}

func TestRun_NoFoldOrRestartStandalone(t *testing.T) {
	repo := initTempRepo(t)
	dataDir := t.TempDir()
	// A draft exists, but we are NOT in a pair pane (empty tag) → no fold.
	os.WriteFile(filepath.Join(dataDir, "draft-mytag.md"), []byte("stray wip\n"), 0o644)
	env := runEnv{pairTag: "", dataDir: dataDir, zellijSession: ""}

	called := 0
	var b strings.Builder
	if err := run(baseArgs(repo), env, fixedClock(),
		strings.NewReader("## NEXT ACTION\n\ndo it\n"), &b,
		func(string, string) error { called++; return nil }); err != nil {
		t.Fatalf("run: %v", err)
	}
	if called != 0 {
		t.Error("standalone write must not restart")
	}
	if strings.Contains(readWrittenContinuation(t, repo), "stray wip") {
		t.Error("standalone write must not fold the draft")
	}
}

func TestRun_TriggersRestartInCompaction(t *testing.T) {
	repo := initTempRepo(t)
	env := runEnv{pairTag: "mytag", dataDir: t.TempDir(), zellijSession: "pair-mytag"}

	var gotPath string
	called := 0
	var b strings.Builder
	if err := run(baseArgs(repo), env, fixedClock(),
		strings.NewReader("## NEXT ACTION\n\ngo\n"), &b,
		func(path, digest string) error {
			gotPath = path
			called++
			if len(digest) != 64 {
				t.Fatalf("missing committed digest %q", digest)
			}
			return nil
		}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if called != 1 || gotPath != strings.TrimSpace(b.String()) || !filepath.IsAbs(gotPath) {
		t.Fatalf("restart not triggered with exact path: called=%d path=%q", called, gotPath)
	}
}

func TestNewContinueRestartCmd_FakesInZellij(t *testing.T) {
	c := newContinueRestartCmd("/opt/pair", "/worktree/checkpoint.md", "expected-digest", nil, nil, nil)
	if len(c.Args) != 4 || c.Args[1] != "continue" || c.Args[2] != "--checkpoint" || c.Args[3] != "/worktree/checkpoint.md" {
		t.Fatalf("args = %v, want [exe continue myslug]", c.Args)
	}
	found := false
	digestFound := false
	for _, e := range c.Env {
		if e == "PAIR_CONTINUATION_DIGEST=expected-digest" {
			digestFound = true
		}
		if e == "PAIR_FAKE_IN_ZELLIJ=1" {
			found = true
		}
	}
	if !found || !digestFound {
		// The agent's command sandbox blocks the proc-ancestry walk InZellijPane
		// uses; without this fake the restart misfires under a sandboxed shell (#105).
		t.Error("restart command must set PAIR_FAKE_IN_ZELLIJ=1")
	}
}

func TestRun_NoRestartFlagSuppressesInCompaction(t *testing.T) {
	repo := initTempRepo(t)
	dataDir := t.TempDir()
	os.WriteFile(filepath.Join(dataDir, "draft-mytag.md"), []byte("wip\n"), 0o644)
	env := runEnv{pairTag: "mytag", dataDir: dataDir, zellijSession: "pair-mytag"}

	a := baseArgs(repo)
	a.noRestart = true
	called := 0
	var b strings.Builder
	if err := run(a, env, fixedClock(),
		strings.NewReader("## NEXT ACTION\n\ngo\n"), &b,
		func(string, string) error { called++; return nil }); err != nil {
		t.Fatalf("run: %v", err)
	}
	if called != 0 {
		t.Error("--no-restart must suppress the restart even in compaction context")
	}
	if strings.Contains(readWrittenContinuation(t, repo), "wip") {
		t.Error("--no-restart must also suppress the fold (deliberate manual write)")
	}
}

func TestRunRestartFailurePreservesCommittedCheckpoint(t *testing.T) {
	repo := initTempRepo(t)
	env := runEnv{pairTag: "mytag", dataDir: t.TempDir(), zellijSession: "pair-mytag"}
	var out strings.Builder
	err := run(baseArgs(repo), env, fixedClock(), strings.NewReader("## NEXT ACTION\nContinue.\n"), &out, func(string, string) error { return errors.New("request refused") })
	if err == nil || !strings.Contains(err.Error(), strings.TrimSpace(out.String())) {
		t.Fatalf("expected actionable durable failure: %v", err)
	}
	path := strings.TrimSpace(out.String())
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", repo, "ls-files", "--error-unmatch", path)
	if raw, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkpoint not committed: %s %v", raw, err)
	}
}
