package webcast

import (
	"context"
	"errors"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCastDraftCapturePreservesEvidenceAndDocument(t *testing.T) {
	prepared := board.PreparedNote{Document: 7, At: recording.NoteAt{Route: "https://original.test/"}, Region: recording.Box{10, 20, 30, 40}}
	at := time.Now()
	browser := &submissionBrowser{FakeBrowser: NewFakeBrowser()}
	browser.capture = func(ctx context.Context, doc int, boxes []core.CaptureRegion, report func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		if doc != 7 || len(boxes) != 1 || boxes[0].Region != prepared.Region {
			t.Fatalf("changed preparation %d %+v", doc, boxes)
		}
		return []recorder.CapturedCrop{{Page: core.PageIdentity{URL: "https://original.test/", Provenance: "cdp"}, PNG: []byte("pixels"), Box: prepared.Region, Document: doc, CapturedAt: at, Settlement: &core.CaptureSettlement{Status: "settled"}}}, nil
	}
	source := castDraftCapture{browser: func(context.Context) (Browser, error) { return browser, nil }}
	evidence, png, err := source.Capture(context.Background(), prepared)
	if err != nil || string(png) != "pixels" || evidence.Crop.CapturedAt != at || evidence.Crop.Document != 7 || evidence.Page.URL != "https://original.test/" || evidence.Crop.Settlement.Status != "settled" {
		t.Fatal(evidence, err)
	}
	browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		return []recorder.CapturedCrop{{Document: 7, Page: core.PageIdentity{URL: "https://changed.test/", Provenance: "cdp"}}}, nil
	}
	if _, _, err = source.Capture(context.Background(), prepared); err == nil {
		t.Fatal("accepted changed route within same document")
	}
	browser.capture = func(context.Context, int, []core.CaptureRegion, func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
		return []recorder.CapturedCrop{{Document: 8}}, nil
	}
	if _, _, err = source.Capture(context.Background(), prepared); err == nil {
		t.Fatal("accepted other document capture")
	}
}

func TestCastDraftServiceExistsBeforeLaunchAndStorageStaysPrivate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "recordings")
	sentinel := errors.New("launch refused")
	_, err := StartCast(context.Background(), CastConfig{RecordingsDir: root, Launch: func(ctx context.Context, options *RecordOptions) (Browser, error) {
		if options == nil || options.NoteDrafts == nil || options.Operator == nil {
			t.Fatal("draft service missing at launch")
		}
		if _, err := os.Stat(root + ".note-captures"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Fatal("preparation store entered recording root", err)
		}
		return nil, sentinel
	}})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
}
