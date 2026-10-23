package couchtty

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// This cast models a failed private console and independently observable live
// display. Confirmed delivery must still reach live viewers in that condition.
type displayPreviewCast struct {
	*submissionPreviewFake
	missingHost bool
	masks       chan core.DeliveredFeedback
}

func (f *displayPreviewCast) SubmissionPreviewHost(ctx context.Context) (core.PreviewHost, <-chan struct{}, error) {
	if f.missingHost {
		return core.PreviewHost{}, nil, errors.New("private host unavailable")
	}
	return f.submissionPreviewFake.SubmissionPreviewHost(ctx)
}
func (f *displayPreviewCast) SetDeliveredFeedback(v core.DeliveredFeedback) error {
	f.muPreview.Lock()
	remaining := slices.DeleteFunc(slices.Clone(f.selection.Notes), func(n board.SelectedNote) bool {
		_, sent := v.Notes[core.PreviewNote{ID: n.ID, Revision: n.Revision}]
		return sent
	})
	if !slices.Equal(remaining, f.selection.Notes) {
		f.selection.Notes = remaining
		f.selection.SketchGeneration++
		f.setVisibleFrame()
	}
	f.muPreview.Unlock()
	f.masks <- v
	return nil
}

func TestSubmissionDeliveredDisplayDoesNotRequirePrivateHost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, base, target := newSubmissionManagerTest(t)
		sub, _, err := base.PrepareSubmission(context.Background(), "display-host", target, board.Selection{}, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = m.store.Prepare(sub, nil); err != nil {
			t.Fatal(err)
		}
		for _, event := range []core.Event{core.Begin, core.Ack} {
			if _, err = m.store.Transition(sub.ID, target, event); err != nil {
				t.Fatal(err)
			}
		}
		f := &displayPreviewCast{submissionPreviewFake: newSubmissionPreviewFake(base), missingHost: true, masks: make(chan core.DeliveredFeedback, 8)}
		f.selection.Sketches = nil
		f.selection.Notes = []board.SelectedNote{{ID: "note"}}
		ctx, cancel := context.WithCancel(m.ctx)
		done := make(chan struct{})
		go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
		synctest.Wait()
		select {
		case mask := <-f.masks:
			if _, ok := mask.Notes[core.PreviewNote{ID: "note"}]; !ok {
				t.Error("confirmed note missing from display mask")
			}
		default:
			t.Error("confirmed display update blocked by unavailable private host")
		}
		cancel()
		synctest.Wait()
		<-done
	})
}

func TestSubmissionDisplayChangesOnlyAfterConfirmedAckAndRefreshesNotes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, base, target := newSubmissionManagerTest(t)
		base.resolveFail = errors.New("annotation resolution waiting")
		sub, _, err := base.PrepareSubmission(context.Background(), "display-ack", target, board.Selection{}, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = m.store.Prepare(sub, nil); err != nil {
			t.Fatal(err)
		}
		f := &displayPreviewCast{submissionPreviewFake: newSubmissionPreviewFake(base), masks: make(chan core.DeliveredFeedback, 8)}
		f.selection.Sketches = nil
		f.selection.Notes = []board.SelectedNote{{ID: "note"}, {ID: "unsent", Revision: 1}}
		m.current = func() (submissionCast, error) { return f, nil }
		ctx, cancel := context.WithCancel(m.ctx)
		done := make(chan struct{})
		go func() { defer close(done); watchSubmissionPreview(m, ctx, f) }()
		synctest.Wait()
		select {
		case mask := <-f.masks:
			if len(mask.Notes) != 0 {
				t.Error("prepared evidence hidden")
			}
		default:
			t.Error("initial mask not applied")
		}
		select {
		case p := <-f.published:
			if len(p.Selection().Notes) != 2 {
				t.Error("initial note preview changed")
			}
		default:
			t.Error("initial feedback not published")
		}
		for _, op := range []string{"begin", "ack"} {
			if _, err = m.request(context.Background(), op, sub.ID, target); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		select {
		case mask := <-f.masks:
			if len(mask.Notes) != 1 {
				t.Errorf("confirmed mask=%v", mask.Notes)
			}
		default:
			t.Error("confirmed ACK did not update live display")
		}
		select {
		case p := <-f.published:
			notes := p.Selection().Notes
			if len(notes) != 1 || notes[0].ID != "unsent" {
				t.Errorf("preview did not use selection after mask application: %v", notes)
			}
		default:
			t.Error("remaining feedback not published")
		}
		cancel()
		synctest.Wait()
		<-done
	})
}
