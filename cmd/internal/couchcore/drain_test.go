package couchcore

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDrainNoticeLiteral(t *testing.T) {
	got, err := DrainNotice("00000000-0000-4000-8000-000000000001", "quoted \"data\"", false)
	want := "Couch drain v1 00000000-0000-4000-8000-000000000001: checkpoint and stop. Finish the current atomic step; commit and push the checkpoint; write where you stopped in the issue Log. Start nothing new. Run couch --drain --ack 00000000-0000-4000-8000-000000000001 after the checkpoint is durable, then end your turn and stay idle. Do not send a separate drained report.\nReason: \"quoted \\\"data\\\"\""
	if err != nil || got != want {
		t.Fatalf("notice = %q, %v", got, err)
	}
	if _, err := DrainNotice("bad", "", false); err == nil {
		t.Fatal("accepted invalid ID")
	}
	if _, err := DrainNotice("00000000-0000-4000-8000-000000000001", "\x1b[31m", false); err == nil {
		t.Fatal("accepted control")
	}
	fast, err := DrainNotice("00000000-0000-4000-8000-000000000001", "", true)
	if err != nil || !strings.Contains(fast, "(fast): stop now and checkpoint. Do not finish the current step.") {
		t.Fatal(fast, err)
	}
}

func drainFixture(t *testing.T) DrainState {
	t.Helper()
	s, fx, err := StartDrain(DrainStart{ID: "00000000-0000-4000-8000-000000000001", Deadline: time.Unix(1000, 0), Members: []DrainMember{{Workspace: "pair:1", Identity: "binding-session"}}})
	if err != nil || len(fx) != 1 || fx[0].Kind != DrainPause {
		t.Fatalf("start %v %v", fx, err)
	}
	return s
}
func stepDrain(s DrainState, e DrainEvent) (DrainState, []DrainEffect) {
	e.ID = s.ID
	return ReduceDrain(s, e)
}
func drainPaused(s DrainState) DrainState {
	s, _ = stepDrain(s, DrainEvent{Kind: DrainQueueObserved, Queue: DrainQueueObservation{Known: true, Complete: true, Paused: true, Owners: []string{s.Owner}}})
	return s
}
func TestDrainAckRequiresSubsequentIdleAndInvalidates(t *testing.T) {
	s := drainPaused(drainFixture(t))
	s, _ = stepDrain(s, DrainEvent{Kind: DrainNoticeResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
	obs := DrainObservation{Known: true, Connected: true, Idle: true}
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: obs})
	if b := ProjectDrainBoard(s, time.Unix(1, 0)); b.State != DrainStuck || b.Rows[0].Reason != "no ack" {
		t.Fatalf("quiet without ack: %+v", b)
	}
	s, _ = stepDrain(s, DrainEvent{Kind: DrainAcknowledged, Workspace: "pair:1", Identity: "binding-session"})
	if ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
		t.Fatal("ack reused pre-ack idle")
	}
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: obs})
	if b := ProjectDrainBoard(s, time.Unix(1, 0)); b.State != DrainDrained {
		t.Fatalf("not drained %+v", b)
	}
	old := s
	obs.Idle = false
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: obs})
	if ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
		t.Fatal("busy reused ack")
	}
	if !old.Members["pair:1"].AckSealed {
		t.Fatal("mutated input")
	}
}
func TestDrainAuthorityMutations(t *testing.T) {
	for _, mutate := range []func(*DrainState){
		func(s *DrainState) { s.Queue.Known = false }, func(s *DrainState) { s.Queue.Complete = false }, func(s *DrainState) { s.Queue.Paused = false }, func(s *DrainState) { s.Queue.Owners = []string{"other"} }, func(s *DrainState) { s.Queue.Active = []DrainActivity{{ID: "a", Workspace: "", Kind: "suite"}} },
		func(s *DrainState) { m := s.Members["pair:1"]; m.Observation.Known = false; s.Members["pair:1"] = m }, func(s *DrainState) { m := s.Members["pair:1"]; m.AckSealed = false; s.Members["pair:1"] = m },
	} {
		s := drainPaused(drainFixture(t))
		s, _ = stepDrain(s, DrainEvent{Kind: DrainNoticeResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
		s, _ = stepDrain(s, DrainEvent{Kind: DrainAcknowledged, Workspace: "pair:1", Identity: "binding-session"})
		s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Idle: true}})
		mutate(&s)
		if ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
			t.Fatal("missing authority allowed drain")
		}
	}
}
func TestDrainStaleAndCancelIsolation(t *testing.T) {
	s := drainPaused(drainFixture(t))
	old := s
	s, fx := ReduceDrain(s, DrainEvent{Kind: DrainAcknowledged, ID: "old", Workspace: "pair:1", Identity: "binding-session"})
	if len(fx) != 0 || s.Members["pair:1"].Ack {
		t.Fatal("stale event")
	}
	s, fx = stepDrain(s, DrainEvent{Kind: DrainCancel})
	if len(fx) != 1 || fx[0].Kind != DrainRevoke {
		t.Fatalf("cancel %v", fx)
	}
	s, _ = stepDrain(s, DrainEvent{Kind: DrainQueueObserved, Queue: DrainQueueObservation{Known: true, Complete: true, Paused: true, Owners: []string{"other"}}})
	if ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainCancelled {
		t.Fatal("cancel released before revoke")
	}
	s, fx = stepDrain(s, DrainEvent{Kind: DrainRevoked, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
	if len(fx) != 1 || fx[0].Kind != DrainResume || fx[0].Owner != old.Owner {
		t.Fatal(fx)
	}
	s, _ = stepDrain(s, DrainEvent{Kind: DrainQueueObserved, Queue: DrainQueueObservation{Known: true, Complete: true, Paused: true, Owners: []string{"other"}}})
	if ProjectDrainBoard(s, time.Unix(1, 0)).State != DrainCancelled {
		t.Fatal("not cancelled")
	}
}

func TestDrainFastUnknownReconcilesWithoutDuplicateEffect(t *testing.T) {
	s := drainFixture(t)
	s.Fast = true
	s, fx := stepDrain(s, DrainEvent{Kind: DrainQueueObserved, Queue: DrainQueueObservation{Known: true, Complete: true, Paused: true, Owners: []string{s.Owner}}})
	if len(fx) != 1 || fx[0].Kind != DrainInterrupt {
		t.Fatalf("fast ordering %v", fx)
	}
	s, fx = stepDrain(s, DrainEvent{Kind: DrainInterruptResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainUnknown})
	if len(fx) != 0 {
		t.Fatal("unknown interrupt retried", fx)
	}
	s, fx = stepDrain(s, DrainEvent{Kind: DrainInterruptResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
	if len(fx) != 1 || fx[0].Kind != DrainSendNotice {
		t.Fatalf("reconciled interrupt did not release notice %v", fx)
	}
	_, fx = stepDrain(s, DrainEvent{Kind: DrainQueueObserved, Queue: s.Queue})
	if len(fx) != 0 {
		t.Fatal("duplicate control", fx)
	}
}

func TestDrainReorderedEvidenceAndReplacement(t *testing.T) {
	s := drainPaused(drainFixture(t))
	s, _ = stepDrain(s, DrainEvent{Kind: DrainNoticeResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Sequence: 2, Submission: 1}})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainAcknowledged, Workspace: "pair:1", Identity: "binding-session"})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Idle: true, Sequence: 1}})
	if ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
		t.Fatal("stale idle sealed ack")
	}
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Idle: true, Sequence: 3, Submission: 1}})
	if ProjectDrainBoard(s, time.Unix(1, 0)).State != DrainDrained {
		t.Fatal("fresh idle failed")
	}
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberJoined, Workspace: "pair:1", Identity: "replacement"})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainAcknowledged, Workspace: "pair:1", Identity: "binding-session"})
	if s.Members["pair:1"].Ack || ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
		t.Fatal("replacement inherited old ack")
	}
}

