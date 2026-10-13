package submission

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture() Submission {
	return Submission{Version: 1, ID: "sample", Target: Target{Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce"}, Source: Source{Kind: "dom", RecordingID: "rec", RecordingPath: "/recording", TabID: "tab", Document: 1}, Notes: []Note{{ID: "note", Crop: Crop{Unavailable: "page changed"}}}, Delivery: Prepared}
}
func TestDelivery(t *testing.T) {
	s := Prepared
	for _, e := range []Event{Begin, Begin, Ack, Ack, Complete, Complete} {
		var err error
		s, err = ApplyDelivery(s, e)
		if err != nil {
			t.Fatal(err)
		}
	}
	if s != Resolved {
		t.Fatal(s)
	}
	for _, s := range []Delivery{Prepared, Delivering, Dispatched, Resolved} {
		if _, err := ApplyDelivery(s, Event("invalid")); err == nil {
			t.Fatal("invalid event")
		}
	}
	if _, err := ApplyDelivery(Prepared, Ack); err == nil {
		t.Fatal("skipped begin")
	}
	if _, err := ApplyDelivery(Dispatched, Begin); err == nil {
		t.Fatal("backwards")
	}
}
func TestValidation(t *testing.T) {
	for _, mutate := range []func(*Submission){func(s *Submission) { s.Version = 2 }, func(s *Submission) { s.ID = "../escape" }, func(s *Submission) { s.Source.RecordingPath = "relative" }, func(s *Submission) { s.Notes[0].Crop.File = "../x.png" }, func(s *Submission) { s.Notes[0].Crop = Crop{} }, func(s *Submission) { s.Notes[0].At.Box = &recording.Box{0, 0, math.Inf(1), 1} }} {
		s := fixture()
		mutate(&s)
		if Validate(s) == nil {
			t.Fatalf("accepted %#v", s)
		}
	}
	for _, b := range []recording.Box{{0, 0, math.NaN(), 1}, {0, 0, 0, 1}, {0, 0, 1e12, 1}, {-1, 0, 1, 1}} {
		if ValidateBox(b) == nil {
			t.Fatal(b)
		}
	}
	if err := Validate(fixture()); err != nil {
		t.Fatal(err)
	}
}
func pngBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestPNG(t *testing.T) {
	if err := ValidatePNG(pngBytes(t)); err != nil {
		t.Fatal(err)
	}
	for _, b := range [][]byte{[]byte("no"), make([]byte, MaxPNGBytes+1)} {
		if ValidatePNG(b) == nil {
			t.Fatal("accepted invalid png")
		}
	}
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2049, 2048)))
	if ValidatePNG(b.Bytes()) == nil {
		t.Fatal("accepted oversized dimensions")
	}
}
func TestStoreLifecycle(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	s := fixture()
	s.Notes[0].Crop = Crop{File: "crop-0.png", Box: recording.Box{0, 0, 2, 2}, CapturedAt: time.Now(), Document: 1}
	p, err := store.Prepare(s, map[string][]byte{"crop-0.png": pngBytes(t)})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(p) {
		t.Fatal(p)
	}
	for _, p := range []string{p, filepath.Join(filepath.Dir(p), "crop-0.png")} {
		st, err := os.Stat(p)
		if err != nil || st.Mode().Perm() != 0600 {
			t.Fatalf("permissions %v %v", st, err)
		}
	}
	if _, err = store.Prepare(s, map[string][]byte{"crop-0.png": pngBytes(t)}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Transition(s.ID, s.Target, Begin); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewStore(store.root)
	if err != nil {
		t.Fatal(err)
	}
	next := fixture()
	next.ID = "next"
	if _, err = reopened.Prepare(next, nil); err != nil {
		t.Fatal("uncertain delivery blocked fresh submission", err)
	}
	wrong := s.Target
	wrong.Nonce = "wrong"
	if _, err = store.Transition(s.ID, wrong, Ack); err == nil {
		t.Fatal("wrong target")
	}
	for _, event := range []Event{Ack, Ack, Complete, Complete} {
		if _, err = store.Transition(s.ID, s.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.Read(s.ID)
	if err != nil || got.Delivery != Resolved {
		t.Fatalf("%v %v", got, err)
	}
}
func TestStoreIgnoresForeignEntries(t *testing.T) {
	for _, tc := range []struct {
		name      string
		directory bool
	}{
		{".DS_Store", false},
		{".metadata", false},
		{"notes.txt", false},
		{"valid-id", false},
		{".hidden", true},
		{"not a submission", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, err := NewStore(tempRoot(t))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(store.root, tc.name)
			if tc.directory {
				err = os.Mkdir(path, 0700)
			} else {
				err = os.WriteFile(path, []byte("foreign metadata"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			sub := fixture()
			if _, err = store.Prepare(sub, nil); err != nil {
				t.Fatalf("foreign entry blocked submission: %v", err)
			}
			pending, err := store.Pending(sub.Target)
			if err != nil || len(pending) != 1 || pending[0].ID != sub.ID {
				t.Fatalf("pending = %+v, err = %v", pending, err)
			}
			if _, err = os.Stat(path); err != nil {
				t.Fatal("foreign entry removed", err)
			}
		})
	}
}

func TestStorePendingRejectsCorruptSubmissionDirectory(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(store.root, "broken-submission"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Pending(fixture().Target); err == nil {
		t.Fatal("corrupt submission directory silently ignored")
	}
	if _, err = store.Prepare(fixture(), nil); err == nil {
		t.Fatal("corrupt submission directory allowed admission")
	}
}

func TestStoreFailuresAndAdmission(t *testing.T) {
	for _, point := range []string{"write", "sync", "rename"} {
		t.Run(point, func(t *testing.T) {
			store, _ := NewStore(tempRoot(t))
			store.fail = func(op string) error {
				if op == point {
					return errors.New("injected")
				}
				return nil
			}
			if _, err := store.Prepare(fixture(), nil); err == nil {
				t.Fatal("expected failure")
			}
			entries, _ := os.ReadDir(store.root)
			if len(entries) != 0 {
				t.Fatal(entries)
			}
		})
	}
	store, _ := NewStore(tempRoot(t))
	store.capacity = 10
	os.WriteFile(filepath.Join(store.root, ".staging-crashed"), make([]byte, 10), 0600)
	if _, err := store.Prepare(fixture(), nil); err == nil {
		t.Fatal("ignored crash residue")
	}
}
func TestStoreTransitionAtomicAndCorrupt(t *testing.T) {
	store, _ := NewStore(tempRoot(t))
	s := fixture()
	p, err := store.Prepare(s, nil)
	if err != nil {
		t.Fatal(err)
	}
	store.fail = func(op string) error {
		if op == "rename" {
			got, e := store.Read(s.ID)
			if e != nil || got.Delivery != Prepared {
				t.Fatalf("partial visible: %v %v", got, e)
			}
			return errors.New("injected")
		}
		return nil
	}
	if _, err = store.Transition(s.ID, s.Target, Begin); err == nil {
		t.Fatal("expected failure")
	}
	got, err := store.Read(s.ID)
	if err != nil || got.Delivery != Prepared {
		t.Fatal(got, err)
	}
	os.WriteFile(p, []byte(`{"version":2}`), 0600)
	if _, err = store.Read(s.ID); err == nil {
		t.Fatal("corrupt manifest")
	}
	os.Remove(p)
	os.Symlink("/etc/passwd", p)
	if _, err = store.Read(s.ID); err == nil {
		t.Fatal("symlink manifest")
	}
	if _, err = NewStore(store.root + "/../bad/.."); err == nil {
		t.Fatal("unclean path")
	}
}

func tempRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStoreRefusesRebindingAndSymlinkCrop(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	path, err := store.Prepare(sub, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed := sub
	changed.Target.Nonce = "different"
	if _, err = store.Prepare(changed, nil); err == nil {
		t.Fatal("rebound target")
	}
	pending, err := store.Pending(sub.Target)
	if err != nil || len(pending) != 1 || pending[0].Delivery != Prepared {
		t.Fatal(pending, err)
	}
	sub.ID = "crop-test"
	sub.Notes[0].Crop = Crop{File: "crop-0.png", Box: recording.Box{0, 0, 2, 2}, CapturedAt: time.Now(), Document: 1}
	path, err = store.Prepare(sub, map[string][]byte{"crop-0.png": pngBytes(t)})
	if err != nil {
		t.Fatal(err)
	}
	crop := filepath.Join(filepath.Dir(path), "crop-0.png")
	if err = os.Remove(crop); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("/etc/passwd", crop); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Read(sub.ID); err == nil {
		t.Fatal("followed crop symlink")
	}
	if _, err = store.Pending(sub.Target); err == nil {
		t.Fatal("ignored corrupt pending bundle")
	}
}
func TestStoreTransitionWriteFailuresLeaveOldManifest(t *testing.T) {
	for _, point := range []string{"write", "sync", "rename"} {
		t.Run(point, func(t *testing.T) {
			store, err := NewStore(tempRoot(t))
			if err != nil {
				t.Fatal(err)
			}
			sub := fixture()
			if _, err = store.Prepare(sub, nil); err != nil {
				t.Fatal(err)
			}
			store.fail = func(op string) error {
				if op == point {
					return errors.New("injected")
				}
				return nil
			}
			if _, err = store.Transition(sub.ID, sub.Target, Begin); err == nil {
				t.Fatal("failure ignored")
			}
			got, err := store.Read(sub.ID)
			if err != nil || got.Delivery != Prepared {
				t.Fatal(got, err)
			}
			entries, err := os.ReadDir(store.root)
			if err != nil || len(entries) != 1 {
				t.Fatal(entries, err)
			}
		})
	}
}

func TestStoreCanonicalizesParentOnly(t *testing.T) {
	root := tempRoot(t)
	link := filepath.Join(root, "alias")
	actual := filepath.Join(root, "actual")
	if err := os.Mkdir(actual, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(actual, link); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(filepath.Join(link, "bundles"))
	if err != nil {
		t.Fatal(err)
	}
	if store.root != filepath.Join(actual, "bundles") {
		t.Fatal(store.root)
	}
	if _, err = NewStore(link); err == nil {
		t.Fatal("followed symlink root")
	}
}

func TestStoreMetadataDoesNotDecodePixels(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	sub.Notes[0].Crop = Crop{File: "crop-0.png", Box: recording.Box{0, 0, 2, 2}, CapturedAt: time.Now(), Document: 1}
	data := pngBytes(t)
	path, err := store.Prepare(sub, map[string][]byte{"crop-0.png": data})
	if err != nil {
		t.Fatal(err)
	}
	crop := filepath.Join(filepath.Dir(path), "crop-0.png")
	if err = os.WriteFile(crop, bytes.Repeat([]byte{'x'}, len(data)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Read(sub.ID); err == nil {
		t.Fatal("full Read accepted corrupt pixels")
	}
	if _, err = store.ReadMetadata(sub.ID); err != nil {
		t.Fatal("metadata read decoded pixels", err)
	}
	if pending, err := store.Pending(sub.Target); err != nil || len(pending) != 1 {
		t.Fatal("pending decoded pixels", pending, err)
	}
	if _, err = store.Prepare(sub, map[string][]byte{"crop-0.png": data}); err == nil {
		t.Fatal("Prepare skipped full evidence validation")
	}
	for _, event := range []Event{Begin, Ack, Complete} {
		if _, err = store.Transition(sub.ID, sub.Target, event); err != nil {
			t.Fatal("transition decoded pixels", err)
		}
	}
	if err = os.Remove(crop); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadMetadata(sub.ID); err == nil {
		t.Fatal("explicit metadata read ignored missing resolved crop")
	}
	if pending, err := store.Pending(sub.Target); err != nil || len(pending) != 0 {
		t.Fatal("pending examined resolved crop", pending, err)
	}
}
func TestStoreMetadataRejectsOversizeAndSymlink(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	sub.Notes[0].Crop = Crop{File: "crop-0.png", Box: recording.Box{0, 0, 2, 2}, CapturedAt: time.Now(), Document: 1}
	path, err := store.Prepare(sub, map[string][]byte{"crop-0.png": pngBytes(t)})
	if err != nil {
		t.Fatal(err)
	}
	crop := filepath.Join(filepath.Dir(path), "crop-0.png")
	if err = os.Truncate(crop, MaxPNGBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadMetadata(sub.ID); err == nil {
		t.Fatal("metadata ignored crop size")
	}
	if err = os.Remove(crop); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("/etc/passwd", crop); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadMetadata(sub.ID); err == nil {
		t.Fatal("metadata followed crop symlink")
	}
}

func TestStoreMetadataBoundsAggregateCropSizes(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	sub.Notes = nil
	crops := map[string][]byte{}
	for i := 0; i < 9; i++ {
		name := fmt.Sprintf("crop-%d.png", i)
		sub.Notes = append(sub.Notes, Note{ID: fmt.Sprintf("note-%d", i), Crop: Crop{File: name, Box: recording.Box{0, 0, 2, 2}, CapturedAt: time.Now(), Document: 1}})
		crops[name] = pngBytes(t)
	}
	path, err := store.Prepare(sub, crops)
	if err != nil {
		t.Fatal(err)
	}
	for name := range crops {
		if err = os.Truncate(filepath.Join(filepath.Dir(path), name), MaxPNGBytes); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = store.ReadMetadata(sub.ID); err == nil {
		t.Fatal("metadata accepted aggregate size over 64 MiB")
	}
}

// A rename is visible before directory fsync confirms durability. Neither
// reopening the Store nor an idempotent request may promote that visibility
// to a successful durability acknowledgment while fsync still fails.
func TestStorePrepareRetriesPublicationSync(t *testing.T) {
	for _, dir := range []string{"sample", "."} {
		t.Run(dir, func(t *testing.T) {
			root := tempRoot(t)
			sub := fixture()
			attempts := 0
			fail := func(op string) error {
				if op == "sync-dir:"+dir {
					attempts++
					return errors.New("directory sync unavailable")
				}
				return nil
			}
			for attempt := 0; attempt < 3; attempt++ {
				store, err := NewStore(root)
				if err != nil {
					t.Fatal(err)
				}
				store.fail = fail
				if _, err = store.Prepare(sub, nil); err == nil {
					t.Fatalf("attempt %d acknowledged publication despite failed directory sync", attempt)
				}
				got, err := store.ReadMetadata(sub.ID)
				if err != nil || got.Delivery != Prepared {
					t.Fatalf("rename must be visible and complete: %v %v", got, err)
				}
			}
			if attempts != 3 {
				t.Fatalf("directory sync attempts = %d, want 3", attempts)
			}
			store, err := NewStore(root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.Prepare(sub, nil); err != nil {
				t.Fatal("successful sync must recover preparation", err)
			}
		})
	}
}
func TestStoreTransitionRetriesPublicationSync(t *testing.T) {
	for _, event := range []Event{Begin, Ack, Complete} {
		for _, dir := range []string{"sample", "."} {
			t.Run(string(event)+"/"+dir, func(t *testing.T) {
				root := tempRoot(t)
				store, err := NewStore(root)
				if err != nil {
					t.Fatal(err)
				}
				sub := fixture()
				if _, err = store.Prepare(sub, nil); err != nil {
					t.Fatal(err)
				}
				for _, prior := range []Event{Begin, Ack, Complete} {
					if prior == event {
						break
					}
					if _, err = store.Transition(sub.ID, sub.Target, prior); err != nil {
						t.Fatal(err)
					}
				}
				attempts := 0
				fail := func(op string) error {
					if op == "sync-dir:"+dir {
						attempts++
						return errors.New("directory sync unavailable")
					}
					return nil
				}
				for attempt := 0; attempt < 3; attempt++ {
					store, err = NewStore(root)
					if err != nil {
						t.Fatal(err)
					}
					store.fail = fail
					if _, err = store.Transition(sub.ID, sub.Target, event); err == nil {
						t.Fatalf("attempt %d acknowledged %s despite failed directory sync", attempt, event)
					}
					got, err := store.ReadMetadata(sub.ID)
					if err != nil {
						t.Fatal(err)
					}
					want := map[Event]Delivery{Begin: Delivering, Ack: Dispatched, Complete: Resolved}[event]
					if got.Delivery != want {
						t.Fatalf("rename must be visible: got %s, want %s", got.Delivery, want)
					}
				}
				if attempts != 3 {
					t.Fatalf("directory sync attempts = %d, want 3", attempts)
				}
				store, err = NewStore(root)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = store.Transition(sub.ID, sub.Target, event); err != nil {
					t.Fatal("successful sync must recover transition", err)
				}
			})
		}
	}
}
func TestStoreAdvancingAndPrepareRetryConfirmEarlierPublication(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := fixture()
	store.fail = func(op string) error {
		if op == "sync-dir:." {
			return errors.New("root sync unavailable")
		}
		return nil
	}
	if _, err = store.Prepare(sub, nil); err == nil {
		t.Fatal("preparation ignored failed root sync")
	}
	if _, err = store.Transition(sub.ID, sub.Target, Begin); err == nil {
		t.Fatal("advancing state bypassed earlier failed bundle publication")
	}
	store.fail = func(op string) error {
		if op == "sync-dir:"+sub.ID {
			return errors.New("bundle sync unavailable")
		}
		return nil
	}
	if _, err = store.Prepare(sub, nil); err == nil {
		t.Fatal("existing preparation bypassed failed transition publication")
	}
	store.fail = nil
	if _, err = store.Prepare(sub, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Transition(sub.ID, sub.Target, Begin); err != nil {
		t.Fatal(err)
	}
}

func TestStoreResolvedAckRetriesPublicationSync(t *testing.T) {
	root := tempRoot(t)
	store, err := NewStore(root)
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
	fail := func(op string) error {
		if op == "sync-dir:"+sub.ID {
			return errors.New("directory sync unavailable")
		}
		return nil
	}
	store.fail = fail
	if _, err = store.Transition(sub.ID, sub.Target, Complete); err == nil {
		t.Fatal("complete ignored failed sync")
	}
	// Ack is also idempotent when resolution has already advanced the state.
	for attempt := 0; attempt < 2; attempt++ {
		store, err = NewStore(root)
		if err != nil {
			t.Fatal(err)
		}
		store.fail = fail
		if _, err = store.Transition(sub.ID, sub.Target, Ack); err == nil {
			t.Fatal("resolved ack bypassed failed completion sync")
		}
	}
	store.fail = nil
	got, err := store.Transition(sub.ID, sub.Target, Ack)
	if err != nil || got.Delivery != Resolved {
		t.Fatal(got, err)
	}
}

func TestStoreFreshSubmissionSupersedesWithoutInventingDelivery(t *testing.T) {
	for _, begun := range []bool{false, true} {
		t.Run(fmt.Sprint(begun), func(t *testing.T) {
			store, err := NewStore(tempRoot(t))
			if err != nil {
				t.Fatal(err)
			}
			old := fixture()
			if _, err = store.Prepare(old, nil); err != nil {
				t.Fatal(err)
			}
			if begun {
				if _, err = store.Transition(old.ID, old.Target, Begin); err != nil {
					t.Fatal(err)
				}
			}
			other := fixture()
			other.ID = "other"
			other.Target.Nonce = "other"
			if _, err = store.Prepare(other, nil); err != nil {
				t.Fatal(err)
			}
			fresh := fixture()
			fresh.ID = "fresh"
			if _, err = store.Prepare(fresh, nil); err != nil {
				t.Fatal("fresh submission blocked", err)
			}
			store, err = NewStore(store.root)
			if err != nil {
				t.Fatal(err)
			}
			pending, err := store.Pending(old.Target)
			if err != nil || len(pending) != 1 || pending[0].ID != fresh.ID {
				t.Fatalf("pending=%+v err=%v", pending, err)
			}
			saved, err := store.Read(old.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := Prepared
			if begun {
				want = Delivering
			}
			if saved.Delivery != want {
				t.Fatalf("invented delivery: %s", saved.Delivery)
			}
			if _, err = store.Transition(old.ID, old.Target, Begin); err == nil {
				t.Fatal("superseded begin accepted")
			}
			if begun {
				for _, event := range []Event{Ack, Ack, Complete, Complete} {
					if _, err = store.Transition(old.ID, old.Target, event); err != nil {
						t.Fatal(err)
					}
				}
			}
			if pending, err = store.Pending(other.Target); err != nil || len(pending) != 1 {
				t.Fatalf("changed other target: %+v %v", pending, err)
			}
			// Retrying an older immutable preparation cannot revoke its successor.
			if _, err = store.Prepare(old, nil); err != nil {
				t.Fatal(err)
			}
			if pending, err = store.Pending(fresh.Target); err != nil || len(pending) != 1 || pending[0].ID != fresh.ID {
				t.Fatalf("retry revoked successor: %+v %v", pending, err)
			}
		})
	}
}

func TestDispositionPreservesDeliveryTruth(t *testing.T) {
	for _, delivery := range []Delivery{Prepared, Delivering, Dispatched, Resolved} {
		for _, superseded := range []bool{false, true} {
			for _, event := range []Event{Supersede, Begin, Ack, Complete, Event("invalid")} {
				t.Run(fmt.Sprintf("%s/%v/%s", delivery, superseded, event), func(t *testing.T) {
					next, marked, err := ApplyDisposition(delivery, superseded, event)
					if event == Supersede {
						eligible := delivery == Prepared || delivery == Delivering
						if (err == nil) != eligible || next != delivery || marked != (superseded || eligible) {
							t.Fatalf("%s %v %v", next, marked, err)
						}
					} else if event == Begin && superseded {
						if err == nil || next != delivery || !marked {
							t.Fatalf("superseded begin: %s %v %v", next, marked, err)
						}
					} else {
						want, wantErr := ApplyDelivery(delivery, event)
						if (err == nil) != (wantErr == nil) || next != want || marked != superseded {
							t.Fatalf("changed delivery rules: %s %v %v", next, marked, err)
						}
					}
				})
			}
		}
	}
}

func TestStoreSupersessionPublicationFailures(t *testing.T) {
	for _, tc := range []struct {
		point      string
		occurrence int
	}{
		{"write", 1}, {"sync", 1}, {"staging-sync", 1}, // successor staging
		{"write", 2}, {"sync", 2}, {"rename", 1}, {"sync-dir:sample", 1}, {"sync-dir:.", 1}, // old marker
		{"rename", 2}, {"sync-dir:fresh", 1}, {"sync-dir:.", 2}, // successor publication
	} {
		t.Run(fmt.Sprintf("%s/%d", tc.point, tc.occurrence), func(t *testing.T) {
			store, err := NewStore(tempRoot(t))
			if err != nil {
				t.Fatal(err)
			}
			old := fixture()
			if _, err = store.Prepare(old, nil); err != nil {
				t.Fatal(err)
			}
			if _, err = store.Transition(old.ID, old.Target, Begin); err != nil {
				t.Fatal(err)
			}
			seen := 0
			store.fail = func(op string) error {
				matches := op == tc.point || (tc.point == "staging-sync" && strings.HasPrefix(op, "sync-dir:.staging-"))
				if matches {
					seen++
					if seen == tc.occurrence {
						return errors.New("injected publication failure")
					}
				}
				return nil
			}
			fresh := fixture()
			fresh.ID = "fresh"
			if _, err = store.Prepare(fresh, nil); err == nil {
				t.Fatal("failure ignored")
			}
			oldSaved, err := store.Read(old.ID)
			if err != nil || oldSaved.Delivery != Delivering {
				t.Fatalf("old evidence changed: %+v %v", oldSaved, err)
			}
			if _, err = store.Read(fresh.ID); err == nil && !oldSaved.Superseded {
				t.Fatal("successor visible before supersession")
			}
			store, err = NewStore(store.root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.Prepare(fresh, nil); err != nil {
				t.Fatal("restart retry", err)
			}
			pending, err := store.Pending(old.Target)
			if err != nil || len(pending) != 1 || pending[0].ID != fresh.ID {
				t.Fatalf("pending after retry: %+v %v", pending, err)
			}
			oldSaved, err = store.Read(old.ID)
			if err != nil || !oldSaved.Superseded || oldSaved.Delivery != Delivering {
				t.Fatalf("old evidence after retry: %+v %v", oldSaved, err)
			}
		})
	}
}

func TestStoreSupersessionRetriesMarkerSync(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	old := fixture()
	if _, err = store.Prepare(old, nil); err != nil {
		t.Fatal(err)
	}
	fresh := fixture()
	fresh.ID = "fresh"
	for attempt := 0; attempt < 3; attempt++ {
		store, err = NewStore(store.root)
		if err != nil {
			t.Fatal(err)
		}
		store.fail = func(op string) error {
			if op == "sync-dir:sample" {
				return errors.New("marker sync unavailable")
			}
			return nil
		}
		if _, err = store.Prepare(fresh, nil); err == nil {
			t.Fatal("acknowledged unsynced marker")
		}
		if _, err = store.Read(fresh.ID); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("published successor before durable marker: %v", err)
		}
	}
	store.fail = nil
	if _, err = store.Prepare(fresh, nil); err != nil {
		t.Fatal(err)
	}
}
