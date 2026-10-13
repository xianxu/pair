package webcast

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
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

type submissionBrowser struct {
	*FakeBrowser
	capture func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error)
}

func (b *submissionBrowser) CaptureCrops(ctx context.Context, doc int, boxes []core.CaptureRegion, progress func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
	return b.capture(ctx, doc, boxes, progress)
}
func castSubmissionFixture(t *testing.T) (*Cast, *submissionBrowser, *recording.Builder, *int) {
	t.Helper()
	b := recording.NewBuilder("recording")
	el := b.El("button", nil, b.Txt("Save"))
	b.Page(recording.SegLive, "https://example.test", el)
	box := recording.Box{20, 20, 100, 40}
	b.At(10).Add(&recording.Note{ID: "n", Author: recording.Author{ID: "person"}, Text: "fix", At: recording.NoteAt{Node: el.ID, Text: "Save", Box: &box}, Region: &box})
	writes := 0
	var writer *board.Writer
	bd := board.New(notes.Structural{}, func(r recording.Record, doc int) error { writes++; return writer.Emit(r, doc) })
	writer = board.NewWriter(bd, nil)
	bd.WithAnnotationCleanup(writer.ClearAnnotations)
	if err := bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	for _, r := range b.Recs {
		if err := writer.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	for name, recs := range recording.Split(b.Recs) {
		var data []byte
		for _, r := range recs {
			line, err := recording.Encode(r)
			if err != nil {
				t.Fatal(err)
			}
			data = append(data, line...)
			data = append(data, '\n')
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	browser := &submissionBrowser{FakeBrowser: NewFakeBrowser()}
	browser.capture = func(ctx context.Context, doc int, boxes []core.CaptureRegion, progress func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		var out []recorder.CapturedCrop
		for _, box := range boxes {
			var data bytes.Buffer
			png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2)))
			out = append(out, recorder.CapturedCrop{Page: core.PageIdentity{URL: "http://localhost:4321/blog", Title: "Local blog", Provenance: "cdp"}, PNG: data.Bytes(), Box: box.Box, BoxText: core.BoxText{Region: box.Region}, Document: doc, CapturedAt: time.Now()})
		}
		return out, nil
	}
	return &Cast{browser: browser, board: bd, recDir: dir, done: make(chan struct{})}, browser, b, &writes
}
func castSubmissionTarget() core.Target {
	return core.Target{Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce"}
}
func TestCastSubmissionPrepareAndResolve(t *testing.T) {
	c, _, _, writes := castSubmissionFixture(t)
	selection, err := c.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	sub, crops, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selection, "https://replay.test/token", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(crops) != 1 || sub.Notes[0].Crop.File != "crop-0.png" || sub.Source.RecordingPath != c.RecordingPath() || *writes != 0 {
		t.Fatal(sub, crops, *writes)
	}
	if sub.Notes[0].Replay != "https://replay.test/token/r/recording/#t=10" {
		t.Fatal(sub.Notes[0].Replay)
	}
	if err = c.ResolveSubmission(context.Background(), sub); err == nil {
		t.Fatal("resolved before dispatch")
	}
	sub.Delivery = core.Dispatched
	if err = c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if *writes != 1 {
		t.Fatal(*writes)
	}
	if err = c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if *writes != 1 {
		t.Fatal("duplicate resolve")
	}
}
func TestCastSubmissionCaptureFailureAndGonePage(t *testing.T) {
	c, browser, _, writes := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		return nil, errors.New("navigation during capture")
	}
	if _, _, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil); err == nil {
		t.Fatal("capture failure ignored")
	}
	if *writes != 0 {
		t.Fatal("resolved on failed preparation")
	}
	browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		return []recorder.CapturedCrop{{Unavailable: "page changed"}}, nil
	}
	sub, crops, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(crops) != 0 || sub.Notes[0].Crop.Unavailable != "page changed" {
		t.Fatal(sub, crops)
	}
	sub.Delivery = core.Dispatched
	sub.Source.RecordingPath = "/another-recording"
	if err = c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if *writes != 0 {
		t.Fatal("resolved another cast")
	}
	close(c.done)
	sub.Source.RecordingPath = c.RecordingPath()
	if err = c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if *writes != 0 {
		t.Fatal("resolved ended cast")
	}
}
func TestCastSubmissionBoundedRecordingAndMalformedLines(t *testing.T) {
	c, _, _, _ := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	path := filepath.Join(c.recDir, recording.FilePointer)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(128<<20 + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, _, err = c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil); err == nil {
		t.Fatal("oversize input accepted")
	}
	if err = os.WriteFile(path, []byte("{partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil); err != nil {
		t.Fatal("incomplete tail should be ignored", err)
	}
	if err = os.WriteFile(path, []byte("{bad}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil); err == nil {
		t.Fatal("complete malformed line accepted")
	}
}

func TestCastSubmissionRejectsRecordingSymlinks(t *testing.T) {
	c, _, _, _ := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	path := filepath.Join(c.recDir, recording.FilePointer)
	if err := os.Symlink("/etc/passwd", path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil); err == nil {
		t.Fatal("followed recording symlink")
	}
}
func TestCastSubmissionLateAckDoesNotResolveNewDocument(t *testing.T) {
	c, _, b, writes := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	sub, _, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	prev := len(b.Recs)
	b.At(30).Page(recording.SegNavigate, "https://example.test/next", b.El("p", nil, b.Txt("Next")))
	for _, r := range b.Recs[prev:] {
		if err = c.board.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	sub.Delivery = core.Dispatched
	if err = c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if *writes != 0 {
		t.Fatal("late ack emitted note resolution")
	}
}

func TestCastSubmissionSettlementAndClipping(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*recorder.CapturedCrop)
		wantErr bool
	}{
		{"settled full crop", func(c *recorder.CapturedCrop) { c.Settlement = &core.CaptureSettlement{Status: "settled"} }, false},
		{"unsettled clipped crop", func(c *recorder.CapturedCrop) {
			c.Box[0] += 10
			c.Box[2] -= 10
			c.ClippedToViewport = true
			c.Settlement = &core.CaptureSettlement{Status: "unsettled", Reason: "timeout"}
		}, false},
		{"clipped without flag", func(c *recorder.CapturedCrop) { c.Box[2] -= 10 }, true},
		{"flag without clipping", func(c *recorder.CapturedCrop) { c.ClippedToViewport = true }, true},
		{"outside requested", func(c *recorder.CapturedCrop) { c.Box[0] -= 1; c.ClippedToViewport = true }, true},
		{"expanded requested", func(c *recorder.CapturedCrop) { c.Box[2] += 1; c.ClippedToViewport = true }, true},
		{"invalid coordinates", func(c *recorder.CapturedCrop) { c.Box[0] = math.NaN(); c.ClippedToViewport = true }, true},
		{"changed document", func(c *recorder.CapturedCrop) { c.Document++ }, true},
		{"invalid settlement", func(c *recorder.CapturedCrop) {
			c.Settlement = &core.CaptureSettlement{Status: "settled", Reason: "timeout"}
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, browser, _, writes := castSubmissionFixture(t)
			base := browser.capture
			var captured recorder.CapturedCrop
			browser.capture = func(ctx context.Context, doc int, regions []core.CaptureRegion, progress func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
				crops, err := base(ctx, doc, regions, progress)
				if err != nil {
					return nil, err
				}
				crops[0].BoxText.VisibleText = []core.BoxTextEntry{{Element: "<button>", Text: "current text"}}
				tc.mutate(&crops[0])
				captured = crops[0]
				return crops, nil
			}
			selected, err := c.SubmissionSelection()
			if err != nil {
				t.Fatal(err)
			}
			sub, pngs, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("accepted invalid capture", sub)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			crop := sub.Notes[0].Crop
			if crop.Settlement == nil || *crop.Settlement != *captured.Settlement || crop.ClippedToViewport != captured.ClippedToViewport || crop.Box != captured.Box {
				t.Fatalf("capture metadata not propagated: %+v", crop)
			}
			if len(pngs) != 1 || *writes != 0 || sub.Notes[0].BoxText.VisibleText[0].Text != "current text" {
				t.Fatal("lost evidence or resolved during capture", sub)
			}
		})
	}
}
func TestCastSubmissionOutsideViewportPreservesNote(t *testing.T) {
	c, browser, _, writes := castSubmissionFixture(t)
	browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		return []recorder.CapturedCrop{{Unavailable: "outside viewport", BoxText: core.BoxText{Unavailable: "outside viewport"}}}, nil
	}
	selected, err := c.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	sub, pngs, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.Notes) != 1 || sub.Notes[0].ID != "n" || sub.Notes[0].At.Text != "Save" || !strings.Contains(sub.Notes[0].Element, "Save") {
		t.Fatal("unavailable capture lost recorded context", sub)
	}
	crop := sub.Notes[0].Crop
	if crop.Unavailable != "outside viewport" || crop.Settlement != nil || crop.ClippedToViewport || len(pngs) != 0 || *writes != 0 {
		t.Fatal("unavailable capture has current pixels or delivery effects", sub)
	}
}
func TestCastSubmissionUnavailableRefusesCurrentMetadata(t *testing.T) {
	for _, mutate := range []func(*recorder.CapturedCrop){
		func(c *recorder.CapturedCrop) { c.Settlement = &core.CaptureSettlement{Status: "settled"} },
		func(c *recorder.CapturedCrop) { c.ClippedToViewport = true },
		func(c *recorder.CapturedCrop) { c.Box = recording.Box{1, 1, 2, 2} },
		func(c *recorder.CapturedCrop) { c.CapturedAt = time.Now() },
		func(c *recorder.CapturedCrop) { c.Document = 1 },
	} {
		c, browser, _, _ := castSubmissionFixture(t)
		browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
			crop := recorder.CapturedCrop{Unavailable: "outside viewport"}
			mutate(&crop)
			return []recorder.CapturedCrop{crop}, nil
		}
		selected, err := c.SubmissionSelection()
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil); err == nil {
			t.Fatal("unavailable capture silently dropped contradictory current metadata")
		}
	}
}

