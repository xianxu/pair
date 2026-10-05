package couchmessage

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// receiptState names one row of the transition table: a receipt in each
// status, or none held.
func receiptIn(status ReceiptStatus, at time.Time) OperationReceipt {
	if status == ReceiptNone {
		return OperationReceipt{}
	}
	r := OperationReceipt{ID: "id", Op: "resume", Target: "pair:1", Status: status, Admitted: at}
	if status.Terminal() {
		r.Finished = at
	}
	return r
}

// receiptEventCase names one column of the table.
type receiptEventCase struct {
	name  string
	event func(at time.Time) ReceiptEvent
}

func allReceiptEventCases() []receiptEventCase {
	return []receiptEventCase{
		{"admit-same", func(at time.Time) ReceiptEvent {
			return ReceiptEvent{Kind: ReceiptAdmit, ID: "id", Op: "resume", Target: "pair:1", At: at}
		}},
		{"admit-other", func(at time.Time) ReceiptEvent {
			return ReceiptEvent{Kind: ReceiptAdmit, ID: "id", Op: "reboot", Target: "pair:1", At: at}
		}},
		{"start", func(at time.Time) ReceiptEvent { return ReceiptEvent{Kind: ReceiptStart, ID: "id", At: at} }},
		{"finish-ok", func(at time.Time) ReceiptEvent {
			return ReceiptEvent{Kind: ReceiptFinish, ID: "id", Outcome: ReceiptOutcome{Status: ReceiptSucceeded, Tag: "t2"}, At: at}
		}},
		{"finish-refused", func(at time.Time) ReceiptEvent {
			return ReceiptEvent{Kind: ReceiptFinish, ID: "id", Outcome: ReceiptOutcome{Status: ReceiptRefused, Code: "not-offered"}, At: at}
		}},
		{"finish-failed", func(at time.Time) ReceiptEvent {
			return ReceiptEvent{Kind: ReceiptFinish, ID: "id", Outcome: ReceiptOutcome{Status: ReceiptFailed, Diagnostic: "attach-failed"}, At: at}
		}},
		{"expire", func(at time.Time) ReceiptEvent {
			return ReceiptEvent{Kind: ReceiptExpire, At: at.Add(ReceiptRetention)}
		}},
		{"status", func(at time.Time) ReceiptEvent { return ReceiptEvent{Kind: ReceiptStatusQuery, ID: "id", At: at} }},
	}
}

// TestApplyReceiptEventTable walks every (state, event) pair against the
// literal table of pair#367 Task 2.1. "err" leaves the receipt unchanged. A
// receipt not held has no op to differ from, so both admits create it.
func TestApplyReceiptEventTable(t *testing.T) {
	const (
		errCode      = "err"
		idConflict   = "id-conflict"
		unchanged    = "same"
		removed      = "removed"
		statusAnswer = "answer"
	)
	want := map[ReceiptStatus]map[string]string{
		ReceiptNone: {"admit-same": "queued", "admit-other": "queued", "start": errCode, "finish-ok": errCode, "finish-refused": errCode,
			"finish-failed": errCode, "expire": unchanged, "status": "unknown"},
		ReceiptQueued: {"admit-same": unchanged, "admit-other": idConflict, "start": "running", "finish-ok": "succeeded", "finish-refused": "refused",
			"finish-failed": "failed", "expire": errCode, "status": statusAnswer},
		ReceiptRunning: {"admit-same": unchanged, "admit-other": idConflict, "start": errCode, "finish-ok": "succeeded", "finish-refused": "refused",
			"finish-failed": "failed", "expire": errCode, "status": statusAnswer},
	}
	terminal := map[string]string{"admit-same": unchanged, "admit-other": idConflict, "start": errCode, "finish-ok": errCode, "finish-refused": errCode,
		"finish-failed": errCode, "expire": removed, "status": statusAnswer}
	at := time.Unix(1000, 0)
	pairs := 0
	for _, status := range AllReceiptStatuses() {
		row := want[status]
		if status.Terminal() {
			row = terminal
		}
		if row == nil {
			t.Fatalf("no table row for %q", status)
		}
		for _, c := range allReceiptEventCases() {
			pairs++
			expect, ok := row[c.name]
			if !ok {
				t.Fatalf("no table cell for %s × %s", status, c.name)
			}
			before := receiptIn(status, at)
			got, err := ApplyReceiptEvent(before, c.event(at))
			label := fmt.Sprintf("%s × %s", orNone(status), c.name)
			switch expect {
			case errCode:
				if err == nil || got != before {
					t.Errorf("%s: got %+v, %v; want an error and no change", label, got, err)
				}
			case idConflict:
				if !errors.Is(err, ErrReceiptIDConflict) || got != before {
					t.Errorf("%s: got %+v, %v; want id-conflict and no change", label, got, err)
				}
			case unchanged, statusAnswer:
				if err != nil || got != before {
					t.Errorf("%s: got %+v, %v; want the receipt unchanged", label, got, err)
				}
			case removed:
				if err != nil || got.Status != ReceiptNone {
					t.Errorf("%s: got %+v, %v; want removed", label, got, err)
				}
			case "unknown":
				if err != nil || got.Status != ReceiptNone || got.Answer() != ReceiptUnknown {
					t.Errorf("%s: got %+v, %v; want unknown", label, got, err)
				}
			default:
				if err != nil || string(got.Status) != expect {
					t.Errorf("%s: got %+v, %v; want %s", label, got, err, expect)
				}
			}
		}
	}
	if pairs != len(AllReceiptStatuses())*len(allReceiptEventCases()) {
		t.Fatalf("walked %d pairs", pairs)
	}
}

