package recording

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #458's additive v1 kinds: hover and ptr are tracked by the reducer, reset
// by a segment, and stay out of Canonical.
func TestHoverAndPtr(t *testing.T) {
	p := newPage()
	before, _ := Reduce(p.b.Recs, 1<<62)
	p.b.Add(&Hover{ID: p.div.ID})
	p.b.Add(&Ptr{X: 12.5, Y: 40})
	s := p.reduce(t)
	if s.Hover != p.div.ID || s.Ptr == nil || *s.Ptr != [2]float64{12.5, 40} {
		t.Fatalf("hover %d ptr %v", s.Hover, s.Ptr)
	}
	if !bytes.Equal(before.Canonical(), s.Canonical()) {
		t.Error("hover and ptr must stay outside Canonical")
	}
	mustFail(t, s, &Hover{ID: 999}, "unknown")
	// The hovered element goes: hover is none, the pointer stays.
	p.b.Add(&Rm{ID: p.div.ID})
	if s = p.reduce(t); s.Hover != 0 || s.Ptr == nil {
		t.Errorf("after rm: hover %d ptr %v", s.Hover, s.Ptr)
	}
	p.b.Add(&Hover{ID: 0})
	p.b.Add(&Stop{Why: StopPause})
	p.b.Add(&Seg{Seg: p.b.NextSeg(), Why: SegNavigate, Tab: Tab{ID: "t"}, VP: Viewport{W: 1, H: 1, DPR: 1}})
	p.b.Snapshot("http://x/2", p.b.El("body", nil))
	if s = p.reduce(t); s.Hover != 0 || s.Ptr != nil {
		t.Errorf("a segment resets hover and ptr: %d %v", s.Hover, s.Ptr)
	}
	// Both live in events.jsonl and round-trip.
	for _, r := range []Record{&Hover{ID: 3}, &Ptr{X: 1, Y: 2}} {
		r.header().K = r.Kind()
		if FileOf(r.Kind()) != FileEvents {
			t.Errorf("%s is not in %s", r.Kind(), FileEvents)
		}
		line, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		back, err := Decode(line)
		if err != nil || back.Kind() != r.Kind() {
			t.Fatalf("%s: %v %v", line, back, err)
		}
	}
	if line, _ := Encode(&Hover{Header: Header{K: "hover"}}); !strings.Contains(string(line), `"id":0`) {
		t.Errorf("hover none must say id 0: %s", line)
	}
}

// A password's value never appears in a recording: an input record says
// secret instead, and a password input carries no value attribute.
func TestPasswordValuesAreRefused(t *testing.T) {
	p := newPage()
	pw := p.b.El("input", []string{"type", "Password"})
	txt := p.b.El("input", []string{"type", "text"})
	p.b.Add(&Ins{P: p.body.ID, Node: pw})
	p.b.Add(&Ins{P: p.body.ID, Node: txt})
	p.b.Add(&Input{ID: pw.ID, V: Str("")}) // empty says nothing
	p.b.Add(&Input{ID: pw.ID, Secret: true})
	p.b.Add(&Input{ID: txt.ID, V: Str("shown")})
	s := p.reduce(t)
	if in := s.Inputs[pw.ID]; in == nil || !in.Secret {
		t.Fatalf("password input state: %+v", in)
	}
	mustFail(t, s, &Input{ID: pw.ID, V: Str("hunter2")}, "password")
	mustFail(t, s, &Input{ID: txt.ID, V: Str("x"), Secret: true}, "secret")
	mustFail(t, s, &Attr{ID: pw.ID, Name: "value", V: Str("hunter2")}, "password")
	mustFail(t, s, &Ins{P: p.body.ID, Node: p.b.El("input", []string{"type", "password", "value", "hunter2"})}, "password")
	mustFail(t, s, &Kids{P: p.ul.ID, C: []Node{p.b.El("input", []string{"type", "password", "value", "x"})}}, "password")

	// Flipping a control with a value attribute to password is refused; the
	// value a text control showed earlier stays recorded (accepted: it was
	// on screen).
	withValue := p.b.El("input", []string{"type", "text", "value", "v"})
	p.b.Add(&Ins{P: p.body.ID, Node: withValue})
	s = p.reduce(t)
	mustFail(t, s, &Attr{ID: withValue.ID, Name: "type", V: Str("password")}, "password")

	// A whole snapshot carrying one is refused too.
	b := NewBuilder("pw")
	b.Page(SegLive, "http://x/", b.El("input", []string{"type", "password", "value", "x"}))
	if _, err := Reduce(b.Recs, 1<<62); err == nil || !strings.Contains(err.Error(), "password") {
		t.Errorf("a doc with a password value: %v", err)
	}
	// secret reaches Canonical's $input.
	if c := string(s.Canonical()); !strings.Contains(c, `"secret": true`) {
		t.Errorf("Canonical lacks secret:\n%s", c)
	}
}

// #457's open advisory: ValidateDir reports anything in frames/ that isn't
// the index or a frame's png, such as a stray .tmp-* from a crash.
func TestValidateDirReportsStrayFrameFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	w, err := NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range newPage().b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.PutFrame(&Frame{File: "frames/0.png", W: 2, H: 2, Scale: 1}, []byte("png")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if fs, _, _ := ValidateDir(dir); len(fs) != 0 {
		t.Fatalf("clean: %v", fs)
	}
	for _, stray := range []string{".tmp-123", "7.png"} { // 7.png: no frame record names it
		if err := os.WriteFile(filepath.Join(dir, "frames", stray), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fs, _, _ := ValidateDir(dir)
	all := ""
	for _, f := range fs {
		all += f.String() + "\n"
	}
	for _, want := range []string{"frames/.tmp-123", "frames/7.png"} {
		if !strings.Contains(all, want) {
			t.Errorf("ValidateDir lacks %s:\n%s", want, all)
		}
	}
}
