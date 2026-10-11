package sessionsync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type rig struct {
	t    *testing.T
	sync *Sync
	p    Paths
	now  time.Time
}

const statusDoc = "---\ntype: session-sync\nowner: ops:0\n---\n\n_Updated 2026-10-10 23:43._\n\n## Needs you\n\nold\n\n## Trains\n\ntrains\n"

// newRig is a session-sync file the agent last wrote (file == shadow),
// a recording Couch store, and a fixed clock.
func newRig(t *testing.T) *rig {
	dir := t.TempDir()
	r := &rig{t: t, p: PathsOf(filepath.Join(dir, "tl.md")), now: t0.Add(time.Hour)}
	r.sync = &Sync{Ledger: &Ledger{Dir: filepath.Join(dir, "store")}, Now: func() time.Time { return r.now },
		Diff: UnifiedDiff, Canonical: Canonical}
	r.put(r.p.File, statusDoc)
	r.put(r.p.Shadow, statusDoc)
	os.Chtimes(r.p.Shadow, t0, t0)
	if err := r.sync.Ledger.MarkRecording(Recording{PID: 1, Started: t0}); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *rig) put(p, s string) {
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) read(p string) string {
	raw, _ := os.ReadFile(p)
	return string(raw)
}

func (r *rig) deliver(id, head string) {
	if err := r.sync.Ledger.Append("ops:0", Entry{ID: id, From: "pair:5", At: t0.Add(time.Minute), Head: head}); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) owedIDs() []string {
	st, err := r.sync.Status(r.p.File)
	if err != nil {
		r.t.Fatal(err)
	}
	var ids []string
	for _, e := range st.Owed {
		ids = append(ids, e.ID+":"+string(e.Kind))
	}
	return ids
}

func wantRefusal(t *testing.T, err error, exit int, code string) {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) || r.Exit != exit || r.Code != code {
		t.Fatalf("want refusal exit %d %s, got %v", exit, code, err)
	}
}

func TestWriteUnderAnOperatorLockRefusesAndChangesNothing(t *testing.T) {
	r := newRig(t)
	r.put(r.p.Lock, "holder: operator\nat: 2026-10-10T23:50\n")
	r.deliver("d1", "report")
	before := []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State), r.read(r.p.Lock)}
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "new", AcceptEdits: true, Fold: []string{"d1"}})
	wantRefusal(t, err, ExitOperatorLock, "operator-lock")
	_, err = r.sync.Release(r.p.File, nil)
	wantRefusal(t, err, ExitOperatorLock, "operator-lock")
	after := []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State), r.read(r.p.Lock)}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("file %d changed under the operator's lock", i)
		}
	}
}

func TestWriteOverUnreadEditsRefusesUntilAcceptEdits(t *testing.T) {
	r := newRig(t)
	edited := strings.Replace(statusDoc, "old", "old 🤖[why?]", 1)
	r.put(r.p.File, edited)
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "new trains"})
	wantRefusal(t, err, ExitUnreadEdits, "unread-edits")
	if r.read(r.p.File) != edited {
		t.Fatal("a refused write changed the file")
	}
	_, err = r.sync.Release(r.p.File, nil)
	wantRefusal(t, err, ExitUnreadEdits, "unread-edits")

	_, diff, err := r.sync.DiffText(r.p.File)
	if err != nil || !strings.Contains(diff, "+old 🤖[why?]") || !strings.HasPrefix(diff, "--- shadow") {
		t.Fatalf("diff=%q err=%v", diff, err)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "answered", AcceptEdits: true}); err != nil {
		t.Fatal(err)
	}
	got := r.read(r.p.File)
	if got != r.read(r.p.Shadow) || !strings.Contains(got, "## Needs you\n\nanswered\n\n## Trains") ||
		!strings.Contains(got, "Updated 2026-10-10 13:00") {
		t.Fatalf("got %q", got)
	}
}

