package couchmessage

import (
	"errors"
	"fmt"
	"time"
)

// ReceiptStatus is a slot-operation receipt's state (pair#367 M2). A receipt
// lives in the running Couch's memory only: queued → running →
// succeeded|refused|failed, then removed ReceiptRetention after it turned
// terminal, or when Couch exits.
type ReceiptStatus string

const (
	// ReceiptNone is no receipt held; a status query answers ReceiptUnknown.
	ReceiptNone      ReceiptStatus = ""
	ReceiptQueued    ReceiptStatus = "queued"
	ReceiptRunning   ReceiptStatus = "running"
	ReceiptSucceeded ReceiptStatus = "succeeded"
	ReceiptRefused   ReceiptStatus = "refused"
	ReceiptFailed    ReceiptStatus = "failed"
	// ReceiptUnknown is an answer, never a held state: the receipt expired,
	// was never admitted here, or Couch restarted. The outcome is uncertain.
	ReceiptUnknown ReceiptStatus = "unknown"
)

// AllReceiptStatuses lists every held state, none first.
func AllReceiptStatuses() []ReceiptStatus {
	return []ReceiptStatus{ReceiptNone, ReceiptQueued, ReceiptRunning, ReceiptSucceeded, ReceiptRefused, ReceiptFailed}
}

// Terminal reports a finished receipt.
func (s ReceiptStatus) Terminal() bool {
	return s == ReceiptSucceeded || s == ReceiptRefused || s == ReceiptFailed
}

// ReceiptRetention is how long a terminal receipt stays readable.
const ReceiptRetention = 5 * time.Minute

// OperationReceipt is the typed outcome of one resume or reboot request.
type OperationReceipt struct {
	ID     string
	Op     string
	Target string
	Status ReceiptStatus
	// Code is a refusal's code (SlotOperationError's, e.g. not-offered).
	Code string `json:",omitempty"`
	// Detail is the refusal's or failure's factual text.
	Detail string `json:",omitempty"`
	// Diagnostic is a failure's ResumeDiagnostic code.
	Diagnostic string `json:",omitempty"`
	// Tag is the thread tag a success left running; Archived is the tag a
	// reboot retired; Warning is a success's advisory.
	Tag      string    `json:",omitempty"`
	Archived string    `json:",omitempty"`
	Warning  string    `json:",omitempty"`
	Admitted time.Time `json:",omitzero"`
	Finished time.Time `json:",omitzero"`
}

// Answer is what a status query reports for this receipt.
func (r OperationReceipt) Answer() ReceiptStatus {
	if r.Status == ReceiptNone {
		return ReceiptUnknown
	}
	return r.Status
}

// ReceiptEventKind is one input of the receipt machine.
type ReceiptEventKind string

const (
	ReceiptAdmit       ReceiptEventKind = "admit"
	ReceiptStart       ReceiptEventKind = "start"
	ReceiptFinish      ReceiptEventKind = "finish"
	ReceiptExpire      ReceiptEventKind = "expire"
	ReceiptStatusQuery ReceiptEventKind = "status"
)

// ReceiptOutcome is a finish event's terminal result.
type ReceiptOutcome struct {
	Status                                           ReceiptStatus
	Code, Detail, Diagnostic, Tag, Archived, Warning string
}

// ReceiptEvent drives ApplyReceiptEvent.
type ReceiptEvent struct {
	Kind           ReceiptEventKind
	ID, Op, Target string // admit names all three; start, finish and status name the ID
	Outcome        ReceiptOutcome
	At             time.Time
}

// ErrReceiptIDConflict refuses an admission reusing an ID for another
// operation or target.
var ErrReceiptIDConflict = errors.New("receipt ID already names another operation")

// ApplyReceiptEvent is the closed transition table (pair#367 Task 2.1). Any
// pair not listed is an error and leaves the receipt unchanged. A duplicate
// admission (same ID, op and target) returns the receipt as it stands, so a
// repeat never makes a second effect.
func ApplyReceiptEvent(r OperationReceipt, e ReceiptEvent) (OperationReceipt, error) {
	refuse := func(why string) (OperationReceipt, error) {
		return r, fmt.Errorf("receipt %s: %s on %s", r.ID, why, orNoReceipt(r.Status))
	}
	if r.Status != ReceiptNone && e.Kind != ReceiptExpire && e.ID != r.ID {
		return refuse("event for receipt " + e.ID)
	}
	switch e.Kind {
	case ReceiptStatusQuery:
		return r, nil
	case ReceiptAdmit:
		if r.Status == ReceiptNone {
			return OperationReceipt{ID: e.ID, Op: e.Op, Target: e.Target, Status: ReceiptQueued, Admitted: e.At}, nil
		}
		if e.Op != r.Op || e.Target != r.Target {
			return r, ErrReceiptIDConflict
		}
		return r, nil
	case ReceiptStart:
		if r.Status != ReceiptQueued {
			return refuse("start")
		}
		r.Status = ReceiptRunning
		return r, nil
	case ReceiptFinish:
		if r.Status != ReceiptQueued && r.Status != ReceiptRunning {
			return refuse("finish")
		}
		if !e.Outcome.Status.Terminal() {
			return refuse("finish without a terminal outcome")
		}
		o := e.Outcome
		r.Status, r.Code, r.Detail, r.Diagnostic, r.Tag, r.Archived, r.Warning, r.Finished = o.Status, o.Code, o.Detail, o.Diagnostic, o.Tag, o.Archived, o.Warning, e.At
		return r, nil
	case ReceiptExpire:
		switch {
		case r.Status == ReceiptNone:
			return r, nil
		case !r.Status.Terminal():
			return refuse("expire before terminal")
		case e.At.Sub(r.Finished) < ReceiptRetention:
			return refuse("expire within retention")
		}
		return OperationReceipt{}, nil
	}
	return refuse("unknown event " + string(e.Kind))
}

func orNoReceipt(s ReceiptStatus) string {
	if s == ReceiptNone {
		return "no receipt"
	}
	return string(s)
}
