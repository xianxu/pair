package couchmessage

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
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

// OperationProgress describes the current stage; repeated reports do not renew Started.
type OperationProgress struct {
	Stage   string
	Started time.Time
}

func validOperationStage(stage string) bool {
	if stage == "" || len(stage) > 256 || !utf8.ValidString(stage) {
		return false
	}
	for _, ch := range stage {
		if unicode.IsControl(ch) {
			return false
		}
	}
	return true
}

// OperationReceipt is the typed outcome of one resume or reboot request.
type OperationReceipt struct {
	Progress *OperationProgress `json:",omitempty"`
	// FailureNonce correlates a terminal receipt with retained lifecycle evidence.
	FailureNonce         string                               `json:",omitempty"`
	IntentDigest         string                               `json:",omitempty"`
	Drive                *agentdrive.Result                   `json:",omitempty"`
	RecoveryConfirmation string                               `json:",omitempty"`
	Confirmation         *couchcore.RecoveryConfirmationOffer `json:",omitempty"`
	ToHub                string                               `json:",omitempty"`
	Persist              bool                                 `json:",omitempty"`
	ID                   string
	Op                   string
	Target               string
	Agent                string `json:",omitempty"`
	CandidateKey         string `json:",omitempty"`
	Status               ReceiptStatus
	// Code is a refusal's code (SlotOperationError's, e.g. not-offered).
	Code string `json:",omitempty"`
	// Detail is the refusal's or failure's factual text.
	Detail string `json:",omitempty"`
	// Diagnostic is a failure's ResumeDiagnostic code.
	Diagnostic string `json:",omitempty"`
	// Tag is the thread tag a success left running; Archived is the tag a
	// reboot retired; Warning is a success's advisory.
	Tag      string `json:",omitempty"`
	Archived string `json:",omitempty"`
	Warning  string `json:",omitempty"`
	// Path is the checkout a prepare admission resolved (pair#434).
	Path     string    `json:",omitempty"`
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
	ReceiptProgress    ReceiptEventKind = "progress"
	ReceiptFinish      ReceiptEventKind = "finish"
	ReceiptExpire      ReceiptEventKind = "expire"
	ReceiptStatusQuery ReceiptEventKind = "status"
)

// ReceiptOutcome is a finish event's terminal result.
type ReceiptOutcome struct {
	FailureNonce                                           string `json:",omitempty"`
	Drive                                                  *agentdrive.Result
	Confirmation                                           *couchcore.RecoveryConfirmationOffer
	Status                                                 ReceiptStatus
	Code, Detail, Diagnostic, Tag, Archived, Warning, Path string
}

// ReceiptEvent drives ApplyReceiptEvent.
type ReceiptEvent struct {
	Stage                               string
	IntentDigest                        string
	RecoveryConfirmation                string
	ToHub                               string
	Persist                             bool
	Kind                                ReceiptEventKind
	ID, Op, Target, Agent, CandidateKey string // admit names all three; start, finish and status name the ID
	Outcome                             ReceiptOutcome
	At                                  time.Time
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
			return OperationReceipt{CandidateKey: e.CandidateKey, RecoveryConfirmation: e.RecoveryConfirmation, IntentDigest: e.IntentDigest, ID: e.ID, Op: e.Op, Target: e.Target, Agent: e.Agent, ToHub: e.ToHub, Persist: e.Persist, Status: ReceiptQueued, Admitted: e.At}, nil
		}
		if e.RecoveryConfirmation != r.RecoveryConfirmation || e.CandidateKey != r.CandidateKey || e.IntentDigest != r.IntentDigest || e.Op != r.Op || e.Target != r.Target || e.Agent != r.Agent || e.ToHub != r.ToHub || e.Persist != r.Persist {
			return r, ErrReceiptIDConflict
		}
		return r, nil
	case ReceiptStart:
		if r.Status != ReceiptQueued {
			return refuse("start")
		}
		r.Status = ReceiptRunning
		return r, nil
	case ReceiptProgress:
		if r.Status != ReceiptRunning {
			return refuse("progress")
		}
		stage := strings.TrimSpace(e.Stage)
		if !validOperationStage(stage) {
			return refuse("invalid progress")
		}
		if r.Progress == nil || r.Progress.Stage != stage {
			r.Progress = &OperationProgress{Stage: stage, Started: e.At}
		}
		return r, nil
	case ReceiptFinish:
		if r.Status != ReceiptQueued && r.Status != ReceiptRunning {
			return refuse("finish")
		}
		if !e.Outcome.Status.Terminal() {
			return refuse("finish without a terminal outcome")
		}
		o := e.Outcome
		r.Drive = boundedDriveResult(o.Drive)
		if r.Drive != nil {
			o.Detail = r.Drive.Detail
		}
		r.Confirmation = o.Confirmation
		r.FailureNonce = o.FailureNonce
		r.Status, r.Code, r.Detail, r.Diagnostic, r.Tag, r.Archived, r.Warning, r.Path, r.Finished = o.Status, o.Code, o.Detail, o.Diagnostic, o.Tag, o.Archived, o.Warning, o.Path, e.At
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

// A receipt owns its result and retains at most 32 KiB of encoded drive data.
func boundedDriveResult(source *agentdrive.Result) *agentdrive.Result {
	if source == nil {
		return nil
	}
	result := *source
	clip := func(value string, limit int) string {
		value = strings.ToValidUTF8(value, "?")
		if len(value) <= limit {
			return value
		}
		value = value[:limit]
		for !utf8.ValidString(value) {
			value = value[:len(value)-1]
		}
		return value
	}
	result.Detail = clip(result.Detail, 1024)
	result.Model = clip(result.Model, 128)
	result.Effort = clip(result.Effort, 32)
	result.Scope = agentdrive.ModelScope(clip(string(result.Scope), 32))
	result.ID = clip(result.ID, 128)
	result.Tail = nil
	for i := len(source.Tail) - 1; i >= 0 && len(result.Tail) < 200; i-- {
		line := clip(source.Tail[i], agentdrive.MaxResultTailBytes/6)
		candidate := append([]string{line}, result.Tail...)
		previous := result.Tail
		result.Tail = candidate
		encoded, _ := json.Marshal(result)
		if len(encoded) > agentdrive.MaxResultTailBytes {
			result.Tail = previous
			break
		}
	}
	return &result
}
