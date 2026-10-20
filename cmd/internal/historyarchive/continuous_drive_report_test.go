package historyarchive

import (
	"encoding/json"
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
