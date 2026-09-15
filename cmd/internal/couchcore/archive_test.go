package couchcore

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func archivableThread(t *testing.T, store *ThreadStore, tag ThreadTag) ThreadRecord {
	t.Helper()
	record := actionableTestThread(tag, time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// Archiving is the operator's "delete": get this out of my switcher. It has to
// be complete -- gone from the manifest, so gone from every projection -- and
// reversible, because "start anew" should not mean "destroy the evidence".
func TestArchiveThreadRemovesItFromTheWorkingSetAndKeepsTheRecord(t *testing.T) {
	store, _ := newTestThreadStore(t)
	kept := archivableThread(t, store, "couch-0000000000000001")
	retired := archivableThread(t, store, "couch-0000000000000002")

	if err := store.ArchiveThread(retired.Address); err != nil {
		t.Fatalf("ArchiveThread: %v", err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 || snapshot.Records[0].Address != kept.Address {
		t.Fatalf("working set = %+v, want only the kept thread", snapshot.Records)
	}
	if _, err := store.GetThread(retired.Address); err == nil {
		t.Fatal("an archived thread is still readable in the working set")
	}

	archived, err := store.ArchivedThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 1 || archived[0].Address != retired.Address {
		t.Fatalf("archive = %+v, want the retired thread", archived)
	}
	// Same shape as it had: restoring is a file move plus a manifest re-add,
	// not a reconstruction.
	if archived[0].WorkingPath != retired.WorkingPath || archived[0].LatestLaunchProfile == nil {
		t.Fatalf("archived record lost detail: %+v", archived[0])
	}
}

// Archiving a thread couch is hosting would leave the console owning a record
// the store no longer lists -- a stale incarnation manufactured on purpose.
func TestArchiveThreadRefusesALiveOrParkingThread(t *testing.T) {
	store, _ := newTestThreadStore(t)

	live := archivableThread(t, store, "couch-0000000000000001")
	updated, err := store.UpdateExistingThread(live.Address, live.Revision, func(record *ThreadRecord) error {
		record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-live", State: IncarnationLive}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	live = updated
	if err := store.ArchiveThread(live.Address); err == nil {
		t.Fatal("archived a live thread")
	}
	if _, err := store.GetThread(live.Address); err != nil {
		t.Fatalf("refused archive still removed the record: %v", err)
	}

	// The mid-park branch this test was named for and never built. A park in
	// flight is a teardown already underway; archiving through it would leave
	// the transaction owning a record the store no longer lists.
	parking := archivableThread(t, store, "couch-0000000000000002")
	parked, err := store.UpdateExistingThread(parking.Address, parking.Revision, func(record *ThreadRecord) error {
		record.Incarnations = []ThreadIncarnation{{PID: 43, Identity: "pair-parking", State: IncarnationLive}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	begun, err := store.BeginPark(parked.Address, parked.Revision, ParkIdentity{
		Nonce: "park-0123456789abcdef", Address: parked.Address, PID: 43, ProcessIdentity: "pair-parking",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ArchiveThread(begun.Address); err == nil {
		t.Fatal("archived a thread with a park in flight")
	}
	// DISCRIMINATING: the refusal must come from the park, not from the live
	// incarnation the park needed in order to exist. Deleting the Park branch
	// left this test green until it asked which rule fired.
	if err := archivableRecord(ThreadRecord{
		Address: begun.Address, Park: begun.Park,
	}); err == nil {
		t.Fatal("a park in flight with no incarnation was archivable -- the park branch is unexercised")
	}
}

// An unusable thread is exactly what the operator wants gone, so no reason
// blocks the move -- the whole point is that debris leaves on their say-so.
func TestArchiveThreadAcceptsAnUnusableThread(t *testing.T) {
	store, _ := newTestThreadStore(t)
	broken := archivableThread(t, store, "couch-0000000000000001")
	if err := store.ArchiveThread(broken.Address); err != nil {
		t.Fatalf("ArchiveThread on debris: %v", err)
	}
}

func TestArchiveThreadRefusesAnUnknownAddress(t *testing.T) {
	store, _ := newTestThreadStore(t)
	err := store.ArchiveThread(ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000009"})
	if err == nil {
		t.Fatal("archived a thread that does not exist")
	}
}

// The whole point of a complete delete: the session stops. A record filed while
// its agent keeps running is the forgotten thread couch exists to prevent, and
// the one-time cleanup produced exactly one of those before this existed.
func TestCouchArchiveStopsTheSessionBeforeMovingTheRecord(t *testing.T) {
	store, _ := newTestThreadStore(t)
	thread := archivableThread(t, store, "couch-0000000000000001")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-"+string(thread.Address.Tag), true)
	artifacts.SetDetachedSession(thread.Address, "pair-"+string(thread.Address.Tag))
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}

	if _, err := couch.ArchiveThread(context.Background(), thread.Address); err != nil {
		t.Fatalf("ArchiveThread: %v", err)
	}
	if got := artifacts.Quiesces(); len(got) != 1 || got[0] != thread.Address {
		t.Fatalf("quiesced %+v, want exactly the archived thread", got)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
}

// Order is the property, not just the pair of effects. Archiving first and
// stopping second would leave a record in the archive with a live session
// behind it -- the exact state this action removes.
func TestCouchArchiveRefusesWhenTheSessionCannotBeStopped(t *testing.T) {
	store, _ := newTestThreadStore(t)
	thread := archivableThread(t, store, "couch-0000000000000001")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-"+string(thread.Address.Tag), false)
	artifacts.QuiesceHook = func(ThreadAddress) error { return errors.New("zellij is not answering") }
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}

	if _, err := couch.ArchiveThread(context.Background(), thread.Address); err == nil {
		t.Fatal("archived a thread whose session could not be stopped")
	}
	if _, err := store.GetThread(thread.Address); err != nil {
		t.Fatalf("a refused archive still moved the record: %v", err)
	}
	archived, _ := store.ArchivedThreads()
	if len(archived) != 0 {
		t.Fatalf("archive = %+v, want nothing", archived)
	}
}

// writeCorruptRecord puts a file the decoder cannot read into a real store,
// listed in the manifest. CreateThread cannot produce this shape, which is why
// no fixture had one -- and why `invalid` was a documented, labelled,
// archive-exit reason that no store could actually produce.
func writeCorruptRecord(t *testing.T, store *ThreadStore, sibling ThreadRecord, tag ThreadTag) ThreadAddress {
	t.Helper()
	address := ThreadAddress{RepoScope: sibling.Address.RepoScope, Tag: tag}
	// Create it validly, then corrupt the bytes on disk: that is how a real
	// store reaches this state (a truncated write, a rolled-back schema).
	record := actionableTestThread(tag, time.Unix(100, 0).UTC())
	record.Address.RepoScope = sibling.Address.RepoScope
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.recordPath(address), []byte(`{"schema_version":99,"nope":`), 0o600); err != nil {
		t.Fatal(err)
	}
	return address
}

// The thesis of #181, applied to the one shape it never tested: a record the
// decoder rejects must still produce a visible row, must not remove other rows,
// and must be archivable. It previously did none of those -- one corrupt file
// made `couch --list` exit 1 and the switcher show nothing.
func TestAnUndecodableRecordIsAVisibleRowAndCanBeArchived(t *testing.T) {
	store, _ := newTestThreadStore(t)
	healthy := archivableThread(t, store, "couch-0000000000000001")
	corrupt := writeCorruptRecord(t, store, healthy, "couch-0000000000000002")

	couch := &Couch{
		Threads: store, Artifacts: NewFakeThreadArtifactCollisionChecker(), Path: NewFakePathOps(nil),
	}
	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatalf("one corrupt record failed the whole inventory: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want both threads listed: %+v", len(rows), rows)
	}
	found := false
	for _, row := range rows {
		if row.Address == corrupt {
			found = true
			// `unreadable`, not `invalid`: couch could not read the bytes at
			// all, which is a different claim from "this record is not valid"
			// -- and the difference matters, because an older couch reading a
			// newer store cannot read ANY record and must not call the
			// operator's live work debris.
			if row.State != ThreadUnusable || row.Reason != ReasonUnreadable {
				t.Fatalf("corrupt row = %+v, want unusable/unreadable", row)
			}
		}
	}
	if !found {
		t.Fatalf("the corrupt record produced no row: %+v", rows)
	}

	// And it can leave: an unusable row the operator cannot remove is worse
	// than one they cannot use.
	// It archives -- a SUCCESS, because the mutation happened -- and reports in
	// the result that it did not stop the session, since quiescing would kill
	// an agent on the strength of a record it just failed to read.
	result, err := couch.ArchiveThread(context.Background(), corrupt)
	if err != nil {
		t.Fatalf("archiving an undecodable record: %v", err)
	}
	if !result.SessionNotStopped || result.Warning() == "" {
		t.Fatalf("archive result = %+v, want it to report the session was left alone", result)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 || archived[0].Address != corrupt {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
}

// Occupancy is one rule now, and this is the case that made it matter: a thread
// mid-start passed the store's narrower guard, so archiving it killed the
// session being created while the spawn was still in flight.
// Unknown stays conservative: an unreadable record is filed, and its session is
// NOT stopped. Quiescing there would kill an agent couch cannot identify, which
// is the version-skew harm the unreadable/invalid split exists to prevent.
func TestArchivingAnUnreadableRecordNeverStopsItsSession(t *testing.T) {
	store, _ := newTestThreadStore(t)
	healthy := archivableThread(t, store, "couch-0000000000000001")
	corrupt := writeCorruptRecord(t, store, healthy, "couch-0000000000000002")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(corrupt, "pair-"+string(corrupt.Tag), true)
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}

	result, err := couch.ArchiveThread(context.Background(), corrupt)
	if err != nil {
		t.Fatalf("archiving an unreadable record: %v", err)
	}
	if !result.SessionNotStopped {
		t.Fatal("archive did not report that it left the session alone")
	}
	if got := artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("quiesced %+v -- couch stopped a session it could not identify", got)
	}
	archived, _ := store.ArchivedThreads()
	if len(archived) != 1 || archived[0].Address != corrupt {
		t.Fatalf("archive = %+v, want the unreadable record filed anyway", archived)
	}
}

// The guard runs BEFORE any effect. It used to run inside the store, after
// Quiesce, so a park-in-flight thread had its session killed and was then
// refused: agent dead, record still listed.
func TestARefusedArchiveStopsNothing(t *testing.T) {
	store, _ := newTestThreadStore(t)
	thread := archivableThread(t, store, "couch-0000000000000001")
	live, err := store.UpdateExistingThread(thread.Address, thread.Revision, func(record *ThreadRecord) error {
		record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-live", State: IncarnationLive}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(live.Address, "pair-live-session", true)
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}

	if _, err := couch.ArchiveThread(context.Background(), live.Address); err == nil {
		t.Fatal("archived a live thread")
	}
	if got := artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("a REFUSED archive stopped %+v", got)
	}
}

func TestArchiveRefusesEveryOccupiedIncarnationNotJustLive(t *testing.T) {
	for _, state := range []IncarnationState{IncarnationLive, IncarnationCreating, IncarnationUnknown} {
		t.Run(string(state), func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			thread := archivableThread(t, store, "couch-0000000000000001")
			updated, err := store.UpdateExistingThread(thread.Address, thread.Revision, func(record *ThreadRecord) error {
				record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-x", State: state}}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := store.ArchiveThread(updated.Address); err == nil {
				t.Fatalf("archived a thread with a %s incarnation", state)
			}
			if _, err := store.GetThread(updated.Address); err != nil {
				t.Fatalf("a refused archive still moved the record: %v", err)
			}
		})
	}
}

// An unreadable record must BLOCK its repository rather than free it. It has no
// working path to match on -- reading it is what would have supplied one -- so
// treating it as absent starts a second thread in a tree that may already hold
// live work, silently, where the old code failed loudly.
func TestAnUnreadableRecordBlocksStartsInItsRepository(t *testing.T) {
	rows := []ActionableThreadSummary{{
		Address: ThreadAddress{RepoScope: "scope", Tag: "couch-0000000000000001"},
		State:   ThreadUnusable, Reason: ReasonUnreadable,
	}}
	if _, blocked := PathHoldsUnreadableThread(rows, "scope"); !blocked {
		t.Fatal("an unreadable record did not block its repository")
	}
	if _, blocked := PathHoldsUnreadableThread(rows, "other-scope"); blocked {
		t.Fatal("an unreadable record blocked a different repository")
	}
	// Every other unusable reason is a KNOWN state, so it does not block:
	// a path whose only rows are debris must stay startable.
	for _, reason := range AllThreadReasons() {
		if reason == ReasonUnreadable {
			continue
		}
		rows[0].Reason = reason
		if _, blocked := PathHoldsUnreadableThread(rows, "scope"); blocked {
			t.Fatalf("reason %q blocked a start; only unreadable is unknown", reason)
		}
	}
}

// The guard this round ADDED, entered by a test at last. The reviewer deleted
// the whole block from spawnResolved and no test outcome changed: the message
// had been fixed and its commands pinned, but nothing proved the refusal fires.
//
// Spawn is the seam every creation entry funnels through, so this is the one
// place that covers `couch <path>`, the TUI start form and SpawnPrepared alike.
func TestSpawnRefusesWhileAnUnreadableRecordIsInTheRepository(t *testing.T) {
	env := newTestEnv(t, "/repo")
	healthy := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	corrupt := writeCorruptRecord(t, env.Couch.Threads, healthy, "couch-0000000000000002")

	_, _, err := env.Couch.Spawn(StartArgs{Worktree: "/repo"})
	if err == nil {
		t.Fatal("started a second thread while a record in this repository could not be read")
	}
	// Every next step the refusal names is checked for presence here and
	// EXECUTED in couchcmd's seam test; a refusal with unnamed steps is the
	// class this issue kept reopening.
	for _, want := range []string{string(corrupt.Tag), "couch --show", "another repository", "the record:"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not mention %q", err, want)
		}
	}
}

func TestRecoveryArchivePreservesPendingCheckpointAndHistory(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := archivableThread(t, store, "couch-0000000000000250")
	request := testContinuationRequest(t, record)
	record, err := store.PublishContinuation(record.Address, record.Revision, request)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(record.Address, "pair-source", false)
	c := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}
	materialized, err := c.materializeContinuation(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ArchiveThread(context.Background(), record.Address); err != nil {
		t.Fatalf("archive missing-session checkpoint: %v", err)
	}
	if _, err := os.Stat(materialized); !os.IsNotExist(err) {
		t.Fatalf("derived checkpoint remains: %v", err)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 || archived[0].Continuation == nil || !reflect.DeepEqual(*archived[0].Continuation, request) {
		t.Fatalf("archive lost pending checkpoint: %+v, %v", archived, err)
	}
}

func TestRecoveryArchiveRetiresOnlyExactDeadSettledHelper(t *testing.T) {
	for _, mode := range []string{"dead", "pid-reused", "live", "unknown", "session-unknown", "creating"} {
		t.Run(mode, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := archivableThread(t, store, "couch-0000000000000250")
			record, err := store.UpdateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error {
				r.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "original", State: IncarnationLive}}
				if mode == "creating" {
					r.Incarnations[0].State = IncarnationCreating
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			proc := NewFakeProcOps()
			if mode == "live" {
				proc.Set(42, "original")
			}
			if mode == "pid-reused" {
				proc.Set(42, "other")
			}
			if mode == "unknown" {
				proc.SetUnknown(42)
			}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(record.Address, "pair-source", false)
			if mode == "session-unknown" {
				artifacts.BeforePairSession = func(ThreadAddress) error { return errors.New("query failed") }
			}
			c := &Couch{Threads: store, Artifacts: artifacts, Proc: proc, Path: NewFakePathOps(nil)}
			_, err = c.ArchiveThread(context.Background(), record.Address)
			want := mode == "dead" || mode == "pid-reused"
			if (err == nil) != want {
				t.Fatalf("archive %s = %v", mode, err)
			}
			if !want {
				after, e := store.GetThread(record.Address)
				if e != nil || after.Revision != record.Revision || len(artifacts.Quiesces()) != 0 {
					t.Fatalf("refusal mutated record/session: %+v %v", after, e)
				}
			}
			if len(proc.Signals) != 0 || len(proc.GroupSignals) != 0 {
				t.Fatal("archive signalled helper")
			}
		})
	}
}

func TestRecoveryArchiveRejectsRecordChangedDuringQuiesce(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := archivableThread(t, store, "couch-0000000000000250")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(record.Address, "pair-source", false)
	artifacts.QuiesceHook = func(ThreadAddress) error {
		_, err := store.UpdateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error { r.Name = "concurrent update"; return nil })
		return err
	}
	c := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}
	_, err := c.ArchiveThread(context.Background(), record.Address)
	var conflict *ThreadRevisionError
	if !errors.As(err, &conflict) {
		t.Fatalf("archive accepted changed record: %v", err)
	}
	if after, e := store.GetThread(record.Address); e != nil || after.Name != "concurrent update" {
		t.Fatalf("archive lost concurrent state: %+v %v", after, e)
	}
}

func TestRecoveryArchiveRefusesSessionAppearingBeforeStop(t *testing.T) {
	for _, initiallyPresent := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent-to-present", true: "session-replaced"}[initiallyPresent], func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := archivableThread(t, store, "couch-0000000000000250")
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(record.Address, "old-session", initiallyPresent)
			if initiallyPresent {
				artifacts.SetDetachedSession(record.Address, "old-session")
			}
			calls := 0
			artifacts.BeforePairSession = func(ThreadAddress) error {
				calls++
				if calls == 2 {
					artifacts.SetPairSession(record.Address, "new-session", true)
					artifacts.SetDetachedSession(record.Address, "new-session")
				}
				return nil
			}
			c := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}
			_, err := c.ArchiveThread(context.Background(), record.Address)
			if err == nil || len(artifacts.Quiesces()) != 0 {
				t.Fatalf("archive touched newly appearing session: err=%v stops=%v", err, artifacts.Quiesces())
			}
			if _, err := store.GetThread(record.Address); err != nil {
				t.Fatal("archive removed raced record")
			}
		})
	}
}

func TestRecoveryArchiveRefusesLiveContinuationReferences(t *testing.T) {
	for _, actor := range []string{"source", "target", "target-unknown", "session"} {
		t.Run(actor, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := archivableThread(t, store, "couch-0000000000000250")
			request := testContinuationRequest(t, record)
			request.Phase = checkpoint.Running
			request.Attempt = "attempt"
			request.SourceAbsence = &checkpoint.SourceAbsence{Session: request.Source.Session, LaunchOrdinal: request.Source.LaunchOrdinal, ObservedAt: record.CreatedAt, RecordRevision: record.Revision}
			request.Target = &checkpoint.Target{Process: checkpoint.Process{PID: 43, Identity: "target"}, ObservedAt: record.CreatedAt}
			record, err := store.UpdateExistingThread(record.Address, record.Revision, func(r *ThreadRecord) error { r.Continuation = &request; return nil })
			if err != nil {
				t.Fatal(err)
			}
			proc := NewFakeProcOps()
			switch actor {
			case "source":
				proc.Set(42, "source")
			case "target":
				proc.Set(43, "target")
			case "target-unknown":
				proc.SetUnknown(43)
			}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(record.Address, "pair-source", actor == "session")
			if actor == "session" {
				artifacts.SetDetachedSession(record.Address, "pair-source")
			}
			c := &Couch{Threads: store, Artifacts: artifacts, Proc: proc, Path: NewFakePathOps(nil)}
			if _, err := c.ArchiveThread(context.Background(), record.Address); err == nil {
				t.Fatal("archive accepted occupied continuation")
			}
			if len(artifacts.Quiesces()) != 0 {
				t.Fatal("archive stopped unfinished continuation")
			}
			after, err := store.GetThread(record.Address)
			if err != nil || after.Revision != record.Revision || !reflect.DeepEqual(after.Continuation, record.Continuation) {
				t.Fatalf("refused archive changed request: %+v %v", after, err)
			}
		})
	}
}

func TestArchiveExpectedUnreadableObservationDoesNotArchiveRepairedRecord(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := archivableThread(t, store, "couch-0000000000000250")
	// Revision zero represents an unreadable observation. A repaired record must
	// be inspected afresh, even if its new state happens to be unoccupied.
	if err := store.ArchiveThreadExpected(record.Address, 0); err == nil {
		t.Fatal("unreadable observation archived a repaired record")
	}
	corrupt := writeCorruptRecord(t, store, record, "couch-0000000000000251")
	if err := store.ArchiveThreadExpected(corrupt, 0); err != nil {
		t.Fatalf("unchanged unreadable record lost archive escape: %v", err)
	}
}

func TestCouchArchiveNeverBoundThreadDoesNotSignal(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprintf("unreadable-index=%v", corrupt), func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := archivableThread(t, store, "couch-0000000000000001")
			global := t.TempDir()
			paths := launcher.NewScopedPaths(global, launcher.RepoScope{Key: record.Address.RepoScope}, string(record.Address.Tag))
			if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
				t.Fatal(err)
			}
			if corrupt {
				if err := os.Mkdir(paths.SessionBindings(), 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(paths.SessionBindings(), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			checker := &archiveScopedChecker{ScopedThreadArtifactCollisionChecker: NewScopedThreadArtifactCollisionChecker(global)}
			deleter := &fakeSessionDeleter{}
			checker.Sessions = deleter
			c := &Couch{Threads: store, Artifacts: checker, Proc: NewFakeProcOps()}
			_, err := c.ArchiveThread(context.Background(), record.Address)
			if corrupt {
				if err == nil {
					t.Fatal("archived with unreadable index")
				}
				if _, err := store.GetThread(record.Address); err != nil {
					t.Fatal(err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				archived, err := store.ArchivedThreads()
				if err != nil || len(archived) != 1 || !reflect.DeepEqual(archived[0], record) {
					t.Fatalf("archive=%+v err=%v", archived, err)
				}
			}
			if len(deleter.deleted) != 0 || checker.stops != 0 {
				t.Fatalf("signalled sessions: %v, quiesce calls: %d", deleter.deleted, checker.stops)
			}
		})
	}
}

// Keep the real index reader while observing the destructive seam and injecting
// a competing record publication between its two exact missing-binding reads.
type archiveScopedChecker struct {
	ScopedThreadArtifactCollisionChecker
	before func()
	stops  int
}

func (a *archiveScopedChecker) PairSessionContext(ctx context.Context, address ThreadAddress) (PairSessionBinding, error) {
	if a.before != nil {
		a.before()
	}
	return a.ScopedThreadArtifactCollisionChecker.PairSessionContext(ctx, address)
}
func (a *archiveScopedChecker) Quiesce(address ThreadAddress) error {
	a.stops++
	return a.ScopedThreadArtifactCollisionChecker.Quiesce(address)
}

func TestCouchArchiveMissingBindingDoesNotBypassRequestOrRevision(t *testing.T) {
	for _, mode := range []string{"request", "revision-race", "index-race"} {
		t.Run(mode, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := archivableThread(t, store, "couch-0000000000000001")
			if mode == "request" {
				var err error
				record, err = store.PublishContinuation(record.Address, record.Revision, testContinuationRequest(t, record))
				if err != nil {
					t.Fatal(err)
				}
			}
			global := t.TempDir()
			checker := &archiveScopedChecker{ScopedThreadArtifactCollisionChecker: NewScopedThreadArtifactCollisionChecker(global)}
			reads := 0
			checker.before = func() {
				reads++
				if reads != 2 {
					return
				}
				if mode == "revision-race" {
					if _, err := store.PublishContinuation(record.Address, record.Revision, testContinuationRequest(t, record)); err != nil {
						t.Fatal(err)
					}
				}
				if mode == "index-race" {
					paths := launcher.NewScopedPaths(global, launcher.RepoScope{Key: record.Address.RepoScope}, string(record.Address.Tag))
					if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
						t.Fatal(err)
					}
					// An unreadable newly published index is unknown, never missing.
					if err := os.Mkdir(paths.SessionBindings(), 0700); err != nil {
						t.Fatal(err)
					}
				}
			}
			c := &Couch{Threads: store, Artifacts: checker, Proc: NewFakeProcOps()}
			if _, err := c.ArchiveThread(context.Background(), record.Address); err == nil {
				t.Fatal("archived despite retained request or changed evidence")
			}
			if _, err := store.GetThread(record.Address); err != nil {
				t.Fatal(err)
			}
			if checker.stops != 0 {
				t.Fatalf("quiesced %d times", checker.stops)
			}
		})
	}
}
