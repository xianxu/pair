package board

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestOperatorParserClosedShapes(t *testing.T) {
	good := []string{
		`{"generation":"1","op":"prepare-note","request":"r1","body":{"seg":1,"path":[0,-1,2],"box":[0,0,10,10],"region":[1,1,5,5]}}`,
		`{"generation":"1","op":"save-note","request":"r2","body":{"draft":"d1","color":0,"text":"hello"}}`,
		`{"generation":"1","op":"stroke","request":"r3","body":{"seg":1,"id":"s","color":0,"pts":[[0,0,0]]}}`,
		`{"generation":"1","op":"resolve","request":"r4","body":{"seg":1,"id":"n","state":"resolved"}}`,
		`{"generation":"1","op":"cancel-draft","request":"r5","body":{"draft":"d1"}}`,
		`{"generation":"1","op":"status","request":"r2","body":{}}`,
	}
	for _, s := range good {
		if _, err := ParseOperatorRequest([]byte(s)); err != nil {
			t.Errorf("valid %s: %v", s, err)
		}
	}
	bad := []string{`null`, `{}`, `{"generation":"1","op":"status","request":"r","body":{"author":"operator"}}`, `{"generation":"1","op":"prepare-note","request":"r","body":{"seg":1,"path":[-2],"box":[0,0,1,1],"region":[0,0,1,1]}}`, `{"generation":"1","op":"stroke","request":"r","body":{"seg":1,"id":"s","color":0,"pts":[[0,0,null]]}}`, `{"generation":"1","op":"save-note","request":"r","body":{"draft":"d","color":0,"text":" "}}`}
	for _, s := range bad {
		if _, err := ParseOperatorRequest([]byte(s)); err == nil {
			t.Errorf("accepted %s", s)
		}
	}
}
func TestOperatorSettlementAndReservation(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	var settle func(OperatorOutcome, error)
	writes := 0
	s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
		writes++
		settle = done
		return nil
	})
	in := NoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 20, 20}, Region: recording.Box{1, 1, 10, 10}, Color: 0, Text: "hi"}
	first := s.Note("1", in)
	if first.Outcome != OperatorPending || len(r.bd.pending) != 1 {
		t.Fatalf("first %+v pending %v", first, r.bd.pending)
	}
	if again := s.Note("1", in); again != first || writes != 1 {
		t.Fatalf("duplicate %+v writes %d", again, writes)
	}
	if busy := s.Note("2", in); busy.Outcome != OperatorRefused || writes != 1 {
		t.Fatalf("busy %+v", busy)
	}
	settle(OperatorRefused, ErrStale)
	settle(OperatorAccepted, nil)
	if got := s.Status("1"); got.Outcome != OperatorRefused || len(r.bd.pending) != 0 {
		t.Fatalf("settled %+v pending %v", got, r.bd.pending)
	}
	if next := s.Note("3", in); next.Outcome != OperatorPending {
		t.Fatalf("next %+v", next)
	}
	s.Close()
	if got := s.Status("3"); got.Outcome != OperatorUnknown {
		t.Fatalf("closed %+v", got)
	}
}
func TestOperatorWritesIdentityAndAdmission(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
		err := r.w.Emit(rec, doc)
		if err != nil {
			done(OperatorRefused, err)
		} else {
			done(OperatorAccepted, nil)
		}
		return nil
	})
	got := s.Stroke("1", StrokeIn{Viewer: ana, Seg: r.seg(), ID: "pen", Color: 1, Pts: [][3]float64{{1, 2, 0}}})
	if got.Outcome != OperatorAccepted {
		t.Fatal(got)
	}
	if r.kinds("stroke")[0].(*recording.Stroke).Author.ID != Operator {
		t.Fatal("claimed viewer leaked")
	}
	r.bd.mu.Lock()
	r.bd.st.Control.Driver = &recording.Author{ID: "ana", Name: "Ana"}
	r.bd.mu.Unlock()
	got = s.Stroke("2", StrokeIn{Seg: r.seg(), ID: "pen", Color: 1, Pts: [][3]float64{{1, 2, 0}}})
	if got.Outcome != OperatorRefused || got.Error != ErrDriving.Error() {
		t.Fatal(got)
	}
}
func TestOperatorViewLatestAndCopies(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	ch, stop := r.bd.SubscribeOperator()
	defer stop()
	_, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 140; i++ {
		r.add(&recording.Stroke{ID: "s", Author: recording.Author{ID: "a"}, Color: "red", Pts: [][3]float64{{1, 2, 0}}})
	}
	v := <-ch
	if len(v.Notes) != 1 || len(v.Strokes) != 128 || v.Strokes[0].Seq >= v.Strokes[127].Seq {
		t.Fatalf("view notes %d strokes %d", len(v.Notes), len(v.Strokes))
	}
	v.Notes[0].Text = "mutated"
	v.Strokes[0].Pts[0][0] = 999
	fresh := r.bd.OperatorView()
	if fresh.Notes[0].Text == "mutated" || fresh.Strokes[0].Pts[0][0] == 999 {
		t.Fatal("view aliases board")
	}
	r.b.At(12000)
	r.add(&recording.Stroke{ID: "later", Author: recording.Author{ID: "a"}, Color: "red", Pts: [][3]float64{{1, 2, 0}}})
	if got := r.bd.OperatorView(); len(got.Strokes) != 1 {
		t.Fatalf("expired strokes %d", len(got.Strokes))
	}
}

