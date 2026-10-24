// Package board is the web cast's collaboration layer (#470): viewers'
// pointer strokes and notes, checked and turned into recording records.
//
// A Board follows the cast's record stream as a recording.Sink, so it knows
// the page every viewer is looking at. A request is checked against that
// state, anchored by the notes model (notes.Model.Anchor, with the viewer's
// own box), and handed to an Emit: the stream's one writer (the recorder's
// queue, #458), which stamps t and re-checks the document when it orders the
// record (recording.State.OnDoc). Every viewer then sees the record in the live stream, and
// the recording keeps it.
//
// Pure: ParseStroke, ParseNote, ParseNoteState. Stateful: Board, and
// Writer, the reference writer that tests use in place of the recorder.
// atlas/webcast-format.md, "Notes and pointer", is the prose.
package board

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"
	"sync"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var (
	// ErrEnded: the cast is over (stop end).
	ErrDisabled = errors.New("board: annotation is disabled")
	ErrEnded    = errors.New("board: the cast has ended")
	// ErrNotLive: no page has been cast yet.
	ErrNotLive = errors.New("board: the cast hasn't shown a page yet")
	// ErrPaused: the cast is paused; viewers do nothing until it resumes
	// (pair#499).
	ErrPaused = errors.New("board: the cast is paused")
	// ErrStale: the viewer was showing another document, the element is
	// gone, or a note's box is off the page.
	ErrStale = errors.New("board: the page changed")
	// ErrColorFull: the color already has broadcast.MaxNotesPerColor open
	// notes on this page.
	ErrColorFull = errors.New("board: too many notes in one color")
	// ErrUnknown: no such note on this page.
	ErrUnknown = errors.New("board: no such note")
	// ErrClosed: the board was closed.
	ErrClosed = errors.New("board: closed")
	// ErrBroken: the board's state parted from the stream it follows (a
	// bug); it takes no more requests this cast.
	ErrBroken = errors.New("board: lost track of the page")
)

// Emit writes r into the stream if the document that began at segment doc
// is still current when the writer orders r (recording.State.OnDoc), stamping
// its t; else it drops r. A writer that knows at once may return ErrStale for a drop;
// an asynchronous one (the recorder's queue) returns nil and drops later.
type Emit func(r recording.Record, doc int) error

// LiveNote is an open note on the current document, for Couch.
type LiveNote struct {
	ID     string
	Author recording.Author
	Text   string
	// Color is the note's palette index (broadcast.PointerPalette), or -1
	// for a color outside it.
	Color int
	T     int64
}

// pending is a note emitted and not yet seen back: it counts toward its
// color's limit until then.
type pending struct {
	doc   int
	color int
}

// Board checks viewers' requests against the stream it follows. mu is a
// leaf lock: Emit and subscribers are never called while it is held.
type Board struct {
	// Logf, when set, reports the board breaking, once.
	Logf    func(format string, args ...any)
	model   notes.Model
	emit    Emit
	palette []string
	newID   func() string

	command              sync.Mutex // serializes annotation validation/dispatch and cleanup
	annotation           capability.CapabilityState
	annotationProblem    string
	annotationCleanup    AnnotationCleanup
	annotationAttempt    *annotationAttempt
	annotationSubs       map[chan capability.Snapshot]struct{}
	mu                   sync.Mutex
	st                   *recording.State
	broken               error // the board's reducer refused a record: it can't trust its state
	pending              map[string]pending
	subs                 map[chan []LiveNote]struct{}
	last                 []LiveNote
	closed               bool
	operatorSubs         map[chan OperatorView]struct{}
	operatorRevision     uint64
	operatorSeq          uint64
	operatorStrokes      []OperatorStroke
	operatorDoc          int
	sketchDelivery       sketchDeliveryJournal
	sketchService        *SketchService
	submissionGeneration uint64
	feedback             feedbackContexts
	feedbackDisplay      feedbackDisplay
	submissionSubs       map[chan struct{}]struct{}
	pageLoad             pageLoadScope
}

