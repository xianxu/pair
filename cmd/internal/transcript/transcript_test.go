package transcript

import (
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
