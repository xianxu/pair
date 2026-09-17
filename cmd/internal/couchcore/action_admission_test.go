package couchcore

import (
	"context"
	"strings"
	"testing"
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