// New makes a board that anchors notes with model and writes through emit.
func New(model notes.Model, emit Emit) *Board {
	return &Board{model: model, emit: emit, palette: broadcast.PointerPalette(), newID: randomID,
		st: recording.NewState(), pending: map[string]pending{}, subs: map[chan []LiveNote]struct{}{}, annotationSubs: map[chan capability.Snapshot]struct{}{}, feedbackDisplay: feedbackDisplay{mask: LiveFeedbackMask{Scope: randomID()}}}
}

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Put follows the stream: every record the writer wrote, in order.
func (b *Board) Put(r recording.Record) error {
	// Put never fails: the board rides on the recording's Tee, and a
	// problem of its own must not fail the recording.
	b.mu.Lock()
	if b.closed || b.broken != nil {
		b.mu.Unlock()
		return nil
	}
	var previousLoad *recording.PageLoad
	_, segment := r.(*recording.Seg)
	if segment && !b.pageLoad.managed {
		previousLoad = b.currentPageLoadLocked()
	}
	sketchBefore := b.sketchBeforeChangeLocked(r)
	if err := b.st.Apply(r); err != nil {
		// The writer applied r to its own reducer first, so this can't
		// happen short of a bug: the board stops taking requests and says
		// so, rather than check them against a page it no longer knows.
		b.broken = err
		b.mu.Unlock()
		if b.Logf != nil {
			b.Logf("web cast notes stopped: %v", err)
		}
		return nil
	}
	if segment && !b.pageLoad.managed && !sameOptionalPageLoad(previousLoad, b.currentPageLoadLocked()) {
		b.resetSketchDeliveryLocked()
	}
	b.refreshSketchDeliveryLocked(r, sketchBefore)
	b.refreshFeedbackContextLocked(r, sketchBefore.duplicate)
	b.pruneFeedbackMaskLocked(r)
	b.retireSketchSourceLocked()
	if recording.SketchReceiptKey(r) != "" {
		// Drawing has its own bounded delta wake. Rebuilding unrelated note
		// projections here would make every pressure chunk scan old notes.
		b.mu.Unlock()
		return nil
	}
	// The board needs no strokes (nothing checks against them), so it
	// doesn't keep them for the cast's life.
	b.st.Strokes = b.st.Strokes[:0]
	if _, ok := r.(*recording.AnnotationClear); ok {
		clear(b.pending)
	}
	if n, ok := r.(*recording.Note); ok {
		delete(b.pending, n.ID)
	}
	for id, p := range b.pending {
		if p.doc != b.st.DocSeg {
			delete(b.pending, id) // dropped by the writer, or on a page that's gone
		}
	}
	b.refreshLocked()
	b.refreshOperatorLocked(r)
	b.mu.Unlock()
	return nil
}

// Close ends every subscription; later requests fail with ErrClosed. The
// writer closes it with the recording (it is one of the Tee's sinks).
func (b *Board) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		b.closed = true
		for ch := range b.submissionSubs {
			close(ch)
		}
		b.submissionSubs = nil
		for ch := range b.feedbackDisplay.subs {
			close(ch)
		}
		b.feedbackDisplay.subs = nil
		b.retireSketchSourceLocked()
		b.closeSketchDeliveryLocked()
		b.annotation, _ = capability.NextCapability(b.annotation, capability.Event{Kind: capability.End})
		b.publishAnnotationLocked()
		for ch := range b.annotationSubs {
			close(ch)
		}
		b.annotationSubs = nil
		for ch := range b.subs {
			close(ch)
		}
		b.subs = nil
		b.refreshOperatorLocked(nil)
		for ch := range b.operatorSubs {
			close(ch)
		}
		b.operatorSubs = nil
	}
	return nil
}

