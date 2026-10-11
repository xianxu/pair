package sessionsync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

// hash is what `read` hands out for a section ("" = the whole file).
func (r *rig) hash(section string) string {
	res, err := r.sync.Read(r.p.File, section)
	if err != nil {
		r.t.Fatal(err)
	}
	return res.Hash
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
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "new", IfHash: r.hash("Needs you"), Fold: []string{"d1"}})
	wantRefusal(t, err, ExitOperatorLock, "operator-lock")
	// Reading under the operator's lock works and reports the holder.
	if res, err := r.sync.Read(r.p.File, "Needs you"); err != nil || res.Holder != HolderOperator {
		t.Fatalf("read under the lock: %+v %v", res, err)
	}
	after := []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State), r.read(r.p.Lock)}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("file %d changed under the operator's lock", i)
		}
	}
}

func TestASectionWriteSucceedsOverAnEditElsewhereAndRefusesStale(t *testing.T) {
	r := newRig(t)
	trains := r.hash("Trains")
	needs := r.hash("Needs you")
	// The operator edits another section; the agent hasn't read it.
	edited := strings.Replace(statusDoc, "old", "old 🤖[why?]", 1)
	r.put(r.p.File, edited)
	if r.hash("Trains") != trains {
		t.Fatal("an edit elsewhere must not change this section's hash")
	}
	res, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "new trains", IfHash: trains})
	if err != nil {
		t.Fatal(err)
	}
	got := r.read(r.p.File)
	if !strings.Contains(got, "old 🤖[why?]") || !strings.Contains(got, "## Trains\n\nnew trains\n") ||
		!strings.Contains(got, "Updated 2026-10-10 13:00") || got != r.read(r.p.Shadow) || res.Hash != r.hash("Trains") {
		t.Fatalf("res %+v got %q", res, got)
	}
	// The edited section's old hash is stale: refused with the current one.
	_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "answered", IfHash: needs})
	wantRefusal(t, err, ExitStale, "stale")
	var ref *Refusal
	if errors.As(err, &ref); ref.Hash != r.hash("Needs you") || r.read(r.p.File) != got {
		t.Fatalf("stale refusal %+v must name the current hash and change nothing", ref)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "answered", IfHash: ref.Hash}); err != nil {
		t.Fatalf("a retry with the current hash wins: %v", err)
	}
}

func TestWriteWithoutIfHashRefuses(t *testing.T) {
	r := newRig(t)
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x"})
	wantRefusal(t, err, ExitRefused, "refused")
	if err == nil || !strings.Contains(err.Error(), "--if-hash") || r.read(r.p.File) != statusDoc {
		t.Fatalf("got %v", err)
	}
}

