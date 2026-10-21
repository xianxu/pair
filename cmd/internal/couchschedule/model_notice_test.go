package couchschedule

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func freezeForTest(s TaskState, at time.Time) TaskState {
	next, _ := Advance(s, Event{Kind: NoticeFrozen, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision, ReceiptID: "schedule-" + s.Active.ID, NoticeBody: "task result", NoticeBinding: json.RawMessage(`{"slot":"pair:0","nonce":"frozen"}`)})
	return next
}
func TestNoticeRequiresFrozenAuthority(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "notifying")
	at := s.Active.StartedAt.Add(time.Second)
	next, effects := Advance(s, Event{Kind: NoticeIntentDurable, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision, HubOnline: true})
	if len(effects) != 0 || next.Active.Stage != "notifying" {
		t.Fatal("unfrozen notice sent")
	}
	frozen := freezeForTest(s, at)
	if frozen.Active.NoticeBody == "" || !s.Active.NoticeMayHaveHappened || len(s.Active.NoticeBinding) != 0 {
		t.Fatal("freeze missing or input mutated")
	}
	next, effects = Advance(frozen, Event{Kind: NoticeIntentDurable, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision, HubOnline: true})
	if len(effects) != 1 || next.Active.Stage != "delivery" || !next.Active.NoticeMayHaveHappened {
		t.Fatal("frozen durable notice not sent")
	}
	changed, _ := Advance(frozen, Event{Kind: NoticeFrozen, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision, ReceiptID: frozen.Active.ReceiptID, NoticeBody: "changed", NoticeBinding: frozen.Active.NoticeBinding})
	if !reflect.DeepEqual(changed, frozen) {
		t.Fatal("frozen payload changed")
	}
	lost, effects := Advance(next, Event{Kind: OwnerUnknown, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision})
	if lost.Active != nil || lost.Last.Notice != "uncertain" || len(effects) != 0 {
		t.Fatal("lost receipt replayed")
	}
}
func TestScheduledResultRecoveryDoesNotExecute(t *testing.T) {
	for _, stage := range []string{"reserved", "running", "uncertain"} {
		for _, notice := range []string{"pending", "submitted", "dropped: hub offline", "uncertain", ""} {
			s := phaseStateFixture(t, "scheduled", stage)
			at := s.Active.StartedAt.Add(time.Second)
			next, effects := Advance(s, Event{Kind: ResultRecovered, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision, Exit: 10, Outcome: "wake", Notice: notice})
			if len(effects) != 0 || next.Last == nil || next.Last.Outcome != "wake" {
				t.Fatalf("%s/%s did not recover", stage, notice)
			}
			if notice == "pending" {
				if next.Active == nil || next.Active.Stage != "notifying" || !next.Active.NoticeDeadline.Equal(at.Add(time.Minute)) {
					t.Fatal("pending recovery lost original result deadline")
				}
			} else if next.Active != nil {
				t.Fatal("terminal recovered record retained execution")
			}
		}
	}
}
func TestScheduledRefusalProducesPendingNotice(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "reserved")
	at := s.Active.StartedAt.Add(time.Second)
	next, effects := Advance(s, Event{Kind: ExecutionRefused, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision, Notice: "build refused"})
	if len(effects) != 0 || next.Active == nil || next.Active.Stage != "notifying" || next.Last.Outcome != "refused" || next.Last.Notice != "pending" {
		t.Fatal("scheduled refusal lost notice", next)
	}
}

