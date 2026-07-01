package slugcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDescendantPIDsIncludesNestedChildren(t *testing.T) {
	children := map[string][]string{
		"10": {"11", "12"},
		"11": {"13"},
		"13": {"14"},
	}
	got := descendantPIDs("10", children)
	want := []string{"10", "11", "12", "13", "14"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("descendantPIDs = %v, want %v", got, want)
	}
}

func TestCodexRolloutPattern(t *testing.T) {
	path := "/Users/x/.codex/sessions/2026/05/31/rollout-2026-05-31T21-36-56-019e8178-79c2-7862-91db-e8fa1be3b162.jsonl"
	if !codexRolloutRE.MatchString(path) {
		t.Fatalf("codexRolloutRE did not match %q", path)
	}
}

func TestResolveLiveCodexTranscriptUsesDescendantLsof(t *testing.T) {
	dataDir := t.TempDir()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "agent-pid-testtag"), []byte("10\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(home, ".codex", "sessions", "2026", "05", "31",
		"rollout-2026-05-31T21-36-56-019e8178-79c2-7862-91db-e8fa1be3b162.jsonl")
	binDir := t.TempDir()
	ps := "#!/bin/sh\nprintf ' 10 1\\n 11 10\\n'\n"
	if err := os.WriteFile(filepath.Join(binDir, "ps"), []byte(ps), 0o755); err != nil {
		t.Fatal(err)
	}
	lsof := "#!/bin/sh\nif [ \"$2\" = \"11\" ]; then printf 'p11\\nn" + path + "\\n'; else printf 'p%s\\n' \"$2\"; fi\n"
	if err := os.WriteFile(filepath.Join(binDir, "lsof"), []byte(lsof), 0o755); err != nil {
		t.Fatal(err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
	got := resolveLiveCodexTranscript(dataDir, "testtag", home)
	if got != path {
		t.Fatalf("resolveLiveCodexTranscript = %q, want %q", got, path)
	}
}