// The hash check and the write are one step under the flock: a write that
// waited for another couch command sees what that command wrote.
func TestAWriteChecksItsHashUnderTheFlock(t *testing.T) {
	r := newRig(t)
	h := r.hash("Trains")
	fl, err := os.OpenFile(r.p.Flock, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(fl.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "late", IfHash: h})
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("the write ran while another command held the flock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	r.put(r.p.File, strings.Replace(statusDoc, "trains", "first", 1))
	fl.Close()
	wantRefusal(t, <-done, ExitStale, "stale")
}

// A write that meets another writer's torn file (truncated mid-rewrite)
// waits for its flock and judges what it wrote, rather than refusing the
// file as not session-sync (pair#444).
func TestAWriteJudgesATornFileUnderTheFlock(t *testing.T) {
	r := newRig(t)
	h := r.hash("Trains")
	fl, err := os.OpenFile(r.p.Flock, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(fl.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	r.put(r.p.File, "")
	done := make(chan error, 1)
	go func() {
		_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "late", IfHash: h})
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("the write judged the torn file without waiting for the flock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	r.put(r.p.File, strings.Replace(statusDoc, "trains", "first", 1))
	fl.Close()
	wantRefusal(t, <-done, ExitStale, "stale")
}

// A first writer can create the flock file and tear the file between
// another command's flock-file probe and its unlocked check; the failed
// check then defers to the lock instead of refusing (pair#444).
func TestAFailedCheckDefersToAFlockCreatedMeanwhile(t *testing.T) {
	r := newRig(t)
	fl, err := openFlock(r.p.File, func() error {
		r.put(r.p.Flock, "")
		return refuse("torn")
	})
	if err != nil {
		t.Fatalf("a check that failed after another writer created the flock file must defer to the lock: %v", err)
	}
	fl.Close()
}

func TestADeliveryStaysOwedUntilAWriteFoldsIt(t *testing.T) {
	r := newRig(t)
	r.deliver("d1", "pair#437 done")
	if ids := r.owedIDs(); len(ids) != 1 || ids[0] != "d1:delivery" {
		t.Fatalf("owed %v", ids)
	}
	// A write that doesn't fold leaves it owed: no implicit clear.
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x", IfHash: r.hash("Trains")}); err != nil {
		t.Fatal(err)
	}
	if ids := r.owedIDs(); len(ids) != 1 {
		t.Fatalf("an unfolded delivery must stay owed: %v", ids)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "y", IfHash: r.hash("Trains"), Fold: []string{"d1"}}); err != nil {
		t.Fatal(err)
	}
	if ids := r.owedIDs(); len(ids) != 0 {
		t.Fatalf("owed %v", ids)
	}
	st, _ := r.sync.Status(r.p.File)
	if st.LastWrite == nil || !st.LastWrite.Equal(r.now) || st.Owner != "ops:0" || st.Holder != HolderNone {
		t.Fatalf("status %+v", st)
	}
}

func TestFoldingAnUnknownIDRefusesAndWritesNothing(t *testing.T) {
	r := newRig(t)
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x", IfHash: r.hash("Trains"), Fold: []string{"nope"}})
	wantRefusal(t, err, ExitRefused, "refused")
	if r.read(r.p.File) != statusDoc {
		t.Fatal("changed")
	}
}

// A TL turn that never runs release leaves the operator free to edit:
// a submit is a ledger row, the agent folds it with a write, and no lock
// appears. A legacy agent lock left by an old parley counts as none.
func TestATurnWithoutReleaseLeavesNoLock(t *testing.T) {
	r := newRig(t)
	r.deliver("s1", SubmitPrefix+r.p.File)
	if ids := r.owedIDs(); len(ids) != 1 || ids[0] != "s1:submit" {
		t.Fatalf("owed %v", ids)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "answered",
		IfHash: r.hash("Needs you"), Fold: []string{"s1"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.p.Lock); !os.IsNotExist(err) {
		t.Fatal("no couch path may create a lock")
	}
	if ids := r.owedIDs(); len(ids) != 0 {
		t.Fatalf("owed %v", ids)
	}
	r.put(r.p.Lock, "holder: agent\n")
	if st, _ := r.sync.Status(r.p.File); st.Holder != HolderNone {
		t.Fatalf("a legacy agent lock is no lock: %q", st.Holder)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x", IfHash: r.hash("Trains")}); err != nil {
		t.Fatal(err)
	}
	if r.read(r.p.Lock) != "holder: agent\n" {
		t.Fatal("couch never touches parley's lock file")
	}
}

func TestWriteAllKeepsGuardsAndFrontmatter(t *testing.T) {
	r := newRig(t)
	h := r.hash("")
	if h != Hash(statusDoc) {
		t.Fatal("--all hashes the whole file")
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Body: "# no frontmatter\n", IfHash: h}); err == nil {
		t.Fatal("dropping the frontmatter must refuse")
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Body: "  \n", IfHash: h}); err == nil {
		t.Fatal("an empty body must refuse")
	}
	next := strings.Replace(statusDoc, "trains", "all new", 1)
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Body: next, IfHash: Hash("other")})
	wantRefusal(t, err, ExitStale, "stale")
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Body: next, IfHash: h}); err != nil {
		t.Fatal(err)
	}
	if got := r.read(r.p.File); !strings.Contains(got, "all new") || !strings.Contains(got, "Updated 2026-10-10 13:00") {
		t.Fatalf("got %q", got)
	}
}

