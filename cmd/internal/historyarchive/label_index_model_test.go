package historyarchive

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func indexLabel() BundleLabel {
	m := pureManifest()
	id, _ := m.ID()
	return BundleLabel{Version: Version, ID: id, Kind: m.Kind, From: m.From, To: m.To, FileCount: 1, RawBytes: 1, Bytes: 17, Compression: m.Compression, Recipients: m.Recipients, SHA256: strings.Repeat("b", 64)}
}

func indexRefresh() IndexRefresh {
	l := indexLabel()
	tm := l.To.Add(time.Hour)
	return IndexRefresh{Backend: "drive", Locator: "pair-backup:backups/pair-history", FolderID: "folder-id", At: tm, Complete: true, Observations: []IndexObservation{{Label: l, Location: BundleLocation{Backend: "drive", Locator: "pair-backup:backups/pair-history", FolderID: "folder-id", BundleKey: l.ID + ".tar.zst.gpg", LabelKey: l.ID + ".json", Available: true, Evidence: "verified", FirstObserved: tm, LastObserved: tm, Proof: LocationProof{Method: "backend-sha256", SHA256: l.SHA256, VerifiedAt: tm}}}}}
}

func TestIndexRefreshRetainsHistoryAndOtherDestinations(t *testing.T) {
	old := LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}
	r := indexRefresh()
	first, err := ApplyIndexRefresh(old, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(old.Records) != 0 {
		t.Fatal("mutated input")
	}
	r.Locator = "second:backups/history"
	r.Observations[0].Location.Locator = r.Locator
	second, err := ApplyIndexRefresh(first, r)
	if err != nil {
		t.Fatal(err)
	}
	r.Observations = nil
	r.At = r.At.Add(time.Hour)
	third, err := ApplyIndexRefresh(second, r)
	if err != nil {
		t.Fatal(err)
	}
	record := third.Records[indexLabel().ID]
	if len(record.Locations) != 2 {
		t.Fatalf("locations: %+v", record.Locations)
	}
	for _, loc := range record.Locations {
		if loc.Available != (loc.Locator != r.Locator) || loc.Proof.Method != "backend-sha256" {
			t.Fatal("availability/history lost", loc)
		}
	}
	if len(first.Records[indexLabel().ID].Locations) != 1 {
		t.Fatal("mutated prior snapshot")
	}
}

func TestIndexRefreshIncompleteConflictAndProofRefused(t *testing.T) {
	old, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, indexRefresh())
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(old)
	for _, mutate := range []func(*IndexRefresh){
		func(r *IndexRefresh) { r.Complete = false },
		func(r *IndexRefresh) { r.Observations[0].Label.Bytes++ },
		func(r *IndexRefresh) { r.Observations[0].Location.Proof.SHA256 = strings.Repeat("a", 64) },
		func(r *IndexRefresh) { r.At = r.At.Add(-time.Hour) },
		func(r *IndexRefresh) { r.Observations = append(r.Observations, r.Observations[0]) },
	} {
		r := indexRefresh()
		mutate(&r)
		if _, err := ApplyIndexRefresh(old, r); err == nil {
			t.Fatal("accepted invalid refresh", r)
		}
		after, _ := json.Marshal(old)
		if string(before) != string(after) {
			t.Fatal("failed refresh changed old index")
		}
	}
}

func TestIndexCodecPreservesNestedFutureFields(t *testing.T) {
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, indexRefresh())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	var tree map[string]any
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	tree["future"] = map[string]any{"nested": []any{true, "value"}}
	record := tree["records"].(map[string]any)[indexLabel().ID].(map[string]any)
	record["future_record"] = "kept"
	record["label"].(map[string]any)["future_label"] = []any{float64(7)}
	for _, value := range record["locations"].(map[string]any) {
		loc := value.(map[string]any)
		loc["future_location"] = true
		loc["proof"].(map[string]any)["future_proof"] = "kept"
	}
	raw, _ = json.Marshal(tree)
	decoded, err := DecodeLabelIndex(raw)
	if err != nil {
		t.Fatal(err)
	}
	round, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(round, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tree, got) {
		t.Fatalf("unknown fields lost\n%s\n%s", raw, round)
	}
}

func TestIndexCodecRejectsKnownFieldAndIdentityMutations(t *testing.T) {
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, indexRefresh())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		`null`, `{}`, string(raw) + ` {}`,
		strings.Replace(string(raw), `"version":1`, `"version":1,"Version":1`, 1),
		strings.Replace(string(raw), `"available":true`, `"available":null`, 1),
		strings.Replace(string(raw), `"available":true`, `"available":"true"`, 1),
		strings.Replace(string(raw), `"evidence":"verified"`, `"evidence":"copied"`, 1),
		strings.Replace(string(raw), `"method":"backend-sha256"`, `"method":"copied"`, 1),
		strings.Replace(string(raw), `"folder_id":"folder-id"`, `"folder_id":""`, 1),
	} {
		if _, err := DecodeLabelIndex([]byte(bad)); err == nil {
			t.Fatal("accepted invalid snapshot", bad)
		}
	}
}

func TestIndexRefreshExtensionConflictsAndMissingHashes(t *testing.T) {
	r := indexRefresh()
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	r.At = r.At.Add(time.Hour)
	r.Observations[0].Location.LastObserved = r.At
	r.Observations[0].Location.Evidence = "observed"
	r.Observations[0].Location.Proof = LocationProof{}
	r.Observations[0].Label.Extensions = map[string]json.RawMessage{"future": json.RawMessage(`{"keep":true}`)}
	next, err := ApplyIndexRefresh(i, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, loc := range next.Records[indexLabel().ID].Locations {
		if loc.Evidence != "observed" || loc.Proof.Method != "backend-sha256" {
			t.Fatal("missing hash erased history or promoted observation", loc)
		}
	}
	r.Observations[0].Label.Extensions["future"] = json.RawMessage(`{"keep":false}`)
	if _, err := ApplyIndexRefresh(next, r); err == nil {
		t.Fatal("conflicting future label field silently discarded")
	}
}

func FuzzLabelIndexCodec(f *testing.F) {
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, indexRefresh())
	if err != nil {
		f.Fatal(err)
	}
	raw, err := json.Marshal(i)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(raw)
	f.Add([]byte(`{"version":1,"records":{}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > MaxMetadataBytes {
			t.Skip()
		}
		i, err := DecodeLabelIndex(raw)
		if err != nil {
			return
		}
		out, err := json.Marshal(i)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeLabelIndex(out); err != nil {
			t.Fatal(err)
		}
	})
}