func TestCastSubmissionPrivacyUsesCaptureAndSurvivesInvalidPolicy(t *testing.T) {
	for _, tc := range []struct{ url, want, reason string }{
		{"https://captured.test", "shareable", ""}, {"https://example.test", "private", ""},
		{"", "private", ""}, {"https://captured.test", "private", "invalid JSON"},
	} {
		t.Run(tc.url+tc.reason, func(t *testing.T) {
			c, b, _, _ := castSubmissionFixture(t)
			original := b.capture
			b.capture = func(ctx context.Context, doc int, boxes []core.CaptureRegion, progress func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
				crops, err := original(ctx, doc, boxes, progress)
				for i := range crops {
					if tc.url == "" {
						crops[i] = recorder.CapturedCrop{Unavailable: "page changed"}
						continue
					}
					crops[i].Page = core.PageIdentity{URL: tc.url, Title: "Captured page", Provenance: "cdp"}
				}
				return crops, err
			}
			reads := 0
			notice := ""
			c.logf = func(format string, args ...any) { notice = fmt.Sprintf(format, args...) }
			c.readOriginPolicy = func() (core.OriginPolicy, string) {
				reads++
				if tc.reason != "" {
					return core.InvalidOriginPolicy(tc.reason), "/couch/webcast/origin-privacy.json: invalid policy; using private"
				}
				p, err := core.ParseOriginPolicy([]byte(`{"origins":[{"origin":"https://captured.test","privacy":"shareable"}]}`))
				if err != nil {
					t.Fatal(err)
				}
				return p, ""
			}
			selected, _ := c.SubmissionSelection()
			sub, crops, err := c.PrepareSubmission(context.Background(), "privacy", castSubmissionTarget(), selected, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(sub.Privacy) != tc.want || reads != 1 {
				t.Fatalf("decision=%+v reads=%d", sub.PrivacyDecision, reads)
			}
			if tc.reason != "" && (sub.PrivacyRule.Invalid != tc.reason || !strings.Contains(notice, "origin-privacy.json")) {
				t.Fatalf("missing invalid-policy evidence %v %s", sub.PrivacyRule, notice)
			}
			store, err := core.NewStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.Prepare(sub, crops); err != nil {
				t.Fatal(err)
			}
			for _, event := range []core.Event{core.Begin, core.Ack, core.Complete} {
				if _, err = store.Transition(sub.ID, sub.Target, event); err != nil {
					t.Fatal("invalid policy blocked delivery", err)
				}
			}
		})
	}
}

func TestCastSubmissionPrivacyDefault(t *testing.T) {
	c, _, _, _ := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	sub, _, err := c.PrepareSubmission(context.Background(), "privacy", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Privacy != core.Private {
		t.Fatalf("default %q", sub.Privacy)
	}
}

func TestCastSubmissionUsesCapturedPageSource(t *testing.T) {
	c, b, _, _ := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	sub, _, err := c.PrepareSubmission(context.Background(), "page", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Source.Page == nil || sub.Source.Page.URL != "http://localhost:4321/blog" || sub.Source.Page.Title != "Local blog" || sub.Notes[0].SourceDocument != sub.Source.Document {
		t.Fatal(sub)
	}
	original := b.capture
	b.capture = func(ctx context.Context, doc int, boxes []core.CaptureRegion, progress func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		crops, err := original(ctx, doc, boxes, progress)
		crops[0].Page = core.PageIdentity{}
		return crops, err
	}
	if _, _, err = c.PrepareSubmission(context.Background(), "missing-page", castSubmissionTarget(), selected, "", nil); err == nil {
		t.Fatal("pixels without page accepted")
	}
}

func TestCastSubmissionMixedViewportAvailability(t *testing.T) {
	for _, reason := range []string{"outside viewport", "page changed"} {
		t.Run(reason, func(t *testing.T) {
			c, browser, b, _ := castSubmissionFixture(t)
			box := recording.Box{900, 20, 100, 40}
			note := &recording.Note{ID: "offscreen", Author: recording.Author{ID: "person"}, Text: "also fix", At: recording.NoteAt{Box: &box}, Region: &box}
			b.At(11).Add(note)
			c.board = board.New(notes.Structural{}, func(recording.Record, int) error { return nil })
			writer := board.NewWriter(c.board, nil)
			for _, r := range b.Recs {
				if err := writer.Put(r); err != nil {
					t.Fatal(err)
				}
			}
			for name, recs := range recording.Split(b.Recs) {
				var data []byte
				for _, r := range recs {
					line, err := recording.Encode(r)
					if err != nil {
						t.Fatal(err)
					}
					data = append(data, line...)
					data = append(data, '\n')
				}
				if err := os.WriteFile(filepath.Join(c.RecordingPath(), name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			base := browser.capture
			browser.capture = func(ctx context.Context, doc int, regions []core.CaptureRegion, progress func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
				crops, err := base(ctx, doc, regions, progress)
				if err != nil {
					return nil, err
				}
				if len(crops) != 2 {
					t.Fatalf("want two selected notes, got %d", len(crops))
				}
				crops[1] = recorder.CapturedCrop{Unavailable: reason, BoxText: core.BoxText{Unavailable: reason}}
				return crops, nil
			}
			selected, err := c.SubmissionSelection()
			if err != nil {
				t.Fatal(err)
			}
			sub, crops, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "", nil)
			if reason == "page changed" {
				if err == nil {
					t.Fatal("mixed document availability accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(crops) != 1 || sub.Source.Page == nil || len(sub.Notes) != 2 || sub.Notes[1].Crop.Unavailable != reason {
				t.Fatalf("lost batch evidence: %+v", sub)
			}
		})
	}
}

func TestCastSubmissionProgressReachesCapture(t *testing.T) {
	c, b, _, _ := castSubmissionFixture(t)
	selected, err := c.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	original := b.capture
	var phases []core.SubmissionProgress
	b.capture = func(ctx context.Context, doc int, boxes []core.CaptureRegion, report func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		if len(phases) != 1 || phases[0].Phase != "context" {
			t.Fatalf("read/build progress before capture=%+v", phases)
		}
		report(core.SubmissionProgress{Phase: "capturing", Current: 1, Total: len(boxes)})
		return original(ctx, doc, boxes, report)
	}
	_, _, err = c.PrepareSubmission(context.Background(), "progress", castSubmissionTarget(), selected, "", func(p core.SubmissionProgress) { phases = append(phases, p) })
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) != 2 || phases[1] != (core.SubmissionProgress{Phase: "capturing", Current: 1, Total: 1}) {
		t.Fatalf("capture progress not forwarded: %+v", phases)
	}
}
