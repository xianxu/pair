package historyarchive

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func restoreModelIndex(n int) (LabelIndex, []string) {
	i := LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}
	ids := make([]string, n)
	for k := range ids {
		label := indexLabel()
		label.ID = fmt.Sprintf("%064x", k+1)
		ids[k] = label.ID
		i.Records[label.ID] = LabelIndexRecord{Label: label, Locations: map[string]BundleLocation{}}
	}
	return i, ids
}

func TestPlanRestoreDeduplicatesAndPreservesLiteralSelection(t *testing.T) {
	index, ids := restoreModelIndex(2)
	req := RestoreRequest{IDs: []string{ids[1], ids[0], ids[1]}, To: "/private/restore/run", Paths: []string{"literal[*]?.md", "nested/二.txt", "literal[*]?.md"}}
	original, _ := json.Marshal(index)
	originalReq := RestoreRequest{IDs: append([]string(nil), req.IDs...), To: req.To, Paths: append([]string(nil), req.Paths...)}
	plan, err := PlanRestore(index, req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Request.IDs, []string{ids[1], ids[0]}) || !reflect.DeepEqual(plan.Request.Paths, []string{"literal[*]?.md", "nested/二.txt"}) {
		t.Fatalf("selection changed: %+v", plan.Request)
	}
	if len(plan.Bundles) != 2 || plan.Bundles[0].Label.ID != ids[1] || plan.Bundles[1].Label.ID != ids[0] {
		t.Fatalf("input order lost: %+v", plan.Bundles)
	}
	if plan.CipherBytes != 34 || plan.RawBytes != 2 {
		t.Fatalf("incorrect admission totals: %+v", plan)
	}
	for _, bundle := range plan.Bundles {
		b, l, err := BundleKeys(bundle.Label.Kind, bundle.Label.From, bundle.Label.To, bundle.Label.ID)
		if err != nil || bundle.BundleKey != b || bundle.LabelKey != l {
			t.Fatalf("incorrect local keys: %+v %v", bundle, err)
		}
	}
	after, _ := json.Marshal(index)
	if string(original) != string(after) || !reflect.DeepEqual(req, originalReq) {
		t.Fatal("planner mutated inputs")
	}
}

func TestPlanRestoreRejectsUnsafeRequests(t *testing.T) {
	index, ids := restoreModelIndex(1)
	for _, req := range []RestoreRequest{
		{To: "/restore"},
		{IDs: []string{"short"}, To: "/restore"},
		{IDs: []string{strings.Repeat("A", 64)}, To: "/restore"},
		{IDs: []string{strings.Repeat("g", 64)}, To: "/restore"},
		{IDs: ids},
		{IDs: ids, To: "relative/restore"},
		{IDs: ids, To: "/"},
		{IDs: ids, To: "/tmp/../restore"},
		{IDs: ids, To: "/tmp/restore/"},
		{IDs: ids, To: "/restore\x00"},
		{IDs: ids, To: "/restore", From: "observed"},
		{IDs: ids, To: "/restore", Paths: []string{"../private"}},
		{IDs: ids, To: "/restore", Paths: []string{"/absolute"}},
		{IDs: ids, To: "/restore", Paths: []string{"a/../b"}},
		{IDs: ids, To: "/restore", Paths: []string{""}},
		{IDs: ids, To: "/restore", Paths: []string{"."}},
		{IDs: ids, To: "/restore", Paths: []string{"a\\b"}},
	} {
		if _, err := PlanRestore(index, req); err == nil {
			t.Fatalf("accepted unsafe request: %+v", req)
		}
	}
	for _, from := range []string{"", "auto", "local", "drive"} {
		if _, err := PlanRestore(index, RestoreRequest{IDs: ids, To: "/restore", From: from}); err != nil {
			t.Fatalf("valid backend %q: %v", from, err)
		}
	}
	unknown := strings.Repeat("f", 64)
	if _, err := PlanRestore(index, RestoreRequest{IDs: []string{unknown}, To: "/restore"}); err == nil || !strings.Contains(err.Error(), "find") || !strings.Contains(err.Error(), "sync") {
		t.Fatalf("unknown bundle should suggest discovery: %v", err)
	}
}

