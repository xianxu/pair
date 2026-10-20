package historyarchive

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestIndexCompleteEmptyObservationSurvivesWithoutRecords(t *testing.T) {
	r := indexRefresh()
	r.FolderID = "folder-id"
	r.Observations = nil
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	key := r.Backend + "\n" + r.Locator
	marker, ok := i.CompleteDestinations[key]
	if !ok || marker.FolderID != r.FolderID || !marker.ObservedAt.Equal(r.At) {
		t.Fatal("empty inventory lost complete observation", i)
	}
	r.At = r.At.Add(-time.Nanosecond)
	if _, err := ApplyIndexRefresh(i, r); err == nil {
		t.Fatal("stale empty refresh accepted")
	}
	marker.Extensions = map[string]json.RawMessage{"future": json.RawMessage(`{"nested":true}`)}
	i.CompleteDestinations[key] = marker
	r.At = r.At.Add(time.Hour)
	r.FolderID = "new-folder"
	next, err := ApplyIndexRefresh(i, r)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(next)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"future":{"nested":true}`) {
		t.Fatal("marker extension lost", string(raw))
	}
	got, err := DecodeLabelIndex(raw)
	if err != nil || got.CompleteDestinations[key].FolderID != "new-folder" {
		t.Fatal(got, err)
	}
}

func TestIndexLegacySnapshotHasNoCompleteObservations(t *testing.T) {
	i, err := DecodeLabelIndex([]byte(`{"version":1,"records":{},"future_root":true}`))
	if err != nil || len(i.CompleteDestinations) != 0 {
		t.Fatal(i, err)
	}
	raw, err := json.Marshal(i)
	if err != nil || !strings.Contains(string(raw), `"future_root":true`) {
		t.Fatal(string(raw), err)
	}
}

func TestIndexCompleteDestinationBound(t *testing.T) {
	i := LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}, CompleteDestinations: map[string]DestinationObservation{}}
	for n := 0; n <= MaxIndexDestinations; n++ {
		locator := fmt.Sprintf("fixture%d:archive/history", n)
		d := DestinationObservation{Backend: "drive", Locator: locator, FolderID: "folder", ObservedAt: indexLabel().To}
		i.CompleteDestinations[d.Identity()] = d
	}
	if err := ValidateLabelIndex(i); err == nil {
		t.Fatal("destination bound accepted")
	}
	delete(i.CompleteDestinations, fmt.Sprintf("drive\nfixture%d:archive/history", MaxIndexDestinations))
	if err := ValidateLabelIndex(i); err != nil {
		t.Fatal("valid boundary refused", err)
	}
}

func TestIndexOptionalDestinationSchemaStaysStrict(t *testing.T) {
	for _, raw := range []string{
		`{"version":1,"records":{},"complete_destinations":null}`,
		`{"version":1,"records":{},"Complete_Destinations":{}}`,
		`{"version":1,"records":{},"complete_destinations":{"wrong":null}}`,
		`{"version":1,"records":{},"complete_destinations":{"drive\nfixture:archive/history":{"backend":"drive","locator":"fixture:archive/history","folder_id":"folder","observed_at":null}}}`,
	} {
		if _, err := DecodeLabelIndex([]byte(raw)); err == nil {
			t.Fatal("invalid optional field admitted", raw)
		}
	}
}

func TestIndexCompleteObservationRejectsContradictoryAvailability(t *testing.T) {
	r := indexRefresh()
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	r.At = r.At.Add(time.Hour)
	r.Observations = nil
	i, err = ApplyIndexRefresh(i, r)
	if err != nil {
		t.Fatal(err)
	}
	record := i.Records[indexLabel().ID]
	for key, loc := range record.Locations {
		loc.Available = true
		record.Locations[key] = loc
	}
	i.Records[indexLabel().ID] = record
	if err := ValidateLabelIndex(i); err == nil {
		t.Fatal("older reader availability contradicted preserved complete marker")
	}
}

func TestIndexOlderEnvelopePreservesNewCompleteMarkers(t *testing.T) {
	r := indexRefresh()
	r.Observations = nil
	i, err := ApplyIndexRefresh(LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	// The prior root codec recognized only these two keys. Its opaque extension
	// path must carry the new marker into a later reader without interpreting it.
	known, extensions, err := decodeExtensibleObjectLimit(raw, []string{"version", "records"}, MaxIndexBytes)
	if err != nil {
		t.Fatal(err)
	}
	base, err := json.Marshal(known)
	if err != nil {
		t.Fatal(err)
	}
	round, err := marshalExtensibleObjectLimit(base, extensions, []string{"version", "records"}, MaxIndexBytes)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeLabelIndex(round)
	if err != nil || len(got.CompleteDestinations) != 1 {
		t.Fatal("complete marker lost across older envelope", got, err)
	}
}