func TestAWrongFileLeavesNoFlockBehind(t *testing.T) {
	r := newRig(t)
	r.put(r.p.File, "# plain markdown\n")
	if _, err := r.sync.Read(r.p.File, ""); err == nil {
		t.Fatal("a non-session-sync file must refuse")
	}
	if _, err := os.Stat(r.p.Flock); !os.IsNotExist(err) {
		t.Fatal("a refused command must not create the flock file")
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

func TestStatusReportsCouchOwnerOrTheLegacyOwner(t *testing.T) {
	for _, c := range []struct{ name, fm, want string }{
		{"couch-owner", "couch-owner: ops:2", "ops:2"},
		{"legacy owner", "owner: ops:0", "ops:0"},
		{"both: couch-owner wins", "owner: ops:0\ncouch-owner: ops:2", "ops:2"},
		{"process-instruction line", "process-instruction: [copy-edit] tighten the prose\ncouch-owner: ops:2", "ops:2"},
	} {
		r := newRig(t)
		r.put(r.p.File, strings.Replace(statusDoc, "owner: ops:0", c.fm, 1))
		st, err := r.sync.Status(r.p.File)
		if err != nil || st.Owner != c.want {
			t.Errorf("%s: owner %q err %v, want %q", c.name, st.Owner, err, c.want)
		}
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

func TestALedgerGapRefusesUntilAcceptedPerFile(t *testing.T) {
	r := newRig(t)
	r.sync.Ledger.Now = func() time.Time { return r.now }
	if err := r.sync.Ledger.MarkGap("ops:0", errors.New("disk full")); err != nil {
		t.Fatal(err)
	}
	_, err := r.sync.Status(r.p.File)
	wantRefusal(t, err, ExitRefused, "delivery-gap")
	_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x", IfHash: r.hash("Trains")})
	wantRefusal(t, err, ExitRefused, "delivery-gap")
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x", IfHash: r.hash("Trains"), Accepts: Accepts{Gap: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Status(r.p.File); err != nil {
		t.Fatalf("an accepted gap stops refusing: %v", err)
	}
	// A later gap refuses again.
	r.now = r.now.Add(time.Minute)
	r.sync.Ledger.MarkGap("ops:0", errors.New("again"))
	_, err = r.sync.Status(r.p.File)
	wantRefusal(t, err, ExitRefused, "delivery-gap")
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "y", IfHash: r.hash("Trains"), Accepts: Accepts{Gap: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Status(r.p.File); err != nil {
		t.Fatal(err)
	}
	// Another owner's file never sees this slot's gap.
	r.put(r.p.File, strings.Replace(r.read(r.p.File), "owner: ops:0", "owner: ops:1", 1))
	r.put(r.p.Shadow, r.read(r.p.File))
	os.Remove(r.p.State)
	if _, err := r.sync.Status(r.p.File); err != nil {
		t.Fatalf("ops:1 must not inherit ops:0's gap: %v", err)
	}
}

func TestAGapBeforeTheHorizonOrExpiredIsIgnored(t *testing.T) {
	r := newRig(t)
	at := t0.Add(-time.Hour) // before the shadow's mtime (t0)
	r.sync.Ledger.Now = func() time.Time { return at }
	r.sync.Ledger.MarkGap("ops:0", errors.New("old"))
	r.sync.Ledger.Now = func() time.Time { return r.now }
	if _, err := r.sync.Status(r.p.File); err != nil {
		t.Fatalf("a gap before the file's horizon hides nothing it owes: %v", err)
	}
	r.sync.Ledger.Now = func() time.Time { return t0.Add(keepFor + 2*time.Hour) }
	if g, _ := r.sync.Ledger.Gap("ops:0"); g != nil {
		t.Fatal("an expired gap is gone")
	}
	if err := r.sync.Ledger.Append("ops:0", Entry{ID: "y", At: t0}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.sync.Ledger.Dir, "ops%3A0.gap")); !os.IsNotExist(err) {
		t.Fatal("the next append removes an expired gap record")
	}
}

func TestAnUnreadableGapIsDatedByItsFileSoItCanBeAccepted(t *testing.T) {
	r := newRig(t)
	r.sync.Ledger.Now = func() time.Time { return r.now }
	gap := filepath.Join(r.sync.Ledger.Dir, "ops%3A0.gap")
	r.put(gap, "{torn")
	at := t0.Add(30 * time.Minute)
	os.Chtimes(gap, at, at)
	_, err := r.sync.Status(r.p.File)
	wantRefusal(t, err, ExitRefused, "delivery-gap")
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "x", IfHash: r.hash("Trains"), Accepts: Accepts{Gap: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Status(r.p.File); err != nil {
		t.Fatalf("an accepted unreadable gap stops refusing: %v", err)
	}
	r.sync.Ledger.Now = func() time.Time { return at.Add(keepFor + time.Hour) }
	if g, _ := r.sync.Ledger.Gap("ops:0"); g != nil {
		t.Fatal("an unreadable gap expires by its mtime")
	}
}
