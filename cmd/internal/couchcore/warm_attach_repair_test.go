package couchcore

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
)

func assertWarmClientRepair(t *testing.T, env *testEnv, before ThreadRecord, actor ActorRecord) {
	t.Helper()
	after, err := env.Couch.Threads.GetThread(before.Address)
	if err != nil {
		t.Fatal(err)
	}
	expected := cloneThreadRecord(before)
	expected.Revision++
	expected.Incarnations[0].PID = actor.PID
	expected.Incarnations[0].Identity = actor.Identity
	expected.Incarnations[0].State = IncarnationLive
	if !reflect.DeepEqual(expected, after) {
		t.Fatalf("repair changed more than client incarnation: before=%+v after=%+v", before, after)
	}
	if actor.AttachmentNotice != "" {
		t.Fatalf("repaired helper reported borrowed: %s", actor.AttachmentNotice)
	}
}

func TestWarmAttachmentRepairsOnlyDeadManagedClient(t *testing.T) {
	env, _, before := liveSlotFixture(t)
	env.Proc.Kill(before.Incarnations[0].PID)
	env.Artifacts.SetPairSession(before.Address, "listed-slot-one", true)
	result, err := env.Couch.ResumeTarget(context.Background(), ResumeTarget{Path: before.StartingPath})
	if err != nil {
		t.Fatal(err)
	}
	start, ok := result.(StartResult)
	if !ok || start.Handle == nil {
		t.Fatalf("result %T", result)
	}
	assertWarmClientRepair(t, env, before, start.Record)
	if len(env.Proc.Signals[before.Incarnations[0].PID]) != 0 {
		t.Fatal("repair signalled old helper")
	}
}

func TestWarmAttachmentBorrowsWithClaimsOrUnprovenOldClient(t *testing.T) {
	for _, reason := range []string{"live", "unknown", "start claim"} {
		t.Run(reason, func(t *testing.T) {
			record := warmDetachedRecord(t)
			record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "old-client", State: IncarnationLive, LaunchProfile: record.LatestLaunchProfile}}
			owner := Dead
			if reason == "live" {
				owner = Live
			}
			if reason == "unknown" {
				owner = Unknown
			}
			if reason == "start claim" {
				record.Incarnations[0].State = IncarnationCreating
				record.Incarnations[0].Start = &ThreadStartClaim{Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor", LaunchProfile: record.LatestLaunchProfile}
				record.Incarnations[0].LaunchProfile = nil
			}
			env, before, session := readonlyWarmFixture(t, record, owner)
			actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
			if err != nil || h == nil {
				t.Fatalf("h=%v err=%v", h, err)
			}
			if !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
				t.Fatalf("missing borrowed reason: %+v", actor)
			}
			assertWarmRecordUnchanged(t, env, before)
		})
	}
}

func TestWarmAttachmentLeavesRecordWhenOldClientProofChangesAtEffect(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	before := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-0123456789abcdef")
	before.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "old-client", State: IncarnationLive}}
	var err error
	before, err = local.CreateThread(before)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetPairSession(before.Address, "listed-slot-one", true)
	// The preflight sees Dead; changing at the staged write proves the final
	// guard must run immediately before atomic publication of the replacement.
	env.Couch.Threads = local
	local.hooks.AfterPublicationWrite = func(string) error { env.Proc.Set(5151, "old-client"); return nil }
	actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, "listed-slot-one", ResumeOptions{})
	if err != nil || h == nil {
		t.Fatalf("h=%v err=%v", h, err)
	}
	if !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
		t.Fatalf("missing changed-proof notice: %s", actor.AttachmentNotice)
	}
	assertWarmRecordUnchanged(t, env, before)
}

func TestWarmAttachmentRepairPreservesProducingLaunchAndParkHistory(t *testing.T) {
	record := warmDetachedRecord(t)
	markActionableParked(&record, time.Unix(100, 0).UTC())
	record.VerifiedPark = nil
	record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationLive, StartedAt: time.Unix(110, 0).UTC(), LaunchProfile: record.LatestLaunchProfile}}
	record.SessionBinding = &couchidentity.SessionBinding{Legacy: true, Name: "pair-producing-launch", ScopeKey: record.Address.RepoScope, Tag: string(record.Address.Tag), StartNonce: "start-0123456789abcdef"}
	env, before, session := readonlyWarmFixture(t, record, Dead)
	actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
	if err != nil || h == nil {
		t.Fatalf("h=%v err=%v", h, err)
	}
	assertWarmClientRepair(t, env, before, actor)
	if env.Artifacts.BindingResolutions() != 0 {
		t.Fatal("repair re-derived native launch identity")
	}
}

func TestWarmAttachmentRepeatedClientRemainsBorrowedWhileFirstClientLives(t *testing.T) {
	env, _, before := liveSlotFixture(t)
	env.Proc.Kill(before.Incarnations[0].PID)
	env.Artifacts.SetPairSession(before.Address, "listed-slot-one", true)
	first, h, err := env.Couch.attachListedTerminal(context.Background(), before, "listed-slot-one", ResumeOptions{})
	if err != nil || h == nil {
		t.Fatal(err)
	}
	env.Proc.Set(first.PID, first.Identity)
	repaired, err := env.Couch.Threads.GetThread(before.Address)
	if err != nil {
		t.Fatal(err)
	}
	second, h2, err := env.Couch.attachListedTerminal(context.Background(), repaired, "listed-slot-one", ResumeOptions{})
	if err != nil || h2 == nil || !strings.Contains(second.AttachmentNotice, "still alive") {
		t.Fatalf("second=%+v h=%v err=%v", second, h2, err)
	}
	assertWarmRecordUnchanged(t, env, repaired)
}

func TestDeadClientRepairDoesNotClaimPublicationIntoMissingStore(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	before := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-0123456789abcdef")
	before.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationLive}}
	var err error
	before, err = local.CreateThread(before)
	if err != nil {
		t.Fatal(err)
	}
	h, err := env.Runner.Start(local.slot.WorktreeRoot, []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := ActorRecord{Thread: before.Address, PID: h.PID(), Identity: h.Identity(), Shape: StartWarmReattach}
	if err := os.Rename(local.root, local.root+"-moved"); err != nil {
		t.Fatal(err)
	}
	err = local.RepairDeadClientHelper(context.Background(), before, client, h, env.Proc)
	if err == nil {
		t.Fatal("missing store reported a client repair without publication")
	}
	if _, err := os.Stat(local.root); !os.IsNotExist(err) {
		t.Fatalf("repair manufactured missing store: %v", err)
	}
}

func TestWarmAttachmentBorrowsVerifiedParkWithoutChangingReceipt(t *testing.T) {
	record := warmDetachedRecord(t)
	markActionableParked(&record, time.Unix(100, 0).UTC())
	record.Incarnations = []ThreadIncarnation{{PID: 5151, Identity: "dead-client", State: IncarnationUnknown, LaunchProfile: record.LatestLaunchProfile}}
	env, before, session := readonlyWarmFixture(t, record, Dead)
	actor, h, err := env.Couch.attachListedTerminal(context.Background(), before, session, ResumeOptions{})
	if err != nil || h == nil || !strings.Contains(actor.AttachmentNotice, "BORROWED ATTACH") {
		t.Fatalf("actor=%+v h=%v err=%v", actor, h, err)
	}
	assertWarmRecordUnchanged(t, env, before)
}
