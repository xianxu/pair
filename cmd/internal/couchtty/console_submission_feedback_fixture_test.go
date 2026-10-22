package couchtty

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// Transaction fakes expose the same preview seam as their capture wrappers.
// Neither construction nor interface detection acknowledges any feedback.
func (f *submissionCastFake) ReadSubmissionVisibility(ctx context.Context, host core.PreviewHost) (core.FeedbackVisibility, error) {
	if f.feedback == nil {
		return core.FeedbackVisibility{}, errors.New("no completed paint")
	}
	return f.feedback.ReadSubmissionVisibility(ctx, host)
}
func (f *submissionCastFake) ResolveSubmissionPreview(ctx context.Context, host core.PreviewHost, selection core.PreviewSelection) (core.PreviewSelection, error) {
	if f.feedback == nil {
		return core.PreviewSelection{}, errors.New("no completed paint")
	}
	return f.feedback.ResolveSubmissionPreview(ctx, host, selection)
}
func (f *submissionCastFake) FreezeSubmissionPreview(host core.PreviewHost, selection core.PreviewSelection) (board.Selection, error) {
	if f.feedback == nil {
		return board.Selection{}, errors.New("test feedback has not been published")
	}
	return f.feedback.FreezeSubmissionPreview(host, selection)
}
func (f *submissionCastFake) SubmissionCandidateUpdates() (<-chan struct{}, func()) {
	if f.feedback == nil {
		return nil, func() {}
	}
	return f.feedback.SubmissionCandidateUpdates()
}
func (f *submissionCastFake) SubmissionPreviewUpdates() (<-chan struct{}, func()) {
	if f.feedback == nil {
		return nil, func() {}
	}
	return f.feedback.SubmissionPreviewUpdates()
}
func (f *submissionCastFake) SubmissionPreviewHost(ctx context.Context) (core.PreviewHost, <-chan struct{}, error) {
	if f.feedback == nil {
		return core.PreviewHost{}, nil, errors.New("test feedback has not been published")
	}
	return f.feedback.SubmissionPreviewHost(ctx)
}
func (f *submissionCastFake) PublishSubmissionPreview(ctx context.Context, p core.PreviewPublication) error {
	if f.feedback == nil {
		return errors.New("test feedback has not been published")
	}
	return f.feedback.PublishSubmissionPreview(ctx, p)
}

// acknowledgeSubmissionFeedback exercises the real publication state machine
// synchronously. Transaction tests own their scheduler, so this fixture starts
// no background workers. The exact current cast (including capture wrappers)
// remains the run's identity and keeps its optional interfaces intact.
func acknowledgeSubmissionFeedback(t *testing.T, m *submissionManager, base *submissionCastFake) {
	t.Helper()
	actual, err := m.current()
	if err != nil {
		t.Fatal(err)
	}
	cast, ok := actual.(submissionPreviewCast)
	if !ok {
		t.Fatal("transaction cast lacks preview interface")
	}
	selected, err := cast.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	f := newSubmissionPreviewFake(base)
	f.selection = selected
	f.setVisibleFrame()
	base.feedback = f
	ctx, cancel := context.WithTimeout(m.ctx, time.Second)
	defer cancel()
	host, _, err := cast.SubmissionPreviewHost(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var candidates []core.PreviewSketchCandidate
	for _, v := range selected.Sketches {
		candidates = append(candidates, core.PreviewSketchCandidate{Item: core.PreviewItem{RecordingID: filepath.Base(cast.RecordingPath()), SketchID: v.ID, Sequence: v.NextSequence, Digest: v.Digest}, Source: v.Source, Finish: v.Finish})
	}
	var notes []core.PreviewNote
	for _, n := range selected.Notes {
		notes = append(notes, core.PreviewNote{ID: n.ID, Revision: n.Revision})
	}
	m.mu.Lock()
	deliveryGeneration := m.deliveredGeneration
	m.mu.Unlock()
	selection, err := core.SelectVisibleFeedback(candidates, notes, core.DeliveredFeedback{}, core.PreviewStamp{Board: selected.SketchGeneration, Delivered: deliveryGeneration}, f.visibility)
	if err != nil {
		t.Fatal(err)
	}
	selection, err = cast.ResolveSubmissionPreview(ctx, host, selection)
	if err != nil {
		t.Fatal(err)
	}
	run := m.newSubmissionPreviewRun(cast, func() {})
	// The synchronous publisher is the entire owned execution scope. Mark it
	// joined before returning so replacing/stopping the preview cannot wait on
	// a worker that was never scheduled.
	defer close(run.done)
	run.host = host
	m.mu.Lock()
	m.previewState, _ = core.NextPreview(m.previewState, core.PreviewHostChanged{Host: host})
	m.previewState, _ = core.NextPreview(m.previewState, core.PreviewCandidatesChanged{Selection: selection})
	var effects []core.PreviewEffect
	m.previewState, effects = core.NextPreview(m.previewState, core.PreviewRefresh{})
	m.mu.Unlock()
	var publication *core.PreviewPublication
	for _, effect := range effects {
		if p, ok := effect.(core.PreviewPublish); ok {
			copy := p.Publication
			publication = &copy
		}
	}
	if publication == nil {
		t.Fatal("feedback selection did not request publication")
	}
	if err := cast.PublishSubmissionPreview(ctx, *publication); err != nil {
		t.Fatal(err)
	}
	select {
	case p := <-f.published:
		if p.Operation() != publication.Operation() {
			t.Fatal("publisher acknowledged another operation")
		}
	default:
		t.Fatal("publisher did not observe feedback")
	}
	m.mu.Lock()
	m.previewState, _ = core.NextPreview(m.previewState, core.PreviewPublishAccepted{Publication: *publication})
	shown := m.previewState.Phase() == core.PreviewShown
	m.mu.Unlock()
	if !shown {
		t.Fatal("feedback publication not acknowledged")
	}
	t.Cleanup(func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.previewRun == run {
			m.previewState, _ = core.NextPreview(m.previewState, core.PreviewDispose{})
			m.previewRun = nil
		}
	})
}
