package board

import (
	"errors"
	"strings"
	"testing"
)

const (
	goodStroke = `{"viewer":{"id":"ana","name":"Ana"},"seg":1,"id":"s1","color":2,"pts":[[1,2,0],[3.5,4,16]]}`
	goodNote   = `{"viewer":{"id":"ana","name":" Ana "},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":"Too small"}`
	goodRegion = `{"viewer":{"id":"ana"},"seg":2,"node":7,"box":[1,2,3,4],"region":[10,20,30,40],"color":5,"text":"here"}`
	goodState  = `{"viewer":{"id":"ana","name":"Ana"},"id":"0123abcd","state":"resolved"}`
)

func TestParseAcceptsWellFormedRequests(t *testing.T) {
	s, err := ParseStroke([]byte(goodStroke))
	if err != nil || s.Viewer != (Viewer{"ana", "Ana"}) || s.Seg != 1 || s.ID != "s1" || s.Color != 2 || len(s.Pts) != 2 || s.Pts[1] != [3]float64{3.5, 4, 16} {
		t.Errorf("stroke %+v %v", s, err)
	}
	n, err := ParseNote([]byte(goodNote))
	if err != nil || n.Viewer.Name != "Ana" || n.Node != 7 || n.Box != [4]float64{1, 2, 3, 4} || n.Region != nil || n.Text != "Too small" {
		t.Errorf("note %+v %v", n, err)
	}
	n, err = ParseNote([]byte(goodRegion))
	if err != nil || n.Region == nil || *n.Region != [4]float64{10, 20, 30, 40} || n.Viewer.Name != "" {
		t.Errorf("region note %+v %v", n, err)
	}
	st, err := ParseNoteState([]byte(goodState))
	if err != nil || st.ID != "0123abcd" || st.State != "resolved" {
		t.Errorf("state %+v %v", st, err)
	}
}

func TestParseRefusals(t *testing.T) {
	long := strings.Repeat("x", 1025)
	pts65 := "[" + strings.TrimSuffix(strings.Repeat("[1,1,0],", 65), ",") + "]"
	for _, c := range []struct {
		name  string
		parse func([]byte) error
		body  string
	}{
		{"stroke: unknown field", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[[1,1,0]],"x":1}`},
		{"stroke: no viewer", stroke, `{"seg":1,"id":"s","color":0,"pts":[[1,1,0]]}`},
		{"stroke: operator", stroke, `{"viewer":{"id":"operator"},"seg":1,"id":"s","color":0,"pts":[[1,1,0]]}`},
		{"stroke: bad viewer id", stroke, `{"viewer":{"id":"a b"},"seg":1,"id":"s","color":0,"pts":[[1,1,0]]}`},
		{"stroke: bad stroke id", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s/1","color":0,"pts":[[1,1,0]]}`},
		{"stroke: seg 0", stroke, `{"viewer":{"id":"a"},"seg":0,"id":"s","color":0,"pts":[[1,1,0]]}`},
		{"stroke: color 6", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":6,"pts":[[1,1,0]]}`},
		{"stroke: no points", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[]}`},
		{"stroke: 65 points", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":` + pts65 + `}`},
		{"stroke: 4-tuple", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[[1,1,0,9]]}`},
		{"stroke: 2-tuple", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[[1,1]]}`},
		{"stroke: null coordinate", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[[null,1,0]]}`},
		{"stroke: huge coordinate", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[[2e6,1,0]]}`},
		{"stroke: overflowing number", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[[1e999,1,0]]}`},
		{"stroke: negative dt", stroke, `{"viewer":{"id":"a"},"seg":1,"id":"s","color":0,"pts":[[1,1,-1]]}`},
		{"stroke: two objects", stroke, goodStroke + goodStroke},
		{"note: no box", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"color":0,"text":"x"}`},
		{"note: 3-box", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"box":[1,2,3],"color":0,"text":"x"}`},
		{"note: negative width", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"box":[1,2,-3,4],"color":0,"text":"x"}`},
		{"note: empty region", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"box":[1,2,3,4],"region":[1,2,0,4],"color":0,"text":"x"}`},
		{"note: node 0", note, `{"viewer":{"id":"a"},"seg":1,"node":0,"box":[1,2,3,4],"color":0,"text":"x"}`},
		{"note: blank text", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":" \n "}`},
		{"note: long text", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":"` + long + `"}`},
		{"note: bidi override", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":"a‮b"}`},
		{"note: control char", note, `{"viewer":{"id":"a"},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":"a\u0007"}`},
		{"note: newline in name", note, `{"viewer":{"id":"a","name":"A\nB"},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":"x"}`},
		{"note: long name", note, `{"viewer":{"id":"a","name":"` + strings.Repeat("n", 41) + `"},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":"x"}`},
		{"note: operator", note, `{"viewer":{"id":"operator"},"seg":1,"node":7,"box":[1,2,3,4],"color":0,"text":"x"}`},
		{"note: invalid UTF-8", note, "{\"viewer\":{\"id\":\"a\"},\"seg\":1,\"node\":7,\"box\":[1,2,3,4],\"color\":0,\"text\":\"\xff\"}"},
		{"state: bad state", state, `{"viewer":{"id":"a"},"id":"n","state":"done"}`},
		{"state: no viewer", state, `{"id":"n","state":"open"}`},
		{"state: operator", state, `{"viewer":{"id":"operator"},"id":"n","state":"open"}`},
		{"state: bad id", state, `{"viewer":{"id":"a"},"id":"","state":"open"}`},
	} {
		if err := c.parse([]byte(c.body)); !errors.Is(err, ErrShape) {
			t.Errorf("%s: %v, want ErrShape", c.name, err)
		}
	}
}

func stroke(b []byte) error { _, err := ParseStroke(b); return err }
func note(b []byte) error   { _, err := ParseNote(b); return err }
func state(b []byte) error  { _, err := ParseNoteState(b); return err }

// The parsers never panic, never echo input in an error, and what they
// accept is within bounds.
func FuzzParse(f *testing.F) {
	for _, s := range []string{goodStroke, goodNote, goodRegion, goodState, `{"viewer":{"id":"operator"}}`, "\xff", `{"pts":[[1e999]]}`, strings.Repeat("[", 10000)} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		if s, err := ParseStroke(body); err == nil {
			if len(s.Pts) == 0 || len(s.Pts) > 64 || s.Viewer.ID == Operator || s.Color < 0 || s.Color > 5 {
				t.Errorf("accepted out of bounds: %+v", s)
			}
		} else if err != ErrShape {
			t.Errorf("error %v", err)
		}
		if n, err := ParseNote(body); err == nil {
			if n.Viewer.ID == Operator || n.Node < 1 || len(n.Text) > 1024 || n.Box[2] < 0 || n.Box[3] < 0 {
				t.Errorf("accepted out of bounds: %+v", n)
			}
		} else if err != ErrShape {
			t.Errorf("error %v", err)
		}
		if s, err := ParseNoteState(body); err == nil && (s.Viewer.ID == Operator || (s.State != "open" && s.State != "resolved")) {
			t.Errorf("accepted %+v", s)
		}
	})
}
