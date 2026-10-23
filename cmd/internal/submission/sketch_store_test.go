package submission

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sketchPNG(t *testing.T, s Sketch) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, s.Image.Plan.Width, s.Image.Plan.Height))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestStoreSketchEvidenceLifecycleAndFileInventory(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := sketchSubmission(t)
	sub.Notes = []Note{{PrivacyDecision: (OriginPolicy{}).Decide("https://example.test/note"), Page: &PageIdentity{URL: "https://example.test/note", Provenance: "cdp"}, CaptureID: "capture", SourceDocument: 1, ID: "note", Crop: Crop{File: "crop-0.png", Box: [4]float64{0, 0, 2, 2}, CapturedAt: time.Now().UTC(), Document: 1}}}
	files := map[string][]byte{"sketch-0.png": sketchPNG(t, sub.Sketches[0]), "crop-0.png": pngBytes(t)}
	if _, err = store.Prepare(sub, files); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ConfirmPrepared(sub, files); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Prepare(sub, files); err != nil {
		t.Fatal("idempotent prepare", err)
	}
	for _, event := range []Event{Begin, Ack, Complete} {
		if _, err = store.Transition(sub.ID, sub.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	read, err := store.Read(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Delivery != Resolved || read.Sketches[0].Image.Plan.Paths[0].Pts[0][2] != 0.2 {
		t.Fatalf("lost sketch/delivery: %#v", read)
	}
	if err = os.Remove(filepath.Join(filepath.Dir(store.ManifestPath(sub.ID)), "sketch-0.png")); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadMetadata(sub.ID); err == nil {
		t.Fatal("metadata ignored missing sketch image")
	}
}
func TestStoreUnknownSketchFieldsSurviveEveryDurableRewrite(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub, err := DecodePreserving(extensionManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"sketch-0.png": sketchPNG(t, sub.Sketches[0])}
	if _, err = store.Prepare(sub, files); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ConfirmPrepared(sub, files); err != nil {
		t.Fatal(err)
	}
	for _, event := range []Event{Begin, Ack, Complete} {
		if _, err = store.Transition(sub.ID, sub.Target, event); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(store.ManifestPath(sub.ID))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(raw, []byte("90071992547409931234567890")) || !bytes.Contains(raw, []byte("future_anchor")) {
			t.Fatalf("rewrite lost extension: %s", raw)
		}
	}
	if _, err = store.Read(sub.ID); err != nil {
		t.Fatal(err)
	}
}
func TestStoreSketchPNGMatchesDeclaredTransform(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := sketchSubmission(t)
	if _, err = store.Prepare(sub, map[string][]byte{"sketch-0.png": pngBytes(t)}); err == nil {
		t.Fatal("accepted PNG dimensions different from sketch transform")
	}
}
func TestSketchManifestLegacyDecoderFailsClosed(t *testing.T) {
	data, err := json.Marshal(sketchSubmission(t))
	if err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(data)
	// Frozen pre-sketch version-3 root contract. It knows every former root field
	// but cannot consume the additive sketch evidence, so it cannot partially ACK.
	var legacy struct {
		PrivacyDecision
		HowToRead  string   `json:"how_to_read"`
		Version    int      `json:"version"`
		ID         string   `json:"id"`
		Target     Target   `json:"target"`
		Source     Source   `json:"source"`
		Notes      []Note   `json:"notes"`
		Delivery   Delivery `json:"delivery"`
		Superseded bool     `json:"superseded,omitempty"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&legacy); err == nil || !strings.Contains(err.Error(), "sketches") {
		t.Fatalf("old decoder did not reject sketches: %v", err)
	}
	if !bytes.Equal(original, data) {
		t.Fatal("old decoder changed original evidence")
	}
}

func TestStoreSketchImageUsesExistingConfinementAndImmutability(t *testing.T) {
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	sub := sketchSubmission(t)
	files := map[string][]byte{"sketch-0.png": sketchPNG(t, sub.Sketches[0])}
	if _, err = store.Prepare(sub, files); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(filepath.Dir(store.ManifestPath(sub.ID)), "sketch-0.png")
	if err = os.Remove(imagePath); err != nil {
		t.Fatal(err)
	}
	// A private regular file elsewhere is still not an admitted evidence file.
	target := filepath.Join(t.TempDir(), "outside.png")
	if err = os.WriteFile(target, files["sketch-0.png"], 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(target, imagePath); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadMetadata(sub.ID); err == nil {
		t.Fatal("symlinked image accepted by metadata")
	}
	if _, err = store.Read(sub.ID); err == nil {
		t.Fatal("symlinked image accepted by read")
	}
	if _, err = store.ConfirmPrepared(sub, files); err == nil {
		t.Fatal("symlinked image confirmed")
	}
	if err = os.Remove(imagePath); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(imagePath, files["sketch-0.png"], 0600); err != nil {
		t.Fatal(err)
	}
	changed := sub
	changed.Sketches = append([]Sketch(nil), sub.Sketches...)
	changed.Sketches[0].Note = "changed immutable note"
	if _, err = store.Prepare(changed, files); err == nil {
		t.Fatal("same ID replaced immutable sketch note")
	}
	if _, err = store.ConfirmPrepared(changed, files); err == nil {
		t.Fatal("changed immutable sketch note confirmed")
	}
}
