package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildPairContext(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair-context")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func TestPairContext_Claude(t *testing.T) {
	bin := buildPairContext(t)
	home := t.TempDir()
	data := filepath.Join(home, "data")
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

	cmd := exec.Command(bin, "T", "claude")
	cmd.Env = append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.TrimSpace(string(out)) != "398k" {
		t.Fatalf("got %q want 398k", out)
	}
}

func TestPairContext_NoConfig_PrintsNothing(t *testing.T) {
	bin := buildPairContext(t)
	home := t.TempDir()
	cmd := exec.Command(bin, "T", "claude")
	cmd.Env = append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+filepath.Join(home, "empty"))
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("want empty, got %q", out)
	}
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
