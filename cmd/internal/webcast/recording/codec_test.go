package recording

import (
	"reflect"
	"strings"
	"testing"
)

// sample is one record of every kind, so a test over it covers the table.
func sample() []Record {
	box := Box{1, 2, 3, 4}
	t, f := true, Pos{5, 1}
	records := []Record{
		&Head{Format: Format, V: Version, ID: "r1", Started: "2026-10-11T23:00:00Z"},
		&Seg{Seg: 1, Why: SegLive, Tab: Tab{ID: "T", URL: "http://x/"}, VP: Viewport{W: 1280, H: 800, DPR: 1}},
		&Doc{Doc: 1, URL: "http://x/", Tree: Node{ID: 1, Y: YDoc, C: []Node{{ID: 2, Y: YDoctype, N: "html"}, {ID: 3, Y: YEl, N: "html"}}}},
		&Ins{P: 3, After: 0, Node: Node{ID: 4, Y: YText, V: Str("")}},
		&Rm{ID: 4},
		&Kids{P: 3, C: []Node{{ID: 6, Y: YComment, V: Str("c")}}},
		&Attr{ID: 3, Name: "lang", V: Str("en")},
		&Attr{ID: 3, Name: "lang"},
		&Text{ID: 6, V: "d"},
		&Shadow{Host: 3, Root: Node{ID: 7, Y: YShadow, Mode: "closed"}},
		&Sheet{ID: 8, Text: "a{}"},
		&Adopt{ID: 1, Sheets: []NodeID{8}},
		&Font{Family: "F", Src: "http://x/f.ttf"},
		&Asset{URL: "http://x/a.css", Hash: strings.Repeat("ab", 32), MIME: "text/css", Size: 3},
		&Input{ID: 3, Checked: &t},
		&Scroll{ID: 1, X: 0, Y: 10},
		&Selection{Anchor: &f, Focus: &f},
		&Nav{URL: "http://x/#a"},
		&Title{V: "T"},
		&VP{Viewport: Viewport{W: 800, H: 600, DPR: 2}},
		&Click{ID: 3, X: 1, Y: 2},
		&Ctl{Op: CtlRequest, Who: Author{ID: "v1", Name: "Ana"}, Control: Control{Pending: []Author{{ID: "v1", Name: "Ana"}}}},
		&Ctl{Op: CtlGrant, Who: Author{ID: "v1", Name: "Ana"}, Control: Control{Driver: &Author{ID: "v1", Name: "Ana"}}},
		&Click{Source: Source{By: "v1"}, ID: 3, X: 1, Y: 2},
		&Focus{ID: 3},
		&Layout{Boxes: map[NodeID]Box{3: box}},
		&Note{ID: "n1", Author: Author{ID: "a"}, Text: "hi", At: NoteAt{Node: 3, Box: &box}},
		&NoteEdit{ID: "n1", Text: "updated"},
		&NoteState{ID: "n1", State: NoteResolved, Who: &Author{ID: "operator"}},
		&Stroke{ID: "s1", Author: Author{ID: "a"}, Pts: [][3]float64{{1, 2, 0}}},
		&Hover{ID: 3},
		&Ptr{X: 1.5, Y: 2},
		&Frame{File: "frames/0.png", W: 640, H: 400, Scale: 0.5},
		&AnnotationClear{Who: Author{ID: OperatorID}},
		&Ctl{Op: CtlReset, Who: Author{ID: OperatorID}},
		&Stop{Why: StopEnd},
	}
	for _, raw := range sketchWireSamples() {
		r, err := Decode([]byte(raw))
		if err != nil {
			panic(err)
		}
		records = append(records, r)
	}
	return records
}

func TestCodecRoundTripsEveryKind(t *testing.T) {
	seen := map[string]bool{}
	for i, r := range sample() {
		r.header().T = int64(i)
		r.header().K = r.Kind()
		line, err := Encode(r)
		if err != nil {
			t.Fatalf("%s: %v", r.Kind(), err)
		}
		got, err := Decode(line)
		if err != nil {
			t.Fatalf("%s: decode %s: %v", r.Kind(), line, err)
		}
		if !reflect.DeepEqual(got, r) {
			t.Errorf("%s: round trip\n got %#v\nwant %#v", r.Kind(), got, r)
		}
		seen[r.Kind()] = true
	}
	for k := range kinds {
		if !seen[k] {
			t.Errorf("kind %q has no sample", k)
		}
	}
}