func TestPlanRestoreUniqueIDAndAggregateAdmission(t *testing.T) {
	index, ids := restoreModelIndex(65)
	if _, err := PlanRestore(index, RestoreRequest{IDs: ids, To: "/restore"}); err == nil {
		t.Fatal("admitted more than 64 unique bundles")
	}
	duplicateIDs := append(append([]string(nil), ids[:64]...), ids[:64]...)
	if _, err := PlanRestore(index, RestoreRequest{IDs: duplicateIDs, To: "/restore"}); err != nil {
		t.Fatalf("duplicates consumed bundle budget: %v", err)
	}
	for _, bound := range []string{"cipher", "raw", "files"} {
		t.Run(bound, func(t *testing.T) {
			i, ids := restoreModelIndex(3)
			for _, id := range ids {
				r := i.Records[id]
				switch bound {
				case "cipher":
					r.Label.Bytes = MaxCiphertextBytes
				case "raw":
					r.Label.Kind = "codex-database"
					r.Label.RawBytes = MaxSnapshotRawBytes
				case "files":
					r.Label.FileCount = MaxFiles
				}
				i.Records[id] = r
			}
			if bound == "files" {
				i, ids = restoreModelIndex(11)
				for _, id := range ids {
					r := i.Records[id]
					r.Label.FileCount = MaxFiles
					i.Records[id] = r
				}
			}
			admitted := 2
			if bound == "files" {
				admitted = 10
			}
			if _, err := PlanRestore(i, RestoreRequest{IDs: ids[:admitted], To: "/restore"}); err != nil {
				t.Fatalf("exact bound refused: %v", err)
			}
			// A literal filter cannot discount whole-bundle admission before validation.
			if _, err := PlanRestore(i, RestoreRequest{IDs: ids, To: "/restore", Paths: []string{"one-file"}}); err == nil {
				t.Fatal("aggregate over-bound admitted")
			}
		})
	}
}

