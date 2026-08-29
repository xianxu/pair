package sessioninventory_test

import (
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

const (
	agySchemaQuery = "pragma table_info(trajectory_meta)"
	agyFactsQuery  = "select cascade_id, typeof(cascade_id), typeof(trajectory_type), typeof(source) from trajectory_meta limit 2"
)

func TestScanAgyV1RootsWithoutInventingParentEdges(t *testing.T) {
	t.Parallel()

	runtime := sessioninventorytest.NewFakeRuntime()
	fixture := filepath.Join("testdata", "native", "agy", "v1")
	loadNativeFixture(t, runtime, sessioninventory.AgentAgy, "agy-conversations", filepath.Join(fixture, "agy-conversations"))
	loadNativeFixture(t, runtime, sessioninventory.AgentAgy, "agy-brain", filepath.Join(fixture, "agy-brain"))
	for _, nativeID := range []string{"55555555-5555-4555-8555-555555555555", "66666666-6666-4666-8666-666666666666"} {
		database := sessioninventory.Artifact{StorageRoot: "agy-conversations", RelativePath: nativeID + ".db"}
		runtime.PutSQLite(database, agySchemaQuery, sessioninventory.SQLiteResult{
			Columns: []string{"cid", "name", "type", "notnull", "dflt_value", "pk"},
			Rows: [][]string{
				{"0", "trajectory_id", "TEXT", "0", "", "1"},
				{"1", "cascade_id", "TEXT", "0", "", "0"},
				{"2", "trajectory_type", "INTEGER", "0", "", "0"},
				{"3", "source", "INTEGER", "0", "", "0"},
			},
		})
		runtime.PutSQLite(database, agyFactsQuery, sessioninventory.SQLiteResult{
			Columns: []string{"cascade_id", "typeof(cascade_id)", "typeof(trajectory_type)", "typeof(source)"},
			Rows:    [][]string{{nativeID, "text", "integer", "integer"}},
		})
	}

	got := inventoryFromScan(sessioninventory.ScanAgy(runtime))
	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 2 {
		t.Fatalf("forests = %#v", got.Forests)
	}
	joined := got.Forests[0].Roots[0]
	if joined.NativeID != "55555555-5555-4555-8555-555555555555" || !joined.Resumable || len(joined.Artifacts) != 2 || len(joined.Children) != 0 {
		t.Fatalf("joined root = %#v", joined)
	}
	if joined.Artifacts[0].Kind != sessioninventory.ArtifactTranscript || joined.Artifacts[1].Kind != sessioninventory.ArtifactDatabase {
		t.Fatalf("joined artifacts = %#v", joined.Artifacts)
	}
	if !diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticParentMissing) {
		t.Fatalf("diagnostics = %#v, want missing transcript warning", got.Diagnostics)
	}
	if diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("diagnostics = %#v, known Agy side artifacts are not candidates", got.Diagnostics)
	}
}

func TestIncrementalAgyJoinsDatabaseAndTranscriptState(t *testing.T) {
	t.Parallel()
	nativeID := "55555555-5555-4555-8555-555555555555"
	runtime, database, transcript := incrementalAgyFixture(nativeID)
	records := []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"type":"USER_INPUT","content":"prompt"}`)}, {Bytes: []byte(`{"type":"PLANNER_RESPONSE","content":"answer"}`)}}
	state, diagnostics, err := sessioninventory.ValidateAgyDelta(runtime, database, transcript, nil, records)
	if err != nil || len(diagnostics) != 0 || state.Disputed || !state.FirstRecordValidated || state.NativeID != nativeID {
		t.Fatalf("state=%#v diagnostics=%#v err=%v", state, diagnostics, err)
	}
	prior := state
	state, diagnostics, err = sessioninventory.ValidateAgyDelta(runtime, database, transcript, &prior, []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{`)}})
	if err != nil || !state.Disputed || !diagnosticPresent(diagnostics, sessioninventory.DiagnosticSchemaNearMiss) || prior.Disputed {
		t.Fatalf("state=%#v diagnostics=%#v prior=%#v err=%v", state, diagnostics, prior, err)
	}
}

func TestIncrementalAgyDatabaseIdentityMismatchFailsClosed(t *testing.T) {
	t.Parallel()
	nativeID := "55555555-5555-4555-8555-555555555555"
	runtime, database, transcript := incrementalAgyFixture(nativeID)
	runtime.PutSQLite(database.Artifact, agyFactsQuery, sessioninventory.SQLiteResult{
		Columns: []string{"cascade_id", "typeof(cascade_id)", "typeof(trajectory_type)", "typeof(source)"},
		Rows:    [][]string{{"66666666-6666-4666-8666-666666666666", "text", "integer", "integer"}},
	})
	state, diagnostics, err := sessioninventory.ValidateAgyDelta(runtime, database, transcript, nil, nil)
	if err != nil || !state.Disputed || !diagnosticPresent(diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("state=%#v diagnostics=%#v err=%v", state, diagnostics, err)
	}
}

func incrementalAgyFixture(nativeID string) (*sessioninventorytest.FakeRuntime, sessioninventory.FileEntry, sessioninventory.FileEntry) {
	runtime := sessioninventorytest.NewFakeRuntime()
	database := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "agy-conversations", RelativePath: nativeID + ".db", Kind: sessioninventory.ArtifactDatabase}}
	transcript := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "agy-brain", RelativePath: nativeID + "/.system_generated/logs/transcript.jsonl", Kind: sessioninventory.ArtifactTranscript}}
	runtime.PutFile(database, []byte("SQLite format 3\x00fixture"))
	runtime.PutFile(transcript, nil)
	runtime.PutSQLite(database.Artifact, agySchemaQuery, sessioninventory.SQLiteResult{
		Columns: []string{"cid", "name", "type"},
		Rows:    [][]string{{"0", "cascade_id", "TEXT"}, {"1", "trajectory_type", "INTEGER"}, {"2", "source", "INTEGER"}},
	})
	runtime.PutSQLite(database.Artifact, agyFactsQuery, sessioninventory.SQLiteResult{
		Columns: []string{"cascade_id", "typeof(cascade_id)", "typeof(trajectory_type)", "typeof(source)"},
		Rows:    [][]string{{nativeID, "text", "integer", "integer"}},
	})
	return runtime, database, transcript
}
