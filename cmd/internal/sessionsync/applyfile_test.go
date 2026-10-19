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

// A round artifact that can't be written after the document moved is a
// warning, not a failure: the reply keeps the new Hash and the Body to commit,
// and no stale artifact remains (close review BR-10).
func TestApplyFileRoundWriteFailureWarns(t *testing.T) {
	r := newFileRig(t, "one two\n")
	if _, err := r.apply(Hash(r.read(r.doc)), EditRecord{Old: "one", Occurrence: 1, New: "1"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(r.path), 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(r.path), 0o755)
	res, err := r.apply(Hash(r.read(r.doc)), EditRecord{Old: "two", Occurrence: 1, New: "2"})
	if err == nil {
		// The previous artifact could not even be removed: that refuses first.
		t.Skip("platform let the read-only dir be written")
	}
	wantRefusal(t, err, ExitRefused, "refused")
	if r.read(r.doc) != "1 two\n" {
		t.Fatal("the document moved although its old round could not be cleared")
	}
	os.Chmod(filepath.Dir(r.path), 0o755)
	os.Remove(r.path)
	os.Chmod(filepath.Dir(r.path), 0o500)
	res, err = r.apply(Hash(r.read(r.doc)), EditRecord{Old: "two", Occurrence: 1, New: "2"})
	if err != nil || res.Warning == "" || res.Hash != Hash(r.read(r.doc)) || res.Body == "" || r.read(r.doc) != "1 2\n" {
		t.Fatalf("res=%+v err=%v doc=%q", res, err, r.read(r.doc))
	}
	if _, err := os.Stat(r.path); !os.IsNotExist(err) {
		t.Fatal("an artifact exists after a failed round write")
	}
}

// A reply with no edits is a round too (pair#690, ariadne#453): an empty
// records list, or records that change nothing, leaves the document's bytes
// alone and writes an empty round, keyed apart from the last one by its time,
// with no Body, so the agent commits nothing.
func TestApplyFileWithNoChangeWritesAnEmptyRound(t *testing.T) {
	for name, edits := range map[string][]EditRecord{
		"no records":   nil,
		"no-op record": {{Old: "beta", Occurrence: 1, New: "beta"}},
	} {
		t.Run(name, func(t *testing.T) {
			r := newFileRig(t, "alpha beta\n")
			if _, err := r.apply(Hash(r.read(r.doc)), EditRecord{Old: "alpha", Occurrence: 1, New: "A"}); err != nil {
				t.Fatal(err)
			}
			before, err := os.Stat(r.doc)
			if err != nil {
				t.Fatal(err)
			}
			r.now = r.now.Add(time.Minute)
			h := Hash(r.read(r.doc))
			res, err := r.apply(h, edits...)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := os.Stat(r.doc)
			if r.read(r.doc) != "A beta\n" || !after.ModTime().Equal(before.ModTime()) {
				t.Fatalf("an empty round wrote the document: %q", r.read(r.doc))
			}
			if res.Hash != h || res.Body != "" || res.Round == nil || len(res.Round.Spans) != 0 ||
				res.Round.BaseHash != h || res.Round.ResultHash != h || !res.Round.At.Equal(r.now) {
				t.Fatalf("res %+v round %+v", res, res.Round)
			}
			artifact, err := ReadRoundArtifact(r.path)
			if err != nil || artifact == nil || artifact.File != r.doc || artifact.Body != "" ||
				len(artifact.Round.Spans) != 0 || !artifact.Round.At.Equal(r.now) || artifact.Round.ResultHash != h {
				t.Fatalf("artifact %+v err %v", artifact, err)
			}
		})
	}
}

// An empty apply is refused like any other: stale, then under the operator
// lock, writing no round.
func TestApplyFileRefusesAnEmptyApplyAlike(t *testing.T) {
	r := newFileRig(t, "alpha beta\n")
	_, err := r.apply("0000000000000000")
	wantRefusal(t, err, ExitStale, "stale")
	if err := os.WriteFile(r.doc+".lock", []byte("holder: operator\nat: 2026-10-18T08:55\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = r.apply(Hash(r.read(r.doc)))
	wantRefusal(t, err, ExitOperatorLock, "operator-lock")
	if _, err := os.Stat(r.path); !os.IsNotExist(err) {
		t.Fatal("a refused empty apply wrote a round")
	}
}

// An empty round that can't be recorded warns, with no Body, and says to
// commit nothing; the document is untouched.
func TestApplyFileEmptyRoundWriteFailureWarns(t *testing.T) {
	r := newFileRig(t, "one two\n")
	if _, err := r.apply(Hash(r.read(r.doc))); err != nil { // creates the flock
		t.Fatal(err)
	}
	os.Remove(r.path)
	if err := os.Chmod(filepath.Dir(r.path), 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(r.path), 0o755)
	res, err := r.apply(Hash(r.read(r.doc)))
	if err != nil || res.Body != "" || !strings.Contains(res.Warning, "commit nothing") || r.read(r.doc) != "one two\n" {
		t.Fatalf("res=%+v err=%v doc=%q", res, err, r.read(r.doc))
	}
}
