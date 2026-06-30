package contextcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunClaude(t *testing.T) {
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

	var stdout bytes.Buffer
	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: data}, &stdout)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "398k" {
		t.Fatalf("stdout = %q, want 398k", stdout.String())
	}
}

func TestRunMissingConfigPrintsNothing(t *testing.T) {
	home := t.TempDir()
	var stdout bytes.Buffer
	code := Run([]string{"T", "claude"}, Env{Home: home, PairDataDir: filepath.Join(home, "empty")}, &stdout)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
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
