package couchschedule

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

const summaryFutureJSON = `{ "nested" : [1, {"value":42}], "enabled": true }`

func reopenSummaryState(t *testing.T, state TaskState) TaskState {
	t.Helper()
	raw, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{state.Key.ID(): state}})
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := DecodeLedger(raw)
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Tasks[state.Key.ID()]
}

func seedSummaryUnknown(state *TaskState) {
	state.Last.Unknown = map[string]json.RawMessage{
		"future_summary": json.RawMessage(summaryFutureJSON),
		"collision":      json.RawMessage(`{"old":true}`),
	}
}

func assertSummaryUnknown(t *testing.T, state TaskState) {
	t.Helper()
	if state.Last == nil || string(state.Last.Unknown["future_summary"]) != summaryFutureJSON {
		t.Fatalf("same-occurrence rewrite lost nested summary metadata: %+v", state.Last)
	}
}

func TestRecoveredSummaryPreservesSameIdentityUnknown(t *testing.T) {
	for _, source := range []string{"manual", "scheduled"} {
		for _, stage := range []string{"reserved", "running", "uncertain", "notifying", "delivery", "passive"} {
			if source == "manual" && (stage == "notifying" || stage == "delivery") {
				continue
			}
			t.Run(source+"/"+stage, func(t *testing.T) {
				fixtureStage := stage
				if stage == "passive" {
					fixtureStage = "running"
				}
				state := phaseStateFixture(t, source, fixtureStage)
				if state.Last == nil {
					state.Last = summary(state.Active, state.Active.StartedAt, nil, "quiet", "none")
				}
				seedSummaryUnknown(&state)
				if stage == "passive" {
					state.Active = nil
				}
				state = reopenSummaryState(t, state)
				code := 10
				canonical := &RunSummary{ID: state.Last.ID, Revision: "canonical-revision", Source: source, DueAt: state.Last.DueAt, StartedAt: state.Last.StartedAt.Add(-time.Second), EndedAt: state.Last.EndedAt.Add(time.Second), Exit: &code, Outcome: "wake", Notice: "submitted", Unknown: map[string]json.RawMessage{"canonical_summary": json.RawMessage(`{"new":[2,3]}`), "collision": json.RawMessage(`{"new":true}`)}}
				event := Event{Kind: ResultRecovered, At: canonical.EndedAt, Recovered: canonical}
				if state.Active != nil {
					event.Attempt, event.Revision = state.Active.ID, state.Active.Revision
				}
				next, effects := Advance(state, event)
				if len(effects) != 0 || next.Active != nil || next.Last.Revision != canonical.Revision || next.Last.Outcome != canonical.Outcome || !next.Last.StartedAt.Equal(canonical.StartedAt) {
					t.Fatalf("canonical known facts not adopted: %+v %v", next, effects)
				}
				assertSummaryUnknown(t, next)
				if string(next.Last.Unknown["collision"]) != `{"new":true}` || string(next.Last.Unknown["canonical_summary"]) != `{"new":[2,3]}` {
					t.Fatal("new summary metadata did not win")
				}
				assertSummaryUnknown(t, reopenSummaryState(t, next))
				next.Last.Unknown["future_summary"][0] = '['
				next.Last.Unknown["canonical_summary"][0] = '['
				next.Last.Unknown["added"] = json.RawMessage(`1`)
				*next.Last.Exit = 2
				assertSummaryUnknown(t, state)
				if state.Last.Unknown["added"] != nil || string(canonical.Unknown["canonical_summary"]) != `{"new":[2,3]}` || canonical.Unknown["added"] != nil || *canonical.Exit != 10 {
					t.Fatal("recovery result aliases persisted input or event")
				}
			})
		}
	}
}

func TestSummaryRewritesPreserveSameIdentityUnknown(t *testing.T) {
	for _, tc := range []struct {
		name, stage         string
		kind                EventKind
		notice              string
		notifyOnly, expired bool
	}{
		{name: "completion", stage: "running", kind: TaskFinished},
		{name: "legacy recovery", stage: "uncertain", kind: ResultRecovered, notice: "submitted"},
		{name: "owner uncertain", stage: "running", kind: OwnerUnknown},
		{name: "refusal", stage: "reserved", kind: ExecutionRefused},
		{name: "reservation expired", stage: "reserved", kind: ReservationDurable, expired: true},
		{name: "notify reservation", stage: "reserved", kind: ReservationDurable, notifyOnly: true},
		{name: "notice intent", stage: "notifying", kind: NoticeIntentDurable},
		{name: "notice observed", stage: "delivery", kind: NoticeObserved, notice: "submitted"},
		{name: "notice owner uncertain", stage: "notifying", kind: OwnerUnknown},
		{name: "notice cancelled", stage: "notifying", kind: CancelRequested},
		{name: "delivery cancelled", stage: "delivery", kind: CancelRequested},
		{name: "catalogue cancelled", stage: "notifying", kind: CatalogChanged},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := phaseStateFixture(t, "scheduled", tc.stage)
			state.Spec.ScriptRun = !tc.notifyOnly
			if tc.kind == NoticeIntentDurable {
				state.Active.ReceiptID = "schedule-phase-run"
				state.Active.NoticeBody = "task result"
				state.Active.NoticeBinding = json.RawMessage(`{"nonce":"frozen"}`)
			}
			if state.Last == nil {
				state.Last = summary(state.Active, state.Active.StartedAt, nil, "quiet", "none")
			}
			seedSummaryUnknown(&state)
			code := 7
			state.Last.Exit = &code
			state = reopenSummaryState(t, state)
			event := Event{Kind: tc.kind, At: state.Active.StartedAt.Add(time.Second), Attempt: state.Active.ID, Revision: state.Active.Revision, Exit: 10, Notice: tc.notice, HubOnline: true, Outcome: "wake"}
			if tc.expired {
				event.At = state.Active.Deadline
			}
			next, _ := Advance(state, event)
			if reflect.DeepEqual(next, state) {
				t.Fatal("test did not exercise an accepted rewrite")
			}
			assertSummaryUnknown(t, next)
			assertSummaryUnknown(t, reopenSummaryState(t, next))
			next.Last.Unknown["future_summary"][0] = '['
			next.Last.Unknown["added"] = json.RawMessage(`1`)
			if next.Last.Exit != nil {
				*next.Last.Exit = 99
			}
			assertSummaryUnknown(t, state)
			if state.Last.Unknown["added"] != nil || *state.Last.Exit != 7 {
				t.Fatal("rewrite aliases persisted input")
			}
		})
	}
}

