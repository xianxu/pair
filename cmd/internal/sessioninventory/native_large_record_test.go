package sessioninventory_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestNativeLargeRecordsScanAndEvents(t *testing.T) {
	// Sequential cases share padding; every existing fixture record keeps all
	// identity/event fields and gains an ignored field exceeding the old 8 MiB
	// ceiling. Codex has separate large-record coverage.
	padding := bytes.Repeat([]byte{'x'}, (8<<20)+1)
	for _, tc := range []struct {
		agent                    sessioninventory.Agent
		root, relative, nativeID string
		scan                     func(sessioninventory.Runtime) sessioninventory.ScanResult
	}{
		{sessioninventory.AgentClaude, "claude-projects", "-repo/11111111-1111-4111-8111-111111111111.jsonl", "11111111-1111-4111-8111-111111111111", sessioninventory.ScanClaude},
		{sessioninventory.AgentQoder, "qoder-projects", "-repo/11111111-1111-4111-8111-111111111111.jsonl", "11111111-1111-4111-8111-111111111111", sessioninventory.ScanQoder},
		{sessioninventory.AgentMuse, "muse-sessions", "2026/08/28/77777777-7777-4777-8777-777777777777/session.jsonl", "77777777-7777-4777-8777-777777777777", sessioninventory.ScanMuse},
		{sessioninventory.AgentAgy, "agy-brain", "55555555-5555-4555-8555-555555555555/.system_generated/logs/transcript.jsonl", "55555555-5555-4555-8555-555555555555", sessioninventory.ScanAgy},
	} {
		t.Run(string(tc.agent), func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "native", string(tc.agent), "v1", tc.root, filepath.FromSlash(tc.relative)))
			if err != nil {
				t.Fatal(err)
			}
			runtime := sessioninventorytest.NewFakeRuntime()
			entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: tc.root, RelativePath: tc.relative, Kind: sessioninventory.ArtifactTranscript}}
			if tc.agent == sessioninventory.AgentAgy {
				// Reuse the keyed SQLite fake, including its real database header,
				// so Agy's transcript is joined to authorized conversation evidence.
				var database sessioninventory.FileEntry
				runtime, database, entry = incrementalAgyFixture(tc.nativeID)
				runtime.AddRoot(sessioninventory.StorageRoot{Agent: tc.agent, Name: database.Artifact.StorageRoot})
			}
			runtime.AddRoot(sessioninventory.StorageRoot{Agent: tc.agent, Name: tc.root})
			runtime.PutFile(entry, raw)
			baseline := tc.scan(runtime)
			if len(baseline.Diagnostics) != 0 || len(baseline.Facts) != 1 || baseline.Facts[0].NativeID != tc.nativeID || baseline.Facts[0].Role != sessioninventory.RoleRoot {
				t.Fatalf("invalid unpadded scan fixture: facts=%v diagnostics=%v", baseline.Facts, baseline.Diagnostics)
			}
			inventory := inventoryFromScan(baseline)
			wantEvents, diagnostics := sessioninventory.NativeEventsWithRuntime(runtime, inventory, tc.agent)
			if len(diagnostics) != 0 || len(wantEvents) == 0 {
				t.Fatalf("invalid unpadded event fixture: count=%d diagnostics=%v", len(wantEvents), diagnostics)
			}
			var padded bytes.Buffer
			lines := bytes.Split(bytes.TrimSpace(raw), []byte{'\n'})
			padded.Grow(len(raw) + len(lines)*(len(padding)+16))
			for _, line := range lines {
				line = bytes.TrimSpace(line)
				if !json.Valid(line) || len(line) < 2 || line[0] != '{' || line[len(line)-1] != '}' {
					t.Fatal("fixture record is not a JSON object")
				}
				padded.Write(line[:len(line)-1])
				padded.WriteString(`,"padding":"`)
				padded.Write(padding)
				padded.WriteString("\"}\n")
			}
			runtime.PutFile(entry, padded.Bytes())
			t.Run("scan", func(t *testing.T) {
				got := tc.scan(runtime)
				if len(got.Diagnostics) != 0 || !reflect.DeepEqual(got.Facts, baseline.Facts) {
					t.Fatalf("oversized %s records changed scan authority: facts=%v diagnostics=%v; want=%v", tc.agent, got.Facts, got.Diagnostics, baseline.Facts)
				}
			})
			t.Run("events", func(t *testing.T) {
				// Use the previously authorized inventory so a scan rejection
				// cannot mask an independent event-reader cutoff.
				got, diagnostics := sessioninventory.NativeEventsWithRuntime(runtime, inventory, tc.agent)
				if len(diagnostics) != 0 || len(got) != len(wantEvents) {
					t.Fatalf("oversized %s events: count=%d want=%d diagnostics=%v", tc.agent, len(got), len(wantEvents), diagnostics)
				}
				for i := range got {
					if got[i].Agent != wantEvents[i].Agent || got[i].RootNodeID != wantEvents[i].RootNodeID || !reflect.DeepEqual(got[i].Event, wantEvents[i].Event) {
						t.Fatalf("oversized record changed event %d: got=%v want=%v", i, got[i].Event, wantEvents[i].Event)
					}
					if i > 0 && got[i].Position <= got[i-1].Position {
						t.Fatalf("event positions lost ordering at %d", i)
					}
				}
			})
		})
	}
}
