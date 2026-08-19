package slugcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/transcript"
)

func TestCodexRolloutPattern(t *testing.T) {
	path := "/Users/x/.codex/sessions/2026/05/31/rollout-2026-05-31T21-36-56-019e8178-79c2-7862-91db-e8fa1be3b162.jsonl"
	if got := transcript.CodexSessionIDFromPath(path); got == "" {
		t.Fatalf("CodexSessionIDFromPath did not match %q", path)
	}
}

func TestResolveLiveCodexTranscriptUsesDescendantLsof(t *testing.T) {
	dataDir := t.TempDir()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "agent-pid-testtag"), []byte("10\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
	rootPath := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
		"rollout-2026-05-31T21-36-56-"+rootSID+".jsonl")
	subPath := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
		"rollout-2026-05-31T22-00-00-"+subSID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(rootPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootPath, []byte(`{"type":"session_meta","payload":{"id":"`+rootSID+`","parent_thread_id":null,"source":"cli"}}`+"\n"), 0o644); err != nil {
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

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
	got := resolveLiveCodexTranscript(dataDir, "testtag", home)
	if got != rootPath {
		t.Fatalf("resolveLiveCodexTranscript = %q, want root %q", got, rootPath)
	}
}
