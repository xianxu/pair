package sessionsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// editFixturesDir holds the before/after fixtures the Go and Lua resolvers
// share (pair#661): one contract, one fixture set.
const editFixturesDir = "../../../testdata/edit-records"

type editFixture struct {
	Name    string          `json:"name"`
	Base    string          `json:"base"`
	Records json.RawMessage `json:"records"`
	Result  *string         `json:"result"`
	Invalid []BadEdit       `json:"invalid"`
}

func TestResolveEditsFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(editFixturesDir, "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no fixtures in %s: %v", editFixturesDir, err)
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fx editFixture
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		t.Run(fx.Name, func(t *testing.T) {
			records, err := ParseEdits(fx.Records)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			next, resolved, bad := ResolveEdits(fx.Base, records)
			if fx.Result != nil {
				if len(bad) != 0 || next != *fx.Result {
					t.Fatalf("next=%q bad=%+v, want %q", next, bad, *fx.Result)
				}
				if len(resolved) != len(records) {
					t.Fatalf("resolved %d of %d records", len(resolved), len(records))
				}
				return
			}
			if !reflect.DeepEqual(bad, fx.Invalid) {
				t.Fatalf("bad=%+v, want %+v", bad, fx.Invalid)
			}
		})
	}
}

func TestResolveEditsRangesAndReasons(t *testing.T) {
	records := []EditRecord{{Old: "three", Occurrence: 1, New: "3", Explain: "digit"}, {Old: "one", Occurrence: 1, New: "1"}}
	next, resolved, bad := ResolveEdits("one two three\n", records)
	if len(bad) != 0 || next != "1 two 3\n" {
		t.Fatalf("next=%q bad=%+v", next, bad)
	}
	want := []ResolvedEdit{{Index: 2, BaseStart: 0, BaseEnd: 3, New: "1"}, {Index: 1, BaseStart: 8, BaseEnd: 13, New: "3", Explain: "digit"}}
	if !reflect.DeepEqual(resolved, want) {
		t.Fatalf("resolved=%+v, want %+v", resolved, want)
	}
}

func TestParseEditsRefusesMalformed(t *testing.T) {
	for _, raw := range []string{`{}`, `"x"`, `[{"old": 3}]`, `{"records": "x"}`, `[{"old":"a","occurrence":1.5}]`, `[{"old":"a","occurrence":0,"new":"b"}]`, `[{"old":"a","occurrence":-1,"new":"b"}]`} {
		if _, err := ParseEdits([]byte(raw)); err == nil {
			t.Errorf("ParseEdits(%s) accepted", raw)
		}
	}
}
