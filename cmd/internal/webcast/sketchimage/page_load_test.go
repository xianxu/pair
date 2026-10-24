package sketchimage

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestSamePageViewIncludesExactComparableLoadIdentity(t *testing.T) {
	scene := pageFixture(t, 1).Scene
	raw, _ := json.Marshal(scene)
	var wire map[string]json.RawMessage
	json.Unmarshal(raw, &wire)
	wire["page_load"] = json.RawMessage(`{"target":"tab1","loader":"one","root":"9223372036854775807"}`)
	raw, _ = json.Marshal(wire)
	var a PageScene
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	wire["page_load"] = json.RawMessage(`{"target":"tab1","loader":"two","root":"9223372036854775807"}`)
	raw, _ = json.Marshal(wire)
	var b PageScene
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if SamePageView(a, b) {
		t.Fatal("different loads entered same page view")
	}
	if a.SketchSourceFacts == b.SketchSourceFacts {
		t.Fatal("scalar source witness ignores load identity")
	}
	var _ = map[query.SketchSourceFacts]bool{a.SketchSourceFacts: true}
}

func TestPageLoadSourceFinishAndCurrentWitnessForgeryRefuses(t *testing.T) {
	known := query.PageLoadFacts{Target: "tab1", Loader: "source", Root: "9223372036854775807"}
	fixture := func() PageBatch {
		b := pageFixture(t, 1)
		b.Records[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "source", Root: math.MaxInt64}
		b.Sketches[0].Source.PageLoad = known
		b.Finishes[0].PageLoad = known
		b.Scene.PageLoad = known
		return b
	}
	if _, err := ValidatePageBatch(fixture()); err != nil {
		t.Fatalf("known load refused: %v", err)
	}
	for _, change := range []func(*PageBatch){
		func(b *PageBatch) { b.Sketches[0].Source.PageLoad.Loader = "forged" },
		func(b *PageBatch) { b.Finishes[0].PageLoad.Root = "9223372036854775806" },
		func(b *PageBatch) { b.Scene.PageLoad.Target = "other" },
		func(b *PageBatch) { b.Scene.PageLoad = query.PageLoadFacts{} },
	} {
		b := fixture()
		change(&b)
		if _, err := ValidatePageBatch(b); err == nil {
			t.Fatal("forged load witness accepted")
		}
	}
	b := fixture()
	b.Current = &CurrentView{Projections: []PathProjection{{SketchID: "s0", PathID: "p", At: b.Sketches[0].History.Strokes[0].Begin.At, Clip: recording.Box{0, 0, 1280, 800}}}}
	if _, err := ValidatePageBatch(b); err != nil {
		t.Fatalf("current known scene refused: %v", err)
	}
	b.Scene.PageLoad.Loader = "forged"
	if _, err := ValidatePageBatch(b); err == nil {
		t.Fatal("forged current scene accepted")
	}
}
