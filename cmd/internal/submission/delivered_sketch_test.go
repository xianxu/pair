package submission

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sketchVersion(s Sketch) PreviewItem {
	return PreviewItem{RecordingID: s.RecordingID, SketchID: s.ID, Sequence: s.NextSequence, Digest: s.Digest}
}
func TestDeliveredSketchVersionsFollowDurableLifecycleAcrossTargets(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := sketchSubmission(t)
	files := map[string][]byte{"sketch-0.png": sketchPNG(t, sub.Sketches[0])}
	if _, err = store.Prepare(sub, files); err != nil {
		t.Fatal(err)
	}
	key := sketchVersion(sub.Sketches[0])
	check := func(want ...PreviewItem) {
		t.Helper()
		got, err := store.DeliveredSketchVersions("rec")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("versions=%v want=%v", got, want)
		}
		for _, key := range want {
			if _, ok := got[key]; !ok {
				t.Fatalf("missing exact version %v", key)
			}
		}
	}
	check()
	if _, err = store.Transition(sub.ID, sub.Target, Begin); err != nil {
		t.Fatal(err)
	}
	check()
	if _, err = store.Transition(sub.ID, sub.Target, Ack); err != nil {
		t.Fatal(err)
	}
	check(key)
	if _, err = store.Transition(sub.ID, sub.Target, Complete); err != nil {
		t.Fatal(err)
	}
	check(key)
	next := sketchSubmission(t)
	next.ID = "next"
	next.Target.Session = "different-target"
	next.Sketches[0].NextSequence++
	next.Sketches[0].Digest = strings.Repeat("b", 64)
	if _, err = store.Prepare(next, files); err != nil {
		t.Fatal(err)
	}
	check(key)
	// An older delivered prefix must never suppress a later edited version.
	nextKey := sketchVersion(next.Sketches[0])
	got, err := store.DeliveredSketchVersions("rec")
	if err != nil {
		t.Fatal(err)
	}
	if _, suppressed := got[nextKey]; suppressed {
		t.Fatal("later version prematurely suppressed")
	}
	if _, err = store.Transition(next.ID, next.Target, Begin); err != nil {
		t.Fatal(err)
	}
	check(key)
	if _, err = store.Transition(next.ID, next.Target, Ack); err != nil {
		t.Fatal(err)
	}
	check(key, nextKey)
	other := sketchSubmission(t)
	other.ID = "other"
	other.Target.Session = "third-target"
	other.Source.RecordingID = "another-recording"
	other.Sketches[0].RecordingID = other.Source.RecordingID
	if _, err = store.Prepare(other, files); err != nil {
		t.Fatal(err)
	}
	for _, event := range []Event{Begin, Ack} {
		if _, err = store.Transition(other.ID, other.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	check(key, nextKey)
}
func TestDeliveredSketchVersionsRefuseUnconfirmedOrMalformedMetadata(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := sketchSubmission(t)
	files := map[string][]byte{"sketch-0.png": sketchPNG(t, sub.Sketches[0])}
	if _, err = store.Prepare(sub, files); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Transition(sub.ID, sub.Target, Begin); err != nil {
		t.Fatal(err)
	}
	store.fail = func(op string) error {
		if strings.HasPrefix(op, "sync-dir:") {
			return errors.New("durability unavailable")
		}
		return nil
	}
	if _, err = store.Transition(sub.ID, sub.Target, Ack); err == nil {
		t.Fatal("injected publication failure missing")
	}
	if versions, err := store.DeliveredSketchVersions("rec"); err == nil || versions != nil {
		t.Fatalf("visible-but-unconfirmed ACK suppressed: %v %v", versions, err)
	}
	store.fail = nil
	if versions, err := store.DeliveredSketchVersions("rec"); err != nil || len(versions) != 1 {
		t.Fatalf("confirmation failed: %v %v", versions, err)
	}
	if err = os.WriteFile(store.ManifestPath(sub.ID), []byte(`{"version":3,"id":"sample","sketches":"bad"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if versions, err := store.DeliveredSketchVersions("rec"); err == nil || versions != nil {
		t.Fatalf("malformed metadata ignored: %v %v", versions, err)
	}
}
func TestDeliveredSketchVersionsRefuseBundleSymlink(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(t.TempDir(), filepath.Join(store.root, "pretend-bundle")); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DeliveredSketchVersions("rec"); err == nil {
		t.Fatal("symlink bundle ignored")
	}
}
