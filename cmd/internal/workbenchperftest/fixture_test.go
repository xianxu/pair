package workbenchperftest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureStreamsValidDistinctOwnerArtifacts(t *testing.T) {
	f := NewFixture(t, 3, 30_000_000)
	if f.Home == "" || f.DataDir == "" || f.Counts == nil || len(f.Owners) != 3 {
		t.Fatalf("incomplete fixture: %+v", f)
	}
	tags := map[string]bool{}
	ids := map[string]bool{}
	for _, o := range f.Owners {
		if tags[o.Tag] || ids[o.NativeID] {
			t.Fatalf("duplicate owner: %+v", o)
		}
		tags[o.Tag] = true
		ids[o.NativeID] = true
		if o.ScopeDir != filepath.Join(f.DataDir, "repos", o.Scope) || o.Agent != "codex" {
			t.Fatalf("bad scope: %+v", o)
		}
		file, err := os.Open(o.Ledger)
		if err != nil {
			t.Fatal(err)
		}
		info, err := file.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() != 30_000_000 {
			t.Fatalf("ledger size=%d", info.Size())
		}
		decoder := json.NewDecoder(bufio.NewReader(file))
		for _, kind := range []string{"launch", "binding"} {
			var record map[string]any
			if err := decoder.Decode(&record); err != nil {
				t.Fatal(err)
			}
			if record["kind"] != kind || record["tag"] != o.Tag || record["scope_key"] != o.Scope {
				t.Fatalf("incorrect ledger record: %+v", record)
			}
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			t.Fatalf("ledger trailing content: %v", err)
		}
		file.Close()
		raw, err := os.ReadFile(o.Transcript)
		if err != nil {
			t.Fatal(err)
		}
		if !json.Valid(bytes.TrimSpace(raw)) || !bytes.Contains(raw, []byte(o.NativeID)) {
			t.Fatalf("invalid native metadata: %s", raw)
		}
	}
	f.Counts.Snapshot().RequireZero(t)
}

func TestFixtureSmallLedgerRetainsRequiredRecords(t *testing.T) {
	f := NewFixture(t, 1, 1)
	raw, err := os.ReadFile(f.Owners[0].Ledger)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"kind":"launch"`)) || !bytes.Contains(raw, []byte(`"kind":"binding"`)) {
		t.Fatalf("required records lost: %s", raw)
	}
}
