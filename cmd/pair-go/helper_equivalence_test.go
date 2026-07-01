package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPairGoContextMatchesLegacyPairContext(t *testing.T) {
	bin := t.TempDir()
	pairContext := filepath.Join(bin, "pair-context")
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairContext, "../pair-context")
	buildCommand(t, pairGo, ".")

	home, data := writeContextFixture(t)
	env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)

	legacy := runCommand(t, env, pairContext, "T", "claude")
	dispatch := runCommand(t, env, pairGo, "context", "T", "claude")
	if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
		t.Fatalf("pair-go context mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
			legacy.code, legacy.stdout, legacy.stderr,
			dispatch.code, dispatch.stdout, dispatch.stderr)
	}
}

func TestPairGoSlugMatchesLegacyPairSlug(t *testing.T) {
	bin := t.TempDir()
	pairSlug := filepath.Join(bin, "pair-slug")
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairSlug, "../pair-slug")
	buildCommand(t, pairGo, ".")

	// Empty data dir → no config → both resolve no session_id and no-op (exit 0,
	// no output, no slug-proposed). Both entry points call the identical
	// slugcmd.Run(), so the `pair slug` route and the pair-slug shim must agree.
	data := t.TempDir()
	env := append(os.Environ(),
		"PAIR_TAG=T", "PAIR_DATA_DIR="+data, "PAIR_AGENT=claude", "PAIR_SLUG_NESTED=")

	legacy := runCommand(t, env, pairSlug)
	dispatch := runCommand(t, env, pairGo, "slug")
	if dispatch.code != legacy.code || dispatch.stdout != legacy.stdout || dispatch.stderr != legacy.stderr {
		t.Fatalf("pair-go slug mismatch\nlegacy:   code=%d stdout=%q stderr=%q\ndispatch: code=%d stdout=%q stderr=%q",
			legacy.code, legacy.stdout, legacy.stderr,
			dispatch.code, dispatch.stdout, dispatch.stderr)
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
