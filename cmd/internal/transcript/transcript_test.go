package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCodexSessionMetaLineLimit = 1 << 20

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

func TestCodexRootSessionID(t *testing.T) {
	sid := "01a00e37-16c4-7100-89fc-42ce26158f71"
	path := filepath.Join("/home/u", ".codex", "sessions", "2026", "08", "16", "rollout-2026-08-16T22-34-46-"+sid+".jsonl")
	tests := []struct {
		name  string
		path  string
		event string
		want  string
	}{
		{name: "cli root", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}`, want: sid},
		{name: "exec root with absent parent", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","source":"exec"}}`, want: sid},
		{name: "subagent", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":"parent","source":{"subagent":{"thread_spawn":{"depth":1}}}}}`},
		{name: "non-null parent", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":"parent","source":"cli"}}`},
		{name: "unknown string source", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"future"}}`},
		{name: "unknown object source", path: path, event: `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":{"other":{}}}}`},
		{name: "mismatched id", path: path, event: `{"type":"session_meta","payload":{"id":"11a00e37-16c4-7100-89fc-42ce26158f71","parent_thread_id":null,"source":"cli"}}`},
		{name: "wrong event type", path: path, event: `{"type":"event_msg","payload":{"id":"` + sid + `","source":"cli"}}`},
		{name: "missing id", path: path, event: `{"type":"session_meta","payload":{"parent_thread_id":null,"source":"cli"}}`},
		{name: "malformed json", path: path, event: `{"type":`},
		{name: "malformed filename", path: "/tmp/not-codex.jsonl", event: `{"type":"session_meta","payload":{"id":"` + sid + `","source":"cli"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodexRootSessionID(tt.path, []byte(tt.event)); got != tt.want {
				t.Fatalf("CodexRootSessionID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadCodexRootSessionIDBoundaries(t *testing.T) {
	home := t.TempDir()
	sid := "01a00e37-16c4-7100-89fc-42ce26158f71"
	path := filepath.Join(home, ".codex", "sessions", "2026", "08", "16", "rollout-2026-08-16T22-34-46-"+sid+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	root := `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}`
	subagent := `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":"parent","source":{"subagent":{}}}}`

	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(root + "\n")
	if got := ReadCodexRootSessionID(path); got != sid {
		t.Fatalf("valid root = %q, want %q", got, sid)
	}
	write(subagent + "\n")
	if got := ReadCodexRootSessionID(path); got != "" {
		t.Fatalf("subagent = %q, want empty", got)
	}
	write("{}\n" + root + "\n")
	if got := ReadCodexRootSessionID(path); got != "" {
		t.Fatalf("later metadata = %q, want empty", got)
	}
	write(root)
	if got := ReadCodexRootSessionID(path); got != "" {
		t.Fatalf("unterminated first line = %q, want empty", got)
	}

	prefix := `{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli","padding":"`
	suffix := `"}}` + "\n"
	lineOfLength := func(n int) string {
		t.Helper()
		padding := n - len(prefix) - len(suffix)
		if padding < 0 {
			t.Fatalf("test line length %d too small", n)
		}
		return prefix + strings.Repeat("x", padding) + suffix
	}
	write(lineOfLength(testCodexSessionMetaLineLimit))
	if got := ReadCodexRootSessionID(path); got != sid {
		t.Fatalf("exact-limit root = %q, want %q", got, sid)
	}
	write(lineOfLength(testCodexSessionMetaLineLimit + 1))
	if got := ReadCodexRootSessionID(path); got != "" {
		t.Fatalf("over-limit root = %q, want empty", got)
	}

	if got := ReadCodexRootSessionID(filepath.Join(home, "missing.jsonl")); got != "" {
		t.Fatalf("missing file = %q, want empty", got)
	}
	if got := ReadCodexRootSessionID(home); got != "" {
		t.Fatalf("directory read = %q, want empty", got)
	}
}

func TestSessionIDValidatesCodexRootMetadata(t *testing.T) {
	home := t.TempDir()
	data := filepath.Join(home, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
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
	if err := os.WriteFile(rootPath, []byte(`{"type":"session_meta","payload":{"id":"`+rootSID+`","parent_thread_id":null,"source":"cli"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subPath, []byte(`{"type":"session_meta","payload":{"id":"`+subSID+`","parent_thread_id":"`+rootSID+`","source":{"subagent":{}}}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(data, "config-work-codex.json")
	if err := os.WriteFile(config, []byte(`{"session_id":"`+rootSID+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := SessionID(data, "work", "codex", home); got != rootSID {
		t.Fatalf("root config = %q, want %q", got, rootSID)
	}
	if err := os.WriteFile(config, []byte(`{"session_id":"`+subSID+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := SessionID(data, "work", "codex", home); got != "" {
		t.Fatalf("subagent config = %q, want empty", got)
	}

	if err := os.WriteFile(filepath.Join(data, "config-work-claude.json"), []byte(`{"session_id":"claude-id"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := SessionID(data, "work", "claude", home); got != "claude-id" {
		t.Fatalf("claude config = %q, want claude-id", got)
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
