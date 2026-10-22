package recording

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sketchWireSamples() []string {
	common := `"t":1,"operation":"op","sequence":0,"digest":"` + strings.Repeat("a", 64) + `"`
	return []string{
		`{` + common + `,"k":"sketch-begin","id":"sk","author":{"id":"operator","name":"Drawer","color":"#abcdef"},"owner":"operator","doc":1,"tab":"tab","vp":{"w":800,"h":600,"dpr":1}}`,
		`{` + common + `,"k":"sketch-stroke-begin","sketch":"sk","id":"s","at":{"node":3,"box":[1,2,30,40]},"width":3}`,
		`{` + common + `,"k":"sketch-points","sketch":"sk","id":"s","chunk":0,"pts":[[1,2,0],[3,4,0.5],[5,6,1]]}`,
		`{` + common + `,"k":"sketch-stroke-end","sketch":"sk","id":"s"}`,
		`{` + common + `,"k":"sketch-undo","sketch":"sk","id":"s"}`,
		`{` + common + `,"k":"sketch-redo","sketch":"sk","id":"s"}`,
		`{` + common + `,"k":"sketch-label","sketch":"sk","label":"Move this button"}`,
		`{` + common + `,"k":"sketch-interrupt","sketch":"sk","reason":"expiry"}`,
	}
}

func TestSketchWireEveryKind(t *testing.T) {
	for _, line := range sketchWireSamples() {
		r, err := Decode([]byte(line))
		if err != nil {
			t.Fatal(err)
		}
		if !Known(r.Kind()) {
			t.Errorf("missing sketch registration: %s", r.Kind())
			continue
		}
		if FileOf(r.Kind()) != FileEvents {
			t.Fatal("sketch not in events")
		}
		out, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		var a, b any
		json.Unmarshal([]byte(line), &a)
		json.Unmarshal(out, &b)
		aa, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		if !bytes.Equal(aa, bb) {
			t.Errorf("roundtrip %s\n%s", aa, bb)
		}
	}
}

