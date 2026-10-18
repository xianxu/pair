package sessionsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fileRig struct {
	t                *testing.T
	doc, flock, path string
	now              time.Time
}

func newFileRig(t *testing.T, content string) *fileRig {
	dir, data := t.TempDir(), t.TempDir()
	r := &fileRig{t: t, doc: filepath.Join(dir, "doc.md"), flock: filepath.Join(data, "apply.flock"),
		path: filepath.Join(data, "review-round-x.json"), now: time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)}
	if err := os.WriteFile(r.doc, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *fileRig) apply(hash string, edits ...EditRecord) (FileApplyResult, error) {
	return ApplyFile(FileApplyRequest{File: r.doc, IfHash: hash, FlockPath: r.flock, RoundPath: r.path, Edits: edits, Now: r.now})
}

func (r *fileRig) read(p string) string {
	raw, _ := os.ReadFile(p)
	return string(raw)
}

func TestApplyFileWritesTheDocumentAndItsRoundArtifact(t *testing.T) {
	r := newFileRig(t, "# Doc\n\nThe quick fox. The quick end.\n")
	res, err := r.apply(Hash(r.read(r.doc)),
		EditRecord{Old: "quick", Occurrence: 2, New: "slow", Explain: "pace"})
	if err != nil {
		t.Fatal(err)
	}
	if r.read(r.doc) != "# Doc\n\nThe quick fox. The slow end.\n" || res.Hash != Hash(r.read(r.doc)) {
		t.Fatalf("doc %q hash %s", r.read(r.doc), res.Hash)
	}
	var artifact RoundArtifact
	if err := json.Unmarshal([]byte(r.read(r.path)), &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.File != r.doc || artifact.Body != res.Body || len(artifact.Round.Spans) != 1 {
		t.Fatalf("artifact %+v", artifact)
	}
	// The body is the pane's commit format, and new_occurrence places the
	// span exactly where the round says it is (resume reconstructs from it).
	block := strings.TrimSuffix(strings.SplitN(res.Body, "```review-records\n", 2)[1], "\n```")
	var records []roundRecord
	if err := json.Unmarshal([]byte(block), &records); err != nil || len(records) != 1 {
		t.Fatalf("body %q: %v", res.Body, err)
	}
	result := r.read(r.doc)
	span := artifact.Round.Spans[0]
	if got := nthOffset(result, records[0].New, records[0].NewOccurrence); got != lineStarts(result)[span.Start[0]]+span.Start[1] || records[0].Explain != "pace" {
		t.Fatalf("record %+v does not anchor span %+v", records[0], span)
	}
}

// Every refusal writes nothing: not the document, not the round artifact.
func TestApplyFileRefusalsWriteNothing(t *testing.T) {
	r := newFileRig(t, "alpha beta\n")
	if _, err := r.apply(Hash(r.read(r.doc)), EditRecord{Old: "alpha", Occurrence: 1, New: "A"}); err != nil {
		t.Fatal(err)
	}
	snap := func() string { return r.read(r.doc) + "\x00" + r.read(r.path) }
	before := snap()
	_, err := r.apply(Hash(r.read(r.doc)), EditRecord{Old: "beta", Occurrence: 1, New: "B"}, EditRecord{Old: "absent", Occurrence: 1, New: "x"})
	wantRefusal(t, err, ExitRefused, "edits-invalid")
	_, err = r.apply("0000000000000000", EditRecord{Old: "beta", Occurrence: 1, New: "B"})
	wantRefusal(t, err, ExitStale, "stale")
	if err := os.WriteFile(r.doc+".lock", []byte("holder: operator\nat: 2026-10-18T08:55\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = r.apply(Hash(r.read(r.doc)), EditRecord{Old: "beta", Occurrence: 1, New: "B"})
	wantRefusal(t, err, ExitOperatorLock, "operator-lock")
	if snap() != before {
		t.Fatal("a refused apply wrote")
	}
}

// A failure after the old round artifact is removed leaves none, never the
// previous round over the new document.
func TestApplyFileNeverLeavesAStaleRoundArtifact(t *testing.T) {
	r := newFileRig(t, "one two\n")
	if _, err := r.apply(Hash(r.read(r.doc)), EditRecord{Old: "one", Occurrence: 1, New: "1"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(r.doc), 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(r.doc), 0o755)
	if _, err := r.apply(Hash(r.read(r.doc)), EditRecord{Old: "two", Occurrence: 1, New: "2"}); err == nil {
		t.Fatal("a document that can't be replaced was reported written")
	}
	if _, err := os.Stat(r.path); !os.IsNotExist(err) {
		t.Fatal("the previous round artifact survived a failed apply")
	}
}
