package recording

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

const loadSegment = `{"t":1,"k":"seg","seg":1,"why":"live","tab":{"id":"T","url":"https://x/"},"vp":{"w":800,"h":600,"dpr":1},"page_load":{"target":"T","loader":"L","root":"9223372036854775807"}}`

func TestSegmentPageLoadStrictWire(t *testing.T) {
	for _, bad := range []string{
		`null`, `{}`, `{"target":"T","loader":"L"}`, `{"target":"","loader":"L","root":"1"}`,
		`{"target":"OTHER","loader":"L","root":"1"}`, `{"target":"T","loader":"","root":"1"}`,
		`{"target":"T","loader":"L","root":1}`, `{"target":"T","loader":"L","root":null}`,
		`{"target":"T","loader":"L","root":"0"}`, `{"target":"T","loader":"L","root":"01"}`,
		`{"target":"T","loader":"L","root":"+1"}`, `{"target":"T","loader":"L","root":" 1"}`,
		`{"target":"T","loader":"L","root":"1e2"}`, `{"target":"T","loader":"L","root":"1.0"}`,
		`{"target":"T","loader":"L","root":"9223372036854775808"}`,
		`{"target":"T","loader":"L","root":"1","root":"2"}`,
		`{"target":"T","loader":"\ud800","root":"1"}`,
		`{"target":"T","loader":"L","root":"1","future":{"x":1,"x":2}}`,
	} {
		line := loadSegment[:strings.Index(loadSegment, `,"page_load"`)] + `,"page_load":` + bad + `}`
		if _, err := Decode([]byte(line)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestSegmentPreservesRecursiveRawExtensionsAndExactHeader(t *testing.T) {
	line := strings.Replace(loadSegment, `"tab":{`, `"future": { "n": 1e9999 },"K":"stop","tab":{"future":[1,  2],`, 1)
	line = strings.Replace(line, `"vp":{`, `"vp":{"future": { "n": 1e9999 },`, 1)
	line = strings.Replace(line, `"loader":"L"`, `"future": { "nested": [1,  2] },"loader":"L"`, 1)
	r, err := Decode([]byte(line))
	if err != nil {
		t.Fatal(err)
	}
	seg, ok := r.(*Seg)
	if !ok {
		t.Fatalf("case alias changed kind to %T", r)
	}
	seg.Why = SegTab
	out, err := Encode(seg)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{ "n": 1e9999 }`, `[1,  2]`, `{ "nested": [1,  2] }`, `"page_load"`, `"why":"tab"`} {
		if !bytes.Contains(out, []byte(raw)) {
			t.Errorf("lost %s in %s", raw, out)
		}
	}
	if bytes.Count(out, []byte(`{ "n": 1e9999 }`)) != 2 {
		t.Fatalf("segment and viewport extensions must both survive: %s", out)
	}
}

// This models an older *preserving* segment schema. Already-shipped plain
// encoding/json readers cannot be retroactively made extension-preserving.
func TestSegmentFrozenPreservingOlderSchemaRewrite(t *testing.T) {
	type olderSegment struct {
		Header
		Seg int      `json:"seg"`
		Why string   `json:"why"`
		Tab Tab      `json:"tab"`
		VP  Viewport `json:"vp"`
	}
	frozen := &segmentSchema{fields: map[string]bool{"t": true, "k": true, "seg": true, "why": true, "tab": true, "vp": true}, children: map[string]*segmentSchema{"tab": segmentTabSchema, "vp": segmentVPSchema}}
	line := strings.Replace(loadSegment, `"loader":"L"`, `"loader":"L","future": { "n": 1e9999 }`, 1)
	clean, wire, err := splitSegmentObject([]byte(line), frozen)
	if err != nil {
		t.Fatal(err)
	}
	var older olderSegment
	if err = json.Unmarshal(clean, &older); err != nil {
		t.Fatal(err)
	}
	older.Why = SegTab
	typed, err := json.Marshal(older)
	if err != nil {
		t.Fatal(err)
	}
	rewritten, err := mergeSegmentObject(typed, wire, frozen)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Decode(rewritten)
	if err != nil {
		t.Fatal(err)
	}
	seg := r.(*Seg)
	if seg.Why != SegTab || seg.PageLoad == nil || seg.PageLoad.Root != math.MaxInt64 {
		t.Fatalf("lost identity: %s", rewritten)
	}
	out, err := Encode(seg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte(`{ "n": 1e9999 }`)) {
		t.Fatalf("lost nested extension: %s", out)
	}
}

func TestSegmentLegacyAcceptanceAndCodecOwnership(t *testing.T) {
	for _, line := range []string{`{"t":0,"k":"seg","seg":1,"why":"live","tab":null,"vp":{"w":1,"h":1}}`, `{"t":0,"k":"seg"}`} {
		if _, err := Decode([]byte(line)); err != nil {
			t.Fatalf("tightened unrelated legacy fields: %v", err)
		}
	}
	r, err := Decode([]byte(loadSegment))
	if err != nil {
		t.Fatal(err)
	}
	seg := r.(*Seg)
	seg.K = "untrusted"
	before := *seg
	out, err := Encode(seg)
	if err != nil {
		t.Fatal(err)
	}
	if seg.K != before.K || !bytes.Contains(out, []byte(`"k":"seg"`)) {
		t.Fatal("encoding mutated or trusted caller kind")
	}
	for _, line := range []string{
		strings.Replace(loadSegment, `"k":"seg"`, `"k":"seg","k":"stop"`, 1),
		strings.Replace(loadSegment, `"seg":1`, `"seg":1,"seg":2`, 1),
		strings.Replace(loadSegment, `"id":"T"`, `"id":"T","future":{"x":1,"x":2}`, 1),
		strings.Replace(loadSegment, `"dpr":1`, `"dpr":1,"future":{"x":1,"x":2}`, 1),
		strings.Replace(loadSegment, `"loader":"L"`, `"Loader":"L"`, 1),
	} {
		if _, err := Decode([]byte(line)); err == nil {
			t.Fatalf("accepted malformed object: %s", line)
		}
	}
}

func FuzzSegmentPageLoadRoundTrip(f *testing.F) {
	f.Add(loadSegment)
	f.Add(strings.Replace(loadSegment, `"vp":{`, `"future": { "n": 1e9999 },"vp":{`, 1))
	f.Add(`{"t":0,"k":"seg","tab":null}`)
	f.Fuzz(func(t *testing.T, line string) {
		r, err := Decode([]byte(line))
		if err != nil {
			return
		}
		seg, ok := r.(*Seg)
		if !ok {
			return
		}
		raw, err := Encode(seg)
		if err != nil {
			t.Fatal(err)
		}
		again, err := Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		repeated, err := Encode(again)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(raw, repeated) {
			t.Fatalf("rewrite not stable: %s vs %s", raw, repeated)
		}
	})
}