func TestFrozenNoticeRejectsMalformedAndPartialAuthority(t *testing.T) {
	for _, tc := range []struct{ id, body, binding string }{
		{"bad/id", "ok", `{"nonce":"x"}`}, {"id", "", `{"nonce":"x"}`}, {"id", "bad\x1btext", `{"nonce":"x"}`},
		{"id", "ok", `null`}, {"id", "ok", `[]`}, {"id", "ok", `{}`}, {"id", "ok", `{"nonce":"x","nonce":"y"}`},
		{"id", strings.Repeat("x", 8193), `{"nonce":"x"}`},
		{"id", "ok", `{"nonce":"` + strings.Repeat("x", 4096) + `"}`},
	} {
		s := phaseStateFixture(t, "scheduled", "notifying")
		next, effects := Advance(s, Event{Kind: NoticeFrozen, At: s.Active.StartedAt.Add(time.Second), Attempt: s.Active.ID, Revision: s.Active.Revision, ReceiptID: tc.id, NoticeBody: tc.body, NoticeBinding: json.RawMessage(tc.binding)})
		if !reflect.DeepEqual(next, s) || len(effects) != 0 {
			t.Fatalf("accepted malformed notice %+v", tc)
		}
	}
}
func TestNoticeDroppedAndKnownNonAdmission(t *testing.T) {
	for _, notice := range []string{"dropped: target replaced", "submitted", "expired", "uncertain", "not-admitted"} {
		s := phaseStateFixture(t, "scheduled", "delivery")
		at := s.Active.StartedAt.Add(time.Second)
		next, effects := Advance(s, Event{Kind: NoticeObserved, At: at, Attempt: s.Active.ID, Revision: s.Active.Revision, HubOnline: true, Notice: notice})
		if len(effects) != 0 {
			t.Fatal("observation emitted send")
		}
		if notice == "not-admitted" {
			if next.Active == nil || next.Active.Stage != "notifying" || next.Active.NoticeBody != s.Active.NoticeBody || next.Active.ReceiptID != s.Active.ReceiptID {
				t.Fatal("retry changed frozen authority")
			}
		} else if next.Active != nil || next.Last.Notice != notice {
			t.Fatal("terminal observation not retained")
		}
	}
}
func TestFrozenNoticeCodecAndM1Compatibility(t *testing.T) {
	for _, stage := range []string{"reserved", "notifying", "delivery"} {
		s := phaseStateFixture(t, "scheduled", stage)
		raw := phaseWire(t, s)
		var top map[string]json.RawMessage
		json.Unmarshal(raw, &top)
		var tasks map[string]json.RawMessage
		json.Unmarshal(top["tasks"], &tasks)
		var task map[string]json.RawMessage
		json.Unmarshal(tasks[s.Key.ID()], &task)
		var active map[string]json.RawMessage
		json.Unmarshal(task["active"], &active)
		delete(active, "notice_body")
		delete(active, "notice_binding")
		task["active"], _ = json.Marshal(active)
		tasks[s.Key.ID()], _ = json.Marshal(task)
		top["tasks"], _ = json.Marshal(tasks)
		legacy, _ := json.Marshal(top)
		_, err := DecodeLedger(legacy)
		if (err != nil) != (stage == "delivery") {
			t.Fatalf("M1 %s decode: %v", stage, err)
		}
	}
	s := phaseStateFixture(t, "scheduled", "delivery")
	s.Active.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{ "v" : [1, 2] }`)}
	l := Ledger{Version: 1, Tasks: map[string]TaskState{s.Key.ID(): s}}
	raw, err := EncodeLedger(l)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeLedger(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := decoded.Tasks[s.Key.ID()]
	if got.Active.NoticeBody != s.Active.NoticeBody || !reflect.DeepEqual(got.Active.NoticeBinding, s.Active.NoticeBinding) || string(got.Active.Unknown["future"]) != string(s.Active.Unknown["future"]) {
		t.Fatal("frozen notice or additive metadata changed")
	}
	s.Active.NoticeBody = ""
	if _, err := EncodeLedger(Ledger{Version: 1, Tasks: map[string]TaskState{s.Key.ID(): s}}); err == nil {
		t.Fatal("encode accepted partial authority")
	}
	if _, err := DecodeLedger(phaseWire(t, s)); err == nil {
		t.Fatal("decode accepted partial authority")
	}
}

func TestFreezeCopiesCallerBindingAndIsIdempotent(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "notifying")
	event := Event{Kind: NoticeFrozen, At: s.Active.StartedAt.Add(time.Second), Attempt: s.Active.ID, Revision: s.Active.Revision, ReceiptID: "schedule-phase-run", NoticeBody: "ok", NoticeBinding: json.RawMessage(`{"nonce":"frozen"}`)}
	frozen, effects := Advance(s, event)
	identical, _ := Advance(frozen, event)
	if len(effects) != 0 || !reflect.DeepEqual(frozen, identical) {
		t.Fatal("freeze not idempotent")
	}
	event.NoticeBinding[2] = 'X'
	if string(frozen.Active.NoticeBinding) != `{"nonce":"frozen"}` {
		t.Fatal("caller can mutate frozen binding")
	}
}
func TestRecoveredWrongIdentityCannotReplaceExecution(t *testing.T) {
	s := phaseStateFixture(t, "scheduled", "running")
	for _, e := range []Event{
		{Kind: ResultRecovered, At: s.Active.StartedAt.Add(time.Second), Attempt: "different", Revision: s.Active.Revision, Outcome: "wake", Notice: "pending"},
		{Kind: ResultRecovered, At: s.Active.StartedAt.Add(time.Second), Attempt: s.Active.ID, Revision: "different", Outcome: "wake", Notice: "pending"},
	} {
		next, effects := Advance(s, e)
		if len(effects) != 0 || !reflect.DeepEqual(next, s) {
			t.Fatal("foreign result replaced execution")
		}
	}
}

func TestExpiredReservedExecutionProducesTimeoutPending(t *testing.T) {
	for _, outcome := range []string{"timeout", "arbitrary", ""} {
		s := phaseStateFixture(t, "scheduled", "reserved")
		next, effects := Advance(s, Event{Kind: ExecutionRefused, At: s.Active.Deadline.Add(time.Second), Attempt: s.Active.ID, Revision: s.Active.Revision, Outcome: outcome, Notice: "admission deadline elapsed"})
		want := "refused"
		if outcome == "timeout" {
			want = "timeout"
		}
		if len(effects) != 0 || next.Active == nil || next.Last.Outcome != want || next.Last.Notice != "pending" {
			t.Fatalf("outcome %q: %+v", outcome, next)
		}
	}
}

func TestNotifyOnlyAdmissionWindowBoundedByOneMinuteAndTaskDeadline(t *testing.T) {
	for _, timeout := range []time.Duration{10 * time.Second, time.Hour} {
		s, due := taskFixture(t, false)
		s.Spec.Timeout = timeout
		s, _ = Advance(s, Event{Kind: ClockObserved, At: due, RunID: "run-window", HubOnline: true})
		next, _ := Advance(s, Event{Kind: ReservationDurable, At: due, Attempt: s.Active.ID, Revision: s.Active.Revision, HubOnline: true})
		window := timeout
		if window > time.Minute {
			window = time.Minute
		}
		if next.Active == nil || !next.Active.NoticeDeadline.Equal(due.Add(window)) {
			t.Fatal("notice admission exceeds execution deadline or one minute")
		}
	}
}

func TestRecoverTerminalNoticeAfterHistoryCommitBeforeLedgerCommit(t *testing.T) {
	for _, stage := range []string{"notifying", "delivery"} {
		for _, notice := range []string{"submitted", "expired", "dropped: hub changed", "uncertain"} {
			s := phaseStateFixture(t, "scheduled", stage)
			if stage == "notifying" {
				s = freezeForTest(s, s.Active.StartedAt.Add(time.Second))
			}
			next, effects := Advance(s, Event{Kind: ResultRecovered, At: s.Last.EndedAt, Attempt: s.Active.ID, Revision: s.Active.Revision, Exit: 10, Outcome: "wake", Notice: notice})
			if len(effects) != 0 || next.Active != nil || next.Last == nil || next.Last.Notice != notice || next.Last.Outcome != "wake" || !next.Last.DueAt.Equal(s.Active.DueAt) || !next.Last.StartedAt.Equal(s.Active.StartedAt) || !next.Last.EndedAt.Equal(s.Last.EndedAt) {
				t.Fatalf("%s/%s failed to adopt saved terminal history: %+v", stage, notice, next)
			}
		}
	}
}
func TestRecoverPendingNoticeDoesNotResetFrozenAuthority(t *testing.T) {
	for _, stage := range []string{"notifying", "delivery"} {
		s := phaseStateFixture(t, "scheduled", stage)
		if stage == "notifying" {
			s = freezeForTest(s, s.Active.StartedAt.Add(time.Second))
		}
		next, effects := Advance(s, Event{Kind: ResultRecovered, At: s.Last.EndedAt, Attempt: s.Active.ID, Revision: s.Active.Revision, Exit: 10, Outcome: "wake", Notice: "pending"})
		if len(effects) != 0 || !reflect.DeepEqual(next, s) {
			t.Fatal("pending recovery reset frozen authority or authorized replay")
		}
	}
}
