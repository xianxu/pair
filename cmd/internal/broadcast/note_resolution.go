package broadcast

import (
	"time"
	"unicode/utf8"
)

const MaxNoteResolutions = 128

// NoteResolution is volatile cast-lifetime attribution, not a durable journal.
type NoteResolution struct {
	ID     int       `json:"id"`
	Author string    `json:"author,omitempty"`
	At     time.Time `json:"at"`
}

type NoteResolutionRequest struct {
	ID     int
	Author string
}

func ParseNoteResolution(body []byte) (NoteResolutionRequest, error) {
	var raw struct {
		ID     *int    `json:"id"`
		Author *string `json:"author"`
	}
	if len(body) > MaxNoteBody || !utf8.Valid(body) || !DecodeExactObject(body, &raw, "id author") || raw.ID == nil || *raw.ID < 1 {
		return NoteResolutionRequest{}, ErrNoteShape
	}
	out := NoteResolutionRequest{ID: *raw.ID}
	if raw.Author != nil {
		var ok bool
		out.Author, ok = ViewerName(*raw.Author)
		if !ok {
			return NoteResolutionRequest{}, ErrNoteText
		}
	}
	return out, nil
}

// Resolutions returns a copy of the latest bounded resolution events. Ending
// the cast drops them; an OFF flip only clears current notes.
func (s *Session) Resolutions() []NoteResolution {
	s.pointer.mu.Lock()
	defer s.pointer.mu.Unlock()
	return append([]NoteResolution(nil), s.pointer.resolutions...)
}
