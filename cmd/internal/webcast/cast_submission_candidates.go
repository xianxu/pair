package webcast

import (
	"context"
	"fmt"
	"path/filepath"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// HasSubmissionFeedback reads only this cast's live Board projection. It does
// not inspect the submission store, browser paint or recording files.
func (c *Cast) HasSubmissionFeedback() (bool, error) {
	if c.board == nil || c.recDir == "" {
		return false, ErrNotRecording
	}
	select {
	case <-c.done:
		return false, ErrEnded
	default:
	}
	return c.board.HasSubmissionFeedback()
}

func (c *Cast) SubmissionCandidateUpdates() (<-chan struct{}, func()) {
	if c.board == nil {
		ch := make(chan struct{})
		close(ch)
		return ch, func() {}
	}
	return c.board.SubscribeSubmissionCandidates()
}

// SetDeliveredFeedback updates only this recording's live display projection.
// Existing manifests remain the delivery authority; no recording event is made.
func (c *Cast) SetDeliveredFeedback(delivered core.DeliveredFeedback) error {
	if delivered.RecordingID == "" || delivered.RecordingID != filepath.Base(c.RecordingPath()) {
		return fmt.Errorf("feedback display recording changed")
	}
	if c.board == nil {
		return nil
	}
	sketches := make([]board.SelectedSketch, 0, len(delivered.Sketches))
	for key := range delivered.Sketches {
		if key.RecordingID != delivered.RecordingID {
			return fmt.Errorf("feedback sketch recording changed")
		}
		sketches = append(sketches, board.SelectedSketch{ID: key.SketchID, NextSequence: key.Sequence, Digest: key.Digest})
	}
	notes := make([]board.SelectedNote, 0, len(delivered.Notes))
	for key := range delivered.Notes {
		notes = append(notes, board.SelectedNote{ID: key.ID, Revision: key.Revision})
	}
	return c.board.SetDeliveredFeedback(sketches, notes)
}

// ReadSubmissionVisibility reads the completed paint from the private console.
func (c *Cast) ReadSubmissionVisibility(ctx context.Context, host core.PreviewHost) (core.FeedbackVisibility, error) {
	if browser, ok := c.browser.(interface {
		ReadSubmissionVisibility(context.Context, core.PreviewHost) (core.FeedbackVisibility, error)
	}); ok {
		return browser.ReadSubmissionVisibility(ctx, host)
	}
	return core.FeedbackVisibility{}, ErrNotRecording
}

// ResolveSubmissionPreview binds painted locators through the current recorder.
func (c *Cast) ResolveSubmissionPreview(ctx context.Context, host core.PreviewHost, selection core.PreviewSelection) (core.PreviewSelection, error) {
	if browser, ok := c.browser.(interface {
		ResolveSubmissionPreview(context.Context, core.PreviewHost, core.PreviewSelection) (core.PreviewSelection, error)
	}); ok {
		return browser.ResolveSubmissionPreview(ctx, host, selection)
	}
	return core.PreviewSelection{}, ErrNotRecording
}