// Stroke writes one chunk of a viewer's stroke. Its record id is the
// viewer's id and the stroke's, so two viewers' strokes never join.
func (b *Board) Stroke(in StrokeIn) error {
	a, err := b.AdmitAnnotation()
	if err != nil {
		return err
	}
	return a.Stroke(in)
}

func (a AnnotationAdmission) Stroke(in StrokeIn) error {
	if a.board == nil {
		return ErrDisabled
	}
	b := a.board
	b.command.Lock()
	defer b.command.Unlock()
	b.mu.Lock()
	if err := b.admissionLocked(a.permit); err != nil {
		b.mu.Unlock()
		return err
	}
	b.mu.Unlock()
	return b.stroke(in, false, b.emit)
}

func (b *Board) stroke(in StrokeIn, operator bool, emit Emit) error {
	if !strokeShape(in) {
		return ErrShape
	}
	b.mu.Lock()
	doc, err := b.annotationPageLocked(in.Seg, operator)
	b.mu.Unlock()
	if err != nil {
		return err
	}
	return emit(&recording.Stroke{ID: in.Viewer.ID + "." + in.ID, Author: b.author(in.Viewer, in.Color),
		Color: b.palette[in.Color], Pts: in.Pts}, doc)
}

// Note writes a viewer's new note and returns its id. The anchor is the
// notes model's for the element now, with the viewer's box; the note's box
// is kept on the page (notes.ClampToPage).
func (b *Board) Note(in NoteIn) (string, error) {
	a, err := b.AdmitAnnotation()
	if err != nil {
		return "", err
	}
	return a.Note(in)
}

func (a AnnotationAdmission) Note(in NoteIn) (string, error) {
	if a.board == nil {
		return "", ErrDisabled
	}
	b := a.board
	b.command.Lock()
	defer b.command.Unlock()
	b.mu.Lock()
	if err := b.admissionLocked(a.permit); err != nil {
		b.mu.Unlock()
		return "", err
	}
	b.mu.Unlock()
	return b.note(in, false, b.emit)
}

func (b *Board) note(in NoteIn, operator bool, emit Emit) (string, error) {
	b.mu.Lock()
	doc, err := b.annotationPageLocked(in.Seg, operator)
	if err != nil {
		b.mu.Unlock()
		return "", err
	}
	if !noteShape(in) {
		b.mu.Unlock()
		return "", ErrShape
	}
	if b.st.Node(in.Node) == nil {
		b.mu.Unlock()
		return "", ErrStale
	}
	if b.openLocked(in.Color) >= broadcast.MaxNotesPerColor {
		b.mu.Unlock()
		return "", ErrColorFull
	}
	region, ok := notes.ClampToPage(b.st, in.Region)
	if !ok {
		b.mu.Unlock()
		return "", ErrStale // the box is off the page
	}
	at, err := b.model.Anchor(b.st, in.Node)
	if err != nil {
		b.mu.Unlock()
		return "", ErrStale // not an element: nothing to pin a note on
	}
	box := in.Box
	at.Box = &box
	id := b.newID()
	b.pending[id] = pending{doc: doc, color: in.Color}
	b.mu.Unlock()

	n := &recording.Note{ID: id, Author: b.author(in.Viewer, in.Color), Owner: in.Owner, Text: in.Text, At: at, Region: &region}
	if err := emit(n, doc); err != nil {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return "", err
	}
	return id, nil
}

// SetState resolves or reopens a recorded note under the current page admission. Anyone may;
// setting the state a note already has does nothing.
func (b *Board) SetState(in StateIn) error {
	a, err := b.AdmitAnnotation()
	if err != nil {
		return err
	}
	return a.SetState(in)
}

func (a AnnotationAdmission) SetState(in StateIn) error {
	if a.board == nil {
		return ErrDisabled
	}
	b := a.board
	b.command.Lock()
	defer b.command.Unlock()
	b.mu.Lock()
	if err := b.admissionLocked(a.permit); err != nil {
		b.mu.Unlock()
		return err
	}
	b.mu.Unlock()
	return b.setState(in, 0, false, b.emit)
}

