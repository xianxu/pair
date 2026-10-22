package recording

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func sketchSourceHub(t *testing.T, n int) (*Hub, *Builder, *SketchBegin, NodeID) {
	t.Helper()
	b := NewBuilder("source")
	var children []Node
	var textID NodeID
	for i := 0; i < n; i++ {
		text := b.Txt(fmt.Sprintf("source-%d", i))
		textID = text.ID
		children = append(children, b.El("p", []string{"data-i", fmt.Sprint(i)}, text))
	}
	b.Page(SegLive, "https://source.test/", children...)
	for i := range children {
		b.Add(&Sheet{ID: b.ID(), Text: fmt.Sprintf("p{--index:%d}", i)})
	}
	h := NewHub(0)
	for _, r := range b.Recs {
		if err := h.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	begin := sketchBegin()
	begin.VP = Viewport{W: 1280, H: 800, DPR: 1}
	begin.SetTime(1)
	return h, b, begin, textID
}

func TestSketchSourcePrefixEndsAtAcknowledgedBegin(t *testing.T) {
	h, b, begin, textID := sketchSourceHub(t, 1)
	source, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit)
	if err != nil {
		t.Fatal(err)
	}
	if source.State().T != begin.Time() || source.State().Node(textID).V != "source-0" {
		t.Fatal("source did not end at the prospective begin")
	}
	if source.Length() != source.Stats().PrefixBytes || source.Stats().Nodes != source.State().NodeCount() {
		t.Fatal("source accounting differs from owned content")
	}
	last, err := Decode(bytes.TrimSpace(source.prefix[bytes.LastIndex(source.prefix[:len(source.prefix)-1], []byte{'\n'})+1:]))
	if err != nil || last.Kind() != "sketch-begin" || last.Time() != 1 {
		t.Fatalf("terminal source record: %v, %v", last, err)
	}
	// Capturing a candidate is a preflight, not a write or caller mutation.
	if h.Retained() != len(b.Recs) || begin.Time() != 1 || begin.K != "" {
		t.Fatal("source preparation mutated the writer input or hub")
	}
	if err := h.Put(begin); err != nil {
		t.Fatal(err)
	}
	if err := h.Put(&Text{Header: Header{T: 2}, ID: textID, V: "later"}); err != nil {
		t.Fatal(err)
	}
	begin.Author.Name = "mutated caller"
	if source.State().Node(textID).V != "source-0" || source.Begin().Author.Name != "" {
		t.Fatal("source aliases the later stream or original begin")
	}
}

func TestSketchSourceCaptureSurvivesNavigationCompaction(t *testing.T) {
	h, b, begin, textID := sketchSourceHub(t, 1)
	source, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit)
	if err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(source.prefix)
	h.Put(begin)
	start := len(b.Recs)
	b.At(2).Page(SegNavigate, "https://source.test/next", b.Txt("replacement"))
	for _, r := range b.Recs[start:] {
		h.Put(r)
	}
	if !bytes.Equal(before, source.prefix) || source.State().Node(textID).V != "source-0" || source.State().DocSeg != 1 {
		t.Fatal("navigation changed the retained scene")
	}
}

func TestSketchSourceIndependentBoundsAndAtomicRefusal(t *testing.T) {
	h, _, begin, _ := sketchSourceHub(t, 8)
	source, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit)
	if err != nil {
		t.Fatal(err)
	}
	stats := source.Stats()
	retained := h.Retained()
	for _, tc := range []struct {
		name          string
		prefix, state int
		want          error
	}{
		{"exact", stats.PrefixBytes, stats.StateBytes, nil},
		{"prefix", stats.PrefixBytes - 1, SketchSourceStateLimit, ErrSketchSourcePrefixLimit},
		{"state", SketchSourcePrefixLimit, stats.StateBytes - 1, ErrSketchSourceStateLimit},
		{"zero", 0, stats.StateBytes, ErrSketchSourcePrefixLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.CandidateSketchSource(begin, tc.prefix, tc.state)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if h.Retained() != retained || begin.Time() != 1 {
				t.Fatal("size preflight advanced the stream")
			}
		})
	}
}

