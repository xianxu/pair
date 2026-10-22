package board

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func feedbackDisplayFixture(t *testing.T) *Board {
	t.Helper()
	b := New(nil, func(recording.Record, int) error { t.Fatal("display mask wrote a recording event"); return nil })
	for _, r := range submissionSketchRecords(2) {
		b.Put(r)
	}
	for i := 0; i < 2; i++ {
		n := &recording.Note{ID: fmt.Sprintf("note%d", i), Text: "annotation", Author: recording.Author{ID: "operator", Color: "#ff0000"}, At: recording.NoteAt{Node: 1, Box: &recording.Box{10, 10, 20, 20}}, Region: &recording.Box{10, 10, 20, 20}}
		n.SetTime(b.st.T)
		b.Put(n)
	}
	if b.broken != nil {
		t.Fatal(b.broken)
	}
	return b
}

func TestFeedbackDisplayBudgetReservesExactWireEnvelope(t *testing.T) {
	mask := LiveFeedbackMask{Scope: "scope<&>", Notes: []FeedbackNoteVersion{{ID: "note<&>", Revision: math.MaxUint64}}, Sketches: []FeedbackSketchVersion{}}
	worst := mask
	worst.Revision = math.MaxUint64
	data, err := json.Marshal(worst)
	if err != nil {
		t.Fatal(err)
	}
	if err := feedbackMaskBudget(mask, len(data)); err != nil {
		t.Fatalf("exact reserved wire budget refused: %v", err)
	}
	if err := feedbackMaskBudget(mask, len(data)-1); !errors.Is(err, ErrFeedbackMaskLimit) {
		t.Fatalf("missing worst revision reserve: %v", err)
	}
	if MaxFeedbackMaskBytes != 64<<20 {
		t.Fatal("feedback budget diverged from bundle envelope")
	}
}

func TestFeedbackDisplayBudgetRefusalAtomicAndNoopPreservesPrevious(t *testing.T) {
	b := feedbackDisplayFixture(t)
	s, _ := b.Selection()
	if err := b.SetDeliveredFeedback(nil, s.Notes[:1]); err != nil {
		t.Fatal(err)
	}
	before := b.FeedbackMask()
	wake, stop := b.SubscribeFeedbackMask()
	defer stop()
	<-wake
	operator, stopOperator := b.SubscribeOperator()
	defer stopOperator()
	<-operator
	candidates, stopCandidates := b.SubscribeSubmissionCandidates()
	defer stopCandidates()
	stamp := b.submissionGeneration
	worst := before
	worst.Revision = math.MaxUint64
	data, err := json.Marshal(worst)
	if err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	err = b.setDeliveredFeedbackLocked(nil, s.Notes, len(data))
	b.mu.Unlock()
	if !errors.Is(err, ErrFeedbackMaskLimit) {
		t.Fatalf("oversized full mask accepted: %v", err)
	}
	if !reflect.DeepEqual(before, b.FeedbackMask()) || stamp != b.submissionGeneration {
		t.Fatal("budget refusal changed mask or candidates")
	}
	b.mu.Lock()
	err = b.setDeliveredFeedbackLocked(nil, s.Notes[:1], len(data))
	b.mu.Unlock()
	if err != nil {
		t.Fatalf("exact budget no-op refused: %v", err)
	}
	for name, ch := range map[string]<-chan LiveFeedbackMask{"mask": wake} {
		select {
		case <-ch:
			t.Fatalf("%s notified on refusal/noop", name)
		default:
		}
	}
	select {
	case <-operator:
		t.Fatal("operator notified on refusal/noop")
	default:
	}
	select {
	case <-candidates:
		t.Fatal("candidates notified on refusal/noop")
	default:
	}
}

func TestFeedbackDisplayAccepts49HistoricalOpenNotes(t *testing.T) {
	r := newRig(t)
	for i := 0; i < 49; i++ {
		r.page(recording.SegLive)
		if _, err := r.note(r.p1.ID, 0); err != nil {
			t.Fatal(err)
		}
	}
	s, err := r.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Notes) != 49 {
		t.Fatalf("historical note fixture=%d", len(s.Notes))
	}
	if err := r.bd.SetDeliveredFeedback(nil, s.Notes); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.FeedbackMask().Notes) != 49 || len(r.bd.OperatorView().Notes) != 0 {
		t.Fatal("historical notes were capped at live-page admission bound")
	}
}

