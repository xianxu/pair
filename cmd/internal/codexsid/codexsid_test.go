package codexsid

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRolloutRE(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/Users/x/.codex/sessions/2024/01/rollout-2024-01-01T00-00-00-aaaa1111-2222-3333-4444-555566667777.jsonl", "aaaa1111-2222-3333-4444-555566667777"},
		{"/other/path.jsonl", ""},
		{"/Users/x/.codex/sessions/rollout-nouuid.jsonl", ""},
	}
	for _, c := range cases {
		got := ""
		if m := rolloutRE.FindStringSubmatch(c.path); m != nil {
			got = m[1]
		}
		if got != c.want {
			t.Errorf("%q -> %q, want %q", c.path, got, c.want)
		}
	}
}

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
