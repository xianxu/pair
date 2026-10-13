package recording

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestCapabilityRecordingPrefixesAndFiles(t *testing.T) {
	data, err := os.ReadFile("testdata/capability-events.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var recs []Record
	files := map[string][]byte{}
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		r, err := Decode(line)
		if err != nil {
			t.Fatal(err)
		}
		recs = append(recs, r)
		file := FileOf(r.Kind())
		files[file] = append(append(files[file], line...), '\n')
		full, err := Reduce(recs, r.Time())
		if err != nil {
			t.Fatal(err)
		}
		joined, err := Reduce(CatchUp(recs), r.Time())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(full.Canonical(), joined.Canonical()) || !reflect.DeepEqual(full.Notes, joined.Notes) || !reflect.DeepEqual(full.Strokes, joined.Strokes) || !full.Control.Equal(joined.Control) {
			t.Fatalf("prefix ending in %s at %d differs after catch-up", r.Kind(), r.Time())
		}
	}
	findings, end := Validate(files)
	if len(findings) != 0 {
		t.Fatal(findings)
	}
	if len(end.Notes) != 1 || end.Notes[0].ID != "new" || len(end.Strokes) != 0 || !end.Control.Equal(Control{}) {
		t.Fatalf("files resurrected cleared state: %+v", end)
	}
}

func TestAnnotationClearTimeline(t *testing.T) {
	for _, paused := range []bool{false, true} {
		b := NewBuilder("clear")
		p := b.El("p", nil)
		b.Page(SegLive, "http://x/", p)
		b.At(1).Add(&Note{ID: "n", Author: ana, At: NoteAt{Node: p.ID}})
		b.Add(&Stroke{ID: "s", Author: ana, Pts: [][3]float64{{1, 2, 0}}})
		if paused {
			b.At(2).Add(&Stop{Why: StopPause})
		}
		b.At(3).Add(&AnnotationClear{Who: Author{ID: OperatorID}})
		b.At(4).Add(&AnnotationClear{Who: Author{ID: OperatorID}})
		if !paused {
			b.At(5).Add(&Stop{Why: StopPause})
		}
		b.At(6).Page(SegResume, "http://x/", b.El("p", nil))
		before, err := Reduce(b.Recs, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(before.Notes) != 1 || len(before.Strokes) != 1 {
			t.Fatal("history lost")
		}
		for _, at := range []int64{3, 4, 6} {
			end := 0
			for end < len(b.Recs) && b.Recs[end].Time() <= at {
				end++
			}
			full, err := Reduce(b.Recs[:end], at)
			if err != nil {
				t.Fatal(err)
			}
			joined, err := Reduce(CatchUp(b.Recs[:end]), at)
			if err != nil {
				t.Fatal(err)
			}
			if len(full.Notes) != 0 || len(full.Strokes) != 0 || len(joined.Notes) != 0 || len(joined.Strokes) != 0 {
				t.Fatalf("paused=%v at=%d resurrected marks", paused, at)
			}
			if !bytes.Equal(full.Canonical(), joined.Canonical()) {
				t.Fatal("catch-up DOM drift")
			}
		}
	}
}

func TestAnnotationClearAdmissionAndCodec(t *testing.T) {
	for _, who := range []Author{{}, {ID: "viewer"}, {ID: OperatorID}} {
		b := NewBuilder("clear")
		b.Page(SegLive, "http://x/")
		b.At(1).Add(&AnnotationClear{Who: who})
		_, err := Reduce(b.Recs, 1)
		if (err == nil) != (who.ID == OperatorID) {
			t.Fatalf("who=%+v err=%v", who, err)
		}
	}
	b := NewBuilder("clear")
	b.Page(SegLive, "http://x/")
	b.At(1).Add(&Stop{Why: StopEnd})
	b.At(2).Add(&AnnotationClear{Who: Author{ID: OperatorID}})
	if _, err := Reduce(b.Recs, 2); err == nil {
		t.Fatal("clear after end accepted")
	}
	r := &AnnotationClear{Who: Author{ID: OperatorID}}
	line, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(line)
	if err != nil || !reflect.DeepEqual(r, got) {
		t.Fatalf("roundtrip %s: %v %+v", line, err, got)
	}
	if FileOf(r.Kind()) != FileEvents {
		t.Fatal("clear must share ordered event stream")
	}
}

func TestControlResetOperatorAndBaseline(t *testing.T) {
	prior := Control{Driver: &ana, Pending: []Author{bo}}
	for _, who := range []Author{{}, {ID: "v1"}, {ID: OperatorID}} {
		next, err := NextControl(prior, CtlReset, who)
		if who.ID != OperatorID {
			if err == nil {
				t.Fatal("viewer reset accepted")
			}
			continue
		}
		if err != nil || !next.Equal(Control{}) {
			t.Fatalf("reset: %+v %v", next, err)
		}
		again, err := NextControl(next, CtlReset, who)
		if err != nil || !again.Equal(next) {
			t.Fatal("reset not idempotent")
		}
	}
	if prior.Driver != &ana || len(prior.Pending) != 1 {
		t.Fatal("input changed")
	}
	for _, state := range []Control{{}, {Driver: &ana}, {Pending: []Author{bo}}} {
		b := NewBuilder("reset")
		b.Page(SegLive, "http://x/")
		b.At(1).Add(&Ctl{Op: CtlReset, Who: Author{ID: OperatorID}, Control: state})
		_, err := Reduce(b.Recs, 1)
		if (err == nil) != state.Equal(Control{}) {
			t.Fatalf("reset baseline %+v err=%v", state, err)
		}
	}
}