func TestSketchSourceRequiresFencedLiveBegin(t *testing.T) {
	h, _, begin, _ := sketchSourceHub(t, 1)
	begin.SetTime(0)
	if _, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit); err == nil {
		t.Fatal("accepted unfenced begin")
	}
	begin.SetTime(2)
	h.Put(&Stop{Header: Header{T: 1}, Why: StopPause})
	if _, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit); err == nil {
		t.Fatal("accepted begin on paused source")
	}
	h.Close()
	if _, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed source: %v", err)
	}
}

func TestSketchSourceReadOwnsOnlyRequestedChunk(t *testing.T) {
	h, _, begin, _ := sketchSourceHub(t, 1)
	source, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit)
	if err != nil {
		t.Fatal(err)
	}
	chunk, err := source.Read(0, 7)
	if err != nil || !bytes.Equal(chunk, source.prefix[:7]) {
		t.Fatalf("read %q: %v", chunk, err)
	}
	chunk[0] = '!'
	if source.prefix[0] == '!' {
		t.Fatal("chunk aliases source bytes")
	}
	for _, args := range [][2]int{{-1, 1}, {0, 0}, {0, SketchSourceChunkSize + 1}, {source.Length() + 1, 1}} {
		if _, err := source.Read(args[0], args[1]); err == nil {
			t.Fatalf("accepted read %v", args)
		}
	}
	end, err := source.Read(source.Length(), 1)
	if err != nil || len(end) != 0 {
		t.Fatal("EOF is not an empty owned read")
	}
}

func TestSketchSourceN2NStateAccountingVisits(t *testing.T) {
	var counts []SketchSourceStats
	for _, n := range []int{8, 16} {
		h, _, begin, _ := sketchSourceHub(t, n)
		source, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit)
		if err != nil {
			t.Fatal(err)
		}
		stats := source.Stats()
		if stats.Nodes != source.State().NodeCount() || stats.Sheets != n {
			t.Fatalf("unexpected work %+v", stats)
		}
		counts = append(counts, stats)
	}
	if counts[1].Nodes != 2*counts[0].Nodes-5 || counts[1].Sheets != 2*counts[0].Sheets || counts[1].StateBytes > 2*counts[0].StateBytes {
		t.Fatalf("nonlinear preparation work: %+v", counts)
	}
}

func TestSketchSourceUsesExistingHistoryBootstrap(t *testing.T) {
	h, _, begin, _ := sketchSourceHub(t, 1)
	h.Put(begin)
	label := &SketchLabel{Header: Header{T: 2}, SketchOperation: sketchOp(1), Sketch: begin.ID, Label: strings.Repeat("old", 8)}
	h.Put(label)
	next := *begin
	next.ID, next.Operation = "next", "next-operation"
	next.SetTime(3)
	source, err := h.CandidateSketchSource(&next, SketchSourcePrefixLimit, SketchSourceStateLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(source.prefix, []byte(`"k":"sketch-snapshot"`)) {
		t.Fatal("source discarded authoritative bootstrap")
	}
	meta, ok := source.State().Sketches.Metadata(begin.ID)
	if !ok || meta.Label != label.Label || !reflect.DeepEqual(source.Begin(), next) {
		t.Fatal("old accepted history or prospective identity lost")
	}
}

func TestSketchSourceAssetBindingsAreFrozen(t *testing.T) {
	h, _, begin, _ := sketchSourceHub(t, 1)
	old := strings.Repeat("a", 64)
	if err := h.Put(&Asset{URL: "https://source.test/image.png", Hash: old, MIME: "image/png"}); err != nil {
		t.Fatal(err)
	}
	source, err := h.CandidateSketchSource(begin, SketchSourcePrefixLimit, SketchSourceStateLimit)
	if err != nil {
		t.Fatal(err)
	}
	if !source.AllowsAsset(old) || source.AllowsAsset(strings.Repeat("b", 64)) {
		t.Fatal("wrong source allowlist")
	}
	h.Put(&Asset{URL: "https://source.test/image.png", Hash: strings.Repeat("b", 64), MIME: "image/png"})
	if !source.AllowsAsset(old) || source.AllowsAsset(strings.Repeat("b", 64)) {
		t.Fatal("live asset binding changed held source")
	}
}
