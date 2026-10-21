package recording

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

var updateSketchCorpus = flag.Bool("update-sketch", false, "regenerate shared sketch prefix contract after an intentional model change")

type sketchCorpusPrefix struct {
	Error      bool             `json:"error"`
	T          int64            `json:"t"`
	Control    Control          `json:"control"`
	Projection SketchProjection `json:"projection"`
}
type sketchCorpusCase struct {
	Name     string               `json:"name"`
	Records  []json.RawMessage    `json:"records"`
	Prefixes []sketchCorpusPrefix `json:"prefixes"`
}
type sketchCanonicalVector struct {
	Record    json.RawMessage `json:"record"`
	Canonical string          `json:"canonical"`
	Digest    string          `json:"digest"`
}
type sketchCorpus struct {
	Cases     []sketchCorpusCase      `json:"cases"`
	Canonical []sketchCanonicalVector `json:"canonical"`
}

func corpusCase(t *testing.T, name string, events ...Record) sketchCorpusCase {
	t.Helper()
	b := NewBuilder(name)
	b.Page(SegLive, "http://x/")
	recs := append(b.Recs, events...)
	s := NewState()
	c := sketchCorpusCase{Name: name, Records: []json.RawMessage{}, Prefixes: []sketchCorpusPrefix{}}
	for i, r := range recs {
		r.SetTime(int64(i))
		raw, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		err = s.Apply(decoded)
		c.Records = append(c.Records, raw)
		c.Prefixes = append(c.Prefixes, sketchCorpusPrefix{Error: err != nil, T: s.T, Control: s.Control, Projection: s.Sketches.Snapshot()})
	}
	return c
}
func makeSketchCorpus(t *testing.T) sketchCorpus {
	p := func(n uint64, id string, chunk uint64, pts ...SketchPoint) *SketchPoints {
		return &SketchPoints{SketchOperation: sketchOp(n), Sketch: "sk", ID: id, Chunk: chunk, Pts: pts}
	}
	end := func(n uint64, id string) *SketchStrokeEnd {
		return &SketchStrokeEnd{SketchOperation: sketchOp(n), Sketch: "sk", ID: id}
	}
	who := Author{ID: "driver", Name: "Ana", Color: "#00ff00"}
	rawBegin, _ := Encode(sketchBegin())
	var extra map[string]json.RawMessage
	_ = json.Unmarshal(rawBegin, &extra)
	extra["future"] = json.RawMessage(`{ "payload" : [1, 2] }`)
	var author map[string]json.RawMessage
	_ = json.Unmarshal(extra["author"], &author)
	author["future"] = json.RawMessage(`{ "x" : true }`)
	extra["author"], _ = json.Marshal(author)
	rawBegin, _ = json.Marshal(extra)
	future, err := Decode(rawBegin)
	if err != nil {
		t.Fatal(err)
	}
	drivenBegin := sketchBegin()
	drivenBegin.Author = who
	drivenBegin.Owner = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	c := sketchCorpus{Cases: []sketchCorpusCase{
		corpusCase(t, "driver-DOM", &Ctl{Op: CtlRequest, Who: who, Control: Control{Pending: []Author{who}}}, &Ctl{Op: CtlGrant, Who: who, Control: Control{Driver: &who}}, drivenBegin, &Click{Source: Source{By: who.ID}, ID: 3, X: 1, Y: 2}),
		corpusCase(t, "driver-reset", &Ctl{Op: CtlRequest, Who: who, Control: Control{Pending: []Author{who}}}, &Ctl{Op: CtlGrant, Who: who, Control: Control{Driver: &who}}, drivenBegin, sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1, 2, .4}), &Ctl{Op: CtlReset, Who: Author{ID: OperatorID}}, &SketchLabel{SketchOperation: sketchOp(3), Sketch: "sk", Label: "Reset"}),
		corpusCase(t, "navigate-open", sketchBegin(), sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1, 2, .4}), &Seg{Seg: 2, Why: SegNavigate, Tab: Tab{ID: "tab2", URL: "http://x/2"}, VP: Viewport{W: 100, H: 100, DPR: 1}}, &SketchLabel{SketchOperation: sketchOp(3), Sketch: "sk", Label: "Navigated"}),
		corpusCase(t, "cast-end-open", sketchBegin(), sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1, 2, .4}), &Stop{Why: StopEnd}, &SketchLabel{SketchOperation: sketchOp(3), Sketch: "sk", Label: "too late"}),
		corpusCase(t, "clear-open", sketchBegin(), sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1, 2, .5}), &AnnotationClear{Who: Author{ID: OperatorID}}),
		corpusCase(t, "nested-unknown", future, sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1, 2, .5})),
		corpusCase(t, "history", sketchBegin(), sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1, 2, 0}, SketchPoint{3, 4, .5}), p(2, "a", 0, SketchPoint{1, 2, 0}, SketchPoint{3, 4, .5}), p(2, "a", 0, SketchPoint{99, 2, 0}), p(3, "a", 8, SketchPoint{5, 6, 1}), p(3, "a", 1, SketchPoint{5, 6, 1}), end(4, "a"), &SketchUndo{SketchOperation: sketchOp(5), Sketch: "sk", ID: "a"}, &SketchRedo{SketchOperation: sketchOp(6), Sketch: "sk", ID: "a"}, &SketchUndo{SketchOperation: sketchOp(7), Sketch: "sk", ID: "a"}, sketchStroke(8, "b"), end(9, "b"), &SketchRedo{SketchOperation: sketchOp(10), Sketch: "sk", ID: "a"}, &SketchLabel{SketchOperation: sketchOp(10), Sketch: "sk", Label: "Saved <design>"}, sketchStroke(11, "c")),
		corpusCase(t, "control-and-clear", sketchBegin(), sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1, 2, .3}), &Ctl{Op: CtlRequest, Who: who, Control: Control{Pending: []Author{who}}}, &Ctl{Op: CtlGrant, Who: who}, &Ctl{Op: CtlGrant, Who: who, Control: Control{Driver: &who}}, end(3, "a"), &SketchLabel{SketchOperation: sketchOp(4), Sketch: "sk", Label: "Partial"}, &Seg{Seg: 2, Why: SegNavigate, Tab: Tab{ID: "tab1", URL: "http://x/2"}, VP: Viewport{W: 100, H: 100, DPR: 1}}, &AnnotationClear{Who: Author{ID: OperatorID}}, &SketchLabel{SketchOperation: sketchOp(5), Sketch: "sk", Label: "too late"}),
		corpusCase(t, "pause-resume-expiry", sketchBegin(), sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{-10, 2, 1}), &Stop{Why: StopPause}, p(3, "a", 1, SketchPoint{1, 1, 1}), end(3, "a"), &Seg{Seg: 2, Why: SegResume, Tab: Tab{ID: "tab1", URL: "http://x/"}, VP: Viewport{W: 100, H: 100, DPR: 1}}, sketchStroke(4, "b"), p(5, "b", 0, SketchPoint{0, 0, 0}), &SketchInterrupt{SketchOperation: sketchOp(6), Sketch: "sk", Reason: SketchInterruptExpiry}, &SketchLabel{SketchOperation: sketchOp(7), Sketch: "sk", Label: "Expired"}, &Stop{Why: StopEnd}),
	}, Canonical: []sketchCanonicalVector{}}
	for _, r := range []Record{sketchBegin(), sketchStroke(1, "a"), p(2, "a", 0, SketchPoint{1e-7, -0., .5}, SketchPoint{1e7, -1e7, 1}), &SketchLabel{SketchOperation: sketchOp(3), Sketch: "sk", Label: "<>&\u2028\u2029 漢字"}} {
		raw, err := Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		canonical, err := SketchCanonical(r)
		if err != nil {
			t.Fatal(err)
		}
		digest, err := SketchDigest(r)
		if err != nil {
			t.Fatal(err)
		}
		c.Canonical = append(c.Canonical, sketchCanonicalVector{raw, string(canonical), digest})
	}
	return c
}
func TestSketchSharedCorpus(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tests/webcast/sketch-corpus.json")
	want := makeSketchCorpus(t)
	if *updateSketchCorpus {
		raw, err := json.MarshalIndent(want, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got sketchCorpus
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	// Marshal compacts RawMessages, making fixture indentation irrelevant.
	normalizedWant, _ := json.Marshal(want)
	normalizedGot, _ := json.Marshal(got)
	if !bytes.Equal(normalizedGot, normalizedWant) {
		t.Fatal("shared corpus differs; inspect semantics before -update-sketch")
	}
}
