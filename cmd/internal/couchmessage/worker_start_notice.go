package couchmessage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
)

const WorkerStartNoticeBody = "You restarted; reread your issue Log and continue where you left off; report to your hub."

type WorkerCueState string

const (
	WorkerCueNeeded    WorkerCueState = "needed"
	WorkerCueCommitted WorkerCueState = "committed"
	WorkerCueSubmitted WorkerCueState = "submitted"
	WorkerCueUncertain WorkerCueState = "uncertain"
)

// WorkerCueReport is a wrapper-owned launch witness. Nil means unsupported or
// unknown, never needed. Once input is committed, uncertainty cannot reopen it.
type WorkerCueReport struct {
	LaunchOrdinal uint64
	State         WorkerCueState
	ID            string `json:",omitempty"`
}

func (r WorkerCueReport) Validate() error {
	if r.LaunchOrdinal == 0 {
		return errors.New("worker cue needs a launch ordinal")
	}
	switch r.State {
	case WorkerCueNeeded:
		if r.ID != "" {
			return errors.New("needed worker cue has no delivery ID")
		}
	case WorkerCueCommitted, WorkerCueSubmitted, WorkerCueUncertain:
		if !validMessageID(r.ID) {
			return errors.New("worker cue needs a delivery ID")
		}
	default:
		return errors.New("invalid worker cue state")
	}
	return nil
}

// CopyWorkerCueReport gives each asynchronous owner an independent snapshot.
func CopyWorkerCueReport(r *WorkerCueReport) *WorkerCueReport {
	if r == nil {
		return nil
	}
	owned := *r
	return &owned
}

// WorkerStartNoticeID distinguishes native re-execs even when every Binding
// field (including PID and process birth) survives. It changes no actor keys.
func WorkerStartNoticeID(b Binding, ordinal uint64) string {
	if ordinal == 0 || b.Validate() != nil {
		return ""
	}
	raw, _ := json.Marshal(struct {
		Binding       Binding
		LaunchOrdinal uint64
	}{b, ordinal})
	return fmt.Sprintf("worker-start-%x", sha256.Sum256(raw))
}

// AdvanceWorkerStartNotice accepts only the canonical server-origin self cue.
// Self provenance stays exact when a worker's owning hub is replaced. The
// coordinator independently requires a verified owner before admission.
func AdvanceWorkerStartNotice(r WorkerCueReport, receipt Receipt) WorkerCueReport {
	if r.Validate() != nil {
		return r
	}
	m := receipt.Message
	if !isWorkerStartNotice(m) || ValidateWorkerStartNotice(m, r.LaunchOrdinal) != nil || (r.ID != "" && r.ID != m.ID) {
		return r
	}
	if r.State == WorkerCueSubmitted {
		return r
	}
	switch receipt.Status {
	case Submitted:
		r.State = WorkerCueSubmitted
	case Queued, Held, Delivering:
		if r.State != WorkerCueNeeded {
			return r
		}
		r.State = WorkerCueCommitted
	case Indeterminate, Cancelled, Expired, Refused:
		r.State = WorkerCueUncertain
	default:
		return r
	}
	r.ID = m.ID
	return r
}

// ValidateWorkerStartNotice fences canonical system self cues before native
// admission. The full Binding/socket survives same-PID exec, so its launch
// ordinal must also agree. Ordinary peer/system messages are unaffected.
func ValidateWorkerStartNotice(m Message, ordinal uint64) error {
	if !isWorkerStartNotice(m) {
		return nil
	}
	id := WorkerStartNoticeID(m.To, ordinal)
	if id == "" || m.ID != id {
		return errors.New("worker start cue does not match the current wrapper launch")
	}
	return nil
}

func isWorkerStartNotice(m Message) bool {
	return m.Origin == OriginSystem && m.From == m.To && m.Body == WorkerStartNoticeBody
}
