package couchcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func TestSessionSyncApplyShadowCLI(t *testing.T) {
	for _, retained := range []bool{false, true} {
		t.Run(map[bool]string{false: "mapped", true: "retained"}[retained], func(t *testing.T) {
			rt, file := sessionSyncRT(t)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			original := string(raw) + "\n## Work\n\nwork\n"
			for _, path := range []string{file, sessionsync.PathsOf(file).Shadow} {
				if err := os.WriteFile(path, []byte(original), 0600); err != nil {
					t.Fatal(err)
				}
			}
			current := strings.Replace(original, "old", "operator unread", 1)
			if err := os.WriteFile(file, []byte(current), 0600); err != nil {
				t.Fatal(err)
			}
			record := map[string]any{"old": "work", "new": "done", "occurrence": 1}
			if retained {
				record["old"] = "operator unread"
				record["new"] = "agent changed"
			}
			records, err := json.Marshal([]any{record})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "edits.json")
			if err := os.WriteFile(path, records, 0600); err != nil {
				t.Fatal(err)
			}
			code, out, stderr := runSS(t, rt, "apply", path, file, "--if-hash", sessionsync.Hash(current), "--json")
			if code != 0 {
				t.Fatalf("exit%d %s %s", code, out, stderr)
			}
			var response struct{ Code, Hash string }
			if err := json.Unmarshal([]byte(out), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != "ok" || response.Hash == "" {
				t.Fatal(out)
			}
			if retained != strings.Contains(stderr, "previous diff baseline") {
				t.Fatalf("warning %q", stderr)
			}
			if retained && !strings.Contains(stderr, "review the remaining edits, then read --all") {
				t.Fatal(stderr)
			}
			code, diff, errout := runSS(t, rt, "diff", file)
			if code != 0 {
				t.Fatalf("diff %s %s", diff, errout)
			}
			if retained {
				if !strings.Contains(diff, "+agent changed") {
					t.Fatal(diff)
				}
			} else if !strings.Contains(diff, "+operator unread") || strings.Contains(diff, "+done") {
				t.Fatal(diff)
			}
		})
	}
}
