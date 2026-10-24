package sessionwatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlog"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// pair#618: a codex thread driven only by Couch peer messages has no
// draft-pane turns. The Pair-log entry pair-wrap records for a submitted
// delivery is what lets the watcher confirm its binding.
func TestRunConfirmsCodexBindingFromARecordedPeerTurn(t *testing.T) {
	envelope := "[Couch peer from ops:0; delivery 6ba71ae7-0000-4000-8000-000000000001]\nTL: take #618 M2 next"
	for _, tc := range []struct {
		name    string
		native  string // the user turn in the rollout
		confirm bool
	}{
		{"the rollout holds the delivered envelope", envelope, true},
		{"the rollout holds an unrelated turn", "a turn this slot never got from Couch", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dataDir := t.TempDir()
			native := sessioninventorytest.NewFakeRuntime()
			nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/home/.codex/sessions"}
			native.AddRoot(nativeRoot)
			sid := "019eff64-6ceb-7e72-9d41-a735a97029ad"
			relative := "2026/08/28/rollout-test-" + sid + ".jsonl"
			artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
			quoted, _ := json.Marshal(tc.native)
			native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen:1", MutationToken: "ctime:1"}, codexRound(sid, strings.Trim(string(quoted), `"`)))
			native.SetProcess("1234", "native-identity", nil, []string{filepath.Join(nativeRoot.Path, filepath.FromSlash(relative))})

			// The log bytes come from the wrapper's writer itself, not a literal.
			logPath := filepath.Join(t.TempDir(), "log.md")
			noManagedOwner := func(string) string { return "" }
			if err := pairlog.PersistPeerTurn(noManagedOwner, logPath, []byte(envelope), time.Now(), "6ba71ae7-0000-4000-8000-000000000001"); err != nil {
				t.Fatal(err)
			}
			log, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}

			paths := mustScopedPaths(t, dataDir, "work")
			runtime := newWatcherRuntime(native)
			runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
			runtime.files[paths.Log()] = log
			runtime.files[paths.AgentPID()] = []byte("1234\n")
			runtime.modTimes[paths.AgentPID()] = runtime.now
			runtime.identities["1234"] = "pair-identity"

			// The unrelated round has been observed when the watcher first sleeps.
			// End its owned process then; no wall-clock probation is needed.
			if !tc.confirm {
				runtime.onSleep = func() { runtime.identities["1234"] = "gone" }
			}

			if err := Run(Options{Agent: "codex", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir}, runtime); err != nil {
				t.Fatal(err)
			}
			confirmed := len(runtime.store.records) == 1 && runtime.store.records[0].RootNativeID == sid
			if confirmed != tc.confirm || (!tc.confirm && len(runtime.store.records) != 0) {
				t.Fatalf("confirm=%v records=%#v", tc.confirm, runtime.store.records)
			}
		})
	}
}
