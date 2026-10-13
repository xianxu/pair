package webcast

import (
	"bytes"
	"context"
	"errors"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type submissionBrowser struct {
	*FakeBrowser
	capture func(context.Context, int, []core.CaptureRegion) ([]recorder.CapturedCrop, error)
}

func (b *submissionBrowser) CaptureCrops(ctx context.Context, doc int, boxes []core.CaptureRegion) ([]recorder.CapturedCrop, error) {
	return b.capture(ctx, doc, boxes)
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
	browser.capture = func(ctx context.Context, doc int, boxes []core.CaptureRegion) ([]recorder.CapturedCrop, error) {
		var out []recorder.CapturedCrop
		for _, box := range boxes {
			var data bytes.Buffer
			png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2)))
			out = append(out, recorder.CapturedCrop{PNG: data.Bytes(), Box: box.Box, BoxText: core.BoxText{Region: box.Region}, Document: doc, CapturedAt: time.Now()})
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
	sub, crops, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selection, "https://replay.test/token")
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
	browser.capture = func(context.Context, int, []core.CaptureRegion) ([]recorder.CapturedCrop, error) {
		return nil, errors.New("navigation during capture")
	}
	if _, _, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, ""); err == nil {
		t.Fatal("capture failure ignored")
	}
	if *writes != 0 {
		t.Fatal("resolved on failed preparation")
	}
	browser.capture = func(context.Context, int, []core.CaptureRegion) ([]recorder.CapturedCrop, error) {
		return []recorder.CapturedCrop{{Unavailable: "page changed"}}, nil
	}
	sub, crops, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "")
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
	if _, _, err = c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, ""); err == nil {
		t.Fatal("oversize input accepted")
	}
	if err = os.WriteFile(path, []byte("{partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, ""); err != nil {
		t.Fatal("incomplete tail should be ignored", err)
	}
	if err = os.WriteFile(path, []byte("{bad}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, ""); err == nil {
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
	if _, _, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, ""); err == nil {
		t.Fatal("followed recording symlink")
	}
}
func TestCastSubmissionLateAckDoesNotResolveNewDocument(t *testing.T) {
	c, _, b, writes := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	sub, _, err := c.PrepareSubmission(context.Background(), "submission", castSubmissionTarget(), selected, "")
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