func TestADeliveryStaysOwedUntilAWriteFoldsIt(t *testing.T) {
	r := newRig(t)
	r.deliver("d1", "pair#437 done")
	if ids := r.owedIDs(); len(ids) != 1 || ids[0] != "d1:delivery" {
		t.Fatalf("owed %v", ids)
	}
	// A write that doesn't fold leaves it owed: no implicit clear.
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x"}); err != nil {
		t.Fatal(err)
	}
	if ids := r.owedIDs(); len(ids) != 1 {
		t.Fatalf("an unfolded delivery must stay owed: %v", ids)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "y", Fold: []string{"d1"}}); err != nil {
		t.Fatal(err)
	}
	if ids := r.owedIDs(); len(ids) != 0 {
		t.Fatalf("owed %v", ids)
	}
	st, _ := r.sync.Status(r.p.File)
	if st.LastWrite == nil || !st.LastWrite.Equal(r.now) || st.Owner != "ops:0" || st.Holder != HolderFree {
		t.Fatalf("status %+v", st)
	}
}

func TestFoldingAnUnknownIDRefusesAndWritesNothing(t *testing.T) {
	r := newRig(t)
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x", Fold: []string{"nope"}})
	wantRefusal(t, err, ExitRefused, "refused")
	if r.read(r.p.File) != statusDoc {
		t.Fatal("changed")
	}
}

func TestReleaseFoldsSubmitsNotDeliveriesAndDeletesAnAgentLock(t *testing.T) {
	r := newRig(t)
	r.deliver("s1", SubmitPrefix+r.p.File)
	r.deliver("d1", "report")
	r.put(r.p.Lock, "holder: agent\n")
	res, err := r.sync.Release(r.p.File, nil)
	if err != nil || !res.Released {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if _, err := os.Stat(r.p.Lock); !os.IsNotExist(err) {
		t.Fatal("the agent lock must be gone")
	}
	if ids := r.owedIDs(); len(ids) != 1 || ids[0] != "d1:delivery" {
		t.Fatalf("owed %v", ids)
	}
	if res, err := r.sync.Release(r.p.File, []string{"d1"}); err != nil || res.Released {
		t.Fatalf("a free file has nothing to release: %+v %v", res, err)
	}
	if ids := r.owedIDs(); len(ids) != 0 {
		t.Fatalf("owed %v", ids)
	}
}

func TestWriteAllKeepsGuardsAndFrontmatter(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Body: "# no frontmatter\n"}); err == nil {
		t.Fatal("dropping the frontmatter must refuse")
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Body: "  \n"}); err == nil {
		t.Fatal("an empty body must refuse")
	}
	next := strings.Replace(statusDoc, "trains", "all new", 1)
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Body: next}); err != nil {
		t.Fatal(err)
	}
	if got := r.read(r.p.File); !strings.Contains(got, "all new") || !strings.Contains(got, "Updated 2026-10-10 13:00") {
		t.Fatalf("got %q", got)
	}
}

func TestStatusRefusesWithoutARecordingCouchOrAValidOwner(t *testing.T) {
	r := newRig(t)
	os.Remove(r.sync.Ledger.recordingPath())
	_, err := r.sync.Status(r.p.File)
	wantRefusal(t, err, ExitRefused, "not-recording")

	r = newRig(t)
	r.put(r.p.File, strings.Replace(statusDoc, "owner: ops:0", "owner: ops", 1))
	if _, err := r.sync.Status(r.p.File); err == nil || !strings.Contains(err.Error(), "not a Couch slot address") {
		t.Fatalf("got %v", err)
	}
	r.put(r.p.File, "# plain markdown\n")
	if _, err := r.sync.Status(r.p.File); err == nil || !strings.Contains(err.Error(), "not a session-sync file") {
		t.Fatalf("got %v", err)
	}
}

func TestStatusBeforeAnyWrite(t *testing.T) {
	r := newRig(t)
	st, err := r.sync.Status(r.p.File)
	if err != nil || st.LastWrite != nil || st.Owed == nil || len(st.Owed) != 0 {
		t.Fatalf("st=%+v err=%v", st, err)
	}
	// The horizon came from the shadow (t0): a delivery before it is not owed.
	r.sync.Ledger.Append("ops:0", Entry{ID: "early", At: t0.Add(-time.Second)})
	if ids := r.owedIDs(); len(ids) != 0 {
		t.Fatalf("owed %v", ids)
	}
}
