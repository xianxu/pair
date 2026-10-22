package broadcast

import (
	"sync"
	"testing"
)

func onPointer(t *testing.T) *PointerState {
	t.Helper()
	p := newPointerState(nil)
	if _, _, err := p.set(true, func() (string, error) { return "tok", nil }); err != nil {
		t.Fatal(err)
	}
	return p
}

func closed(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}

func TestPointerNotesAddResolve(t *testing.T) {
	p := onPointer(t)
	_, v0, changed := p.Notes()
	n, err := p.addNote(Note{Box: CellBox{1, 1, 2, 1}, Text: "a"}, 80, 24, 1, p.generation())
	if err != nil {
		t.Fatal(err)
	}
	notes, v1, _ := p.Notes()
	if v1 <= v0 || !closed(changed) || len(notes) != 1 {
		t.Fatalf("add: version %d→%d, closed %v, %d notes", v0, v1, closed(changed), len(notes))
	}
	_, _, changed = p.Notes()
	if _, err := p.addNote(Note{Box: CellBox{1, 1, 2, 1}, Text: "b"}, 80, 24, 1, p.generation()); err != ErrNoteOverlap {
		t.Fatalf("overlap: %v", err)
	}
	if ok, err := p.resolveNote(NoteResolution{ID: 999}); ok || err != nil {
		t.Fatalf("unknown id: %v %v", ok, err)
	}
	if _, v, _ := p.Notes(); v != v1 || closed(changed) {
		t.Fatal("a refused add or an unknown resolve changed the board")
	}
	if ok, err := p.resolveNote(NoteResolution{ID: n.ID}); !ok || err != nil {
		t.Fatalf("resolve: %v %v", ok, err)
	}
	if notes, v, _ := p.Notes(); len(notes) != 0 || v <= v1 || !closed(changed) {
		t.Fatal("resolve didn't change the board")
	}
}

func TestPointerNotesRefusedWhileOff(t *testing.T) {
	p := newPointerState(nil)
	if _, err := p.addNote(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1, p.generation()); err != ErrPointerOff {
		t.Fatalf("add before on: %v", err)
	}
	if _, err := p.resolveNote(NoteResolution{ID: 1}); err != ErrPointerOff {
		t.Fatalf("resolve before on: %v", err)
	}
}

// Every flip to off empties the board in the same critical section.
func TestPointerNotesClearedOnEveryOff(t *testing.T) {
	offs := map[string]func(p *PointerState){
		"set":             func(p *PointerState) { p.set(false, nil) },
		"offIfGeneration": func(p *PointerState) { p.offIfGeneration(p.generation()) },
		"stop":            func(p *PointerState) { p.stop() },
	}
	for name, off := range offs {
		t.Run(name, func(t *testing.T) {
			p := onPointer(t)
			p.addNote(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1, p.generation())
			p.addNote(Note{Box: CellBox{4, 0, 2, 1}, Text: "b"}, 80, 24, 1, p.generation())
			_, v, changed := p.Notes()
			off(p)
			notes, v2, _ := p.Notes()
			if len(notes) != 0 || v2 <= v || !closed(changed) {
				t.Fatalf("after off: %d notes, version %d→%d, closed %v", len(notes), v, v2, closed(changed))
			}
		})
	}
}

func TestPointerNotesStaleWatchClearsNothing(t *testing.T) {
	p := onPointer(t)
	stale := p.generation()
	p.set(false, nil)
	p.set(true, nil)
	p.addNote(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1, p.generation())
	if p.offIfGeneration(stale) {
		t.Fatal("a stale report flipped pointing")
	}
	if notes, _, _ := p.Notes(); len(notes) != 1 {
		t.Fatal("a stale report cleared the board")
	}
}

// Every interleaving of one add and one flip leaves no note while pointing
// is off: the generation the add carries is the seam.
func TestPointerNotesAddFlipInterleavings(t *testing.T) {
	t.Run("flip, then add", func(t *testing.T) {
		p := onPointer(t)
		gen := p.generation()
		p.set(false, nil)
		if _, err := p.addNote(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1, gen); err != ErrPointerOff {
			t.Fatalf("add after off: %v", err)
		}
	})
	t.Run("off and on, then add", func(t *testing.T) {
		p := onPointer(t)
		gen := p.generation()
		p.set(false, nil)
		p.set(true, nil)
		if _, err := p.addNote(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1, gen); err != ErrPointerOff {
			t.Fatalf("add checked before the flips: %v", err)
		}
		if notes, _, _ := p.Notes(); len(notes) != 0 {
			t.Fatal("a stale add landed")
		}
	})
	t.Run("add, then flip", func(t *testing.T) {
		p := onPointer(t)
		if _, err := p.addNote(Note{Box: CellBox{0, 0, 2, 1}, Text: "a"}, 80, 24, 1, p.generation()); err != nil {
			t.Fatal(err)
		}
		_, v, _ := p.Notes()
		p.set(false, nil)
		if notes, v2, _ := p.Notes(); len(notes) != 0 || v2 <= v {
			t.Fatal("the flip didn't clear the add")
		}
	})
}

func TestPointerNotesRaceStress(t *testing.T) {
	p := onPointer(t)
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				if n, err := p.addNote(Note{Box: CellBox{(i % 30) * 2, w, 2, 1}, Color: w, Text: "x"}, 80, 24, 1, p.generation()); err == nil && i%2 == 0 {
					p.resolveNote(NoteResolution{ID: n.ID})
				}
			}
		}(w)
	}
	for i := 0; i < 1000; i++ {
		p.set(i%2 == 1, nil)
	}
	close(stop)
	wg.Wait()
	p.set(false, nil)
	if notes, _, _ := p.Notes(); len(notes) != 0 {
		t.Fatalf("%d notes left with pointing off", len(notes))
	}
}
