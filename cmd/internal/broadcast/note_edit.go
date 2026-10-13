package broadcast

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// NoteEditRequest carries trusted Owner separately from parsed note data.
type NoteEditRequest struct {
	ID    int
	Text  string
	Owner string
}

func ParseNoteEditRequest(body []byte) (NoteEditRequest, error) {
	var raw struct {
		ID   *int    `json:"id"`
		Text *string `json:"text"`
	}
	if len(body) > MaxNoteBody || !utf8.Valid(body) || !DecodeExactObject(body, &raw, "id text") || raw.ID == nil || *raw.ID < 1 || raw.Text == nil {
		return NoteEditRequest{}, ErrNoteShape
	}
	if !PlainText(*raw.Text) || strings.TrimSpace(*raw.Text) == "" {
		return NoteEditRequest{}, ErrNoteText
	}
	return NoteEditRequest{ID: *raw.ID, Text: *raw.Text}, nil
}

// DecodeExactObject rejects duplicate keys and case aliases, in addition to
// DecodeStrict's unknown fields. fields is the space-separated wire key set.
func DecodeExactObject(body []byte, out any, fields string) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	for dec.More() {
		token, err = dec.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] || !strings.Contains(" "+fields+" ", " "+key+" ") {
			return false
		}
		seen[key] = true
		var value json.RawMessage
		if dec.Decode(&value) != nil {
			return false
		}
	}
	return DecodeStrict(body, out)
}

// Edit replaces only text on an existing note. A public attribution name or
// colour is never ownership proof; Owner is supplied by validated admission.
func (b *NoteBoard) Edit(in NoteEditRequest) error {
	if !PlainText(in.Text) || strings.TrimSpace(in.Text) == "" {
		return ErrNoteText
	}
	for i := range b.notes {
		n := &b.notes[i]
		if n.ID != in.ID {
			continue
		}
		if in.Owner == "" || in.Owner != n.Owner {
			return ErrNoteOwner
		}
		n.Text, n.Edited = in.Text, true
		return nil
	}
	return ErrNoteGone
}

func (p *PointerState) editNote(in NoteEditRequest, gen uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.on || p.gen != gen {
		return ErrPointerOff
	}
	if err := p.board.Edit(in); err != nil {
		return err
	}
	p.notesChangedLocked()
	return nil
}

func (s *Session) editNote(in NoteEditRequest) error {
	if err := s.pointer.editNote(in, s.pointer.generation()); err != nil {
		return err
	}
	s.notesChanged()
	return nil
}
