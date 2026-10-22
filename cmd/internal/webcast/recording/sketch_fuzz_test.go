package recording

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// Seed each concrete variant so byte mutations reach each typed decoder,
// including nested preserving objects, rather than only unknown envelopes.
func FuzzSketchCodec(f *testing.F) {
	for _, line := range sketchWireSamples() {
		f.Add([]byte(line))
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			f.Fatal(err)
		}
		for key := range fields {
			mutated := make(map[string]json.RawMessage, len(fields))
			for k, v := range fields {
				mutated[k] = v
			}
			mutated[key] = json.RawMessage("null")
			raw, err := json.Marshal(mutated)
			if err != nil {
				f.Fatal(err)
			}
			f.Add(raw)
		}
	}
	for _, line := range []string{
		strings.Replace(sketchWireSamples()[3], `"id":"s"`, `"id":"s","truncated":true`, 1),
		strings.Replace(sketchWireSamples()[3], `"id":"s"`, `"id":"s","truncated":null`, 1),
		strings.Replace(sketchWireSamples()[0], `"id":"sk"`, `"id":"\ud800"`, 1),
		strings.Replace(sketchWireSamples()[0], `"id":"sk"`, `"id":"\udfff"`, 1),
		strings.Replace(sketchWireSamples()[0], `"id":"sk"`, `"id":"\ud83d\ude00"`, 1),
		strings.Replace(sketchWireSamples()[0], `"name":"Drawer"`, `"name":"`+string([]byte{0xff})+`"`, 1),
		strings.Replace(sketchWireSamples()[1], `"node":3`, `"node":3,"stable":{"custom":"\ud800"}`, 1),
		strings.Replace(sketchWireSamples()[1], `"node":3`, `"node":3,"stable":{"\ud800":"value"}`, 1),
		strings.Replace(sketchWireSamples()[1], `"node":3`, `"node":3,"path":["\udfff"]`, 1),
		strings.Replace(sketchWireSamples()[0], `"author":{`, `"future":"\ud800","author":{`, 1),
		strings.Replace(sketchWireSamples()[0], `"author":{`, `"future":"`+string([]byte{0xff})+`","author":{`, 1),

		strings.Replace(sketchWireSamples()[0], `"author":{`, `"unknown": { "array": [1,  2, {"key":true}] },"author":{"future": { "spacing":  1 },`, 1),
		strings.Replace(sketchWireSamples()[0], `"vp":{`, `"vp":{"future": [1,  2],`, 1),
		strings.Replace(sketchWireSamples()[1], `"node":3`, `"node":3,"future": { "nested": [true,  false] },"stable":{"id":"button","future": { "nested": true }}`, 1),
		strings.Replace(sketchWireSamples()[0], `"author":{`, `"author":{"duplicate":1,"duplicate":2,`, 1),
		strings.Replace(sketchWireSamples()[1], `"node":3`, `"node":null`, 1),
		strings.Replace(sketchWireSamples()[1], `[1,2,30,40]`, `[1,2,30,40,50]`, 1),
		strings.Replace(sketchWireSamples()[2], `[1,2,0]`, `[1,2,null]`, 1),
		strings.Replace(sketchWireSamples()[2], `[1,2,0]`, `[1,2]`, 1),
		strings.Replace(sketchWireSamples()[2], `[1,2,0]`, `[1,2,0,9]`, 1),
		strings.Replace(sketchWireSamples()[2], `"sequence":0`, `"sequence":9007199254740992`, 1),
	} {
		f.Add([]byte(line))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		r, err := Decode(raw)
		if err != nil || sketchMetadata(r) == nil {
			return
		}
		if err := r.check(); err != nil {
			t.Fatalf("Decode accepted invalid sketch: %v", err)
		}
		semantic, err := SketchSemantic(r)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := Encode(r)
		if err != nil {
			t.Fatalf("accepted sketch cannot encode: %v", err)
		}
		again, err := Decode(encoded)
		if err != nil {
			t.Fatalf("encoded sketch cannot decode: %v; %s", err, encoded)
		}
		if err := again.check(); err != nil {
			t.Fatalf("roundtrip invalid: %v", err)
		}
		semanticAgain, err := SketchSemantic(again)
		if err != nil {
			t.Fatal(err)
		}
		if semantic != semanticAgain {
			t.Fatalf("semantic changed: %s -> %s", semantic, semanticAgain)
		}
		reencoded, err := Encode(again)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded, reencoded) {
			t.Fatalf("preserved representation changed on rewrite:\n%s\n%s", encoded, reencoded)
		}
	})
}