func TestFeedbackDisplayMaskSelectiveOwnedAndNoRecordingMutation(t *testing.T) {
	b := feedbackDisplayFixture(t)
	before, err := b.Selection()
	if err != nil {
		t.Fatal(err)
	}
	inkBefore, _ := b.OperatorSketchSince(SketchDeliveryCursor{})
	channel, stop := b.SubscribeFeedbackMask()
	defer stop()
	initial := <-channel
	if initial.Scope == "" || initial.Revision != 0 || len(initial.Sketches) != 0 || len(initial.Notes) != 0 {
		t.Fatalf("initial full mask %+v", initial)
	}
	operator, stopOperator := b.SubscribeOperator()
	defer stopOperator()
	<-operator
	sketches := append([]SelectedSketch(nil), before.Sketches[:1]...)
	notes := append([]SelectedNote(nil), before.Notes[:1]...)
	if err := b.SetDeliveredFeedback(sketches, notes); err != nil {
		t.Fatal(err)
	}
	mask := <-channel
	if mask.Scope != initial.Scope || mask.Revision != 1 || len(mask.Sketches) != 1 || mask.Sketches[0].ID != sketches[0].ID || len(mask.Notes) != 1 || mask.Notes[0].ID != notes[0].ID {
		t.Fatalf("wrong selective mask %+v", mask)
	}
	view := <-operator
	if view.FeedbackMask.Revision != 1 || len(view.Notes) != 1 || view.Notes[0].ID != before.Notes[1].ID {
		t.Fatalf("operator projection retained masked note %+v", view)
	}
	after, _ := b.Selection()
	if len(after.Notes) != 1 || after.Notes[0] != before.Notes[1] || !reflect.DeepEqual(after.Sketches, before.Sketches) {
		t.Fatalf("wrong candidate filtering %+v", after)
	}
	inkAfter, _ := b.OperatorSketchSince(SketchDeliveryCursor{})
	if !reflect.DeepEqual(inkBefore.Bootstrap.Display, inkAfter.Bootstrap.Display) || inkBefore.Bootstrap.Seq != inkAfter.Bootstrap.Seq || b.st.Notes[0].State != recording.NoteOpen {
		t.Fatal("display mutation changed recorded history/ink/note state")
	}
	sketches[0].ID = "caller mutation"
	notes[0].ID = "caller mutation"
	mask.Sketches[0].ID = "subscriber mutation"
	view.FeedbackMask.Notes[0].ID = "operator mutation"
	fresh := b.FeedbackMask()
	if fresh.Sketches[0].ID != before.Sketches[0].ID || fresh.Notes[0].ID != before.Notes[0].ID {
		t.Fatal("mask aliases inputs or projections")
	}
	// Exact same set is a no-op, even with duplicate inputs.
	if err := b.SetDeliveredFeedback([]SelectedSketch{before.Sketches[0], before.Sketches[0]}, before.Notes[:1]); err != nil {
		t.Fatal(err)
	}
	select {
	case <-channel:
		t.Fatal("no-op mask notified")
	default:
	}
	select {
	case <-operator:
		t.Fatal("no-op mask refreshed operator")
	default:
	}
	if b.FeedbackMask().Revision != 1 {
		t.Fatal("no-op advanced revision")
	}
}

func TestFeedbackDisplayNewerVersionsRemainVisibleAndUpdatesCoalesce(t *testing.T) {
	b := feedbackDisplayFixture(t)
	s, _ := b.Selection()
	updates, stop := b.SubscribeFeedbackMask()
	defer stop()
	<-updates
	if err := b.SetDeliveredFeedback(s.Sketches, s.Notes); err != nil {
		t.Fatal(err)
	}
	edit := &recording.NoteEdit{ID: s.Notes[0].ID, Text: "new unsent revision"}
	edit.SetTime(b.st.T)
	b.Put(edit)
	current, _ := b.Selection()
	if len(current.Notes) != 1 || current.Notes[0].ID != s.Notes[0].ID || current.Notes[0].Revision == s.Notes[0].Revision {
		t.Fatalf("new revision hidden %+v", current)
	}
	stale := append([]SelectedSketch(nil), s.Sketches...)
	stale[0].NextSequence++
	stale[1].Digest = "different"
	if err := b.SetDeliveredFeedback(stale, s.Notes); err != nil {
		t.Fatal(err)
	}
	mask := <-updates
	if len(mask.Sketches) != 0 || len(mask.Notes) != 1 || mask.Notes[0].ID != s.Notes[1].ID || mask.Revision != b.FeedbackMask().Revision {
		t.Fatalf("stale versions retained/coalescing lost latest %+v", mask)
	}
	encoded, err := json.Marshal(mask)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Revision string `json:"revision"`
		Notes    []struct {
			Revision string `json:"revision"`
		} `json:"notes"`
	}
	if err := json.Unmarshal(encoded, &wire); err != nil || wire.Revision == "" || wire.Notes[0].Revision != "0" {
		t.Fatalf("uint64 mask wire identity: %s %v", encoded, err)
	}
	b.Close()
	if _, open := <-updates; open {
		t.Fatal("mask subscription survived close")
	}
	if err := b.SetDeliveredFeedback(nil, nil); err != ErrClosed {
		t.Fatalf("closed mutation=%v", err)
	}
}

