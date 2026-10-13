package submission

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"strings"
	"testing"
	"time"
)

func TestReadingGuideNamesBundleAndRecordingCommands(t *testing.T) {
	path := "/recordings/a space's recording"
	guide := ReadingGuide(path)
	for _, want := range []string{"submission.json", "crop-N.png", path, "couch --broadcast-web-query DIR notes|dom|events|link", "start with notes", "dom --at <t> --sel @N", "events --from <t-10000> --to <t>", "couch --skill"} {
		if !strings.Contains(guide, want) {
			t.Errorf("guide missing %q: %s", want, guide)
		}
	}
	data, err := json.Marshal(Submission{HowToRead: guide})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["how_to_read"] != guide {
		t.Fatal("guide missing from manifest", string(data))
	}
}

func TestBoxTextLimits(t *testing.T) {
	valid := BoxText{Region: [4]float64{0, 0, 20, 20}, VisibleText: []BoxTextEntry{{Selector: "#a", Text: "abc"}}}
	if err := ValidateBoxText(valid); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BoxText){
		func(b *BoxText) { b.VisibleText[0].Text = strings.Repeat("x", MaxBoxTextBytes) },
		func(b *BoxText) {
			b.AccessibleNames = []BoxTextEntry{{Kind: "title", Text: strings.Repeat("x", MaxBoxTextBytes)}}
		},
		func(b *BoxText) { b.AccessibleNames = make([]BoxTextEntry, MaxBoxTextEntries) },
		func(b *BoxText) { b.Region[2] = -1 },
		func(b *BoxText) { b.Unavailable = "page changed" },
	} {
		b := valid
		b.VisibleText = append([]BoxTextEntry(nil), valid.VisibleText...)
		mutate(&b)
		if ValidateBoxText(b) == nil {
			t.Fatalf("accepted out of bounds: %#v", b)
		}
	}
	if err := ValidateBoxText(BoxText{Unavailable: "page changed"}); err != nil {
		t.Fatal(err)
	}
}

func TestBoxTextUnsupportedClippingIsExplicit(t *testing.T) {
	b := BoxText{Region: [4]float64{0, 0, 10, 10}, Truncated: true, Skipped: []string{"unsupported clipping"}}
	if err := ValidateBoxText(b); err != nil {
		t.Fatal(err)
	}
	b.Truncated = false
	if ValidateBoxText(b) == nil {
		t.Fatal("skipped clipping needs explicit truncation")
	}
}

func TestCropSettlementValidation(t *testing.T) {
	valid := Crop{File: "crop-0.png", Box: recording.Box{0, 0, 2, 2}, CapturedAt: time.Now(), Document: 1}
	for _, settlement := range []*CaptureSettlement{nil, {Status: "settled"}, {Status: "unsettled", Reason: "timeout"}, {Status: "unsettled", Reason: "paused animation"}, {Status: "unsettled", Reason: "infinite animation"}, {Status: "unsettled", Reason: "inspection limit"}, {Status: "unsettled", Reason: "inspection unavailable"}} {
		crop := valid
		crop.Settlement = settlement
		if err := ValidateCrop(crop); err != nil {
			t.Fatalf("valid settlement %+v: %v", settlement, err)
		}
	}
	for _, settlement := range []*CaptureSettlement{{}, {Status: "unknown"}, {Status: "settled", Reason: "timeout"}, {Status: "unsettled"}, {Status: "unsettled", Reason: "other"}} {
		crop := valid
		crop.Settlement = settlement
		if ValidateCrop(crop) == nil {
			t.Fatalf("accepted malformed settlement %+v", settlement)
		}
	}
	for _, mutate := range []func(*Crop){func(c *Crop) { c.Settlement = &CaptureSettlement{Status: "settled"} }, func(c *Crop) { c.ClippedToViewport = true }, func(c *Crop) { c.Box = valid.Box }, func(c *Crop) { c.CapturedAt = valid.CapturedAt }, func(c *Crop) { c.Document = 1 }, func(c *Crop) { c.File = valid.File }} {
		crop := Crop{Unavailable: "outside viewport"}
		mutate(&crop)
		if ValidateCrop(crop) == nil {
			t.Fatalf("unavailable crop accepted current metadata %+v", crop)
		}
	}
}
func TestCropSettlementOldJSONAndStoreRoundTrip(t *testing.T) {
	var old Crop
	if err := json.Unmarshal([]byte(`{"file":"crop-0.png","box":[0,0,2,2],"captured_at":"2026-10-12T00:00:00Z","document":1}`), &old); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCrop(old); err != nil {
		t.Fatal(err)
	}
	if old.Settlement != nil || old.ClippedToViewport {
		t.Fatal("old crop inferred settlement or clipping")
	}
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "settlement") || strings.Contains(string(data), "clipped_to_viewport") {
		t.Fatal("unknown fields should remain absent", string(data))
	}
	sub := fixture()
	sub.Notes[0].Crop = old
	sub.Notes[0].Crop.Settlement = &CaptureSettlement{Status: "unsettled", Reason: "timeout"}
	sub.Notes[0].Crop.ClippedToViewport = true
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Prepare(sub, map[string][]byte{"crop-0.png": pngBytes(t)}); err != nil {
		t.Fatal(err)
	}
	restored, err := store.Read(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	crop := restored.Notes[0].Crop
	if crop.Settlement == nil || *crop.Settlement != *sub.Notes[0].Crop.Settlement || !crop.ClippedToViewport {
		t.Fatalf("lost settlement evidence: %+v", crop)
	}
	data, err = json.Marshal(crop)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"settlement":{"status":"unsettled","reason":"timeout"}`, `"clipped_to_viewport":true`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing JSON field %s: %s", want, data)
		}
	}
}
