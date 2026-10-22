package submission

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeliveredFeedbackIncludesOnlyConfirmedExactVersionsAcrossTargets(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	sub.Notes[0].Revision = 3
	if _, err = store.Prepare(sub, nil); err != nil {
		t.Fatal(err)
	}
	check := func(notes, sketches int) DeliveredFeedback {
		t.Helper()
		got, err := store.DeliveredFeedbackVersions("rec")
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Notes) != notes || len(got.Sketches) != sketches {
			t.Fatalf("got %+v want notes=%d sketches=%d", got, notes, sketches)
		}
		return got
	}
	check(0, 0)
	for _, event := range []Event{Begin, Ack, Complete} {
		if _, err = store.Transition(sub.ID, sub.Target, event); err != nil {
			t.Fatal(err)
		}
		want := 1
		if event == Begin {
			want = 0
		}
		check(want, 0)
	}
	if _, ok := check(1, 0).Notes[PreviewNote{ID: "note", Revision: 3}]; !ok {
		t.Fatal("exact note revision absent")
	}
	mixed := sketchSubmission(t)
	mixed.ID = "mixed"
	mixed.Target.Session = "other-target"
	mixed.Notes = []Note{{PrivacyDecision: (OriginPolicy{}).Decide(""), SourceDocument: 1, ID: "note", Revision: 4, Crop: Crop{Unavailable: "page changed"}}}
	files := map[string][]byte{"sketch-0.png": sketchPNG(t, mixed.Sketches[0])}
	if _, err = store.Prepare(mixed, files); err != nil {
		t.Fatal(err)
	}
	check(1, 0)
	for _, event := range []Event{Begin, Ack} {
		if _, err = store.Transition(mixed.ID, mixed.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	got := check(2, 1)
	if _, ok := got.Notes[PreviewNote{ID: "note", Revision: 4}]; !ok {
		t.Fatal("later delivered note revision absent")
	}
	if _, ok := got.Sketches[sketchVersion(mixed.Sketches[0])]; !ok {
		t.Fatal("exact sketch version absent")
	}
	if _, ok := got.Notes[PreviewNote{ID: "note", Revision: 5}]; ok {
		t.Fatal("undelivered edit suppressed")
	}
	// Duplicate versions in another target do not inflate either set.
	duplicate := mixed
	duplicate.ID = "duplicate"
	duplicate.Target.Session = "third-target"
	if _, err = store.Prepare(duplicate, files); err != nil {
		t.Fatal(err)
	}
	for _, event := range []Event{Begin, Ack} {
		if _, err = store.Transition(duplicate.ID, duplicate.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	check(2, 1)
	// A different recording may reuse exact note IDs/revisions without leaking.
	other := fixture()
	other.ID = "other"
	other.Source.RecordingID = "another-recording"
	other.Notes[0].Revision = 90
	if _, err = store.Prepare(other, nil); err != nil {
		t.Fatal(err)
	}
	for _, event := range []Event{Begin, Ack} {
		if _, err = store.Transition(other.ID, other.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	check(2, 1)
	// Returned sets are owned by the caller; mutating them changes no authority.
	got = check(2, 1)
	clear(got.Notes)
	clear(got.Sketches)
	got.Notes[PreviewNote{ID: "forged", Revision: 9}] = struct{}{}
	check(2, 1)
	old, err := store.DeliveredSketchVersions("rec")
	if err != nil || len(old) != 1 {
		t.Fatalf("compatibility wrapper %v %v", old, err)
	}
}
func TestDeliveredFeedbackRefusesUncertainNotesOnlyAckAndMalformedMetadata(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	if _, err = store.Prepare(sub, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Transition(sub.ID, sub.Target, Begin); err != nil {
		t.Fatal(err)
	}
	store.fail = func(op string) error {
		if strings.HasPrefix(op, "sync-dir:") {
			return errors.New("uncertain durability")
		}
		return nil
	}
	if _, err = store.Transition(sub.ID, sub.Target, Ack); err == nil {
		t.Fatal("injected ACK error absent")
	}
	if got, err := store.DeliveredFeedbackVersions("rec"); err == nil || got.RecordingID != "" || got.Notes != nil || got.Sketches != nil {
		t.Fatalf("unconfirmed notes suppressed %+v %v", got, err)
	}
	store.fail = nil
	if got, err := store.DeliveredFeedbackVersions("rec"); err != nil || len(got.Notes) != 1 {
		t.Fatalf("durable confirmation failed %+v %v", got, err)
	}
	if err = os.WriteFile(store.ManifestPath(sub.ID), []byte(`{"version":3,"notes":"bad"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := store.DeliveredFeedbackVersions("rec"); err == nil || got.RecordingID != "" || got.Notes != nil || got.Sketches != nil {
		t.Fatalf("malformed metadata ignored %+v %v", got, err)
	}
}
func TestDeliveredFeedbackRefusesSymlinkAndEmptyRecording(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.DeliveredFeedbackVersions(""); err == nil || got.RecordingID != "" || got.Notes != nil || got.Sketches != nil {
		t.Fatalf("empty recording accepted %+v %v", got, err)
	}
	if err = os.Symlink(t.TempDir(), filepath.Join(store.root, "pretend-bundle")); err != nil {
		t.Fatal(err)
	}
	if got, err := store.DeliveredFeedbackVersions("rec"); err == nil || got.RecordingID != "" || got.Notes != nil || got.Sketches != nil {
		t.Fatalf("symlink accepted %+v %v", got, err)
	}
}

func TestDeliveredFeedbackReturnsNoPartialSetsOnUnrelatedMalformedBundle(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	if _, err = store.Prepare(sub, nil); err != nil {
		t.Fatal(err)
	}
	for _, event := range []Event{Begin, Ack} {
		if _, err = store.Transition(sub.ID, sub.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	unrelated := fixture()
	unrelated.ID = "unrelated"
	unrelated.Source.RecordingID = "elsewhere"
	if _, err = store.Prepare(unrelated, nil); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(store.ManifestPath(unrelated.ID), []byte(`{"version":1,"Version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := store.DeliveredFeedbackVersions("rec"); err == nil || got.RecordingID != "" || got.Notes != nil || got.Sketches != nil {
		t.Fatalf("partial suppression escaped unrelated bad metadata %+v %v", got, err)
	}
}

func TestDeliveredFeedbackCarriesExactRecordingIdentity(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.DeliveredFeedbackVersions("rec")
	if err != nil {
		t.Fatal(err)
	}
	if got.RecordingID != "rec" {
		t.Fatalf("identity=%q", got.RecordingID)
	}
	got.RecordingID = "another-recording"
	again, err := store.DeliveredFeedbackVersions("rec")
	if err != nil || again.RecordingID != "rec" {
		t.Fatalf("caller changed derivation identity: %+v %v", again, err)
	}
	failed, err := store.DeliveredFeedbackVersions("")
	if err == nil || failed.RecordingID != "" || failed.Notes != nil || failed.Sketches != nil {
		t.Fatalf("failed derivation retained authority %+v %v", failed, err)
	}
}
