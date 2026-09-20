package sessioninventory_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestCodexLargeRecordTargetValidation(t *testing.T) {
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	const relative = "2026/08/28/rollout-root-" + nativeID + ".jsonl"
	prefix, err := os.ReadFile(filepath.Join("testdata", "native", "codex", "v1", "codex-sessions", filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	// Share one padded record between sequential subtests. Its JSON body is
	// exactly 8 MiB + 1, independently of the production record-limit policy.
	const recordBytes = (8 << 20) + 1
	const head = `{"type":"response_item","payload":{"type":"function_call_output","call_id":"large-record","output":"`
	const tail = `"}}`
	record := bytes.Repeat([]byte{'x'}, recordBytes+1)
	copy(record, head)
	copy(record[recordBytes-len(tail):], tail)
	record[recordBytes] = '\n'
	if !json.Valid(record) {
		t.Fatal("large record fixture is not valid JSON")
	}

	for _, appended := range []bool{false, true} {
		name := "initial"
		if appended {
			name = "appended"
		}
		t.Run(name, func(t *testing.T) {
			runtime := sessioninventorytest.NewFakeRuntime()
			root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"}
			entry := sessioninventory.FileEntry{
				Artifact:     sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript},
				StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1",
			}
			runtime.AddRoot(root)
			runtime.PutFile(entry, prefix)
			observe := func() []sessioninventory.ArtifactObservation {
				t.Helper()
				observations, diagnostics := sessioninventory.ObserveAgentMetadata(runtime, sessioninventory.AgentCodex)
				if len(diagnostics) != 0 || len(observations) != 1 {
					t.Fatalf("metadata observations=%d diagnostics=%v", len(observations), diagnostics)
				}
				return observations
			}
			var prior sessioninventory.TargetValidation
			if appended {
				validations, diagnostics := sessioninventory.ValidateTargetWork(runtime, sessioninventory.AgentCodex, observe())
				if len(diagnostics) != 0 || len(validations) != 1 {
					t.Fatalf("small prefix validation count=%d diagnostics=%v", len(validations), diagnostics)
				}
				prior = validations[0]
			}
			runtime.AppendFile(entry.Artifact, record, "ctime:2")
			var got sessioninventory.TargetValidation
			if appended {
				var diagnostics []sessioninventory.Diagnostic
				var err error
				got, diagnostics, err = sessioninventory.AdvanceTargetValidation(runtime, prior, observe())
				if err != nil || len(diagnostics) != 0 {
					t.Fatalf("AdvanceTargetValidation rejected valid %d-byte Codex record: err=%v diagnostics=%v", recordBytes, err, diagnostics)
				}
			} else {
				validations, diagnostics := sessioninventory.ValidateTargetWork(runtime, sessioninventory.AgentCodex, observe())
				if len(diagnostics) != 0 || len(validations) != 1 {
					t.Fatalf("ValidateTargetWork rejected valid %d-byte Codex record: count=%d diagnostics=%v", recordBytes, len(validations), diagnostics)
				}
				got = validations[0]
			}
			if !got.State.FirstRecordValidated || got.State.Disputed || got.State.NativeID != nativeID || got.Fact.NativeID != nativeID || got.Fact.Role != sessioninventory.RoleRoot {
				t.Fatal("large record lost scanner authority or root identity")
			}
			if len(got.Events) != 3 || got.Events[2].Event.Kind != sessioninventory.EventToolResult {
				t.Fatalf("large tool-result event dropped/duplicated: count=%d", len(got.Events))
			}
			if len(got.Results) != 1 {
				t.Fatalf("result count=%d, want one transcript", len(got.Results))
			}
			for _, result := range got.Results {
				end := int64(len(prefix) + len(record))
				if result.Disputed || result.RawObservedOffset != end || result.FrameState.ParserCompleteOffset != end || len(result.FrameState.IncompleteTail) != 0 {
					t.Fatalf("large record did not reach stable parser-complete EOF: raw=%d parsed=%d tail=%d disputed=%v", result.RawObservedOffset, result.FrameState.ParserCompleteOffset, len(result.FrameState.IncompleteTail), result.Disputed)
				}
				wantRecords := bytes.Count(prefix, []byte{'\n'}) + 1
				if appended {
					wantRecords = 1 // incremental validation must consume only the suffix
				}
				if len(result.Records) != wantRecords {
					t.Fatalf("framed records=%d, want %d", len(result.Records), wantRecords)
				}
				last := result.Records[len(result.Records)-1]
				if last.Offset != int64(len(prefix)) || !bytes.Equal(last.Bytes, record[:recordBytes]) {
					t.Fatalf("large record truncated or misplaced: offset=%d bytes=%d", last.Offset, len(last.Bytes))
				}
			}
		})
	}
}
