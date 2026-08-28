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
