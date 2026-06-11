package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair-continuation")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// gitInitWithBareOrigin makes a repo at root with a local bare origin and an
// initial commit on main, with upstream tracking set — so the writer's
// add/commit/push run for real (process-level realism, not a mock).
func gitInitWithBareOrigin(t *testing.T, root string) {
	t.Helper()
	bare := t.TempDir()
	git(t, bare, "init", "--bare", "-b", "main")
	git(t, root, "init", "-b", "main")
	git(t, root, "config", "user.email", "t@example.com")
	git(t, root, "config", "user.name", "tester")
	git(t, root, "remote", "add", "origin", bare)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "README.md")
	git(t, root, "commit", "-m", "seed")
	git(t, root, "push", "-u", "origin", "main")
}

func TestWriter_WritesCommitsPushes(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	gitInitWithBareOrigin(t, root)

	bodyFile := filepath.Join(t.TempDir(), "body.md")
	body := "# Continuation: robotics\n\n## NEXT ACTION\nRun make test.\n"
	if err := os.WriteFile(bodyFile, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(bin,
		"-repo-root", root, "-slug", "robotics", "-agent", "claude",
		"-session-id", "7f3a", "-issues", "000071, 000073", "-branch", "main",
		"-body-file", bodyFile,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("writer failed: %v\n%s", err, out)
	}
	path := strings.TrimSpace(string(out))

	// 1. file exists with conformant frontmatter + body
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	cs := string(content)
	for _, want := range []string{
		"---\ntype: continuation", "slug: robotics", "agent: claude",
		"session_id: 7f3a", "issues: [000071, 000073]",
		"## NEXT ACTION", "Run make test.",
	} {
		if !strings.Contains(cs, want) {
			t.Fatalf("written file missing %q:\n%s", want, cs)
		}
	}
	// name shape: workshop/continuation/<ts>-robotics.md
	if !strings.Contains(filepath.ToSlash(path), "workshop/continuation/") ||
		!strings.HasSuffix(path, "-robotics.md") {
		t.Fatalf("unexpected path: %s", path)
	}

	// 2. committed locally
	if log := git(t, root, "log", "--oneline", "-1"); !strings.Contains(log, "continuation: robotics") {
		t.Fatalf("commit not found, got: %s", log)
	}

	// 3. pushed to the bare origin (file present in origin/main's tree)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatal(err)
	}
	originLs := git(t, root, "ls-tree", "-r", "--name-only", "origin/main")
	if !strings.Contains(originLs, filepath.ToSlash(rel)) {
		t.Fatalf("file not pushed to origin/main:\n%s", originLs)
	}
}

func TestWriter_MissingRequiredFlags(t *testing.T) {
	bin := buildBinary(t)
	root := t.TempDir()
	gitInitWithBareOrigin(t, root)
	bodyFile := filepath.Join(t.TempDir(), "b.md")
	if err := os.WriteFile(bodyFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// no -slug → validation error, non-zero exit, nothing committed
	out, err := exec.Command(bin, "-repo-root", root, "-agent", "claude",
		"-issues", "000071", "-body-file", bodyFile).CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure on missing slug; output: %s", out)
	}
}
