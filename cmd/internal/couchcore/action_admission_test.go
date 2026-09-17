package couchcore

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestArchiveRefusesAThreadCouchHostsWithNoIncarnation is the archive half of
// the class TestSwitchAgentRefusesAThreadCouchHostsWithNoIncarnation pins.
//
// #256 M1 made couch's own hosting the live proof and stopped consulting the
// record's incarnation, so a thread couch hosts can carry no incarnation at
// all. `archivableRecord` asked only the record, so this row passed its
// occupancy rule -- and archive's next move is Quiesce, which kills the session
// the agent is running in. The guard has to ask the classification, because the
// record has nothing to say about it.
func TestArchiveRefusesAThreadCouchHostsWithNoIncarnation(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	record.Reservation = false
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Couch.reg = env.Couch.reg.Insert(ActorRecord{
		ID: ActorID("hosted-actor"), Thread: created.Address,
		Args: StartArgs{Worktree: Worktree(created.StartingPath), Cwd: created.WorkingPath},
		PID:  4242, Identity: "hosted",
	})

	state, reason, err := env.Couch.classifyForAction(context.Background(), created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if state != ThreadLive {
		t.Fatalf("fixture classified %q/%q, not live -- it no longer exercises the hosted shape", state, reason)
	}
	if _, err := env.Couch.ArchiveThread(context.Background(), created.Address); err == nil {
		t.Fatal("archived a thread couch is hosting")
	} else if !strings.Contains(err.Error(), string(ThreadLive)) {
		// DISCRIMINATING: any refusal would pass a bare err != nil, and the
		// record-shaped guards below would produce one for an unrelated reason.
		// The classification is what must refuse here, so it must be named.
		t.Fatalf("refusal does not name the classification that produced it: %v", err)
	}
}

// SelectResumableRoot's eligibility and ResumableState must be the same
// question. They were two hand-written lists of the same two states, which is
// the drift the action tables exist to catch, one layer below the switcher.
func TestStartupSelectionDerivesFromResumableState(t *testing.T) {
	for _, state := range AllThreadStates() {
		for _, reason := range append(AllThreadReasons(), "") {
			if (state == ThreadUnusable) != (reason != "") {
				continue
			}
			rows := []ActionableThreadSummary{{
				Address:     ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"},
				State:       state,
				Reason:      reason,
				WorkingPath: "/repo",
			}}
			_, selected := SelectResumableRoot(rows, "816fc349d3faebf8", "/repo")
			if selected != ResumableState(state, reason) {
				t.Errorf("%s/%s: startup selects=%v, the resumable rule says %v",
					state, reason, selected, ResumableState(state, reason))
			}
		}
	}
}

// Every classification archive refuses must have something to say about itself.
// A refusal with an empty or generic message is how "it cannot be archived"
// became the answer for four different situations with four different fixes.
func TestArchiveRefusalCoversEveryRefusedClassification(t *testing.T) {
	seen := map[string]ActionableThreadState{}
	refused := 0
	for _, state := range AllThreadStates() {
		for _, reason := range append(AllThreadReasons(), "") {
			if (state == ThreadUnusable) != (reason != "") {
				continue
			}
			if ArchivableState(state, reason) {
				continue
			}
			refused++
			message := archiveRefusal(state, reason)
			if message == "" {
				t.Errorf("%s/%s: archive refuses it and says nothing", state, reason)
				continue
			}
			if other, clash := seen[message]; clash && other != state {
				t.Errorf("%s and %s share the refusal %q; two situations, one answer", other, state, message)
			}
			seen[message] = state
		}
	}
	if refused == 0 {
		t.Fatal("no classification is refused, so this table proves nothing")
	}
}

// The store's archive guard asks only what a DECODED RECORD proves on its own.
//
// It used to also refuse an occupied incarnation, and that is a claim about a
// PROCESS: the store cannot probe one, and since M1 the record's incarnation
// does not answer the question anyway -- it names couch's launcher, which dies
// with couch. That refusal moved to Couch.ArchiveThread, which classifies.
//
// What stays is couch's own unfinished bookkeeping, which the record does prove
// by carrying it: an open park transaction and an outstanding start claim.
// Archiving through either strands a transaction pointing at a record the
// working set no longer lists.
func TestStoreArchiveGuardAsksOnlyWhatARecordProves(t *testing.T) {
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"}

	for _, state := range []IncarnationState{IncarnationLive, IncarnationCreating, IncarnationUnknown} {
		record := ThreadRecord{Address: address, Incarnations: []ThreadIncarnation{
			{PID: 42, Identity: "pair-live", State: state},
		}}
		if err := archivableRecord(record); err != nil {
			t.Errorf("incarnation %q: the store refused on a process it cannot probe: %v", state, err)
		}
	}

	park := ThreadRecord{Address: address, Park: &ParkTransaction{Identity: ParkIdentity{
		Nonce: "park-0123456789abcdef", Address: address, PID: 43, ProcessIdentity: "pair-parking",
	}}}
	if err := archivableRecord(park); err == nil {
		t.Error("the store archived through an open park transaction")
	} else if !strings.Contains(err.Error(), "park") {
		t.Errorf("park refusal does not name the park: %v", err)
	}

	claim := ThreadRecord{Address: address, Incarnations: []ThreadIncarnation{{
		PID: 44, Identity: "pair-starting", State: IncarnationCreating,
		Start: &ThreadStartClaim{Nonce: "start-0123456789abcdef"},
	}}}
	if err := archivableRecord(claim); err == nil {
		t.Error("the store archived through an outstanding start claim")
	} else if !strings.Contains(err.Error(), "start") {
		t.Errorf("start-claim refusal does not name the claim: %v", err)
	}
}

// A record whose incarnation is `unknown` and whose process is PROVED DEAD is
// archivable, and could not be archived at all before #256 M3.
//
// `markLiveRecordUnknown` produces the shape in production: a start reaches a
// live helper, the console attach then fails, and the incarnation is marked
// unproven. That helper is couch's own child and dies with couch, so the row
// arrives at the next couch classified `detached` or `parked` -- archive offered
// -- and clearLifecycleDebris refuses it EVERY time, because the only retirement
// transition available demands a `live` incarnation.
//
// RetireIncarnation's refusal of `unknown` is right for the caller it was
// written for: detach has no death proof, and retiring an unproven incarnation
// there would let it present as cleanly detached. Archive is the other caller
// and it does have the proof -- exact PID and identity token, observed Dead
// immediately above. So the proof gets a transition of its own rather than a
// widened one.
func TestArchiveRetiresAnUnprovenIncarnationItProvedDead(t *testing.T) {
	store, _ := newTestThreadStore(t)
	thread := archivableThread(t, store, "couch-0000000000000001")
	unproven, err := store.UpdateExistingThread(thread.Address, thread.Revision, func(record *ThreadRecord) error {
		record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-x", State: IncarnationUnknown}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(unproven.Address, "pair-"+string(unproven.Address.Tag), true)
	artifacts.SetDetachedSession(unproven.Address, "pair-"+string(unproven.Address.Tag))
	// pid 42 is absent from the table, so the probe answers Dead -- not unknown.
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}

	if _, err := couch.ArchiveThread(context.Background(), unproven.Address); err != nil {
		t.Fatalf("a row whose helper is provably gone could not be archived: %v", err)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
}

// The other direction, which is why the transition is separate rather than a
// widened RetireIncarnation: an `unknown` incarnation whose process CANNOT be
// proved dead stays exactly where it is.
//
// Since Task 9 the refusal arrives one layer EARLIER than it used to, and that
// is the point rather than an accident: an unprovable recorded process makes the
// row classify `unusable/unknown`, so archive declines at the admission rule,
// having asked nothing further and written nothing. clearLifecycleDebris's own
// death-proof screen is still there and still refuses with a code --
// TestReAdoptionExitsAreTotalAndCoded drives it over every record shape -- but
// this path no longer reaches it, so asserting that code here would pin a
// layer this test does not exercise.
func TestArchiveKeepsAnUnprovenIncarnationItCouldNotProveDead(t *testing.T) {
	store, _ := newTestThreadStore(t)
	thread := archivableThread(t, store, "couch-0000000000000001")
	unproven, err := store.UpdateExistingThread(thread.Address, thread.Revision, func(record *ThreadRecord) error {
		record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-x", State: IncarnationUnknown}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(unproven.Address, "pair-"+string(unproven.Address.Tag), true)
	artifacts.SetDetachedSession(unproven.Address, "pair-"+string(unproven.Address.Tag))
	proc := NewFakeProcOps()
	proc.SetUnknown(42)
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: proc, Path: NewFakePathOps(nil)}

	_, err = couch.ArchiveThread(context.Background(), unproven.Address)
	if err == nil {
		t.Fatal("archived a thread whose recorded helper could not be proved dead")
	}
	// DISCRIMINATING: the classification is what must refuse, so its answer has
	// to be in the message. A bare err != nil would pass on any of four guards.
	if !strings.Contains(err.Error(), ReasonUnknown.Label()) {
		t.Fatalf("refusal does not name the unresolved classification: %v", err)
	}
	if got := artifacts.Quiesces(); len(got) != 0 {
		t.Fatalf("a REFUSED archive stopped %+v", got)
	}
}

// Each retirement transition takes exactly the incarnation state it names.
//
// Written as a table over BOTH, because the risk of splitting a transition in
// two is that one of them quietly becomes a superset of the other and the split
// stops meaning anything -- and clearLifecycleDebris routes by state, so a
// widened precondition would be unobservable from there.
func TestRetirementTransitionsTakeExactlyTheStateTheyName(t *testing.T) {
	transitions := []struct {
		name     string
		accepts  IncarnationState
		transfer func(*ThreadStore, ThreadAddress, uint64, ProcessIdentity, time.Time) (ThreadRecord, error)
	}{
		{"RetireIncarnation", IncarnationLive, (*ThreadStore).RetireIncarnation},
		{"RetireUnprovenIncarnation", IncarnationUnknown, (*ThreadStore).RetireUnprovenIncarnation},
	}
	for _, transition := range transitions {
		for _, state := range []IncarnationState{IncarnationLive, IncarnationCreating, IncarnationUnknown} {
			t.Run(transition.name+"/"+string(state), func(t *testing.T) {
				store, _ := newTestThreadStore(t)
				thread := archivableThread(t, store, "couch-0000000000000001")
				with, err := store.UpdateExistingThread(thread.Address, thread.Revision, func(record *ThreadRecord) error {
					record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-x", State: state}}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				retired, err := transition.transfer(store, with.Address, with.Revision,
					ProcessIdentity{PID: 42, Identity: "pair-x"}, time.Unix(300, 0).UTC())
				if state == transition.accepts {
					if err != nil {
						t.Fatalf("%s refused the %s incarnation it is named for: %v", transition.name, state, err)
					}
					if len(retired.Incarnations) != 0 {
						t.Fatalf("%s left %d incarnation(s)", transition.name, len(retired.Incarnations))
					}
					return
				}
				if err == nil {
					t.Fatalf("%s retired a %s incarnation", transition.name, state)
				}
				// DISCRIMINATING: the identity and park screens would also
				// produce an error, and the state precondition is the subject.
				if !strings.Contains(err.Error(), string(transition.accepts)) {
					t.Fatalf("%s refused for some other reason: %v", transition.name, err)
				}
			})
		}
	}
}
