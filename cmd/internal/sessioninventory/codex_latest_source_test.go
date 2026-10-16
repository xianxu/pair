package sessioninventory_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// rust-v0.162.1 exec::all_thread_source_kinds names every API filter,
// but app-server::filters::source_kind_matches excludes Custom and Internal.
// state::extract::enum_to_string persists these data variants as JSON objects.
func TestCodexLatestExecExcludesNativeCustomAndInternalSources(t *testing.T) {
	for _, source := range []string{`{"custom":"atlas"}`, `{"custom":"chatgpt"}`, `{"internal":"guardian"}`, `{"internal":"memory_consolidation"}`} {
		t.Run(source, func(t *testing.T) {
			f := newCodexLatestFixture(t)
			want := f.add(1, 100, "cli", f.cwd, "")
			newer := f.add(2, 200, source, f.cwd, "")
			path := filepath.Join(f.state, "sessions", "2026", "08", "28", "rollout-test-"+newer+".jsonl")
			raw := strings.Replace(codexLatestValidMeta(newer, f.cwd), `"source":"cli"`, `"source":`+source, 1)
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			got := f.query(si.CodexModeExec, false)
			if got.Status != si.NativeSelectorResolved || got.NativeID != want {
				t.Fatalf("native excludes newer %s; got %+v, want older cli %s", source, got, want)
			}
		})
	}
}
