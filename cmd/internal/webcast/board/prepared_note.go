package board

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// PreparedNote freezes the anchor once, before capture and authored text. The
// private authority binds it to the issuing board, generation and surface.
// It is a local service value, never an accepted client payload.
type PreparedNote struct {
	Document  int
	TabID     string
	T         int64
	At        recording.NoteAt
	Region    recording.Box
	board     *Board
	authority annotationAuthority
	operator  bool
}

type PrepareNoteIn struct {
	Seg         int
	Node        recording.NodeID
	Box, Region recording.Box
}

// NoteReservation owns a pending color slot until its note is observed by the
// board or the writer proves absence and its owner releases the reservation.
type NoteReservation struct {
	Document int
	Note     recording.Note
}

func (a AnnotationAdmission) PrepareNote(in PrepareNoteIn, operator bool) (PreparedNote, error) {
	if a.board == nil {
		return PreparedNote{}, ErrDisabled
	}
	b := a.board
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.surfaceAdmissionLocked(a.authority, operator); err != nil {
		return PreparedNote{}, err
	}
	doc, err := b.annotationPageLocked(in.Seg, operator)
	if err != nil {
		return PreparedNote{}, err
	}
	if !segOK(in.Seg) || in.Node <= 0 || !validBox(in.Box, 0) || !validBox(in.Region, 1) {
		return PreparedNote{}, ErrShape
	}
	region, ok := notes.ClampToPage(b.st, in.Region)
	if !ok {
		return PreparedNote{}, ErrStale
	}
	at, err := b.model.Anchor(b.st, in.Node)
	if err != nil {
		return PreparedNote{}, ErrStale
	}
	at = cloneNote(&recording.Note{At: at}).At
	box := in.Box
	at.Box = &box
	return PreparedNote{Document: doc, TabID: b.st.Tab.ID, T: b.st.T, At: at, Region: region, board: b, authority: a.authority, operator: operator}, nil
}

// ReservePreparedNote never looks up the old node. Losing an element after its
// capture must not replace or invalidate the evidence the operator annotated.
// The caller persists its save intent before emitting this reserved record.
func (a AnnotationAdmission) ReservePreparedNote(p PreparedNote, author Viewer, color int, text, captureRef string, operator bool) (NoteReservation, error) {
	if a.board == nil || p.board != a.board || p.authority != a.authority || p.operator != operator {
		return NoteReservation{}, ErrDisabled
	}
	b := a.board
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.surfaceAdmissionLocked(a.authority, operator); err != nil {
		return NoteReservation{}, err
	}
	doc, err := b.annotationPageLocked(p.Document, operator)
	if err != nil {
		return NoteReservation{}, err
	}
	if doc != p.Document || b.st.Tab.ID != p.TabID {
		return NoteReservation{}, ErrStale
	}
	if !colorOK(color) || !noteText(text) || (captureRef != "" && !idPattern.MatchString(captureRef)) || p.At.Node <= 0 || p.At.Box == nil || !validBox(*p.At.Box, 0) || !validBox(p.Region, 1) {
		return NoteReservation{}, ErrShape
	}
	if operator {
		author = Viewer{ID: Operator, Name: "Operator"}
	} else {
		var valid bool
		author, valid = viewer(&wireViewer{ID: &author.ID, Name: &author.Name})
		if !valid {
			return NoteReservation{}, ErrShape
		}
	}
	if b.openLocked(color) >= broadcast.MaxNotesPerColor {
		return NoteReservation{}, ErrColorFull
	}
	id := b.newID()
	b.pending[id] = pending{doc: doc, color: color}
	note := cloneNote(&recording.Note{ID: id, Author: b.author(author, color), Text: text, At: p.At, Region: &p.Region, Capture: captureRef})
	return NoteReservation{Document: doc, Note: *note}, nil
}

// ReleaseNoteReservation is idempotent. A timeout is not proof of absence;
// callers release only after the writer has definitely refused or reconciled.
func (b *Board) ReleaseNoteReservation(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.pending, id)
}

// ObservedNote reports only positive board evidence. false is not proof that an
// in-flight writer did not commit; recovery requires a writer barrier or replay.
func (b *Board) ObservedNote(id string) (*recording.Note, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, n := range b.st.Notes {
		if n.ID == id {
			return cloneNote(n.Note), true
		}
	}
	return nil, false
}

// RestorePreparedNote accepts only a service-authenticated preparation token.
// The caller must verify its signature, cast/surface scope, expiration and
// original annotation generation before invoking this method. It restores
// local authority, not the anchor: a missing/replaced node is never consulted.
func (a AnnotationAdmission) RestorePreparedNote(p PreparedNote, operator bool) (PreparedNote, error) {
	if a.board == nil {
		return PreparedNote{}, ErrDisabled
	}
	b := a.board
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.surfaceAdmissionLocked(a.authority, operator); err != nil {
		return PreparedNote{}, err
	}
	doc, err := b.annotationPageLocked(p.Document, operator)
	if err != nil {
		return PreparedNote{}, err
	}
	if doc != p.Document || p.TabID != b.st.Tab.ID {
		return PreparedNote{}, ErrStale
	}
	if p.T < 0 || p.T > b.st.T || p.At.Node <= 0 || p.At.Kind != recording.AnchorElement || p.At.Box == nil || !validBox(*p.At.Box, 0) || !validBox(p.Region, 1) {
		return PreparedNote{}, ErrShape
	}
	encoded, err := json.Marshal(p.At)
	if err != nil || len(encoded) > MaxOperatorBody {
		return PreparedNote{}, ErrShape
	}
	p.At = cloneNote(&recording.Note{At: p.At}).At
	p.board, p.authority, p.operator = b, a.authority, operator
	return p, nil
}
