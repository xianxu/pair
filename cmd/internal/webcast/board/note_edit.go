package board

import (
	"errors"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var ErrOwner = errors.New("board: only the note's original session can edit it")

type EditIn struct {
	Seg             int
	ID, Text, Owner string
}

func ParseNoteEdit(body []byte) (EditIn, error) {
	var raw struct {
		Seg  *int    `json:"seg"`
		ID   *string `json:"id"`
		Text *string `json:"text"`
	}
	if len(body) > MaxNoteBody || !utf8.Valid(body) || !broadcast.DecodeExactObject(body, &raw, "seg id text") || raw.Seg == nil || raw.ID == nil || raw.Text == nil {
		return EditIn{}, ErrShape
	}
	in := EditIn{Seg: *raw.Seg, ID: *raw.ID, Text: *raw.Text}
	if !editShape(in) {
		return EditIn{}, ErrShape
	}
	return in, nil
}

func editShape(in EditIn) bool {
	return segOK(in.Seg) && idPattern.MatchString(in.ID) && noteText(in.Text)
}

func (b *Board) Edit(in EditIn) error {
	a, err := b.AdmitAnnotation()
	if err != nil {
		return err
	}
	return a.Edit(in)
}

func (a AnnotationAdmission) Edit(in EditIn) error {
	if a.board == nil {
		return ErrDisabled
	}
	b := a.board
	b.command.Lock()
	defer b.command.Unlock()
	b.mu.Lock()
	err := b.surfaceAdmissionLocked(a.authority, false)
	b.mu.Unlock()
	if err != nil {
		return err
	}
	return b.edit(in, false, b.emit)
}

func (b *Board) edit(in EditIn, operator bool, emit Emit) error {
	if !editShape(in) {
		return ErrShape
	}
	b.mu.Lock()
	doc, err := b.annotationPageLocked(in.Seg, operator)
	if err != nil {
		b.mu.Unlock()
		return err
	}
	var owner string
	found := false
	for _, n := range b.st.Notes {
		if n.ID == in.ID && n.State == recording.NoteOpen {
			owner = n.Owner
			found = true
			break
		}
	}
	b.mu.Unlock()
	if !found {
		return ErrUnknown
	}
	if in.Owner == "" || in.Owner != owner {
		return ErrOwner
	}
	return emit(&recording.NoteEdit{ID: in.ID, Text: in.Text}, doc)
}