func TestPlanRestoreReusesLabelValidationAndIgnoresAvailabilityAsAuthority(t *testing.T) {
	for _, mutate := range []func(*BundleLabel){
		func(l *BundleLabel) { l.ID = strings.Repeat("A", 64) },
		func(l *BundleLabel) { l.Kind = "unknown-kind" },
		func(l *BundleLabel) { l.To = l.From.Add(-time.Hour) },
		func(l *BundleLabel) { l.From = time.Time{} },
		func(l *BundleLabel) { l.FileCount = MaxFiles + 1 },
		func(l *BundleLabel) { l.Bytes = MaxCiphertextBytes + 1 },
		func(l *BundleLabel) { l.RawBytes = MaxBundleBytes + 1 },
	} {
		index, ids := restoreModelIndex(1)
		r := index.Records[ids[0]]
		mutate(&r.Label)
		index.Records[ids[0]] = r
		if _, err := PlanRestore(index, RestoreRequest{IDs: ids, To: "/restore"}); err == nil {
			t.Fatal("invalid label bypassed admission")
		}
	}
	index, ids := restoreModelIndex(1)
	// A valid label without any known available location still admits a request:
	// acquisition rechecks configured roots and verifies the ciphertext itself.
	if _, err := PlanRestore(index, RestoreRequest{IDs: ids, To: "/restore"}); err != nil {
		t.Fatalf("availability incorrectly became authority: %v", err)
	}
	verified, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, indexRefresh())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanRestore(verified, RestoreRequest{IDs: []string{indexLabel().ID}, To: "/restore"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NextRestore(RestoreState{Phase: RestorePlanned}, RestorePublishOutput); err == nil {
		t.Fatal("verified discovery evidence skipped validation stages")
	}
}

func TestNextRestoreStrictTransitionMatrix(t *testing.T) {
	allowed := map[RestorePhase]map[RestoreEvent]RestorePhase{
		RestorePlanned:        {RestoreVerifyInputs: RestoreInputsVerified, RestoreFail: RestoreRefused, RestoreCancel: RestoreCancelled},
		RestoreInputsVerified: {RestoreBeginExtraction: RestoreExtracting, RestoreFail: RestoreRefused, RestoreCancel: RestoreCancelled},
		RestoreExtracting:     {RestoreValidateOutput: RestoreValidated, RestoreFail: RestoreRefused, RestoreCancel: RestoreCancelled},
		RestoreValidated:      {RestorePublishOutput: RestorePublished, RestorePublishUncertain: RestorePublicationUncertain, RestoreFail: RestoreRefused, RestoreCancel: RestoreCancelled},
	}
	phases := []RestorePhase{RestorePlanned, RestoreInputsVerified, RestoreExtracting, RestoreValidated, RestorePublished, RestoreRefused, RestoreCancelled, RestorePublicationUncertain, RestorePhase(255)}
	events := []RestoreEvent{RestoreVerifyInputs, RestoreBeginExtraction, RestoreValidateOutput, RestorePublishOutput, RestoreFail, RestoreCancel, RestorePublishUncertain, RestoreEvent(255)}
	for _, phase := range phases {
		for _, event := range events {
			state := RestoreState{Phase: phase}
			got, err := NextRestore(state, event)
			want, ok := allowed[phase][event]
			if ok {
				if err != nil || got.Phase != want {
					t.Fatalf("phase=%d event=%d got=%+v err=%v", phase, event, got, err)
				}
			} else if err == nil || got != state {
				t.Fatalf("invalid transition changed state: phase=%d event=%d got=%+v err=%v", phase, event, got, err)
			}
		}
	}
	state := RestoreState{Phase: RestorePlanned}
	for _, event := range []RestoreEvent{RestoreVerifyInputs, RestoreBeginExtraction, RestoreValidateOutput, RestorePublishOutput} {
		next, err := NextRestore(state, event)
		if err != nil {
			t.Fatal(err)
		}
		state = next
	}
	if state.Phase != RestorePublished {
		t.Fatalf("complete ordered sequence did not publish: %+v", state)
	}
}

func TestPlanRestoreRetainsDeterministicIndependentDriveLocations(t *testing.T) {
	index, ids := restoreModelIndex(1)
	record := index.Records[ids[0]]
	for n, id := range []string{"z", "a", "old", "unavailable"} {
		loc := BundleLocation{Backend: "drive", Locator: "backup:pair-history", FolderID: id, BundleKey: ids[0] + ".tar.zst.gpg", LabelKey: ids[0] + ".json", Available: id != "unavailable", Evidence: "observed", FirstObserved: record.Label.From, LastObserved: record.Label.From.Add(time.Hour), Extensions: map[string]json.RawMessage{"future": json.RawMessage(`{"x":1}`)}, Proof: LocationProof{Extensions: map[string]json.RawMessage{"future": json.RawMessage(`1`)}}}
		if n == 2 {
			loc.LastObserved = loc.FirstObserved
		}
		record.Locations[loc.Identity()] = loc
	}
	index.Records[ids[0]] = record
	plan, err := PlanRestore(index, RestoreRequest{IDs: ids, To: "/restore"})
	if err != nil {
		t.Fatal(err)
	}
	locations := plan.Bundles[0].DriveLocations
	if len(locations) != 4 || locations[0].FolderID != "a" || locations[1].FolderID != "z" || locations[2].FolderID != "old" || locations[3].FolderID != "unavailable" {
		t.Fatalf("unstable preference: %+v", locations)
	}
	locations[0].Extensions["future"][0] = '['
	locations[0].Proof.Extensions["future"][0] = '2'
	original := index.Records[ids[0]].Locations[locations[0].Identity()]
	if string(original.Extensions["future"]) != `{"x":1}` || string(original.Proof.Extensions["future"]) != `1` {
		t.Fatal("planner aliases raw location/proof fields")
	}
}

func TestRestoreDriveAcquisitionReservesSharedGroupBudgetsBeforeIO(t *testing.T) {
	for _, kind := range []string{"entries", "bytes"} {
		t.Run(kind, func(t *testing.T) {
			index, ids := restoreModelIndex(3)
			plan, err := PlanRestore(index, RestoreRequest{IDs: ids, To: "/private/restore"})
			if err != nil {
				t.Fatal(err)
			}
			for n := range plan.Bundles {
				bundle := &plan.Bundles[n]
				loc := BundleLocation{Backend: "drive", Locator: "backup:pair-history", FolderID: fmt.Sprintf("folder-%d", n), BundleKey: bundle.Label.ID + ".tar.zst.gpg", LabelKey: bundle.Label.ID + ".json", Evidence: "observed", FirstObserved: bundle.Label.From, LastObserved: bundle.Label.From}
				bundle.DriveLocations = []BundleLocation{loc}
			}
			budget := RestoreExtractionBudget{Entries: MaxRestoreEntries - 29}
			if kind == "bytes" {
				budget.Entries = 0
				for n := range plan.Bundles {
					plan.Bundles[n].Label.Bytes = MaxCiphertextBytes
				}
			}
			r := Restorer{Config: ContinuousConfig{DriveRemote: "backup", DriveFolder: "pair-history"}}
			err = r.acquireDriveRestoreInputs(context.Background(), plan, plan.Bundles, &budget)
			if err == nil || !strings.Contains(err.Error(), "budget exhausted") {
				t.Fatalf("shared %s budget not refused before IO: %v", kind, err)
			}
		})
	}
}
