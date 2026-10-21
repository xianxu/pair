package historyarchive

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func driveCommandJSON(t *testing.T) map[string]any {
	t.Helper()
	raw, err := json.Marshal(validCommandReport())
	if err != nil {
		t.Fatal(err)
	}
	var r map[string]any
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	r["max_drive_files"] = 20000
	r["report"].(map[string]any)["Drive"] = map[string]any{"Inventory": map[string]any{"destination": "pair-backup:backups/pair-history", "folder_id": "folder-1", "phase": "after", "files": 0}}
	return r
}

func TestContinuousDriveReportRejectsInvalidObservation(t *testing.T) {
	cases := map[string]func(map[string]any, map[string]any){
		"missing count":       func(r, o map[string]any) { delete(o, "files") },
		"null count":          func(r, o map[string]any) { o["files"] = nil },
		"negative count":      func(r, o map[string]any) { o["files"] = -1 },
		"unknown phase":       func(r, o map[string]any) { o["phase"] = "estimated" },
		"missing phase":       func(r, o map[string]any) { delete(o, "phase") },
		"missing destination": func(r, o map[string]any) { delete(o, "destination") },
		"bad destination":     func(r, o map[string]any) { o["destination"] = "../wrong" },
		"missing folder":      func(r, o map[string]any) { delete(o, "folder_id") },
		"empty folder":        func(r, o map[string]any) { o["folder_id"] = "" },
		"zero threshold":      func(r, o map[string]any) { r["max_drive_files"] = 0 },
		"missing threshold":   func(r, o map[string]any) { delete(r, "max_drive_files") },
		"negative threshold":  func(r, o map[string]any) { r["max_drive_files"] = -1 },
		"null threshold":      func(r, o map[string]any) { r["max_drive_files"] = nil },
		"unconfigured": func(r, o map[string]any) {
			r["outcome"] = ContinuousNotConfigured
			p := r["report"].(map[string]any)
			p["Databases"] = nil
			p["Archived"] = nil
		},
		"busy": func(r, o map[string]any) {
			r["outcome"] = ContinuousAlreadyRunning
			p := r["report"].(map[string]any)
			p["Databases"] = nil
			p["Archived"] = nil
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := driveCommandJSON(t)
			o := r["report"].(map[string]any)["Drive"].(map[string]any)["Inventory"].(map[string]any)
			mutate(r, o)
			raw, _ := json.Marshal(r)
			if _, err := DecodeContinuousReport(raw); err == nil {
				t.Fatalf("accepted invalid report: %s", raw)
			}
		})
	}
}

