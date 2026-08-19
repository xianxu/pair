package reviewcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"/repo/doc.md":        "doc",
		"Foo Bar.md":          "foo-bar",
		"x.test.md":           "x-test",
		"/a/--Weird__Name.md": "weird-name",
		"plain":               "plain",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTargetJSON(t *testing.T) {
	// Round-trips with spaces/quotes in the path (Go json handles it; the shell
	// needed jq -n --arg).
	s := targetJSON(`/r/doc "q".md`, "ready", "sid1")
	var d targetDoc
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		t.Fatalf("not valid JSON: %v (%s)", err, s)
	}
	if d.File != `/r/doc "q".md` || d.Status != "ready" || d.Session != "sid1" {
		t.Fatalf("round-trip = %+v", d)
	}
}

func TestOSRuntimeConfiguredSessionIDRejectsCodexSubagent(t *testing.T) {
	home := t.TempDir()
	data := filepath.Join(home, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	sid := "01a017b6-af00-7c91-a656-0611a3750669"
	parent := "019e8178-79c2-7862-91db-e8fa1be3b162"
	rollout := filepath.Join(home, ".codex", "sessions", "2026", "08", "18", "rollout-sub-"+sid+".jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, []byte(`{"type":"session_meta","payload":{"id":"`+sid+`","parent_thread_id":"`+parent+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "config-t-codex.json"), []byte(`{"session_id":"`+sid+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if got := NewOSRuntime().ConfiguredSessionID(data, "t", "codex"); got != "" {
		t.Fatalf("ConfiguredSessionID = %q, want empty", got)
	}
}
