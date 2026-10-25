package recorder

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
)

type heldDraftCapture struct{ entered, release chan struct{} }

func (c heldDraftCapture) Capture(ctx context.Context, p board.PreparedNote) (notedraft.Evidence, []byte, error) {
	close(c.entered)
	select {
	case <-ctx.Done():
		return notedraft.Evidence{}, nil, ctx.Err()
	case <-c.release:
		return notedraft.Evidence{Crop: core.Crop{Unavailable: notedraft.CaptureBusy}}, nil, nil
	}
}
func TestOperatorDraftCaptureDoesNotBlockRecorder(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		name := "viewer-off"
		if allowed {
			name = "viewer-on"
		}
		t.Run(name, func(t *testing.T) { testOperatorDraftCaptureDoesNotBlockRecorder(t, allowed) })
	}
}

func testOperatorDraftCaptureDoesNotBlockRecorder(t *testing.T, allowed bool) {
	h, b, ch := operatorHarnessAnnotation(t, allowed)
	generation := b.AnnotationCapability().Generation
	capture := heldDraftCapture{make(chan struct{}), make(chan struct{})}
	svc, err := notedraft.New(notedraft.Options{Root: t.TempDir(), RecordingPath: filepath.Join(t.TempDir(), "recording"), Capture: capture, Write: h.r.WritePrepared, Observe: h.r.ObserveNote, Release: b.ReleaseNoteReservation})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	h.r.ask(context.Background(), func() { h.r.opts.NoteDrafts = svc })
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "prepare1", map[string]any{"seg": 1, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}, generation), 7))
	first := takeOperator(t, ch)
	if first.State != "pending" || first.Draft == "" {
		t.Fatal(first)
	}
	<-capture.entered
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err = h.r.Stats(ctx); err != nil {
		t.Fatal("capture blocked recorder", err)
	}
	h.r.Send(operatorEvent(operatorRequest("draft-status", "status1", map[string]any{"draft": "prepare1"}, generation), 7))
	if got := takeOperator(t, ch); got.State != "pending" {
		t.Fatal(got)
	}
	close(capture.release)
	var ready OperatorReply
	for end := time.Now().Add(time.Second); time.Now().Before(end); {
		h.r.Send(operatorEvent(operatorRequest("draft-status", "status2", map[string]any{"draft": first.Draft}, generation), 7))
		ready = takeOperator(t, ch)
		if ready.State == "ready" {
			break
		}
	}
	if ready.State != "ready" || ready.Unavailable == "" {
		t.Fatal(ready)
	}
	h.r.Send(operatorEvent(operatorRequest("save-note", "save1", map[string]any{"draft": first.Draft, "color": 0, "text": "saved capture"}, generation), 7))
	got := takeOperator(t, ch)
	if got.Outcome == board.OperatorRefused {
		t.Fatal(got)
	}
	for end := time.Now().Add(time.Second); got.State != "saved" && time.Now().Before(end); {
		h.r.Send(operatorEvent(operatorRequest("draft-status", "saved-status", map[string]any{"draft": first.Draft}, generation), 7))
		got = takeOperator(t, ch)
	}
	if got.State != "saved" || got.ID == "" {
		t.Fatal(got)
	}
	n, ok := b.ObservedNote(got.ID)
	if !ok || n.Owner != broadcast.OperatorNoteOwner || n.CropUnavailable == "" || n.Capture == "" {
		t.Fatal(n, ok)
	}
}