func TestDrainCountedFanoutAndIndependentSequenceOracle(t *testing.T) {
	for _, n := range []int{8, 16} {
		s := drainFixture(t)
		s.Members = make(map[string]DrainMember)
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("pair:%d", i)
			s.Members[key] = DrainMember{Workspace: key, Identity: "identity"}
		}
		s, fx := stepDrain(s, DrainEvent{Kind: DrainQueueObserved, Queue: DrainQueueObservation{Known: true, Complete: true, Paused: true, Owners: []string{s.Owner}}})
		if len(fx) != n {
			t.Fatalf("%d members: %d effects", n, len(fx))
		}
		total := len(fx)
		noticed := map[string]bool{}
		acked := map[string]bool{}
		idleAfter := map[string]bool{}
		// Seeded event permutation is independent of the reducer's state fields.
		rng := rand.New(rand.NewSource(int64(n)))
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("pair:%d", rng.Intn(n))
			kind := rng.Intn(3)
			e := DrainEvent{Workspace: key, Identity: "identity"}
			switch kind {
			case 0:
				e.Kind = DrainNoticeResult
				e.Outcome = DrainDone
				noticed[key] = true
			case 1:
				e.Kind = DrainAcknowledged
				if noticed[key] {
					if !acked[key] {
						idleAfter[key] = false
					}
					acked[key] = true
				}
			case 2:
				e.Kind = DrainMemberObserved
				idle := rng.Intn(2) == 0
				e.Observation = DrainObservation{Known: true, Connected: true, Idle: idle}
				if !idle && idleAfter[key] {
					acked[key] = false
					idleAfter[key] = false
				}
				if idle && acked[key] {
					idleAfter[key] = true
				}
			}
			s, fx = stepDrain(s, e)
			total += len(fx)
			if ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
				for key := range s.Members {
					if !acked[key] || !idleAfter[key] {
						t.Fatalf("false drained at event %d member %s", i, key)
					}
				}
			}
		}
		if total != n {
			t.Fatalf("%d members generated %d control calls", n, total)
		}
	}
}

