package broadcast

import "github.com/xianxu/pair/cmd/internal/terminal"

// Message is one unit of the broadcast: rendered terminal bytes at the
// geometry a viewer must have before writing them.
type Message struct {
	Cols, Rows int
	Data       []byte
}

// Stream holds the broadcast's only frame state, the last frame sent. Each
// frame is rendered once and the diff shared by every viewer that has seen its
// predecessor; Join renders the current frame from nothing for any other.
type Stream struct {
	last    terminal.Frame
	started bool
}

// Next renders the change from the last frame to f and advances. ok is false
// when nothing changed.
func (s *Stream) Next(f terminal.Frame) (m Message, ok bool, err error) {
	data, err := terminal.Render(s.last, f)
	if err != nil {
		return Message{}, false, err
	}
	s.last, s.started = f, true
	if len(data) == 0 {
		return Message{}, false, nil
	}
	return Message{Cols: f.Geometry.Cols, Rows: f.Geometry.Rows, Data: data}, true, nil
}

// Join renders the current frame in full, for a viewer starting now or one
// that missed diffs. ok is false before any frame.
func (s *Stream) Join() (m Message, ok bool, err error) {
	if !s.started {
		return Message{}, false, nil
	}
	data, err := terminal.Render(terminal.Frame{}, s.last)
	if err != nil {
		return Message{}, false, err
	}
	return Message{Cols: s.last.Geometry.Cols, Rows: s.last.Geometry.Rows, Data: data}, true, nil
}
