package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Binary-level route smokes (#104 M3: the standalone helpers are gone, so these
// assert the built `pair` binary's subcommand routes reach their runners rather
// than comparing against a deleted standalone — the in-process routing is
// covered by dispatcher_test.go).

func TestPairGoContextRouteReachesRunner(t *testing.T) {
	bin := t.TempDir()
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairGo, ".")

	home, data := writeContextFixture(t)
	env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)

	r := runCommand(t, env, pairGo, "context", "T", "claude")
	if r.code != 0 {
		t.Fatalf("pair context route exit = %d, want 0\nstderr=%q", r.code, r.stderr)
	}
	if strings.TrimSpace(r.stdout) != "398k" {
		t.Fatalf("pair context route stdout = %q, want 398k", r.stdout)
	}
}

func TestPairGoSlugRouteReachesRunner(t *testing.T) {
	bin := t.TempDir()
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairGo, ".")

	// Empty data dir → no config → slugcmd.Run resolves no session_id and no-ops
	// (exit 0, no output, no slug-proposed file).
	data := t.TempDir()
	env := append(os.Environ(),
		"PAIR_TAG=T", "PAIR_DATA_DIR="+data, "PAIR_AGENT=claude", "PAIR_SLUG_NESTED=")

	r := runCommand(t, env, pairGo, "slug")
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("pair slug route: code=%d stdout=%q stderr=%q, want 0/empty/empty", r.code, r.stdout, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(data, "slug-proposed-T")); !os.IsNotExist(err) {
		t.Fatalf("no-session slug must not write a proposal")
	}
}

func buildCommand(t *testing.T, out, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", out, pkg)
	if body, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, string(body))
	}
}

type commandResult struct {
	code   int
	stdout string
	stderr string
}

func runCommand(t *testing.T, env []string, name string, args ...string) commandResult {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %s: %v", name, err)
		}
		code = exit.ExitCode()
	}
	return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func writeContextFixture(t *testing.T) (home, data string) {
	t.Helper()
	home = t.TempDir()
	data = filepath.Join(home, "data")
	cwd := filepath.Join(home, "repo")
	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
	proj := filepath.Join(home, ".claude", "projects", enc)
	mustMkdir(t, data)
	mustMkdir(t, cwd)
	mustMkdir(t, proj)
	mustWrite(t, filepath.Join(data, "config-T-claude.json"), `{"session_id":"sid1"}`)
	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
	mustWrite(t, filepath.Join(proj, "sid1.jsonl"),
		`{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`)
	return home, data
}

func mustMkdir(t *testing.T, d string) {
	t.Helper()
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
