package broadcast

import (
	"errors"
	"time"
)

// Note requests (#447) share one budget per broadcast, apart from points.
const (
	NoteRatePerSec = 2
	NoteBurst      = 5
)

// ErrPointerOff refuses a note change while pointing is off, or a note
// checked against a screen from before the latest flip.
var ErrPointerOff = errors.New("broadcast: pointing is off")

// Notes is the open notes, their version, and a channel closed at the next
// change.
func (p *PointerState) Notes() ([]Note, uint64, <-chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.board.List(), p.notesVersion, p.notesChanged
}

// addNote opens n, placed on a cols×rows grid, if pointing is on and
// hasn't flipped since gen, the generation the caller read before checking
// the screen.
func (p *PointerState) addNote(n Note, cols, rows, chromeRows int, gen uint64) (Note, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.on || p.gen != gen {
		return Note{}, ErrPointerOff
	}
	n, err := p.board.Add(n, cols, rows, chromeRows)
	if err != nil {
		return Note{}, err
	}
	p.notesChangedLocked()
	return n, nil
}

// resolveNote removes note id while pointing is on, reporting whether there
// was one.
func (p *PointerState) resolveNote(event NoteResolution) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.on {
		return false, ErrPointerOff
	}
	found := false
	for _, n := range p.board.notes {
		if n.ID == event.ID {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}
	if len(p.resolutions) == MaxNoteResolutions {
		copy(p.resolutions, p.resolutions[1:])
		p.resolutions = p.resolutions[:MaxNoteResolutions-1]
	}
	p.resolutions = append(p.resolutions, event)
	p.board.Resolve(event.ID)
	p.notesChangedLocked()
	return true, nil
}

// allowNote takes one request from the note budget.
func (p *PointerState) allowNote(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.noteLimit.Allow(now)
}

// clearNotesLocked empties the board on a flip to off.
func (p *PointerState) clearNotesLocked() {
	if len(p.board.notes) == 0 {
		return
	}
	p.board.Clear()
	p.notesChangedLocked()
}

func (p *PointerState) notesChangedLocked() {
	p.notesVersion++
	close(p.notesChanged)
	p.notesChanged = make(chan struct{})
}
