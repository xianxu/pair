package board

import (
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func submissionFeedback(t *testing.T, b *Board) (bool, error) {
	t.Helper()
	return b.HasSubmissionFeedback()
}
func requireSubmissionFeedback(t *testing.T, b *Board, want bool) {
	t.Helper()
	got, err := submissionFeedback(t, b)
	if err != nil || got != want {
		t.Fatalf("feedback=%v, %v; want %v", got, err, want)
	}
}

func TestHasSubmissionFeedbackNotesUseCurrentOpenRevision(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	requireSubmissionFeedback(t, r.bd, false)
	id, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	requireSubmissionFeedback(t, r.bd, true)
	selected, err := r.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.bd.SetDeliveredFeedback(nil, selected.Notes); err != nil {
		t.Fatal(err)
	}
	requireSubmissionFeedback(t, r.bd, false)
	for _, step := range []struct {
		record recording.Record
		want   bool
	}{
		{&recording.NoteEdit{ID: id, Text: "new revision"}, true},
		{&recording.NoteState{ID: id, State: recording.NoteResolved}, false},
		{&recording.NoteState{ID: id, State: recording.NoteOpen}, true},
	} {
		if err := r.w.Emit(step.record, r.bd.st.DocSeg); err != nil {
			t.Fatal(err)
		}
		requireSubmissionFeedback(t, r.bd, step.want)
	}
}

func TestHasSubmissionFeedbackRequiresFinishedLabelAndExactDeliveryMask(t *testing.T) {
	b := New(nil, nil)
	records := submissionSketchRecords(1)
	for _, r := range records[:len(records)-1] {
		b.Put(r)
	}
	requireSubmissionFeedback(t, b, false)
	b.Put(records[len(records)-1])
	requireSubmissionFeedback(t, b, true)
	selection, err := b.Selection()
	if err != nil {
		t.Fatal(err)
	}
	exact := selection.Sketches[0]
	for _, delivered := range []SelectedSketch{
		{ID: exact.ID, NextSequence: exact.NextSequence - 1, Digest: "older"},
		{ID: exact.ID, NextSequence: exact.NextSequence, Digest: "wrong"},
		{ID: exact.ID, NextSequence: exact.NextSequence + 1, Digest: "future"},
	} {
		if err := b.SetDeliveredFeedback([]SelectedSketch{delivered}, nil); err != nil {
			t.Fatal(err)
		}
		requireSubmissionFeedback(t, b, true)
	}
	wake, stop := b.SubscribeSubmissionCandidates()
	defer stop()
	if err := b.SetDeliveredFeedback([]SelectedSketch{exact}, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case <-wake:
	default:
		t.Fatal("delivery mask did not wake feedback projection")
	}
	requireSubmissionFeedback(t, b, false)
	// An unchanged confirmed mask emits no repeated wake.
	if err := b.SetDeliveredFeedback([]SelectedSketch{exact}, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case <-wake:
		t.Fatal("unchanged mask woke projection")
	default:
	}
}

func TestHasSubmissionFeedbackRefusesUnavailableBoard(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*Board)
		want  error
	}{
		{"not-live", func(*Board) {}, ErrNotLive},
		{"closed", func(b *Board) { b.Close() }, ErrClosed},
		{"broken", func(b *Board) { b.broken = errors.New("broken fixture") }, ErrBroken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := New(nil, nil)
			tc.setup(b)
			got, err := submissionFeedback(t, b)
			if got || !errors.Is(err, tc.want) {
				t.Fatalf("feedback=%v err=%v, want %v", got, err, tc.want)
			}
		})
	}
}