func TestEncodeSetsKindAndEnvelopeFirst(t *testing.T) {
	line, err := Encode(&Rm{Header: Header{T: 7, K: "wrong"}, ID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(line), `{"t":7,"k":"rm","id":3}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestSubmissionRevisionGuardNeverEntersRecordingWire(t *testing.T) {
	revision := uint64(42)
	guarded := &NoteState{ID: "n1", State: NoteResolved, IfRevision: &revision}
	line, err := Encode(guarded)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Encode(&NoteState{ID: "n1", State: NoteResolved})
	if err != nil {
		t.Fatal(err)
	}
	if string(line) != string(want) {
		t.Fatalf("process-only revision guard leaked into wire: %s", line)
	}
	decoded, err := Decode(line)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.(*NoteState).IfRevision != nil {
		t.Fatal("decoded wire acquired a process-only revision guard")
	}
}

func TestAttributionEncodesOnlyWhenSet(t *testing.T) {
	for _, c := range []struct {
		r    Record
		want string
	}{
		{&Click{ID: 3, X: 1, Y: 2}, `{"t":0,"k":"click","id":3,"x":1,"y":2}`},
		{&Click{Source: Source{By: "v1"}, ID: 3, X: 1, Y: 2}, `{"t":0,"k":"click","by":"v1","id":3,"x":1,"y":2}`},
		{&Ctl{Op: CtlRevoke, Who: Author{ID: "v1"}}, `{"t":0,"k":"ctl","op":"revoke","who":{"id":"v1"}}`},
	} {
		line, err := Encode(c.r)
		if err != nil || string(line) != c.want {
			t.Errorf("got %s %v, want %s", line, err, c.want)
		}
	}
}

func TestAttrRemovalEncodesNull(t *testing.T) {
	line, _ := Encode(&Attr{ID: 3, Name: "x"})
	if !strings.Contains(string(line), `"v":null`) {
		t.Errorf("removal must carry v:null, got %s", line)
	}
	r, _ := Decode(line)
	if r.(*Attr).V != nil {
		t.Error("v:null must decode to nil")
	}
}

func TestUnknownKindIsKeptVerbatim(t *testing.T) {
	line := []byte(`{"t":3,"k":"wiggle","id":9,"extra":[1,2]}`)
	r, err := Decode(line)
	if err != nil {
		t.Fatal(err)
	}
	u, ok := r.(*Unknown)
	if !ok || u.Kind() != "wiggle" || u.Time() != 3 {
		t.Fatalf("got %#v", r)
	}
	out, _ := Encode(r)
	if string(out) != string(line) {
		t.Errorf("re-encoded %s, want %s", out, line)
	}
}

func TestDecodeNeedsEnvelope(t *testing.T) {
	for _, line := range []string{`{"k":"rm","id":1}`, `{"t":1,"id":1}`, `{"t":1,"k":""}`, `not json`} {
		if _, err := Decode([]byte(line)); err == nil {
			t.Errorf("%s: want an error", line)
		}
	}
}

func TestFileOf(t *testing.T) {
	for k, want := range map[string]string{"attr": FileEvents, "note": FileNotes, "note-state": FileNotes, "stroke": FilePointer, "frame": FileFrames, "hover": FileEvents, "wiggle": FileEvents} {
		if got := FileOf(k); got != want {
			t.Errorf("FileOf(%q) = %q, want %q", k, got, want)
		}
	}
}

func TestChecksRejectMissingFields(t *testing.T) {
	for _, r := range []Record{
		&Head{V: 1, ID: "x", Started: "s"}, // wrong format
		&Seg{Seg: 1, Why: "jump", VP: Viewport{W: 1, H: 1}},
		&Doc{Doc: 1, Tree: Node{ID: 2, Y: YDoc}},
		&Ins{P: 1, Node: Node{ID: 2, Y: YEl}},
		&Ins{P: 1, Node: Node{ID: 2, Y: YText}},
		&Shadow{Host: 1, Root: Node{ID: 2, Y: YShadow}},
		&Asset{URL: "u", Hash: "ABC", MIME: "x"},
		&Input{ID: 1},
		&Note{ID: "n", Author: Author{ID: "a"}},
	} {
		if r.check() == nil {
			t.Errorf("%s %#v: want a check error", r.Kind(), r)
		}
	}
	for _, r := range sample() {
		if err := r.check(); err != nil {
			t.Errorf("%s: sample fails its check: %v", r.Kind(), err)
		}
	}
}

func TestEncodeDoesNotMutateSharedRecords(t *testing.T) {
	for _, r := range sample() {
		r.header().K = "deliberately-wrong"
		data, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		if r.header().K != "deliberately-wrong" {
			t.Errorf("%s mutated the shared envelope", r.Kind())
		}
		decoded, err := Decode(data)
		if err != nil || decoded.Kind() != r.Kind() {
			t.Fatalf("kind correction: %s %v", data, err)
		}
	}
}
