package codexsid

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveNoAgentPid(t *testing.T) {
	if got := ResolveSessionID(t.TempDir(), "tag"); got != "" {
		t.Fatalf("no pidfile -> empty, got %q", got)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agent-pid-tag"), []byte("\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := ResolveSessionID(dir, "tag"); got != "" {
		t.Fatalf("empty pidfile -> empty, got %q", got)
	}
}

func TestResolveSessionIDSkipsSubagentRollout(t *testing.T) {
	dataDir := t.TempDir()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "agent-pid-tag"), []byte("10\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
	dir := filepath.Join(home, ".codex", "sessions", "2026", "05", "31")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rootPath := filepath.Join(dir, "rollout-root-"+rootSID+".jsonl")
	subPath := filepath.Join(dir, "rollout-sub-"+subSID+".jsonl")
	if err := os.WriteFile(rootPath, []byte(`{"type":"session_meta","payload":{"id":"`+rootSID+`","parent_thread_id":null,"source":"exec"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subPath, []byte(`{"type":"session_meta","payload":{"id":"`+subSID+`","parent_thread_id":"`+rootSID+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	ps := "#!/bin/sh\nprintf ' 10 1\\n 11 10\\n'\n"
	if err := os.WriteFile(filepath.Join(binDir, "ps"), []byte(ps), 0o755); err != nil {
		t.Fatal(err)
	}
	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + subPath + "\\nn" + rootPath + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
	if err := os.WriteFile(filepath.Join(binDir, "lsof"), []byte(lsof), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if got := ResolveSessionID(dataDir, "tag"); got != rootSID {
		t.Fatalf("ResolveSessionID = %q, want root %q", got, rootSID)
	}
}
