package board

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// rig is a board behind the reference writer: the "recorder" writes the
// builder's records with Put, the board's go through Emit, both into one
// MemSink and the board itself (the Tee #458 builds).
type rig struct {
	t      *testing.T
	b      *recording.Builder
	w      *Writer
	mem    *recording.MemSink
	bd     *Board
	fed    int
	before func() // runs inside each Emit, before the writer orders it
	p1, p2 recording.Node
}

func newRig(t *testing.T) *rig {
	return newRigWithAnnotation(t, true)
}

func newRigWithAnnotation(t *testing.T, allowed bool) *rig {
	r := &rig{t: t, b: recording.NewBuilder("rec"), mem: &recording.MemSink{}}
	r.bd = New(notes.Structural{}, func(rec recording.Record, doc int) error {
		if r.before != nil {
			r.before()
		}
		return r.w.Emit(rec, doc)
	})
	r.w = NewWriter(recording.Tee(r.mem, r.bd), nil)
	r.bd.WithAnnotationCleanup(r.w.ClearAnnotations)
	if allowed {
		if err := r.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// page casts a page with two paragraphs.
func (r *rig) page(why string) *rig {
	r.p1 = r.b.El("p", []string{"id", "one"}, r.b.Txt("First"))
	r.p2 = r.b.El("p", nil, r.b.Txt("Second"))
	r.b.Page(why, "http://app/", r.p1, r.p2)
	return r.flush()
}

func (r *rig) add(rec recording.Record) *rig {
	r.b.Add(rec)
	return r.flush()
}

// flush writes what the builder added since the last flush.
func (r *rig) flush() *rig {
	r.t.Helper()
	for ; r.fed < len(r.b.Recs); r.fed++ {
		if err := r.w.Put(r.b.Recs[r.fed]); err != nil {
			r.t.Fatalf("writer refused %s: %v", r.b.Recs[r.fed].Kind(), err)
		}
	}
	return r
}

func (r *rig) seg() int { return r.bd.st.Seg }

var ana = Viewer{ID: "ana", Name: "Ana"}

func (r *rig) note(node recording.NodeID, color int) (string, error) {
	return r.bd.Note(NoteIn{Viewer: ana, Seg: r.seg(), Node: node, Box: recording.Box{10, 20, 30, 40}, Region: recording.Box{12, 22, 20, 10}, Color: color, Text: "hi"})
}

func (r *rig) kinds(k string) []recording.Record {
	var out []recording.Record
	for _, rec := range r.mem.Records() {
		if rec.Kind() == k {
			out = append(out, rec)
		}
	}
	return out
}

func TestNoteIsAnchoredWithTheViewersBox(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.b.At(500)
	id, err := r.note(r.p1.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	ns := r.kinds("note")
	if len(ns) != 1 {
		t.Fatalf("notes written: %d", len(ns))
	}
	n := ns[0].(*recording.Note)
	want := recording.Author{ID: "ana", Name: "Ana", Color: broadcast.PointerPalette()[2]}
	if n.ID != id || n.Author != want || n.Text != "hi" || n.At.Node != r.p1.ID || n.At.Sel != "#one" || n.At.Text != "First" {
		t.Errorf("note %+v", n)
	}
	if n.At.Box == nil || *n.At.Box != (recording.Box{10, 20, 30, 40}) {
		t.Errorf("the anchor's box must be the viewer's: %v", n.At.Box)
	}
	if live := r.bd.Live(); len(live) != 1 || live[0].ID != id || live[0].Color != 2 {
		t.Errorf("live %+v", live)
	}
	if _, err := recording.Reduce(r.mem.Records(), 1<<62); err != nil {
		t.Errorf("the written recording must reduce: %v", err)
	}
}

func TestNoteKeepsItsBoxOnThePage(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	reg := recording.Box{5, 6, 70, 80}
	if _, err := r.bd.Note(NoteIn{Viewer: ana, Seg: 1, Node: r.p2.ID, Box: recording.Box{1, 2, 3, 4}, Region: reg, Color: 0, Text: "here"}); err != nil {
		t.Fatal(err)
	}
	if n := r.kinds("note")[0].(*recording.Note); n.Region == nil || *n.Region != reg {
		t.Errorf("region %v", n.Region)
	}
	// A box over the page's edge is cut to it (#470 BR-4); one off it is refused.
	if _, err := r.bd.Note(NoteIn{Viewer: ana, Seg: 1, Node: r.p2.ID, Box: recording.Box{1, 2, 3, 4}, Region: recording.Box{-50, -50, 100, 100}, Color: 0, Text: "edge"}); err != nil {
		t.Fatal(err)
	}
	if n := r.kinds("note")[1].(*recording.Note); n.Region == nil || *n.Region != (recording.Box{0, 0, 50, 50}) {
		t.Errorf("clamped region %v, want [0 0 50 50]", n.Region)
	}
	if _, err := r.bd.Note(NoteIn{Viewer: ana, Seg: 1, Node: r.p2.ID, Box: recording.Box{1, 2, 3, 4}, Region: recording.Box{-500, 0, 100, 100}, Color: 0, Text: "off"}); !errors.Is(err, ErrStale) {
		t.Errorf("a box off the page: %v, want ErrStale", err)
	}
}

func TestRefusals(t *testing.T) {
	r := newRig(t)
	if _, err := r.note(1, 0); !errors.Is(err, ErrNotLive) {
		t.Errorf("before any page: %v", err)
	}
	r.page(recording.SegLive)
	if _, err := r.note(r.p1.C[0].ID, 0); !errors.Is(err, ErrStale) {
		t.Errorf("a text node: %v", err)
	}
	if _, err := r.note(9999, 0); !errors.Is(err, ErrStale) {
		t.Errorf("an unknown node: %v", err)
	}
	r.add(&recording.Rm{ID: r.p2.ID})
	if _, err := r.note(r.p2.ID, 0); !errors.Is(err, ErrStale) {
		t.Errorf("a removed node: %v", err)
	}
	if _, err := r.bd.Note(NoteIn{Viewer: ana, Seg: 2, Node: r.p1.ID, Region: recording.Box{1, 1, 10, 10}, Color: 0, Text: "x"}); !errors.Is(err, ErrStale) {
		t.Errorf("a segment that hasn't begun: %v", err)
	}
	if err := r.bd.SetState(StateIn{Viewer: ana, ID: "nope", State: recording.NoteResolved}); !errors.Is(err, ErrUnknown) {
		t.Errorf("an unknown note: %v", err)
	}
	r.add(&recording.Stop{Why: recording.StopEnd})
	if _, err := r.note(r.p1.ID, 0); !errors.Is(err, ErrEnded) {
		t.Errorf("after the end: %v", err)
	}
	if err := r.bd.Stroke(StrokeIn{Viewer: ana, Seg: 1, ID: "s", Pts: [][3]float64{{1, 2, 0}}}); !errors.Is(err, ErrEnded) {
		t.Errorf("a stroke after the end: %v", err)
	}
}

// While paused, viewers do nothing (pair#499, reversing #470's notes while
// paused): a note, a stroke and a note-state are refused, and nothing from
// the pause appears after the resume. Notes made before it carry across the
// resume (the same document); a request made on the old document after a
// navigation is stale, while the old document's open notes remain listed.
func TestDocumentRules(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	id, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	r.add(&recording.Stop{Why: recording.StopPause})
	if _, err := r.note(r.p2.ID, 0); !errors.Is(err, ErrPaused) {
		t.Errorf("a note while paused: %v", err)
	}
	if err := r.bd.Stroke(StrokeIn{Viewer: ana, Seg: 1, ID: "s", Pts: [][3]float64{{1, 2, 0}}}); !errors.Is(err, ErrPaused) {
		t.Errorf("a stroke while paused: %v", err)
	}
	if err := r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteResolved}); !errors.Is(err, ErrPaused) {
		t.Errorf("a note-state while paused: %v", err)
	}
	r.b.Add(&recording.Seg{Seg: r.b.NextSeg(), Why: recording.SegResume, Tab: recording.Tab{ID: "tab1"}, VP: recording.Viewport{W: 1280, H: 800}})
	r.b.Snapshot("http://app/", r.b.El("html", nil, r.b.El("body", nil, r.p1Again())))
	r.flush()
	if got := r.bd.Live(); len(got) != 1 || got[0].ID != id {
		t.Errorf("live notes after a resume: %+v, want only the open one from before the pause", got)
	}
	if n := len(r.kinds("stroke")); n != 0 {
		t.Errorf("%d strokes from the pause recorded", n)
	}
	if _, err := r.bd.Note(NoteIn{Viewer: ana, Seg: 1, Node: r.p1.ID, Region: recording.Box{1, 1, 10, 10}, Color: 0, Text: "x"}); err != nil {
		t.Errorf("a viewer one resume behind is on the same document: %v", err)
	}
	r.page(recording.SegNavigate)
	if got := len(r.bd.Live()); got != 2 {
		t.Errorf("open notes after a navigation: %d, want 2", got)
	}
	if _, err := r.bd.Note(NoteIn{Viewer: ana, Seg: 2, Node: r.p1.ID, Region: recording.Box{1, 1, 10, 10}, Color: 0, Text: "x"}); !errors.Is(err, ErrStale) {
		t.Errorf("a note made on the old document: %v", err)
	}
}

// p1Again is p1 with its id kept, as a resume of the same document keeps
// ids.
func (r *rig) p1Again() recording.Node {
	return recording.Node{ID: r.p1.ID, Y: recording.YEl, N: "p", A: map[string]string{"id": "one"}, C: []recording.Node{{ID: r.p1.C[0].ID, Y: recording.YText, V: recording.Str("First")}}}
}

func TestColorLimitAndResolve(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	var first string
	for i := 0; i < broadcast.MaxNotesPerColor; i++ {
		id, err := r.note(r.p1.ID, 1)
		if err != nil {
			t.Fatalf("note %d: %v", i, err)
		}
		if i == 0 {
			first = id
		}
	}
	if _, err := r.note(r.p1.ID, 1); !errors.Is(err, ErrColorFull) {
		t.Errorf("a ninth note in one color: %v", err)
	}
	if _, err := r.note(r.p1.ID, 3); err != nil {
		t.Errorf("another color: %v", err)
	}
	n := len(r.mem.Records())
	if err := r.bd.SetState(StateIn{Viewer: ana, ID: first, State: recording.NoteOpen}); err != nil || len(r.mem.Records()) != n {
		t.Errorf("setting the state a note has must write nothing: %v", err)
	}
	if err := r.bd.SetState(StateIn{Viewer: ana, ID: first, State: recording.NoteResolved}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.note(r.p1.ID, 1); err != nil {
		t.Errorf("a resolve frees a place in its color: %v", err)
	}
}

// Concurrent notes at the limit: pending notes count, so exactly the limit
// land. Run with -race.
func TestColorLimitHoldsUnderConcurrency(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.note(r.p1.ID, 4); err == nil {
				ok.Add(1)
			} else if !errors.Is(err, ErrColorFull) {
				t.Errorf("note: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := ok.Load(); got != broadcast.MaxNotesPerColor {
		t.Errorf("%d notes landed in one color, want %d", got, broadcast.MaxNotesPerColor)
	}
}

// The check/emit race (PQ-1): a navigation the writer orders between the
// board's check and the note drops the note, and its pending entry ends.
func TestNavigationBetweenCheckAndWriteDropsTheNote(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.before = func() {
		r.before = nil
		r.page(recording.SegNavigate)
	}
	if _, err := r.note(r.p1.ID, 5); !errors.Is(err, ErrStale) {
		t.Fatalf("a note overtaken by a navigation: %v", err)
	}
	if len(r.kinds("note")) != 0 {
		t.Errorf("the note must not be written on the new page")
	}
	for i := 0; i < broadcast.MaxNotesPerColor; i++ {
		if _, err := r.note(r.p1.ID, 5); err != nil {
			t.Fatalf("the dropped note must not hold a place in its color (note %d): %v", i, err)
		}
	}
}

// An asynchronous writer drops silently: its pending entries end when the
// board sees the new document.
func TestSilentDropEndsPendingAtTheNextDocument(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	emit := r.bd.emit
	r.bd.emit = func(recording.Record, int) error { return nil } // queued, never written
	for i := 0; i < broadcast.MaxNotesPerColor; i++ {
		if _, err := r.note(r.p1.ID, 0); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.note(r.p1.ID, 0); !errors.Is(err, ErrColorFull) {
		t.Fatalf("pending notes count: %v", err)
	}
	r.bd.emit = emit
	r.page(recording.SegNavigate)
	if _, err := r.note(r.p1.ID, 0); err != nil {
		t.Errorf("pending notes of the old document must end: %v", err)
	}
}

func TestEmitErrorEndsPending(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	boom := errors.New("queue full")
	emit := r.bd.emit
	r.bd.emit = func(recording.Record, int) error { return boom }
	for i := 0; i < broadcast.MaxNotesPerColor+1; i++ {
		if _, err := r.note(r.p1.ID, 0); !errors.Is(err, boom) {
			t.Fatalf("note %d: %v", i, err)
		}
	}
	r.bd.emit = emit
	if _, err := r.note(r.p1.ID, 0); err != nil {
		t.Errorf("failed emits must not hold places: %v", err)
	}
}

func TestStrokeChunksShareOneID(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	for i := 0; i < 2; i++ {
		if err := r.bd.Stroke(StrokeIn{Viewer: ana, Seg: 1, ID: "s1", Color: 3, Pts: [][3]float64{{float64(i), 2, 0}, {3, 4, 16}}}); err != nil {
			t.Fatal(err)
		}
	}
	ss := r.kinds("stroke")
	if len(ss) != 2 {
		t.Fatalf("strokes %d", len(ss))
	}
	for _, s := range ss {
		s := s.(*recording.Stroke)
		if s.ID != "ana.s1" || s.Color != broadcast.PointerPalette()[3] || s.Author.ID != "ana" {
			t.Errorf("stroke %+v", s)
		}
	}
}

func TestSubscribeFollowsTheOpenNotes(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	ch, cancel := r.bd.Subscribe()
	defer cancel()
	if got := <-ch; len(got) != 0 {
		t.Errorf("first delivery %v", got)
	}
	id, _ := r.note(r.p1.ID, 0)
	if got := <-ch; len(got) != 1 || got[0].ID != id {
		t.Errorf("after a note %v", got)
	}
	r.page(recording.SegNavigate)
	select {
	case got := <-ch:
		t.Errorf("unchanged notes should not publish after navigation: %v", got)
	default:
	}
	if err := r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteResolved}); err != nil {
		t.Fatal(err)
	}
	if got := <-ch; len(got) != 0 {
		t.Errorf("after historical resolve %v", got)
	}
	r.bd.Close()
	if _, open := <-ch; open {
		t.Errorf("Close must close subscriptions")
	}
	if _, err := r.note(r.p1.ID, 0); !errors.Is(err, ErrClosed) {
		t.Errorf("after Close: %v", err)
	}
}

func TestWriterStampsNoEarlierThanTheLastRecord(t *testing.T) {
	mem := &recording.MemSink{}
	now := int64(100)
	w := NewWriter(mem, func() int64 { return now })
	b := recording.NewBuilder("rec")
	b.At(500).Page(recording.SegLive, "http://app/", b.El("p", nil))
	for _, rec := range b.Recs {
		w.Put(rec)
	}
	s := &recording.Stroke{ID: "a.s", Author: recording.Author{ID: "a"}, Pts: [][3]float64{{1, 1, 0}}}
	if err := w.Emit(s, 1); err != nil || s.T != 500 {
		t.Errorf("stamped %d (%v), want 500: never before the last record", s.T, err)
	}
	now = 900
	s2 := &recording.Stroke{ID: "a.s", Author: recording.Author{ID: "a"}, Pts: [][3]float64{{1, 1, 0}}}
	if err := w.Emit(s2, 1); err != nil || s2.T != 900 {
		t.Errorf("stamped %d (%v), want the clock's 900", s2.T, err)
	}
	if err := w.Emit(s2, 2); !errors.Is(err, ErrStale) {
		t.Errorf("a document that isn't current: %v", err)
	}
}

// The board over a real fixture: a note made before a pause stays live
// across the resume and the tab switch; the recording with it
// still validates.
func TestBoardOverTheSegmentsFixture(t *testing.T) {
	rec, err := recording.ReadDir(recording.Fixture("segments"))
	if err != nil {
		t.Fatal(err)
	}
	r := newRig(t)
	var id string
	for _, x := range rec.Records {
		if x.Time() > 1600 && id == "" {
			q := firstElement(r.bd.st.Top, "input")
			if q == nil {
				t.Fatal("no input on the page")
			}
			if id, err = r.bd.Note(NoteIn{Viewer: ana, Seg: 1, Node: q.ID, Box: recording.Box{1, 1, 1, 1}, Region: recording.Box{1, 1, 1, 1}, Color: 0, Text: "mine"}); err != nil {
				t.Fatal(err)
			}
		}
		if x.Time() > 6000 && x.Time() < 7000 && !hasLive(r.bd.Live(), id) {
			t.Errorf("my note must be live after the resume (t=%d)", x.Time())
		}
		if err := r.w.Put(x); err != nil {
			t.Fatalf("t=%d %s: %v", x.Time(), x.Kind(), err)
		}
	}
	if !hasLive(r.bd.Live(), id) {
		t.Errorf("my open note must remain listed after the tab switch")
	}
	if _, err := recording.Reduce(r.mem.Records(), 1<<62); err != nil {
		t.Errorf("recording with the note: %v", err)
	}
}

func hasLive(ls []LiveNote, id string) bool {
	for _, l := range ls {
		if l.ID == id {
			return true
		}
	}
	return false
}

func firstElement(n *recording.DOMNode, tag string) *recording.DOMNode {
	if n == nil {
		return nil
	}
	if n.Y == recording.YEl && n.N == tag {
		return n
	}
	for _, k := range n.Kids {
		if f := firstElement(k, tag); f != nil {
			return f
		}
	}
	return nil
}

// A board whose state parts from the stream stops taking requests, says so
// once, and never fails the recording's Tee; it keeps no strokes.
func TestBrokenBoardStopsAndSaysSo(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	if err := r.bd.Stroke(StrokeIn{Viewer: ana, Seg: 1, ID: "s", Pts: [][3]float64{{1, 1, 0}}}); err != nil {
		t.Fatal(err)
	}
	if n := len(r.bd.st.Strokes); n != 0 {
		t.Errorf("the board keeps %d strokes", n)
	}
	var logged []string
	r.bd.Logf = func(f string, a ...any) { logged = append(logged, f) }
	if err := r.bd.Put(&recording.Rm{ID: 9999}); err != nil {
		t.Errorf("Put must never fail the Tee: %v", err)
	}
	r.bd.Put(&recording.Rm{ID: 9998})
	if _, err := r.note(r.p1.ID, 0); !errors.Is(err, ErrBroken) {
		t.Errorf("a broken board: %v", err)
	}
	if len(logged) != 1 {
		t.Errorf("logged %d times, want once", len(logged))
	}
	r.bd.Close()
	if err := r.bd.Put(&recording.Stop{}); err != nil {
		t.Errorf("Put after Close: %v", err)
	}
}

// A record the board admitted just before a pause reaches the writer after
// it: the writer drops it and says the cast is paused, not that the page
// changed (pair#499 close review); after a navigation it is stale.
func TestWriterRefusesARecordDuringAPause(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.add(&recording.Stop{Why: recording.StopPause})
	stroke := &recording.Stroke{ID: "ana.s", Author: recording.Author{ID: "ana"}, Pts: [][3]float64{{1, 2, 0}}}
	if err := r.w.Emit(stroke, 1); !errors.Is(err, ErrPaused) {
		t.Errorf("an emit during a pause: %v", err)
	}
	r.page(recording.SegNavigate)
	if err := r.w.Emit(stroke, 1); !errors.Is(err, ErrStale) {
		t.Errorf("an emit for a document that has gone: %v", err)
	}
	if n := len(r.kinds("stroke")); n != 0 {
		t.Errorf("%d strokes written", n)
	}
}
