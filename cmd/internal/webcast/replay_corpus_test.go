package webcast

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var updateCatchUp = flag.Bool("update-catchup", false, "rewrite tests/webcast/catchup_corpus.json's kept (after a deliberate change to recording.CatchUp; then port it to web/replay.js)")

// The catch-up rule has two implementations: recording.CatchUp, and its
// port in web/replay.js that the replay seeks with (#471). They share one
// corpus that reaches every branch of the rule; this test holds the
// corpus's answers to Go's rule, and tests/webcast/replay.test.mjs holds
// the port to the same answers. A change to Go's rule fails here first.
func TestCatchUpCorpus(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "../../../tests/webcast/catchup_corpus.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		About   string            `json:"about"`
		Records []json.RawMessage `json:"records"`
		Kept    [][]int           `json:"kept"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	var recs []recording.Record
	for i, line := range corpus.Records {
		var compact bytes.Buffer
		json.Compact(&compact, line)
		r, err := recording.Decode(compact.Bytes())
		if err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
		recs = append(recs, r)
	}
	// The corpus must reach every branch: each kind the rule treats
	// specially, before and in the latest segment.
	kinds := map[string]bool{}
	for _, r := range recs {
		kinds[r.Kind()] = true
	}
	for _, k := range []string{"head", "seg", "asset", "note", "note-state", "stroke", "ctl", "hover", "ptr", "stop"} {
		if !kinds[k] {
			t.Errorf("the corpus has no %s record: that branch of the rule is untested", k)
		}
	}
	var kept [][]int
	for i := 1; i <= len(recs); i++ {
		var idx []int
		for _, r := range recording.CatchUp(recs[:i]) {
			idx = append(idx, slices.Index(recs[:i], r))
		}
		kept = append(kept, idx)
	}
	if *updateCatchUp {
		corpus.Kept = kept
		out, _ := json.MarshalIndent(corpus, "", " ")
		if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	for i := range kept {
		if i >= len(corpus.Kept) || !slices.Equal(corpus.Kept[i], kept[i]) {
			t.Fatalf("cut %d: recording.CatchUp keeps %v, the corpus says %v: if the rule changed on purpose, rerun with -update-catchup and port it to web/replay.js", i+1, kept[i], corpus.Kept[min(i, len(corpus.Kept)-1)])
		}
	}
	// And every cut's catch-up reduces to what the prefix does.
	for i := 1; i <= len(recs); i++ {
		whole, err := recording.Reduce(recs[:i], 1<<62)
		if err != nil {
			t.Fatalf("cut %d: %v", i, err)
		}
		caught, err := recording.Reduce(recording.CatchUp(recs[:i]), 1<<62)
		if err != nil {
			t.Fatalf("cut %d catch-up: %v", i, err)
		}
		if !bytes.Equal(whole.Canonical(), caught.Canonical()) || whole.DocSeg != caught.DocSeg {
			t.Fatalf("cut %d: the catch-up reduces differently", i)
		}
	}
}