func orNone(s ReceiptStatus) string {
	if s == ReceiptNone {
		return "(none)"
	}
	return string(s)
}

// A terminal receipt is kept for the retention window, then removed.
func TestReceiptExpiryNeedsTheRetentionWindow(t *testing.T) {
	at := time.Unix(1000, 0)
	done := receiptIn(ReceiptSucceeded, at)
	if _, err := ApplyReceiptEvent(done, ReceiptEvent{Kind: ReceiptExpire, At: at.Add(ReceiptRetention - time.Second)}); err == nil {
		t.Fatal("expired before the retention window")
	}
	if got, err := ApplyReceiptEvent(done, ReceiptEvent{Kind: ReceiptExpire, At: at.Add(ReceiptRetention)}); err != nil || got.Status != ReceiptNone {
		t.Fatalf("not removed at the window: %+v %v", got, err)
	}
}

func TestReceiptSequences(t *testing.T) {
	at := time.Unix(1000, 0)
	var r OperationReceipt
	step := func(e ReceiptEvent) {
		t.Helper()
		next, err := ApplyReceiptEvent(r, e)
		if err != nil {
			t.Fatalf("%v on %+v: %v", e.Kind, r, err)
		}
		r = next
	}
	step(ReceiptEvent{Kind: ReceiptAdmit, ID: "id", Op: "reboot", Target: "pair:2", At: at})
	admitted := r
	step(ReceiptEvent{Kind: ReceiptAdmit, ID: "id", Op: "reboot", Target: "pair:2", At: at.Add(time.Second)})
	if r != admitted {
		t.Fatalf("duplicate admit changed the receipt: %+v", r)
	}
	step(ReceiptEvent{Kind: ReceiptStart, ID: "id", At: at.Add(2 * time.Second)})
	step(ReceiptEvent{Kind: ReceiptFinish, ID: "id", At: at.Add(3 * time.Second), Outcome: ReceiptOutcome{Status: ReceiptSucceeded, Tag: "t9", Archived: "t1", Warning: "w"}})
	if r.Status != ReceiptSucceeded || r.Tag != "t9" || r.Archived != "t1" || r.Warning != "w" || !r.Finished.Equal(at.Add(3*time.Second)) {
		t.Fatalf("finish = %+v", r)
	}
	step(ReceiptEvent{Kind: ReceiptStatusQuery, ID: "id"})
	if r.Answer() != ReceiptSucceeded {
		t.Fatalf("status = %s", r.Answer())
	}
	step(ReceiptEvent{Kind: ReceiptExpire, At: at.Add(3*time.Second + ReceiptRetention)})
	if r.Answer() != ReceiptUnknown {
		t.Fatalf("after expiry status = %s", r.Answer())
	}
	// Couch exit drops every receipt (memory only): the next status is unknown.
	r = OperationReceipt{}
	step(ReceiptEvent{Kind: ReceiptStatusQuery, ID: "id"})
	if r.Answer() != ReceiptUnknown {
		t.Fatalf("after couch exit status = %s", r.Answer())
	}
}

// A finish names a terminal outcome; anything else is refused.
func TestReceiptFinishNeedsATerminalOutcome(t *testing.T) {
	at := time.Unix(1000, 0)
	for _, status := range []ReceiptStatus{ReceiptNone, ReceiptQueued, ReceiptRunning, ReceiptUnknown} {
		if _, err := ApplyReceiptEvent(receiptIn(ReceiptRunning, at), ReceiptEvent{Kind: ReceiptFinish, ID: "id", Outcome: ReceiptOutcome{Status: status}, At: at}); err == nil {
			t.Errorf("finish with %q accepted", status)
		}
	}
	if _, err := ApplyReceiptEvent(receiptIn(ReceiptRunning, at), ReceiptEvent{Kind: ReceiptStart, ID: "other", At: at}); err == nil {
		t.Error("an event for another ID was applied")
	}
}
