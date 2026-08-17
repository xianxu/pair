package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveClaudeEncodesCwd(t *testing.T) {
	got := Resolve("claude", "abc", "/Users/x/work.dir", "/home")
	want := filepath.Join("/home", ".claude", "projects", "-Users-x-work-dir", "abc.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveAgy(t *testing.T) {
	got := Resolve("agy", "sid1", "", "/home")
	want := filepath.Join("/home", ".gemini", "antigravity-cli", "brain", "sid1", ".system_generated", "logs", "transcript.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCodexSessionIDFromPath(t *testing.T) {
	sid := "01a00e37-16c4-7100-89fc-42ce26158f71"
	path := filepath.Join("/home/u", ".codex", "sessions", "2026", "08", "16", "rollout-2026-08-16T22-34-46-"+sid+".jsonl")
	if got := CodexSessionIDFromPath(path); got != sid {
		t.Fatalf("CodexSessionIDFromPath = %q, want %q", got, sid)
	}
	if got := CodexSessionIDFromPath("/tmp/not-codex.jsonl"); got != "" {
		t.Fatalf("non-codex path = %q, want empty", got)
	}
}

func TestResolveMuseIgnoresSubagent(t *testing.T) {
	home := t.TempDir()
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	subSid := "123e4567-e89b-12d3-a456-426614174000"
	// Root session is resumable
	rootPath := filepath.Join(home, ".local", "share", "muse", "sessions", "2026", "08", "14", sid, "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(rootPath), 0o755); err != nil {
		t.Fatalf("create root dir: %v", err)
	}
	if err := os.WriteFile(rootPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("create root: %v", err)
	}
	got := Resolve("muse", sid, "", home)
	if got != rootPath {
		t.Fatalf("Resolve muse root = %q, want %q", got, rootPath)
	}
	// Subagent id should not resolve to a root session (Glob depth mismatch)
	if got2 := Resolve("muse", subSid, "", home); got2 != "" {
		t.Fatalf("Resolve muse subagent id = %q, want empty", got2)
	}
	// If both root and subagent files exist, root id still resolves to root path
	subRoot := filepath.Join(home, ".local", "share", "muse", "sessions", "2026", "08", "14", sid, "subagent", subSid, "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(subRoot), 0o755); err != nil {
		t.Fatalf("create subagent dir: %v", err)
	}
	if err := os.WriteFile(subRoot, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("create subagent: %v", err)
	}
	if got3 := Resolve("muse", sid, "", home); got3 != rootPath {
		t.Fatalf("Resolve after subagent creation = %q, want root %q", got3, rootPath)
	}
}
