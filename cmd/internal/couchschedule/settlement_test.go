package couchschedule

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestSettlementCanonicalTerminalWinsBeforeCatalogueEdit(t *testing.T) {
	for _, edit := range []string{"removed", "revision", "owner-policy"} {
		t.Run(edit, func(t *testing.T) {
			old := phaseStateFixture(t, "scheduled", "delivery")
			old.Last.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{"a": 1}`)}
			canonical := *cloneRunSummary(old.Last)
			canonical.Notice = "submitted"
			at := canonical.EndedAt.Add(time.Second)
			event := Event{Kind: CatalogChanged, At: at}
			if edit != "removed" {
				event.Declaration = &Declaration{Key: old.Key, Revision: old.Revision, Spec: old.Spec, Allowed: old.Allowed}
				if edit == "revision" {
					event.Declaration.Revision += "-new"
				} else {
					event.Declaration.Allowed = false
					event.Declaration.Reason = "owner unavailable"
				}
			}
			// A failed ledger commit leaves the original authority intact. Each retry
			// must independently choose the already-committed terminal history.
			for retry := 0; retry < 3; retry++ {
				got := PlanSettlement(old, &canonical, event)
				if got.Busy || got.State.Active != nil || got.State.Last.Notice != "submitted" || len(got.Effects) != 0 || got.Next != SettlementPersist {
					t.Fatalf("retry %d: terminal history lost to edit: %+v", retry, got)
				}
				if !reflect.DeepEqual(got.State.Last, &canonical) {
					t.Fatalf("canonical facts changed: %+v", got.State.Last)
				}
				if old.Active == nil || old.Last.Notice != "pending" {
					t.Fatal("failed commit mutated prior authority")
				}
			}
		})
	}
}

func TestSettlementCatalogueWaitsForOccurrence(t *testing.T) {
	for _, stage := range []string{"reserved", "running", "stopping", "notifying", "delivery", "uncertain"} {
		t.Run(stage, func(t *testing.T) {
			old := phaseStateFixture(t, "scheduled", stage)
			got := PlanSettlement(old, nil, Event{Kind: CatalogChanged, At: old.Active.StartedAt.Add(time.Second)})
			if !got.Busy || !reflect.DeepEqual(got.State, old) || len(got.Effects) != 0 {
				t.Fatalf("catalogue cancelled retained occurrence: %+v", got)
			}
			d := Declaration{Key: old.Key, Revision: old.Revision, Spec: old.Spec, Allowed: old.Allowed, Reason: old.Refusal}
			same := PlanSettlement(old, nil, Event{Kind: CatalogChanged, At: old.Active.StartedAt.Add(time.Second), Declaration: &d})
			if same.Busy || !reflect.DeepEqual(same.State, old) {
				t.Fatalf("unchanged discovery was busy/mutated: %+v", same)
			}
		})
	}
}

func TestSettlementPendingCanonicalDoesNotAdmitEdit(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "delivery")
	got := PlanSettlement(old, old.Last, Event{Kind: CatalogChanged, At: old.Last.EndedAt.Add(time.Second)})
	if !got.Busy || got.State.Active == nil || got.State.Active.Stage != "delivery" || got.State.Last.Notice != "pending" || got.Kind != SettlementAwaitingReceipt {
		t.Fatalf("pending result allowed edit: %+v", got)
	}
}

func TestSettlementEvidenceAndDuplicateFacts(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "running")
	at := old.Active.StartedAt.Add(time.Second)
	stale := PlanSettlement(old, nil, Event{Kind: TaskFinished, At: at, Attempt: "earlier", Revision: old.Revision, Exit: 0})
	if !reflect.DeepEqual(stale.State, old) || len(stale.Effects) != 0 {
		t.Fatalf("stale result advanced: %+v", stale)
	}
	canonical := summary(old.Active, at, nil, "quiet", "")
	settled := PlanSettlement(old, canonical, Event{Kind: TaskFinished, At: at, Attempt: old.Active.ID, Revision: old.Active.Revision, Exit: 10, HubOnline: true})
	if settled.State.Active != nil || settled.State.Last.Outcome != "quiet" || len(settled.Effects) != 0 {
		t.Fatalf("late result overwrote canonical: %+v", settled)
	}
	duplicate := PlanSettlement(settled.State, canonical, Event{Kind: TaskFinished, At: at, Attempt: old.Active.ID, Revision: old.Active.Revision, Exit: 10, HubOnline: true})
	if !reflect.DeepEqual(duplicate.State, settled.State) || len(duplicate.Effects) != 0 {
		t.Fatalf("duplicate changed settled result: %+v", duplicate)
	}
	invalid := *canonical
	invalid.Revision = "other"
	refused := PlanSettlement(old, &invalid, Event{At: at})
	if refused.Kind != SettlementQuarantined || !reflect.DeepEqual(refused.State, old) || len(refused.Effects) != 0 {
		t.Fatalf("foreign evidence accepted: %+v", refused)
	}
}

func TestSettlementUnknownAndMissingHistoryDoNotReplay(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "running")
	at := old.Active.StartedAt.Add(time.Second)
	got := PlanSettlement(old, nil, Event{Kind: OwnerUnknown, At: at, Attempt: old.Active.ID, Revision: old.Active.Revision})
	if got.Kind != SettlementUnknown || got.State.Active == nil || got.State.Active.Stage != "uncertain" || got.State.Last.Outcome != "unknown" || len(got.Effects) != 0 {
		t.Fatalf("missing history replayed: %+v", got)
	}
	retry := PlanSettlement(got.State, nil, Event{Kind: ReservationDurable, At: at, Attempt: old.Active.ID, Revision: old.Active.Revision})
	if retry.Kind != SettlementUnknown || len(retry.Effects) != 0 {
		t.Fatalf("unknown replayed: %+v", retry)
	}
}

func TestSettlementClonesRetainedAndCanonicalFacts(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "delivery")
	old.Unknown = map[string]json.RawMessage{"task-extra": json.RawMessage(`1`)}
	old.Active.Unknown = map[string]json.RawMessage{"attempt-extra": json.RawMessage(`2`)}
	old.Last.Unknown = map[string]json.RawMessage{"summary-extra": json.RawMessage(`3`)}
	canonical := cloneRunSummary(old.Last)
	canonical.Notice = "submitted"
	got := PlanSettlement(old, canonical, Event{At: old.Last.EndedAt.Add(time.Second)})
	got.State.Last.Unknown["summary-extra"][0] = '9'
	got.State.Unknown["task-extra"][0] = '9'
	retained := PlanSettlement(old, nil, Event{At: old.Last.EndedAt.Add(time.Second)})
	retained.State.Active.Unknown["attempt-extra"][0] = '9'
	retained.State.Active.NoticeBinding[0] = ' '
	if string(old.Unknown["task-extra"]) != "1" || string(old.Active.Unknown["attempt-extra"]) != "2" || string(old.Last.Unknown["summary-extra"]) != "3" || string(canonical.Unknown["summary-extra"]) != "3" || old.Active.NoticeBinding[0] != '{' {
		t.Fatal("settlement aliases caller authority/evidence")
	}
}

func TestSettlementOrdersDerivedFactsBeforeLedgerCommit(t *testing.T) {
	running := phaseStateFixture(t, "scheduled", "running")
	at := running.Active.StartedAt.Add(time.Second)
	completed := PlanSettlement(running, nil, Event{Kind: TaskFinished, At: at, Attempt: running.Active.ID, Revision: running.Active.Revision, Exit: 10, HubOnline: true})
	if completed.Kind != SettlementSettling || completed.Next != SettlementPublishHistory || completed.State.Last.Notice != "pending" || len(completed.Effects) != 0 {
		t.Fatalf("completion did not require history publication: %+v", completed)
	}
	delivery := phaseStateFixture(t, "scheduled", "delivery")
	terminal := PlanSettlement(delivery, nil, Event{Kind: NoticeObserved, At: at, Attempt: delivery.Active.ID, Revision: delivery.Active.Revision, Notice: "submitted"})
	if terminal.Kind != SettlementSettling || terminal.Next != SettlementFinalizeHistory || terminal.State.Active != nil || terminal.State.Last.Notice != "submitted" {
		t.Fatalf("receipt did not require monotonic history finalization: %+v", terminal)
	}
	reserved := phaseStateFixture(t, "scheduled", "reserved")
	start := PlanSettlement(reserved, nil, Event{Kind: ReservationDurable, At: reserved.Active.StartedAt, Attempt: reserved.Active.ID, Revision: reserved.Active.Revision})
	if start.Next != SettlementPersist || len(start.Effects) != 1 || start.Effects[0].Kind != StartTask {
		t.Fatalf("script release not fenced by ledger commit: %+v", start)
	}
}

func TestSettlementRejectsRecoveryWithoutCanonicalDurableEvidence(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "running")
	result := summary(old.Active, old.Active.StartedAt.Add(time.Second), nil, "quiet", "")
	got := PlanSettlement(old, nil, NewRecoveryEvent(result.EndedAt, old.Active.ID, old.Active.Revision, *result))
	if got.Kind != SettlementQuarantined || !reflect.DeepEqual(got.State, old) || len(got.Effects) != 0 {
		t.Fatalf("raw recovery bypassed durable-evidence argument: %+v", got)
	}
}

func TestSettlementUnknownCanonicalRemainsRetained(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "running")
	canonical := summary(old.Active, old.Active.StartedAt.Add(time.Second), nil, "unknown", "execution owner unavailable; no replay")
	got := PlanSettlement(old, canonical, Event{Kind: CatalogChanged, At: canonical.EndedAt})
	if got.Kind != SettlementUnknown || !got.Busy || got.State.Active == nil || got.State.Active.Stage != "uncertain" || len(got.Effects) != 0 {
		t.Fatalf("unknown canonical lost retained authority: %+v", got)
	}
}

func TestSettlementRecoveryCannotOverrideDurableCanonicalArgument(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "delivery")
	durable := cloneRunSummary(old.Last)
	conflicting := cloneRunSummary(old.Last)
	conflicting.Notice = "expired"
	got := PlanSettlement(old, durable, NewRecoveryEvent(old.Last.EndedAt.Add(time.Second), old.Active.ID, old.Active.Revision, *conflicting))
	if got.State.Active == nil || got.State.Last.Notice != "pending" || got.Next == SettlementFinalizeHistory {
		t.Fatalf("event recovery overrode durable evidence: %+v", got)
	}
}

func TestSettlementPriorCanonicalDoesNotConfirmNewTerminalFact(t *testing.T) {
	old := phaseStateFixture(t, "scheduled", "delivery")
	got := PlanSettlement(old, old.Last, Event{Kind: NoticeObserved, At: old.Last.EndedAt.Add(time.Second), Attempt: old.Active.ID, Revision: old.Active.Revision, Notice: "submitted"})
	if got.Next != SettlementFinalizeHistory || got.Kind != SettlementSettling {
		t.Fatalf("pending canonical authorized unfinalized terminal ledger: %+v", got)
	}
}