func TestFeedbackDisplayMatcherVisitsInputsAndCandidatesOnce(t *testing.T) {
	for _, n := range []int{32, 64} {
		var current, delivered []SelectedSketch
		var notes []SelectedNote
		for i := 0; i < n; i++ {
			v := SelectedSketch{ID: fmt.Sprint(i), NextSequence: 5, Digest: "exact"}
			current = append(current, v)
			delivered = append(delivered, v)
			notes = append(notes, SelectedNote{ID: fmt.Sprint(i)})
		}
		mask, visits := matchDeliveredFeedback(current, notes, delivered, notes)
		if len(mask.Sketches) != n || len(mask.Notes) != n || visits != 4*n {
			t.Fatalf("n=%d visits=%d sketch=%d notes=%d", n, visits, len(mask.Sketches), len(mask.Notes))
		}
	}
}

func TestFeedbackDisplayMatcherKeepsExactMatchAmongOlderDeliveredVersions(t *testing.T) {
	current := SelectedSketch{ID: "shape", NextSequence: 10, Digest: "new"}
	older := SelectedSketch{ID: "shape", NextSequence: 5, Digest: "old"}
	note := SelectedNote{ID: "note", Revision: 2}
	mask, _ := matchDeliveredFeedback([]SelectedSketch{current}, []SelectedNote{note}, []SelectedSketch{current, older}, []SelectedNote{note, {ID: "note", Revision: 1}})
	if len(mask.Sketches) != 1 || mask.Sketches[0].NextSequence != 10 || len(mask.Notes) != 1 || mask.Notes[0].Revision != 2 {
		t.Fatalf("older delivery displaced exact current identity: %+v", mask)
	}
}

func TestFeedbackDisplayDeliveryMetadataTracksEveryAcceptedOperation(t *testing.T) {
	b := New(nil, nil)
	cursor := b.OperatorSketchCursor()
	for _, r := range submissionSketchRecords(1) {
		b.Put(r)
		if recording.SketchReceiptKey(r) == "" {
			continue
		}
		metadata, ok := b.st.Sketches.Metadata("a")
		if !ok {
			t.Fatal("missing metadata")
		}
		frame, err := b.OperatorSketchSince(cursor)
		if err != nil || frame.Batch == nil || len(frame.Batch.Patches) != 1 {
			t.Fatalf("operation %T delivery %v %+v", r, err, frame)
		}
		var patch struct {
			Header struct {
				NextSequence string `json:"next_sequence"`
				Digest       string `json:"digest"`
			} `json:"header"`
		}
		if err := json.Unmarshal(frame.Batch.Patches[0], &patch); err != nil {
			t.Fatal(err)
		}
		if patch.Header.NextSequence != fmt.Sprint(metadata.NextSequence) || patch.Header.Digest != metadata.Digest || metadata.Digest == "" {
			t.Fatalf("operation %T stale exact version: %+v metadata=%+v", r, patch, metadata)
		}
		cursor = SketchDeliveryCursor{Epoch: frame.Batch.Epoch, Seq: frame.Batch.To}
	}
	bootstrap, err := b.OperatorSketchSince(SketchDeliveryCursor{})
	if err != nil {
		t.Fatal(err)
	}
	v := bootstrap.Bootstrap.Display.Sketches[0]
	metadata, _ := b.st.Sketches.Metadata(v.Begin.ID)
	if v.NextSequence != metadata.NextSequence || v.Digest != metadata.Digest {
		t.Fatalf("bootstrap version differs %+v %+v", v, metadata)
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		NextSequence string `json:"next_sequence"`
	}
	if err := json.Unmarshal(data, &wire); err != nil || wire.NextSequence != "5" {
		t.Fatalf("bootstrap sequence wire %s %v", data, err)
	}
}
