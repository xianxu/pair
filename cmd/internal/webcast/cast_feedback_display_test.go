package webcast

import (
	"path/filepath"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
)

func TestCastDeliveredFeedbackIsScopedAndDoesNotResolveHistory(t *testing.T) {
	c, _, _, writes := castSubmissionFixture(t)
	selected, err := c.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Notes) != 1 {
		t.Fatal(selected)
	}
	note := selected.Notes[0]
	confirmed := core.DeliveredFeedback{RecordingID: "other", Notes: map[core.PreviewNote]struct{}{{ID: note.ID, Revision: note.Revision}: {}}}
	if err = c.SetDeliveredFeedback(confirmed); err == nil {
		t.Fatal("foreign recording note mask admitted")
	}
	if len(c.board.FeedbackMask().Notes) != 0 {
		t.Fatal("foreign note hidden")
	}
	confirmed.RecordingID = filepath.Base(c.RecordingPath())
	before := *writes
	if err = c.SetDeliveredFeedback(confirmed); err != nil {
		t.Fatal(err)
	}
	if *writes != before {
		t.Fatal("display projection wrote recording history")
	}
	if len(c.board.FeedbackMask().Notes) != 1 || len(c.board.OperatorView().Notes) != 0 {
		t.Fatal("confirmed exact note remains on live display")
	}
}