func TestSummaryDifferentIdentityDoesNotInheritUnknown(t *testing.T) {
	for _, kind := range []EventKind{TaskFinished, ResultRecovered, ClockObserved} {
		t.Run(string(kind), func(t *testing.T) {
			state := phaseStateFixture(t, "scheduled", "running")
			state.Last = summary(state.Active, state.Active.StartedAt, nil, "quiet", "none")
			state.Last.ID = "previous-occurrence"
			seedSummaryUnknown(&state)
			state = reopenSummaryState(t, state)
			event := Event{Kind: kind, At: state.Active.StartedAt.Add(time.Second), Attempt: state.Active.ID, Revision: state.Active.Revision, Exit: 0}
			if kind == ResultRecovered {
				event.Recovered = summary(state.Active, event.At, nil, "quiet", "none")
			}
			if kind == ClockObserved {
				state.Active = nil
				event.At = state.NextDue
				event.RunID = "next-occurrence"
				state.Allowed = false
			}
			next, _ := Advance(state, event)
			if next.Last == nil || next.Last.ID == state.Last.ID {
				t.Fatal("test did not replace occurrence")
			}
			if len(next.Last.Unknown) != 0 {
				t.Fatalf("different occurrence inherited metadata: %v", next.Last.Unknown)
			}
			if len(reopenSummaryState(t, next).Last.Unknown) != 0 {
				t.Fatal("inherited metadata persisted")
			}
		})
	}
}

func TestRecoveredNonterminalSummaryPreservesUnknown(t *testing.T) {
	for _, tc := range []struct{ source, outcome, notice, wantStage string }{
		{"manual", "unknown", "owner unavailable", "uncertain"},
		{"scheduled", "unknown", "owner unavailable", "uncertain"},
		{"scheduled", "wake", "pending", "notifying"},
	} {
		t.Run(tc.source+"/"+tc.outcome, func(t *testing.T) {
			state := phaseStateFixture(t, tc.source, "uncertain")
			seedSummaryUnknown(&state)
			state = reopenSummaryState(t, state)
			canonical := summary(state.Active, state.Last.EndedAt.Add(time.Second), nil, tc.outcome, tc.notice)
			next, effects := Advance(state, Event{Kind: ResultRecovered, At: canonical.EndedAt, Attempt: state.Active.ID, Revision: state.Active.Revision, Recovered: canonical})
			if len(effects) != 0 || next.Active == nil || next.Active.Stage != tc.wantStage || next.Last.Outcome != tc.outcome || next.Last.Notice != tc.notice {
				t.Fatalf("canonical authority not adopted: %+v %v", next, effects)
			}
			assertSummaryUnknown(t, next)
			assertSummaryUnknown(t, reopenSummaryState(t, next))
			next.Last.Unknown["future_summary"][0] = '['
			assertSummaryUnknown(t, state)
			if canonical.Unknown != nil {
				t.Fatal("summary metadata leaked into canonical event")
			}
		})
	}
}

func TestPassiveSummaryRewritePreservesSameIdentityUnknown(t *testing.T) {
	for _, kind := range []EventKind{ManualRequested, ClockObserved} {
		t.Run(string(kind), func(t *testing.T) {
			state, due := taskFixture(t, false)
			state.Allowed = false
			state, _ = Advance(state, Event{Kind: kind, At: due, RunID: "passive-run"})
			seedSummaryUnknown(&state)
			state = reopenSummaryState(t, state)
			next, effects := Advance(state, Event{Kind: kind, At: due.Add(time.Minute), RunID: state.Last.ID})
			if len(effects) != 0 || next.Last.EndedAt.Equal(state.Last.EndedAt) {
				t.Fatal("test did not rewrite passive summary")
			}
			assertSummaryUnknown(t, next)
			assertSummaryUnknown(t, reopenSummaryState(t, next))
		})
	}
}

func TestAdvanceAttemptMetadataDetached(t *testing.T) {
	state := phaseStateFixture(t, "scheduled", "delivery")
	state.Active.Unknown = map[string]json.RawMessage{"future": json.RawMessage(summaryFutureJSON)}
	state = reopenSummaryState(t, state)
	next, _ := Advance(state, Event{Kind: NoticeObserved, At: state.Active.StartedAt.Add(time.Second), Attempt: state.Active.ID, Revision: state.Active.Revision, Notice: "not-admitted", HubOnline: true})
	if next.Active == nil || next.Active.Stage != "notifying" {
		t.Fatal("notice retry did not retain attempt")
	}
	next.Active.Unknown["future"][0] = '['
	next.Active.Unknown["added"] = json.RawMessage(`1`)
	next.Active.NoticeBinding[0] = '['
	if string(state.Active.Unknown["future"]) != summaryFutureJSON || state.Active.Unknown["added"] != nil || string(state.Active.NoticeBinding) != `{"nonce":"frozen"}` {
		t.Fatal("advanced attempt aliases persisted metadata")
	}
}
