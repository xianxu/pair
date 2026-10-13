package submission

import (
	"bytes"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
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
	if _, err = reopened.Prepare(next, nil); err == nil {
		t.Fatal("uncertain delivery allowed duplicate")
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