func TestContinuousDriveReportPreservesObservedZeroAndAdditions(t *testing.T) {
	r := driveCommandJSON(t)
	r["future"] = map[string]any{"integer": json.Number("9999999999999999999")}
	r["report"].(map[string]any)["Drive"].(map[string]any)["Inventory"].(map[string]any)["future"] = true
	raw, _ := json.Marshal(r)
	got, err := DecodeContinuousReport(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeContinuousReport(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"files":0`) || !strings.Contains(string(encoded), `"max_drive_files":20000`) {
		t.Fatalf("lost known observation: %s", encoded)
	}
}

func TestContinuousDriveReportLegacyUnknown(t *testing.T) {
	raw, _ := json.Marshal(validCommandReport())
	r, err := DecodeContinuousReport(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Report.Drive != nil && r.Report.Drive.Inventory != nil {
		t.Fatal("legacy report invented observation")
	}
}

func TestContinuousDriveReportEncoderRejectsInvalidObservation(t *testing.T) {
	raw, _ := json.Marshal(driveCommandJSON(t))
	base, err := DecodeContinuousReport(raw)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*ContinuousCommandReport){
		"negative threshold": func(r *ContinuousCommandReport) { r.MaxDriveFiles = -1 },
		"missing threshold":  func(r *ContinuousCommandReport) { r.MaxDriveFiles = 0 },
		"negative count":     func(r *ContinuousCommandReport) { r.Report.Drive.Inventory.Files = -1 },
		"invalid root":       func(r *ContinuousCommandReport) { r.Report.Drive.Inventory.Destination = "relative" },
		"empty folder":       func(r *ContinuousCommandReport) { r.Report.Drive.Inventory.FolderID = "" },
		"unknown phase":      func(r *ContinuousCommandReport) { r.Report.Drive.Inventory.Phase = "estimate" },
	} {
		t.Run(name, func(t *testing.T) {
			r := base
			drive := *base.Report.Drive
			observation := *drive.Inventory
			r.Report.Drive = &drive
			drive.Inventory = &observation
			mutate(&r)
			if _, err := EncodeContinuousReport(r); err == nil {
				t.Fatal("accepted invalid value")
			}
		})
	}
	base.Report.Drive.Inventory.Files = 200001
	if _, err := EncodeContinuousReport(base); err != nil {
		t.Fatalf("aggregate count unnecessarily capped: %v", err)
	}
	base.Report.Drive = nil
	base.MaxDriveFiles = -1
	if _, err := EncodeContinuousReport(base); err == nil {
		t.Fatal("negative threshold accepted without inventory")
	}
}

func TestContinuousDriveReportRejectsKnownAliasesAndNulls(t *testing.T) {
	raw, _ := json.Marshal(driveCommandJSON(t))
	for _, pair := range [][2]string{
		{`"files":0`, `"files":0,"Files":1`},
		{`"max_drive_files":20000`, `"max_drive_files":20000,"MAX_DRIVE_FILES":9`},
		{`"phase":"after"`, `"phase":null`},
		{`"folder_id":"folder-1"`, `"folder_id":null`},
		{`"destination":"pair-backup:backups/pair-history"`, `"destination":null`},
	} {
		malformed := strings.Replace(string(raw), pair[0], pair[1], 1)
		if malformed == string(raw) {
			t.Fatalf("mutation missed %s", pair[0])
		}
		if _, err := DecodeContinuousReport([]byte(malformed)); err == nil {
			t.Fatalf("accepted ambiguous/null observation %s", malformed)
		}
	}
}

func TestContinuousDriveReportFolderObservations(t *testing.T) {
	observation := func(dest, id, phase string, files any) map[string]any {
		return map[string]any{"destination": dest, "folder_id": id, "phase": phase, "files": files}
	}
	root := func() map[string]any { return observation("remote:root", "root-id", "before", 2) }
	leaf := func() map[string]any { return observation("remote:root/2026/10", "month-id", "after", 8) }
	for name, tc := range map[string]struct {
		items any
		bad   bool
	}{
		"null list": {nil, false}, "empty": {[]any{}, false},
		"additive":         {[]any{root(), leaf()}, false},
		"malformed list":   {map[string]any{}, true},
		"fractional count": {[]any{observation("remote:root", "root-id", "before", 1.5)}, true},
		"null item":        {[]any{nil}, true}, "negative": {[]any{observation("remote:root", "root-id", "before", -1)}, true},
		"missing count":           {[]any{map[string]any{"destination": "remote:root", "folder_id": "root-id", "phase": "before"}}, true},
		"null count":              {[]any{observation("remote:root", "root-id", "before", nil)}, true},
		"invalid destination":     {[]any{observation("../root", "root-id", "before", 0)}, true},
		"invalid ID":              {[]any{observation("remote:root", "", "before", 0)}, true},
		"additive unknown":        {[]any{map[string]any{"destination": "remote:root", "folder_id": "root-id", "phase": "before", "files": 0, "future": map[string]any{"nested": true}}}, false},
		"rebound":                 {[]any{root(), observation("remote:root", "other-id", "after", 4)}, true},
		"aliased folder":          {[]any{root(), observation("remote:root/2026/10", "root-id", "after", 4)}, true},
		"duplicate":               {[]any{root(), root()}, false},
		"contradictory duplicate": {[]any{root(), observation("remote:root", "root-id", "before", 3)}, true},
		"before after":            {[]any{root(), observation("remote:root", "root-id", "after", 4), leaf()}, false},
	} {
		t.Run(name, func(t *testing.T) {
			r := driveCommandJSON(t)
			d := r["report"].(map[string]any)["Drive"].(map[string]any)
			delete(d, "Inventory")
			d["Inventories"] = tc.items
			raw, _ := json.Marshal(r)
			got, err := DecodeContinuousReport(raw)
			if (err != nil) != tc.bad {
				t.Fatalf("decode error=%v bad=%v JSON=%s", err, tc.bad, raw)
			}
			if !tc.bad {
				if _, err := EncodeContinuousReport(got); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	for _, threshold := range []any{nil, 0, -1} {
		r := driveCommandJSON(t)
		r["max_drive_files"] = threshold
		r["report"].(map[string]any)["Drive"] = map[string]any{"Inventories": []any{leaf()}}
		raw, _ := json.Marshal(r)
		if _, err := DecodeContinuousReport(raw); err == nil {
			t.Fatalf("accepted additive inventory with invalid threshold %v", threshold)
		}
	}
	for _, outcome := range []string{ContinuousNotConfigured, ContinuousAlreadyRunning} {
		r := driveCommandJSON(t)
		r["outcome"] = outcome
		p := r["report"].(map[string]any)
		p["Databases"] = nil
		p["Archived"] = nil
		p["Drive"] = map[string]any{"Inventories": []any{leaf()}}
		raw, _ := json.Marshal(r)
		if _, err := DecodeContinuousReport(raw); err == nil {
			t.Fatalf("accepted %s with folder observation", outcome)
		}
	}
}

func TestContinuousDriveReportFolderBoundsAndLegacyConflicts(t *testing.T) {
	r := validCommandReport()
	r.MaxDriveFiles = 20
	root := DriveInventoryObservation{Destination: "remote:root", FolderID: "root-id", Phase: "before", Files: 0}
	r.Report.Drive = &DriveReport{}
	for i := 0; i < MaxIndexDestinations; i++ {
		r.Report.Drive.Inventories = append(r.Report.Drive.Inventories, DriveInventoryObservation{Destination: fmt.Sprintf("remote:root/folder-%d", i), FolderID: fmt.Sprintf("id-%d", i), Phase: "after", Files: 0})
	}
	if _, err := EncodeContinuousReport(r); err != nil {
		t.Fatalf("exact destination bound: %v", err)
	}
	r.Report.Drive.Inventory = &root
	if _, err := EncodeContinuousReport(r); err == nil {
		t.Fatal("accepted too many destinations including legacy")
	}
	r.Report.Drive = &DriveReport{Inventory: &root, Inventories: []DriveInventoryObservation{root}}
	if _, err := EncodeContinuousReport(r); err != nil {
		t.Fatalf("matching legacy: %v", err)
	}
	r.Report.Drive.Inventories[0].Files++
	if _, err := EncodeContinuousReport(r); err == nil {
		t.Fatal("accepted conflicting legacy count")
	}
	r.Report.Drive.Inventories = make([]DriveInventoryObservation, 2*MaxIndexDestinations+1)
	for i := range r.Report.Drive.Inventories {
		r.Report.Drive.Inventories[i] = root
	}
	raw, _ := json.Marshal(r)
	if _, err := DecodeContinuousReport(raw); err == nil {
		t.Fatal("decoder accepted too many repeated observations")
	}
	if _, err := EncodeContinuousReport(r); err == nil {
		t.Fatal("encoder accepted too many repeated observations")
	}
}

func FuzzContinuousDriveFolderReport(f *testing.F) {
	r := validCommandReport()
	r.MaxDriveFiles = 20
	root := DriveInventoryObservation{Destination: "remote:root", FolderID: "root-id", Phase: "before", Files: 0}
	after := root
	after.Phase = "after"
	after.Files = 30
	r.Report.Drive = &DriveReport{Inventory: &after, Inventories: []DriveInventoryObservation{root, after, {Destination: "remote:root/2026/10", FolderID: "month-id", Phase: "after", Files: 40}}}
	seed, _ := json.Marshal(r)
	f.Add(seed)
	for _, pair := range [][2]string{{`"files":0`, `"files":null`}, {`"files":0`, `"files":-1`}, {`"folder_id":"month-id"`, `"folder_id":"root-id"`}, {`"files":0`, `"files":0,"Files":1`}} {
		f.Add([]byte(strings.Replace(string(seed), pair[0], pair[1], 1)))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		got, err := DecodeContinuousReport(raw)
		if err != nil {
			return
		}
		encoded, err := EncodeContinuousReport(got)
		if err != nil {
			// Materialized optional fields can make an accepted input grow.
			materialized, marshalErr := json.Marshal(got)
			if marshalErr == nil && len(materialized) >= MaxContinuousReportBytes {
				return
			}
			t.Fatal(err)
		}
		again, err := DecodeContinuousReport(encoded)
		if err != nil {
			t.Fatal(err)
		}
		projected := LatestDriveInventories(got.Report.Drive)
		if !reflect.DeepEqual(projected, LatestDriveInventories(again.Report.Drive)) {
			t.Fatal("projection changed on round trip")
		}
		seen := map[string]bool{}
		for _, o := range projected {
			if seen[o.Destination] {
				t.Fatal("duplicate physical destination")
			}
			seen[o.Destination] = true
		}
		if len(seen) > MaxIndexDestinations {
			t.Fatal("unbounded physical destinations")
		}
	})
}
