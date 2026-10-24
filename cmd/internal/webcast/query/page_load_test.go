package query

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestQueryNotesExposeHistoricalPageLoadsAndOmitLegacyUnknown(t *testing.T) {
	b := recording.NewBuilder("loads")
	for i, load := range []*recording.PageLoad{{Target: "tab1", Loader: "first", Root: math.MaxInt64}, {Target: "tab1", Loader: "reloaded", Root: 2}, {Target: "other", Loader: "first", Root: math.MaxInt64}, nil} {
		b.At(int64(i + 1))
		node := b.El("button", nil)
		before := len(b.Recs)
		b.Page(recording.SegNavigate, "https://same/", node)
		seg := b.Recs[before].(*recording.Seg)
		seg.PageLoad = load
		if load != nil {
			seg.Tab.ID = load.Target
		}
		b.Add(&recording.Note{ID: string(rune('a' + i)), Author: recording.Author{ID: "operator"}, Text: "retained", At: recording.NoteAt{Node: node.ID}})
	}
	for _, surface := range []Surface{New(b.Recs, notes.FakeModel{}), FakeSurface{Records: b.Recs, Model: notes.FakeModel{}}} {
		views, err := surface.Notes(5000)
		if err != nil || len(views) != 4 {
			t.Fatalf("history: %+v, %v", views, err)
		}
		var out bytes.Buffer
		if err = notesJSON(views, &out); err != nil {
			t.Fatal(err)
		}
		var wire []map[string]json.RawMessage
		if err = json.Unmarshal(out.Bytes(), &wire); err != nil {
			t.Fatal(err)
		}
		for i, want := range []string{`{"target":"tab1","loader":"first","root":"9223372036854775807"}`, `{"target":"tab1","loader":"reloaded","root":"2"}`, `{"target":"other","loader":"first","root":"9223372036854775807"}`} {
			var got, expected map[string]string
			if err = json.Unmarshal(wire[i]["page_load"], &got); err != nil {
				t.Fatalf("note %d missing provenance: %s", i, out.Bytes())
			}
			json.Unmarshal([]byte(want), &expected)
			if got["target"] != expected["target"] || got["loader"] != expected["loader"] || got["root"] != expected["root"] {
				t.Fatalf("note %d: %+v", i, got)
			}
			if text := NoteText(views[i], 5000); !strings.Contains(text, `root="`+expected["root"]+`"`) || !strings.Contains(text, `loader="`+expected["loader"]+`"`) {
				t.Fatalf("text lacks load provenance: %s", text)
			}
		}
		if _, ok := wire[3]["page_load"]; ok {
			t.Fatalf("legacy note invented identity: %s", wire[3]["page_load"])
		}
	}
}

func TestSketchSourcePageLoadJSONAndUnchangedCanonicalRequests(t *testing.T) {
	for _, known := range []bool{false, true} {
		recs, _ := sketchRecords(1)
		if known {
			recs[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "original", Root: math.MaxInt64}
		}
		canonical := map[int]string{}
		for i, r := range recs {
			if raw, err := recording.SketchCanonical(r); err == nil {
				canonical[i] = string(raw)
			}
		}
		views, err := New(recs, notes.Structural{}).SketchesAt(math.MaxInt64)
		if err != nil || len(views) != 1 {
			t.Fatalf("history %+v, %v", views, err)
		}
		raw, err := json.Marshal(views[0].Source)
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]json.RawMessage
		json.Unmarshal(raw, &wire)
		load, ok := wire["page_load"]
		if ok != known {
			t.Fatalf("known=%v source=%s", known, raw)
		}
		if known {
			var got map[string]string
			if err = json.Unmarshal(load, &got); err != nil {
				t.Fatal(err)
			}
			if got["root"] != "9223372036854775807" || got["loader"] != "original" || got["target"] != "tab1" {
				t.Fatalf("rounded identity: %s", load)
			}
		}
		for i, want := range canonical {
			raw, err := recording.SketchCanonical(recs[i])
			if err != nil || string(raw) != want {
				t.Fatal("query altered canonical sketch")
			}
		}
	}
}

func TestPageLoadFactsAreComparableOwnedSemanticValues(t *testing.T) {
	for _, load := range []*recording.PageLoad{nil, {}, {Target: "T", Loader: "L", Root: 0}, {Target: "T", Loader: "", Root: 1}} {
		if got := PageLoadFactsOf(load); got != (PageLoadFacts{}) {
			t.Fatalf("invalid load became provenance: %+v", got)
		}
	}
	raw := `{"t":1,"k":"seg","seg":1,"why":"live","tab":{"id":"T","url":"https://x"},"vp":{"w":1,"h":1},"page_load":{"target":"T","loader":"L","root":"9223372036854775807","future":{"n":1e9999}}}`
	record, err := recording.Decode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	load := record.(*recording.Seg).PageLoad
	facts := PageLoadFactsOf(load)
	if facts != (PageLoadFacts{Target: "T", Loader: "L", Root: "9223372036854775807"}) {
		t.Fatalf("semantic projection %+v", facts)
	}
	load.Loader = "mutated"
	if facts.Loader != "L" {
		t.Fatal("projection aliases source")
	}
	wire, err := json.Marshal(struct {
		PageLoad PageLoadFacts `json:"page_load,omitzero"`
	}{facts})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(wire, []byte("future")) || !bytes.Contains(wire, []byte(`"root":"9223372036854775807"`)) {
		t.Fatalf("wire projection %s", wire)
	}
	var _ = map[SketchSourceFacts]bool{{PageLoad: facts}: true}
}

func TestFrozenNoteDrawContextUsesItsDrawBoundaryPageLoad(t *testing.T) {
	b := recording.NewBuilder("draw-context")
	node := b.El("button", nil)
	b.Page(recording.SegLive, "https://same/", node)
	b.Recs[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "draw", Root: math.MaxInt64}
	b.At(10).Page(recording.SegNavigate, "https://same/", b.El("p", nil))
	b.Recs[3].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "save", Root: 2}
	note := &recording.Note{Header: recording.Header{T: 11}, ID: "frozen", Author: recording.Author{ID: "operator"}, Text: "drawn earlier", At: recording.NoteAt{Node: node.ID}}
	view, err := New(b.Recs, notes.FakeModel{}).ContextForNote(note, 1, 5000)
	if err != nil || view.PageLoad != (PageLoadFacts{Target: "tab1", Loader: "draw", Root: "9223372036854775807"}) {
		t.Fatalf("draw context used save boundary: %+v, %v", view, err)
	}
}
