package couchcore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSlotCurrentObservationMigratesLegacyRecordVersions(t *testing.T) {
	for _, version := range []int{1, 2, 3} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			s := testLocalThreadStore(t)
			record := recordAtCheckout(t, s.slot.WorktreeRoot, s.slot.WorktreeRoot, "couch-0123456789abcdef")
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]json.RawMessage
			if err = json.Unmarshal(raw, &wire); err != nil {
				t.Fatal(err)
			}
			wire["schema_version"], _ = json.Marshal(version)
			if version == 1 {
				delete(wire, "last_active_at")
			}
			raw, err = json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(s.root, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(s.root, "thread.json")
			if err = os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			observed, err := s.observeSlotCurrent()
			if err != nil || observed.Err != nil || observed.Unsupported || observed.Record == nil {
				t.Fatalf("legacy observation: %+v %v", observed, err)
			}
			if observed.Record.SchemaVersion != ThreadSchemaVersion || observed.Record.RecoveryContext != nil {
				t.Fatal("invalid migration")
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(raw) {
				t.Fatal("read-only observation rewrote legacy bytes")
			}
		})
	}
}
