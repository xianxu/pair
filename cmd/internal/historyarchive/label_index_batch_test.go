package historyarchive

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestIndexRefreshBatchMatchesIndependentDestinationSequence(t *testing.T) {
	for _, count := range []int{1, 8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			old := LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}, Extensions: map[string]json.RawMessage{"future_index": json.RawMessage(`42`)}}
			var refreshes []IndexRefresh
			for i := 0; i < count; i++ {
				r := indexRefresh()
				r.Locator = fmt.Sprintf("pair-backup:backups/pair-history/%04d/01", 2000+i)
				r.FolderID = fmt.Sprintf("folder-%d", i)
				r.Observations[0].Location.Locator = r.Locator
				r.Observations[0].Location.FolderID = r.FolderID
				r.Observations[0].Label.Extensions = map[string]json.RawMessage{"future_label": json.RawMessage(`{"large":12345678901234567890}`)}
				var err error
				old, err = ApplyIndexRefresh(old, r)
				if err != nil {
					t.Fatal(err)
				}
				r.At = r.At.Add(time.Hour)
				if i%2 == 0 {
					r.Observations = nil
				} else {
					r.Observations[0].Location.LastObserved = r.At
					r.Observations[0].Location.Evidence = "observed"
					r.Observations[0].Location.Proof = LocationProof{}
				}
				refreshes = append(refreshes, r)
			}
			before, _ := json.Marshal(old)
			want := old
			for _, r := range refreshes {
				var err error
				want, err = ApplyIndexRefresh(want, r)
				if err != nil {
					t.Fatal(err)
				}
			}
			got, err := applyIndexRefreshes(old, refreshes)
			gotRaw, _ := json.Marshal(got)
			wantRaw, _ := json.Marshal(want)
			if err != nil || string(gotRaw) != string(wantRaw) {
				t.Fatalf("batch differs from destination sequence: %v", err)
			}
			for _, loc := range got.Records[indexLabel().ID].Locations {
				if loc.Proof.Method != "backend-sha256" {
					t.Fatal("batch erased historic proof")
				}
			}
			after, _ := json.Marshal(old)
			if string(before) != string(after) {
				t.Fatal("batch mutated input")
			}
			refreshes[len(refreshes)-1].At = refreshes[len(refreshes)-1].At.Add(-24 * time.Hour)
			if _, err = applyIndexRefreshes(old, refreshes); err == nil {
				t.Fatal("late stale folder accepted")
			}
			after, _ = json.Marshal(old)
			if string(before) != string(after) {
				t.Fatal("failed batch mutated input")
			}
		})
	}
}

func TestMissingIndexDriveFolderRetainsCompleteBinding(t *testing.T) {
	r := indexRefresh()
	r.Locator += "/2025/01"
	r.Observations[0].Location.Locator = r.Locator
	old, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	// A newer unavailable receipt for an old binding cannot replace the
	// complete observation's current binding when the folder later disappears.
	record := old.Records[indexLabel().ID]
	for key, loc := range record.Locations {
		delete(record.Locations, key)
		loc.FolderID = "old-receipt-folder"
		loc.Available = false
		loc.LastObserved = loc.LastObserved.Add(time.Hour)
		record.Locations[loc.Identity()] = loc
	}
	old.Records[indexLabel().ID] = record
	root, _ := ParseDriveDestination("pair-backup", "backups/pair-history")
	refreshes, err := missingIndexDriveFolders(old, root, []indexDriveFolder{{dest: root}}, r.At.Add(2*time.Hour))
	if err != nil || len(refreshes) != 1 || refreshes[0].FolderID != r.FolderID {
		t.Fatalf("absence replaced complete binding with receipt: %+v %v", refreshes, err)
	}
}