func TestSketchDecodeRefusesMalformedOwnedFields(t *testing.T) {
	begin, stroke, points, label := sketchWireSamples()[0], sketchWireSamples()[1], sketchWireSamples()[2], sketchWireSamples()[6]
	cases := []string{
		strings.Replace(begin, `"operation":"op",`, "", 1),
		strings.Replace(begin, `"sequence":0`, `"sequence":null`, 1),
		strings.Replace(begin, `"sequence":0`, `"sequence":9007199254740992`, 1),
		strings.Replace(begin, `"sequence":0`, `"sequence":0.5`, 1),
		strings.Replace(begin, `"doc":1`, `"doc":0`, 1),
		strings.Replace(begin, `"t":1`, `"t":9007199254740992`, 1),
		strings.Replace(begin, `"color":"#abcdef"`, `"color":"red"`, 1),
		strings.Replace(begin, `"owner":"operator"`, `"owner":"secret"`, 1),
		strings.Replace(begin, `"vp":{"w":800,"h":600,"dpr":1}`, `"vp":{"w":800,"w":600,"h":600,"dpr":1}`, 1),
		strings.Replace(begin, `"vp":{"w":800,"h":600,"dpr":1}`, `"vp":{"w":800,"h":600}`, 1),
		strings.Replace(stroke, `[1,2,30,40]`, `[1,2,30]`, 1),
		strings.Replace(stroke, `[1,2,30,40]`, `[1,2,30,40,50]`, 1),
		strings.Replace(stroke, `[1,2,30,40]`, `[1,2,0,40]`, 1),
		strings.Replace(stroke, `"width":3`, `"width":0`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"stable":{"id":null}`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"path":[null]`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"classes":[null]`, 1),
		strings.Replace(points, `[1,2,0]`, `[1,2]`, 1),
		strings.Replace(points, `[1,2,0]`, `[1,2,0,9]`, 1),
		strings.Replace(points, `[1,2,0]`, `[1,2,1.1]`, 1),
		strings.Replace(points, `[1,2,0]`, `[10000001,2,0]`, 1),
		strings.Replace(points, `"chunk":0`, `"chunk":-1`, 1),
		strings.Replace(label, `Move this button`, strings.Repeat("a", 1025), 1),
		strings.Replace(label, `Move this button`, `   `, 1),
		strings.Replace(label, `Move this button`, `\u202eattack`, 1),
		strings.Replace(begin, `"tab":"tab"`, `"tab":"bad\u0001"`, 1),
		strings.Replace(begin, `"tab":"tab"`, `"tab":"`+strings.Repeat("x", 129)+`"`, 1),
		strings.Replace(sketchWireSamples()[7], `"expiry"`, `"arbitrary"`, 1),
	}
	for i, line := range cases {
		if _, err := Decode([]byte(line)); err == nil {
			t.Errorf("case %d accepted malformed: %s", i, line)
		}
	}
	for _, line := range sketchWireSamples() {
		var m map[string]json.RawMessage
		json.Unmarshal([]byte(line), &m)
		for key := range m {
			copy := map[string]json.RawMessage{}
			for k, v := range m {
				copy[k] = v
			}
			delete(copy, key)
			bad, _ := json.Marshal(copy)
			if _, err := Decode(bad); err == nil {
				t.Errorf("accepted missing %s in %s", key, line)
			}
		}
	}
}

func TestSketchNestedUnknownRawAndOwnedEdits(t *testing.T) {
	line := strings.Replace(sketchWireSamples()[0], `"author":{`, `"future": { "nested": [1,  2] },"author":{"extra": { "a":  1 },`, 1)
	line = strings.Replace(line, `"vp":{`, `"vp":{"extra":[true,  false],`, 1)
	r, err := Decode([]byte(line))
	if err != nil {
		t.Fatal(err)
	}
	if !Known(r.Kind()) {
		t.Fatal("sketch-begin is unknown")
	}
	out, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{ "nested": [1,  2] }`, `{ "a":  1 }`, `[true,  false]`} {
		if !bytes.Contains(out, []byte(raw)) {
			t.Errorf("lost raw %s: %s", raw, out)
		}
	}
	stroke := strings.Replace(sketchWireSamples()[1], `"node":3`, `"node":3,"future": { "v":  7 },"stable":{"id":"save","future": { "structured": true }}`, 1)
	r, err = Decode([]byte(stroke))
	if err != nil {
		t.Fatal(err)
	}
	out, err = Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{ "v":  7 }`, `{ "structured": true }`} {
		if !bytes.Contains(out, []byte(raw)) {
			t.Errorf("lost anchor raw %s: %s", raw, out)
		}
	}
	for _, bad := range []string{strings.Replace(line, `"extra": { "a":  1 }`, `"extra":{"x":1,"x":2}`, 1), strings.Replace(stroke, `"node":3`, `"node":null`, 1)} {
		if _, err := Decode([]byte(bad)); err == nil {
			t.Errorf("accepted duplicate/null: %s", bad)
		}
	}
}

// Pin the pre-sketch registry at its actual decoder seam. Its unchanged
// DecodeFile, ReadDir and Validate must tolerate additive v1 event kinds;
// copying a reducer here would merely test a second implementation.
func TestSketchFrozenLegacyReaderCompatibility(t *testing.T) {
	b := NewBuilder("legacy-reader")
	b.Page(SegLive, "http://example.test/")
	baseline := append([]Record(nil), b.Recs...)
	pointer := &Stroke{Header: Header{T: 100}, ID: "pointer", Author: Author{ID: "operator"}, Pts: [][3]float64{{1, 2, 0}}}
	baseline = append(baseline, pointer)
	want, err := Reduce(baseline, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var events bytes.Buffer
	for _, r := range b.Recs {
		line, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		events.Write(line)
		events.WriteByte('\n')
	}
	for i, line := range sketchWireSamples() {
		var m map[string]json.RawMessage
		json.Unmarshal([]byte(line), &m)
		m["t"] = json.RawMessage([]byte(stringInt(10 + i)))
		encoded, _ := json.Marshal(m)
		events.Write(encoded)
		events.WriteByte('\n')
	}
	ptr, err := Encode(pointer)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{FileEvents: events.Bytes(), FilePointer: append(ptr, '\n')}
	original := kinds
	frozen := make(map[string]struct {
		new  func() Record
		file string
	})
	for _, k := range strings.Fields(`annotation-clear head seg stop doc ins rm kids attr text shadow sheet adopt font asset input scroll selection nav title vp click focus layout hover ptr ctl note note-edit note-state stroke frame`) {
		frozen[k] = original[k]
	}
	kinds = frozen
	defer func() { kinds = original }()
	located, bad := DecodeFile(FileEvents, files[FileEvents])
	if len(bad) != 0 {
		t.Fatal(bad)
	}
	unknown := 0
	for _, l := range located {
		if _, ok := l.Record.(*Unknown); ok {
			unknown++
		}
	}
	if unknown != 8 {
		t.Fatalf("legacy decoded %d unknown kinds", unknown)
	}
	findings, got := Validate(files)
	if errs := Errors(findings); len(errs) != 0 {
		t.Fatal(errs)
	}
	if len(findings) != 8 {
		t.Fatalf("legacy findings: %v", findings)
	}
	if !bytes.Equal(want.Canonical(), got.Canonical()) || len(got.Strokes) != 1 || got.Strokes[0].Stroke.ID != "pointer" {
		t.Fatal("legacy DOM/pointer semantics changed")
	}
	dir := t.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	rec, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	reduced, err := Reduce(rec.Records, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want.Canonical(), reduced.Canonical()) || len(reduced.Strokes) != 1 {
		t.Fatal("legacy directory reduction changed")
	}
}

func stringInt(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestSketchOwnedUnicodeIsStrictAndUnknownUnicodePreserved(t *testing.T) {
	begin, stroke, label := sketchWireSamples()[0], sketchWireSamples()[1], sketchWireSamples()[6]
	cases := []string{
		strings.Replace(begin, `"id":"sk"`, `"id":"\ud800"`, 1),
		strings.Replace(begin, `"tab":"tab"`, `"tab":"\udfff"`, 1),
		strings.Replace(begin, `"operation":"op"`, `"operation":"\ud800x"`, 1),
		strings.Replace(begin, `"name":"Drawer"`, `"name":"\ud800\ud800"`, 1),
		strings.Replace(begin, `"name":"Drawer"`, `"name":"`+string([]byte{0xff})+`"`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"tag":"\ud800"`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"stable":{"id":"\ud800"}`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"stable":{"custom":"\udfff"}`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"stable":{"\ud800":"value"}`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"classes":["\ud800"]`, 1),
		strings.Replace(stroke, `"node":3`, `"node":3,"path":["\udfff"]`, 1),
		strings.Replace(label, `Move this button`, `\ud800`, 1),
	}
	for i, raw := range cases {
		if _, err := Decode([]byte(raw)); err == nil {
			t.Errorf("accepted invalid owned Unicode case %d: %q", i, raw)
		}
	}
	for _, raw := range []string{
		strings.Replace(begin, `"id":"sk"`, `"id":"\ud83d\ude00"`, 1),
		strings.Replace(label, `Move this button`, `valid literal � and escaped \ufffd and emoji \ud83d\ude00`, 1),
		strings.Replace(label, `Move this button`, `literal escape \\ud800`, 1),
	} {
		if _, err := Decode([]byte(raw)); err != nil {
			t.Errorf("refused legal Unicode %q: %v", raw, err)
		}
	}
	unknowns := []struct{ raw, value string }{
		{strings.Replace(begin, `"author":{`, `"future":"\ud800","author":{`, 1), `"future":"\ud800"`},
		{strings.Replace(begin, `"author":{`, `"author":{"future":"\udfff",`, 1), `"future":"\udfff"`},
		{strings.Replace(begin, `"vp":{`, `"vp":{"future":["\ud800"],`, 1), `"future":["\ud800"]`},
		{strings.Replace(stroke, `"node":3`, `"node":3,"future":"\ud800"`, 1), `"future":"\ud800"`},
		{strings.Replace(stroke, `"node":3`, `"node":3,"stable":{"future":{"raw":"\ud800"}}`, 1), `"future":{"raw":"\ud800"}`},
		{strings.Replace(begin, `"author":{`, `"future":"`+string([]byte{0xff})+`","author":{`, 1), `"future":"` + string([]byte{0xff}) + `"`},
	}
	for _, tc := range unknowns {
		r, err := Decode([]byte(tc.raw))
		if err != nil {
			t.Errorf("tightened unknown Unicode: %v", err)
			continue
		}
		encoded, err := Encode(r)
		if err != nil || !bytes.Contains(encoded, []byte(tc.value)) {
			t.Errorf("lost raw unknown Unicode: %q %v", encoded, err)
		}
	}
}