func TestOperatorIDsBoundedAndPayloadBound(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	writes := 0
	s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
		writes++
		done(OperatorAccepted, nil)
		return nil
	})
	in := StrokeIn{Seg: r.seg(), ID: "s", Color: 0, Pts: [][3]float64{{1, 2, 0}}}
	for _, id := range []string{"0", "01", "-1", "9007199254740992", "abc"} {
		if got := s.Stroke(id, in); got.Outcome != OperatorRefused {
			t.Fatalf("id %s %+v", id, got)
		}
	}
	if got := s.Stroke("1", in); got.Outcome != OperatorAccepted {
		t.Fatal(got)
	}
	changed := in
	changed.ID = "other"
	if got := s.Stroke("1", changed); got.Outcome != OperatorRefused {
		t.Fatal("same ID accepted changed payload")
	}
	for i := 2; i <= 1000; i++ {
		if got := s.Stroke(strconv.Itoa(i), in); got.Outcome != OperatorAccepted {
			t.Fatal(got)
		}
	}
	if got := s.Stroke("1", in); got.Outcome != OperatorRefused || writes != 1000 {
		t.Fatalf("old replay %+v writes %d", got, writes)
	}
}
func TestOperatorUnknownBlocksAndAsyncRaces(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	var done func(OperatorOutcome, error)
	s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, settle func(OperatorOutcome, error)) error {
		done = settle
		return nil
	})
	in := NoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 10, 10}, Region: recording.Box{1, 1, 10, 10}, Color: 0, Text: "hi"}
	s.Note("1", in)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			done(OperatorUnknown, errors.New("sink failed"))
			s.Status("1")
			s.Note("1", in)
		}()
	}
	wg.Wait()
	if got := s.Note("2", in); got.Outcome != OperatorRefused || got.Error != ErrOperatorBusy.Error() {
		t.Fatal(got)
	}
	if got := s.Status("1"); got.Outcome != OperatorUnknown {
		t.Fatal(got)
	}
}
func TestOperatorPauseStaleAndResolve(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
		err := r.w.Emit(rec, doc)
		if err != nil {
			done(OperatorRefused, err)
		} else {
			done(OperatorAccepted, nil)
		}
		return nil
	})
	in := NoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 10, 10}, Region: recording.Box{1, 1, 10, 10}, Color: 0, Text: "hi"}
	got := s.Note("1", in)
	if got.Outcome != OperatorAccepted {
		t.Fatal(got)
	}
	id := got.ID
	if got = s.SetState("2", r.seg(), StateIn{ID: id, State: recording.NoteResolved}); got.Outcome != OperatorAccepted {
		t.Fatal(got)
	}
	if len(r.bd.OperatorView().Notes) != 0 {
		t.Fatal("resolved note projected")
	}
	r.add(&recording.Stop{Why: recording.StopPause})
	if got = s.Note("3", in); got.Error != ErrPaused.Error() {
		t.Fatal(got)
	}
	r.page(recording.SegLive)
	if got = s.Note("4", in); got.Error != ErrStale.Error() {
		t.Fatal(got)
	}
	if len(r.bd.OperatorView().Strokes) != 0 {
		t.Fatal("old strokes crossed doc")
	}
}
func TestOperatorViewWallTimeAndClose(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.add(&recording.Stroke{ID: "s", Author: recording.Author{ID: "a"}, Pts: [][3]float64{{0, 0, 0}}})
	r.bd.mu.Lock()
	r.bd.operatorStrokes[0].received = time.Now().Add(-time.Minute)
	r.bd.mu.Unlock()
	if len(r.bd.OperatorView().Strokes) != 0 {
		t.Fatal("wall-time expired stroke remains")
	}
	ch, cancel := r.bd.SubscribeOperator()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); cancel() }()
	go func() { defer wg.Done(); r.bd.Close() }()
	wg.Wait()
	for range ch {
	}
	if !r.bd.OperatorView().Ended {
		t.Fatal("closed board view live")
	}
}
func FuzzOperatorParser(f *testing.F) {
	f.Add([]byte(`{"generation":"1","op":"status","request":"1","body":{}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		v, err := ParseOperatorRequest(b)
		if err == nil {
			if !idPattern.MatchString(v.Request) {
				t.Fatal("bad admitted request")
			}
			if v.Stroke != nil && !strokeShape(*v.Stroke) {
				t.Fatal("bad stroke")
			}
		}
	})
}

func TestOperatorParserRejectsAmbiguousKeys(t *testing.T) {
	for _, raw := range []string{
		`{"generation":"1","op":"status","op":"status","request":"1","body":{}}`,
		`{"Op":"status","request":"1","body":{}}`,
		`{"generation":"1","op":"save-note","request":"1","body":{"draft":"d","color":0,"text":"one","text":"two"}}`,
		`{"generation":"1","op":"save-note","request":"1","body":{"Draft":"d","color":0,"text":"one"}}`,
	} {
		if _, err := ParseOperatorRequest([]byte(raw)); err == nil {
			t.Errorf("accepted ambiguous input: %s", raw)
		}
	}
}
func TestOperatorViewPruningAdvancesRevision(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.add(&recording.Stroke{ID: "s", Author: recording.Author{ID: "a"}, Pts: [][3]float64{{0, 0, 0}}})
	old := r.bd.OperatorView()
	r.bd.mu.Lock()
	r.bd.operatorStrokes[0].received = time.Now().Add(-time.Minute)
	r.bd.mu.Unlock()
	if next := r.bd.OperatorView(); next.Revision <= old.Revision {
		t.Fatalf("changed view retained revision %d", next.Revision)
	}
}

func TestOperatorUnknownNoteReconcilesOnlyObservedStableID(t *testing.T) {
	for _, observed := range []bool{false, true} {
		name := "absent"
		if observed {
			name = "observed"
		}
		t.Run(name, func(t *testing.T) {
			r := newRig(t).page(recording.SegLive)
			var submitted recording.Record
			var doc int
			writes := 0
			s := r.bd.NewOperator(func(rec recording.Record, d int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
				writes++
				submitted = rec
				doc = d
				done(OperatorUnknown, errors.New("later fanout failed"))
				return nil
			})
			in := NoteIn{Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 1, 10, 10}, Region: recording.Box{1, 1, 10, 10}, Color: 0, Text: "stable"}
			first := s.Note("1", in)
			if first.Outcome != OperatorUnknown || first.ID == "" {
				t.Fatal(first)
			}
			if got := s.Status("1"); got.Outcome != OperatorUnknown {
				t.Fatalf("invented evidence: %+v", got)
			}
			// A different observed note proves nothing about this stable ID.
			if _, err := r.note(r.p1.ID, 1); err != nil {
				t.Fatal(err)
			}
			if got := s.Status("1"); got.Outcome != OperatorUnknown {
				t.Fatalf("other note reconciled operation: %+v", got)
			}
			if observed {
				if err := r.w.Emit(submitted, doc); err != nil {
					t.Fatal(err)
				}
			}
			var wg sync.WaitGroup
			for i := 0; i < 8; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					got := s.Status("1")
					want := OperatorUnknown
					if observed {
						want = OperatorAccepted
					}
					if got.Outcome != want || got.ID != first.ID {
						t.Errorf("status %+v want %s", got, want)
					}
					if observed && got.Error != "" {
						t.Errorf("accepted retains uncertainty: %+v", got)
					}
				}()
			}
			wg.Wait()
			got := s.Note("1", in)
			if writes != 1 {
				t.Fatalf("reconciliation retried write %d", writes)
			}
			if observed {
				if got.Outcome != OperatorAccepted {
					t.Fatal(got)
				}
				s.Note("2", in)
				if writes != 2 {
					t.Fatal("reconciliation did not release outstanding operation")
				}
			} else {
				if got := s.Note("2", in); got.Outcome != OperatorRefused || writes != 1 {
					t.Fatalf("unknown allowed a second write %+v", got)
				}
			}
		})
	}
}
func TestOperatorUnknownResolveRemainsConservative(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	id, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	s := r.bd.NewOperator(func(rec recording.Record, doc int, _ AnnotationAdmission, done func(OperatorOutcome, error)) error {
		if err := r.w.Emit(rec, doc); err != nil {
			return err
		}
		done(OperatorUnknown, errors.New("later fanout failed"))
		return nil
	})
	s.SetState("1", r.seg(), StateIn{ID: id, State: recording.NoteResolved})
	if got := s.Status("1"); got.Outcome != OperatorUnknown {
		t.Fatalf("target state alone attributed to this resolve: %+v", got)
	}
}

func TestOperatorDefaultOffStillRequiresLivePage(t *testing.T) {
	b := New(nil, nil)
	calls := 0
	op := b.NewOperator(func(recording.Record, int, AnnotationAdmission, func(OperatorOutcome, error)) error {
		calls++
		return nil
	})
	got := op.Stroke("1", StrokeIn{Seg: 1, ID: "pen", Pts: [][3]float64{{1, 2, 0}}})
	if got.Outcome != OperatorRefused || got.Error != ErrNotLive.Error() || calls != 0 {
		t.Fatalf("operator before first page: %+v calls=%d", got, calls)
	}
}

func TestOperatorProjectionClearsTransientStrokes(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	if err := r.bd.Stroke(StrokeIn{Viewer: ana, Seg: r.seg(), ID: "pen", Color: 0, Pts: [][3]float64{{1, 2, 0}}}); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.OperatorView().Strokes) != 1 {
		t.Fatal("missing stroke")
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.OperatorView().Strokes) != 0 {
		t.Fatal("clear left operator stroke replay")
	}
}

func TestOperatorGenerationRequiresCanonicalUint64String(t *testing.T) {
	for _, generation := range []string{``, `,"generation":null`, `,"generation":1`, `,"generation":"01"`, `,"generation":"-1"`, `,"generation":"18446744073709551616"`, `,"generation":""`} {
		body := `{"op":"stroke","request":"1","body":{"seg":1,"id":"pen","color":0,"pts":[[1,2,0]]}` + generation + `}`
		if _, err := ParseOperatorRequest([]byte(body)); err == nil {
			t.Fatalf("admitted generation %s", generation)
		}
	}
	body := `{"op":"stroke","request":"1","generation":"18446744073709551615","body":{"seg":1,"id":"pen","color":0,"pts":[[1,2,0]]}}`
	req, err := ParseOperatorRequest([]byte(body))
	if err != nil || req.Generation != ^uint64(0) {
		t.Fatalf("lost uint64 precision: %+v %v", req, err)
	}
	for _, body := range []string{`{"op":"status","request":"1","body":{}}`, `{"op":"cancel-draft","request":"1","body":{"draft":"held"}}`} {
		if _, err := ParseOperatorRequest([]byte(body)); err != nil {
			t.Fatalf("reconciliation needs no grant: %v", err)
		}
	}
}

// pair#849: a caret re-set many times a second (echo's live board) must not
// push the operator page a fresh view each time; a real change still does.
func TestOperatorViewIgnoresSelectionRecords(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	ch, stop := r.bd.SubscribeOperator()
	defer stop()
	before := <-ch
	for i := 0; i < 5; i++ {
		r.add(&recording.Selection{Anchor: &recording.Pos{3, 3}, Focus: &recording.Pos{3, 3}})
	}
	select {
	case v := <-ch:
		t.Fatalf("selection pushed view %d after %d", v.Revision, before.Revision)
	default:
	}
	if _, err := r.note(r.p1.ID, 0); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-ch:
		if len(v.Notes) != 1 {
			t.Fatalf("note view %+v", v.Notes)
		}
	default:
		t.Fatal("a note pushed no view")
	}
}
