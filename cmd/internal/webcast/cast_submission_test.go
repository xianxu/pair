package webcast

import (
	"bytes"
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"sync/atomic"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
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
	b.Recs[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "submission-load", Root: 1}
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
	if len(crops) != 0 || sub.Notes[0].Crop.Unavailable != "not captured when drawn" || sub.Source.RecordingPath != c.RecordingPath() || *writes != 0 {
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
func TestCastSubmissionNeverCapturesLegacyNotes(t *testing.T) {
	c, browser, _, _ := castSubmissionFixture(t)
	selected, _ := c.SubmissionSelection()
	browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		t.Fatal("submission took a fresh screenshot")
		return nil, nil
	}
	sub, crops, err := c.PrepareSubmission(context.Background(), "legacy", castSubmissionTarget(), selected, "", nil)
	if err != nil || len(crops) != 0 || sub.Notes[0].Crop.Unavailable != "not captured when drawn" || sub.Privacy != core.Private {
		t.Fatal(sub, err)
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

type storedSubmissionFixture struct {
	captureRoot string
	pageURL     string
	c           *Cast
	b           *recording.Builder
	w           *board.Writer
	browser     *submissionBrowser
	captures    atomic.Int32
	pixels      []byte
	node        recording.NodeID
}

func newStoredSubmissionFixture(t *testing.T) *storedSubmissionFixture {
	t.Helper()
	f := &storedSubmissionFixture{b: recording.NewBuilder("recording")}
	dir := filepath.Join(t.TempDir(), "recording")
	disk, err := recording.NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	bd := board.New(notes.Structural{}, func(r recording.Record, doc int) error { return f.w.Emit(r, doc) })
	f.w = board.NewWriter(recording.Tee(disk, bd), nil)
	if err = f.w.Put(f.b.Recs[0]); err != nil {
		t.Fatal(err)
	}
	bd.WithAnnotationCleanup(f.w.ClearAnnotations)
	if err = bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	f.browser = &submissionBrowser{FakeBrowser: NewFakeBrowser()}
	var pixels bytes.Buffer
	if err = png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	f.pixels = pixels.Bytes()
	f.browser.capture = func(ctx context.Context, doc int, regions []core.CaptureRegion, progress func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		f.captures.Add(1)
		return []recorder.CapturedCrop{{Page: core.PageIdentity{URL: f.pageURL, Title: "draw-time page", Provenance: "cdp"}, Box: regions[0].Box, BoxText: core.BoxText{Region: regions[0].Region, VisibleText: []core.BoxTextEntry{{Element: "button", Text: "original page text"}}}, PNG: f.pixels, Document: doc, CapturedAt: time.Now().UTC(), Settlement: &core.CaptureSettlement{Status: "settled"}}}, nil
	}
	f.captureRoot = filepath.Join(t.TempDir(), "captures")
	svc, err := notedraft.New(notedraft.Options{Root: f.captureRoot, RecordingPath: dir, Capture: castDraftCapture{browser: func(context.Context) (Browser, error) { return f.browser, nil }}, Release: bd.ReleaseNoteReservation,
		Write: func(ctx context.Context, a board.AnnotationAdmission, n board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
			if err := a.Check(); err != nil {
				return err
			}
			err := f.w.Emit(&n.Note, n.Document)
			if err != nil {
				settle(board.OperatorRefused, err)
			} else {
				settle(board.OperatorAccepted, nil)
			}
			return nil
		}, Observe: func(ctx context.Context, id string) (*recording.Note, bool, error) {
			n, ok := bd.ObservedNote(id)
			return n, ok, nil
		}})
	if err != nil {
		t.Fatal(err)
	}
	f.c = &Cast{browser: f.browser, board: bd, recDir: dir, noteDrafts: svc, done: make(chan struct{})}
	t.Cleanup(func() { svc.Close(); f.w.Close() })
	f.page(t, "https://original.test/")
	return f
}
func (f *storedSubmissionFixture) page(t *testing.T, url string) {
	f.pageURL = url
	prev := len(f.b.Recs)
	el := f.b.El("button", []string{"id", "target"}, f.b.Txt("original anchor"))
	f.node = el.ID
	why := recording.SegLive
	if prev > 1 {
		why = recording.SegNavigate
	}
	f.b.At(int64(prev+1)*10).Page(why, url, el)
	seg := f.b.Recs[prev].(*recording.Seg)
	seg.PageLoad = &recording.PageLoad{Target: seg.Tab.ID, Loader: fmt.Sprintf("stored-load-%d", seg.Seg), Root: int64(seg.Seg)}
	for _, r := range f.b.Recs[prev:] {
		if err := f.w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
}
func (f *storedSubmissionFixture) draw(t *testing.T, request string) notedraft.Status {
	t.Helper()
	a, err := f.c.board.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.PrepareNote(board.PrepareNoteIn{Seg: f.c.board.OperatorView().Seg, Node: f.node, Box: recording.Box{10, 20, 30, 40}, Region: recording.Box{10, 20, 20, 20}}, true)
	if err != nil {
		t.Fatal(err)
	}
	owner := notedraft.Owner{Scope: "private-operator", Generation: 1}
	status, err := f.c.noteDrafts.Prepare(context.Background(), owner, request, a, p, true)
	if err != nil {
		t.Fatal(err)
	}
	for end := time.Now().Add(time.Second); status.State == "pending" && time.Now().Before(end); {
		time.Sleep(time.Millisecond)
		status, err = f.c.noteDrafts.Status(owner, status.Draft)
		if err != nil {
			t.Fatal(err)
		}
	}
	if status.State != "ready" {
		t.Fatal(status)
	}
	status, err = f.c.noteDrafts.Save(context.Background(), owner, "save-"+request, status.Draft, a, board.Viewer{}, 0, "authored "+request, nil)
	if err != nil || status.State != "saved" {
		t.Fatal(status, err)
	}
	return status
}
func TestCastSubmissionStoredCaptureSurvivesMutationAndPublication(t *testing.T) {
	f := newStoredSubmissionFixture(t)
	saved := f.draw(t, "first")
	selected, err := f.c.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	f.page(t, "https://different.test/")
	f.browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		t.Fatal("submission captured current page")
		return nil, nil
	}
	var progress []core.SubmissionProgress
	sub, crops, err := f.c.PrepareSubmission(context.Background(), "stored", castSubmissionTarget(), selected, "https://replay.test/token", func(p core.SubmissionProgress) { progress = append(progress, p) })
	if err != nil {
		t.Fatal(err)
	}
	n := sub.Notes[0]
	if sub.Version != 3 || n.ID != saved.NoteID || n.CaptureID != saved.Draft || n.Page == nil || n.Page.URL != "https://original.test/" || n.SourceDocument != 1 || !bytes.Equal(crops[n.Crop.File], f.pixels) || n.BoxText.VisibleText[0].Text != "original page text" || f.captures.Load() != 1 {
		t.Fatalf("lost draw evidence %+v", sub)
	}
	if len(progress) != 1 || progress[0].Phase != "context" {
		t.Fatal(progress)
	}
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Prepare(sub, crops); err != nil {
		t.Fatal(err)
	}
	obs, err := store.ConfirmPrepared(sub, crops)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.c.ReconcilePreparedSubmission(context.Background(), sub, obs); err != nil {
		t.Fatal(err)
	}
	again, crops2, err := f.c.PrepareSubmission(context.Background(), "second", castSubmissionTarget(), selected, "", nil)
	if err != nil || !bytes.Equal(crops2[again.Notes[0].Crop.File], f.pixels) {
		t.Fatal("bundle retry lost original", err)
	}
	if err = f.c.ReconcilePreparedSubmission(context.Background(), again, core.PublicationObservation{State: core.PublicationAbsent}); err != nil {
		t.Fatal(err)
	}
	sub.Delivery = core.Dispatched
	if err = f.c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if err = f.c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatal("resolve not idempotent", err)
	}
}
func TestCastSubmissionIndependentPagePressesPreservePerNotePrivacy(t *testing.T) {
	f := newStoredSubmissionFixture(t)
	first := f.draw(t, "first")
	firstPress, err := f.c.SubmissionSelection()
	if err != nil || len(firstPress.Notes) != 1 || firstPress.Notes[0].ID != first.NoteID {
		t.Fatalf("first press: %+v, %v", firstPress, err)
	}
	f.page(t, "https://second.test/")
	second := f.draw(t, "second")
	secondPress, err := f.c.SubmissionSelection()
	if err != nil || len(secondPress.Notes) != 1 || secondPress.Notes[0].ID != second.NoteID {
		t.Fatalf("second press selected another load: %+v, %v", secondPress, err)
	}
	policy, err := core.ParseOriginPolicy([]byte(`{"origins":[{"origin":"https://original.test","privacy":"shareable"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	f.c.readOriginPolicy = func() (core.OriginPolicy, string) { return policy, "" }
	original, firstCrops, err := f.c.PrepareSubmission(context.Background(), "original-press", castSubmissionTarget(), firstPress, "", nil)
	if err != nil || len(firstCrops) != 1 || len(original.Notes) != 1 || original.Privacy != core.Shareable || original.Notes[0].Privacy != core.Shareable || original.Notes[0].ID != first.NoteID {
		t.Fatalf("earlier admitted source privacy lost: %+v, %v", original, err)
	}
	current, secondCrops, err := f.c.PrepareSubmission(context.Background(), "current-press", castSubmissionTarget(), secondPress, "", nil)
	if err != nil || len(secondCrops) != 1 || len(current.Notes) != 1 || current.Privacy != core.Private || current.Notes[0].Privacy != core.Private || current.Notes[0].ID != second.NoteID || current.Notes[0].SourceDocument == original.Notes[0].SourceDocument {
		t.Fatalf("independent page press privacy/context: %+v, %v", current, err)
	}
	// Historical parsing remains unfiltered independently of current admission.
	records, err := readSubmissionRecording(context.Background(), f.c.RecordingPath())
	if err != nil {
		t.Fatal(err)
	}
	history, err := query.New(records, notes.Structural{}).Notes(5000)
	if err != nil || len(history) != 2 || history[0].Note.ID != first.NoteID || history[1].Note.ID != second.NoteID || history[0].PageLoad == history[1].PageLoad {
		t.Fatalf("historical documents lost: %+v, %v", history, err)
	}
	for _, sub := range []core.Submission{original, current} {
		if err = f.c.ReconcilePreparedSubmission(context.Background(), sub, core.PublicationObservation{State: core.PublicationAbsent}); err != nil {
			t.Fatal(err)
		}
	}
	policy, err = core.ParseOriginPolicy([]byte(`{"origins":[{"origin":"https://original.test","privacy":"shareable"},{"origin":"https://second.test","privacy":"shareable"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for i, press := range []board.Selection{firstPress, secondPress} {
		sub, _, err := f.c.PrepareSubmission(context.Background(), fmt.Sprintf("shareable-%d", i), castSubmissionTarget(), press, "", nil)
		if err != nil || sub.Privacy != core.Shareable {
			t.Fatalf("shareable retry: %+v, %v", sub, err)
		}
	}
	if f.captures.Load() != 2 {
		t.Fatal("policy or navigation retries recaptured source evidence")
	}
}
func TestCastSubmissionMissingCaptureIsExplicitAndNoRecapture(t *testing.T) {
	f := newStoredSubmissionFixture(t)
	saved := f.draw(t, "first")
	// The durable note survives a missing source; never manufacture a replacement.
	if err := f.c.noteDrafts.Resolve(saved.NoteID); err != nil {
		t.Fatal(err)
	}
	selected, _ := f.c.SubmissionSelection()
	sub, crops, err := f.c.PrepareSubmission(context.Background(), "missing", castSubmissionTarget(), selected, "", nil)
	if err != nil || len(crops) != 0 || sub.Notes[0].Crop.Unavailable == "" || f.captures.Load() != 1 {
		t.Fatal(sub, err)
	}
}
func TestCastSubmissionClaimErrorKeepsIdentityForReconciliation(t *testing.T) {
	f := newStoredSubmissionFixture(t)
	f.draw(t, "first")
	selected, _ := f.c.SubmissionSelection()
	first, _, err := f.c.PrepareSubmission(context.Background(), "first", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	failed, _, err := f.c.PrepareSubmission(context.Background(), "other", castSubmissionTarget(), selected, "", nil)
	if err == nil || failed.ID != "other" || len(failed.Notes) != 1 {
		t.Fatal(failed, err)
	}
	if err = f.c.ReconcilePreparedSubmission(context.Background(), first, core.PublicationObservation{State: core.PublicationUnknown}); err != nil {
		t.Fatal(err)
	}
	if err = f.c.ReconcilePreparedSubmission(context.Background(), first, core.PublicationObservation{State: core.PublicationAbsent}); err != nil {
		t.Fatal(err)
	}
	_, _, err = f.c.PrepareSubmission(context.Background(), "retry", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCastSubmissionPartialClaimErrorCanReleaseOnlyItsClaim(t *testing.T) {
	f := newStoredSubmissionFixture(t)
	first := f.draw(t, "first")
	second := f.draw(t, "second")
	if _, err := f.c.noteDrafts.Claim(context.Background(), "held", []string{second.Draft}); err != nil {
		t.Fatal(err)
	}
	selected, _ := f.c.SubmissionSelection()
	partial, _, err := f.c.PrepareSubmission(context.Background(), "partial", castSubmissionTarget(), selected, "", nil)
	if err == nil || partial.ID != "partial" || len(partial.Notes) != 2 {
		t.Fatal(partial, err)
	}
	if err = f.c.ReconcilePreparedSubmission(context.Background(), partial, core.PublicationObservation{State: core.PublicationAbsent}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.c.noteDrafts.Claim(context.Background(), "first-again", []string{first.Draft}); err != nil {
		t.Fatal("first claim stranded", err)
	}
	if _, err = f.c.noteDrafts.Claim(context.Background(), "second-again", []string{second.Draft}); err == nil {
		t.Fatal("released another submission's uncertain claim")
	}
}
func TestCastSubmissionCorruptPixelsAreUnavailable(t *testing.T) {
	f := newStoredSubmissionFixture(t)
	saved := f.draw(t, "first")
	if err := os.WriteFile(filepath.Join(f.captureRoot, saved.Draft, "capture.png"), []byte("not pixels"), 0600); err != nil {
		t.Fatal(err)
	}
	selected, _ := f.c.SubmissionSelection()
	sub, crops, err := f.c.PrepareSubmission(context.Background(), "corrupt", castSubmissionTarget(), selected, "", nil)
	if err != nil || len(crops) != 0 || sub.Notes[0].Crop.Unavailable == "" || f.captures.Load() != 1 {
		t.Fatal(sub, err)
	}
}
func TestCastDraftCleanupFailureDoesNotReverseRecordedAcknowledgement(t *testing.T) {
	f := newStoredSubmissionFixture(t)
	saved := f.draw(t, "first")
	if err := os.Rename(f.captureRoot, f.captureRoot+".held"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.captureRoot, []byte("blocked path"), 0600); err != nil {
		t.Fatal(err)
	}
	logged := false
	sink := &draftEvidenceSink{sink: f.w, notes: f.c.noteDrafts, ids: map[string]bool{saved.NoteID: true}, logf: func(string, ...any) { logged = true }}
	state := &recording.NoteState{ID: saved.NoteID, State: recording.NoteResolved}
	state.SetTime(1000)
	if err := sink.Put(state); err != nil {
		t.Fatal("cleanup changed successful record ack", err)
	}
	if !logged {
		t.Fatal("cleanup failure not reported")
	}
	recs, err := readSubmissionRecording(context.Background(), f.c.RecordingPath())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range recs {
		if n, ok := r.(*recording.NoteState); ok && n.ID == saved.NoteID && n.State == recording.NoteResolved {
			found = true
		}
	}
	if !found {
		t.Fatal("resolution record missing")
	}
}
