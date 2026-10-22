package webcast

import (
	"fmt"
	"path/filepath"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

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