func (b *Board) setState(in StateIn, seg int, operator bool, emit Emit) error {
	if !stateShape(in) {
		return ErrShape
	}
	b.mu.Lock()
	if !operator {
		seg = b.st.DocSeg
	}
	doc, err := b.annotationPageLocked(seg, operator)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	var cur *recording.NoteView
	load := b.currentPageLoadLocked()
	for _, n := range b.st.Notes {
		if n.ID == in.ID {
			if !b.st.OnPageLoad(n.Seg, load) {
				b.mu.Unlock()
				return ErrStale
			}
			copy := *n
			cur = &copy
		}
	}
	unchanged := cur != nil && cur.State == in.State
	b.mu.Unlock()
	switch {
	case cur == nil:
		return ErrUnknown
	case unchanged:
		return nil
	}
	update := &recording.NoteState{ID: in.ID, State: in.State, Who: &recording.Author{ID: in.Viewer.ID, Name: in.Viewer.Name}}
	if in.State == recording.NoteResolved {
		revision := cur.Revision
		update.IfRevision = &revision
	}
	return emit(update, doc)
}

// Live is open notes on the confirmed current page load, oldest first.
func (b *Board) Live() []LiveNote {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.last)
}

// Subscribe delivers the open notes whenever they change, starting with
// the current ones. The channel holds one value and never blocks the
// board: a slow reader gets the latest. It closes at Close.
func (b *Board) Subscribe() (<-chan []LiveNote, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []LiveNote, 1)
	ch <- slices.Clone(b.last)
	if b.closed {
		close(ch)
		return ch, func() {}
	}
	b.subs[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[ch]; ok {
			delete(b.subs, ch)
			close(ch)
		}
	}
}

// pageLocked admits a request made on segment seg: the cast is live on a
// page (not paused, not ended), and seg is within the current document (a resume keeps the
// document, so a viewer a pause behind is still on it). It returns the
// document's first segment, for the writer's check.
func (b *Board) pageLocked(seg int) (int, error) {
	switch {
	case b.closed:
		return 0, ErrClosed
	case b.broken != nil:
		return 0, ErrBroken
	case b.st.Ended:
		return 0, ErrEnded
	case b.st.Seg == 0:
		return 0, ErrNotLive
	case b.st.Paused():
		return 0, ErrPaused
	case b.pageLoad.managed && b.currentPageLoadLocked() == nil:
		return 0, ErrStale
	case seg < b.st.DocSeg || seg > b.st.Seg:
		return 0, ErrStale
	}
	return b.st.DocSeg, nil
}

// openLocked counts a color's open notes on this page, pending included.
func (b *Board) openLocked(color int) int {
	n := 0
	for _, l := range b.model.Live(b.st) {
		if l.State == recording.NoteOpen && l.Author.Color == b.palette[color] {
			n++
		}
	}
	for _, p := range b.pending {
		if p.color == color {
			n++
		}
	}
	return n
}

func (b *Board) author(v Viewer, color int) recording.Author {
	return recording.Author{ID: v.ID, Name: v.Name, Color: b.palette[color]}
}

// refreshLocked recomputes the open notes and, when they changed, delivers
// them. Only the board sends on a subscription, and it empties the one-slot
// buffer first, so the send never blocks under the lock.
func (b *Board) refreshLocked() {
	var open []LiveNote
	load := b.currentPageLoadLocked()
	for _, n := range b.st.Notes {
		if n.State == recording.NoteOpen && b.st.OnPageLoad(n.Seg, load) {
			open = append(open, LiveNote{ID: n.ID, Author: n.Author, Text: n.Text, Color: slices.Index(b.palette, n.Author.Color), T: n.T})
		}
	}
	if slices.Equal(open, b.last) {
		return
	}
	b.last = open
	for ch := range b.subs {
		select { // keep only the latest: drop a value nobody read yet
		case <-ch:
		default:
		}
		ch <- slices.Clone(open)
	}
}
