package recording

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func TestSketchDigestCanonicalAndSemanticIdentity(t *testing.T) {
	r, err := Decode([]byte(sketchWireSamples()[2]))
	if err != nil {
		t.Fatal(err)
	}
	r.(*SketchPoints).Pts = []SketchPoint{{math.Copysign(0, -1), 1e-7, 0.5}}
	canonical, err := SketchCanonical(r)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"chunk":0,"id":"s","operation":"op","pts":[[0,1e-7,0.5]],"sequence":0,"sketch":"sk"}`
	if string(canonical) != want {
		t.Fatalf("canonical %s want %s", canonical, want)
	}
	digest, _ := SketchDigest(r)
	if digest != "53df46280d4b6c08373997c33095bc153785f44a8aa64513eff7187f911dce10" {
		t.Fatal(digest)
	}
	semantic, _ := SketchSemantic(r)
	r.SetTime(99)
	r.header().K = "wrong"
	r.(*SketchPoints).Digest = "changed"
	got, _ := SketchCanonical(r)
	if !bytes.Equal(canonical, got) {
		t.Fatalf("writer metadata changed digest: %s", got)
	}
	if next, _ := SketchSemantic(r); next != semantic {
		t.Fatal("writer metadata changed receipt semantics")
	}
	r.(*SketchPoints).Pts[0][0] = 1
	if changed, _ := SketchDigest(r); changed == digest {
		t.Fatal("different body kept digest")
	}
	r2, err := Decode([]byte(sketchWireSamples()[6]))
	if err != nil {
		t.Fatal(err)
	}
	r2.(*SketchLabel).Label = "<>&\u2028\u2029 café 😀"
	got, _ = SketchCanonical(r2)
	if !bytes.Contains(got, []byte(`\u003c\u003e\u0026\u2028\u2029 café 😀`)) {
		t.Fatalf("escaping drift: %s", got)
	}
}

func TestSketchOwnedEditsPreserveUnknownAndEncodeIsImmutable(t *testing.T) {
	raw := bytes.Replace([]byte(sketchWireSamples()[0]), []byte(`"id":"operator"`), []byte(`"id":"operator","future": { "spacing":  2 }`), 1)
	r, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	begin := r.(*SketchBegin)
	begin.Author.Name = "Updated"
	begin.K = "wrong"
	before, _ := json.Marshal(begin)
	out, err := Encode(begin)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(begin)
	if !bytes.Equal(before, after) {
		t.Fatal("Encode mutated shared record")
	}
	if !bytes.Contains(out, []byte(`{ "spacing":  2 }`)) || !bytes.Contains(out, []byte(`"name":"Updated"`)) {
		t.Fatalf("lost edit or unknown: %s", out)
	}
	again, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if again.(*SketchBegin).Author.Name != "Updated" {
		t.Fatal("owned edit lost")
	}
}

func TestSketchTypedValidationBounds(t *testing.T) {
	r, err := Decode([]byte(sketchWireSamples()[2]))
	if err != nil {
		t.Fatal(err)
	}
	p := r.(*SketchPoints)
	p.Sequence = MaxSketchInteger
	p.Chunk = MaxSketchInteger
	p.T = int64(MaxSketchInteger)
	if err = p.check(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(){func() { p.Pts = nil }, func() { p.Pts = make([]SketchPoint, 65) }, func() { p.Pts = []SketchPoint{{math.NaN(), 0, 0}} }, func() { p.Pts = []SketchPoint{{0, 0, math.Inf(1)}} }, func() { p.Pts = []SketchPoint{{0, 0, -0.1}} }} {
		p.Pts = []SketchPoint{{1, 2, 0.5}}
		mutate()
		if p.check() == nil {
			t.Fatal("accepted invalid typed point")
		}
	}
}
