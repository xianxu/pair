package webcast

import (
	"errors"
	"path/filepath"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
)

func castSubmissionFeedback(t *testing.T, c *Cast) (bool, error) {
	t.Helper()
	return c.HasSubmissionFeedback()
}

func TestCastHasSubmissionFeedbackUsesLiveBoardMask(t *testing.T) {
	c, _, _, writes := castSubmissionFixture(t)
	got, err := castSubmissionFeedback(t, c)
	if err != nil || !got {
		t.Fatalf("feedback=%v err=%v", got, err)
	}
	selected, err := c.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	note := selected.Notes[0]
	before := *writes
	err = c.SetDeliveredFeedback(core.DeliveredFeedback{RecordingID: filepath.Base(c.RecordingPath()), Notes: map[core.PreviewNote]struct{}{{ID: note.ID, Revision: note.Revision}: {}}})
	if err != nil {
		t.Fatal(err)
	}
	got, err = castSubmissionFeedback(t, c)
	if err != nil || got {
		t.Fatalf("delivered feedback=%v err=%v", got, err)
	}
	if *writes != before {
		t.Fatal("feedback projection changed recording")
	}
}

func TestCastHasSubmissionFeedbackRejectsEndedAndUnrecordedCasts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Cast)
		want   error
	}{
		{"no-board", func(c *Cast) { c.board = nil }, ErrNotRecording},
		{"no-recording", func(c *Cast) { c.recDir = "" }, ErrNotRecording},
		{"ended", func(c *Cast) { close(c.done) }, ErrEnded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, _, _ := castSubmissionFixture(t)
			tc.change(c)
			got, err := castSubmissionFeedback(t, c)
			if got || !errors.Is(err, tc.want) {
				t.Fatalf("feedback=%v err=%v, want %v", got, err, tc.want)
			}
		})
	}
}