func TestDrainRecoveryAndDeadlineNeverRelease(t *testing.T) {
	s := drainPaused(drainFixture(t))
	if b := ProjectDrainBoard(s, time.Unix(1001, 0)); b.State != DrainStuck {
		t.Fatal(b)
	}
	s, fx := stepDrain(s, DrainEvent{Kind: DrainRecover})
	if len(fx) != 0 || ProjectDrainBoard(s, time.Unix(1, 0)).State != DrainStuck {
		t.Fatal("recovery claimed proof")
	}
	if s.Phase == DrainFinished {
		t.Fatal("recovery released owner")
	}
}

func TestDrainSubmissionInvalidatesProvisionalAndSealedAck(t *testing.T) {
	for _, sealed := range []bool{false, true} {
		s := drainPaused(drainFixture(t))
		s, _ = stepDrain(s, DrainEvent{Kind: DrainNoticeResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
		s, _ = stepDrain(s, DrainEvent{Kind: DrainAcknowledged, Workspace: "pair:1", Identity: "binding-session"})
		if sealed {
			s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Idle: true}})
		}
		s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Idle: true, Submission: 1}})
		if s.Members["pair:1"].Ack || ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
			t.Fatal("submission inherited checkpoint", sealed)
		}
	}
}

func TestDrainBoundsAndBoardIsolation(t *testing.T) {
	s := drainPaused(drainFixture(t))
	b := ProjectDrainBoard(s, time.Unix(1, 0))
	b.Queue.Owners[0] = "other"
	b.Rows[0].Identity = "other"
	if s.Queue.Owners[0] != s.Owner || s.Members["pair:1"].Identity != "binding-session" {
		t.Fatal("mutable board alias")
	}
	members := make([]DrainMember, MaxDrainMembers+1)
	for i := range members {
		members[i] = DrainMember{Workspace: fmt.Sprint(i), Identity: "id"}
	}
	_, _, err := StartDrain(DrainStart{ID: s.ID, Deadline: s.Deadline, Members: members})
	if err == nil {
		t.Fatal("unbounded census")
	}
}

func FuzzDrainNoticeReason(f *testing.F) {
	f.Add("literal data")
	f.Add("\x1b[31m")
	f.Add("\"; rm -rf /; \"")
	f.Fuzz(func(t *testing.T, reason string) {
		id := "00000000-0000-4000-8000-000000000001"
		notice, err := DrainNotice(id, reason, false)
		if err != nil {
			return
		}
		if !strings.HasPrefix(notice, "Couch drain v1 "+id+": checkpoint and stop.") {
			t.Fatal("changed protocol")
		}
		if reason != "" && !strings.HasSuffix(notice, "\nReason: "+strconv.Quote(reason)) {
			t.Fatal("reason escaped data boundary")
		}
	})
}

func TestDrainUnknownEvidenceInvalidatesProvisionalAck(t *testing.T) {
	s := drainPaused(drainFixture(t))
	s, _ = stepDrain(s, DrainEvent{Kind: DrainNoticeResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainAcknowledged, Workspace: "pair:1", Identity: "binding-session"})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{}})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Idle: true}})
	if s.Members["pair:1"].Ack || ProjectDrainBoard(s, time.Unix(1, 0)).State == DrainDrained {
		t.Fatal("observation gap reused ack")
	}
}

func TestDrainRetryCancelReleasesOnlySameOwner(t *testing.T) {
	s := drainPaused(drainFixture(t))
	s, _ = stepDrain(s, DrainEvent{Kind: DrainCancel})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainRevoked, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainQueueObserved, Queue: DrainQueueObservation{Known: true, Complete: true, Paused: true, Owners: []string{s.Owner, "other"}}})
	_, fx := stepDrain(s, DrainEvent{Kind: DrainCancel})
	if len(fx) != 1 || fx[0].Kind != DrainResume || fx[0].Owner != s.Owner {
		t.Fatalf("retry cancel missing owner-scoped resume: %v", fx)
	}
}

func TestDrainPreparingDeadlineIsBounded(t *testing.T) {
	s := drainFixture(t)
	b := ProjectDrainBoard(s, time.Unix(1001, 0))
	if b.State != DrainStuck || !strings.Contains(b.Detail, "deadline") {
		t.Fatalf("unconfirmed pause never timed out: %+v", b)
	}
}

func TestDrainGlobalStuckRecoveryUsesInspectionAndKeepsRowRecovery(t *testing.T) {
	s := drainPaused(drainFixture(t))
	s, _ = stepDrain(s, DrainEvent{Kind: DrainNoticeResult, Workspace: "pair:1", Identity: "binding-session", Outcome: DrainDone})
	s, _ = stepDrain(s, DrainEvent{Kind: DrainMemberObserved, Workspace: "pair:1", Identity: "binding-session", Observation: DrainObservation{Known: true, Connected: true, Idle: true}})
	b := ProjectDrainBoard(s, time.Unix(1, 0))
	if b.State != DrainStuck || b.Recovery != "inspect status" || b.Rows[0].Recovery != "checkpoint and ack" {
		t.Fatalf("stuck recovery did not distinguish global inspection from row action: %+v", b)
	}
}
