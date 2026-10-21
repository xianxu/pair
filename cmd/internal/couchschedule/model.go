package couchschedule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// TaskKey routes a task to its configured hub checkout, independently of slots.
type TaskKey struct {
	Hub  string `json:"hub"`
	Task string `json:"task"`
}

func (k TaskKey) ID() string {
	sum := sha256.Sum256([]byte(k.Hub + "\x00" + k.Task))
	return hex.EncodeToString(sum[:])
}
func (k TaskKey) HubID() string {
	sum := sha256.Sum256([]byte(k.Hub))
	return hex.EncodeToString(sum[:])
}

type Declaration struct {
	Key      TaskKey
	Revision string
	Spec     TaskSpec
	Allowed  bool
	Reason   string
}

type RunSummary struct {
	ID        string                     `json:"id"`
	Source    string                     `json:"source"`
	DueAt     time.Time                  `json:"due_at"`
	StartedAt time.Time                  `json:"started_at"`
	EndedAt   time.Time                  `json:"ended_at"`
	Exit      *int                       `json:"exit"`
	Outcome   string                     `json:"outcome"`
	Notice    string                     `json:"notice"`
	Unknown   map[string]json.RawMessage `json:"-"`
}

type Attempt struct {
	ID                    string                     `json:"id"`
	Revision              string                     `json:"revision"`
	Source                string                     `json:"source"`
	Stage                 string                     `json:"stage"`
	DueAt                 time.Time                  `json:"due_at"`
	StartedAt             time.Time                  `json:"started_at"`
	Deadline              time.Time                  `json:"deadline"`
	NoticeDeadline        time.Time                  `json:"notice_deadline"`
	ControllerPID         int                        `json:"controller_pid"`
	ControllerStart       string                     `json:"controller_start"`
	GroupPID              int                        `json:"group_pid"`
	ParentPID             int                        `json:"parent_pid"`
	ReceiptID             string                     `json:"receipt_id"`
	NoticeMayHaveHappened bool                       `json:"notice_may_have_happened"`
	Unknown               map[string]json.RawMessage `json:"-"`
}

// TaskState is the pure transition unit. A ledger composes tasks without
// copying or traversing all other tasks on each event.
type TaskState struct {
	Key      TaskKey                    `json:"key"`
	Revision string                     `json:"revision"`
	Spec     TaskSpec                   `json:"spec"`
	Allowed  bool                       `json:"allowed"`
	Refusal  string                     `json:"refusal"`
	NextDue  time.Time                  `json:"next_due"`
	Active   *Attempt                   `json:"active"`
	Last     *RunSummary                `json:"last"`
	Unknown  map[string]json.RawMessage `json:"-"`
}

type EventKind string

const (
	CatalogChanged      EventKind = "catalog-changed"
	ClockObserved       EventKind = "clock-observed"
	ManualRequested     EventKind = "manual-requested"
	ReservationDurable  EventKind = "reservation-durable"
	TaskFinished        EventKind = "task-finished"
	NoticeIntentDurable EventKind = "notice-intent-durable"
	NoticeObserved      EventKind = "notice-observed"
	OwnerUnknown        EventKind = "owner-unknown"
	CancelRequested     EventKind = "cancel-requested"
)

type Event struct {
	Kind        EventKind
	At          time.Time
	Declaration *Declaration
	Attempt     string
	Revision    string
	RunID       string
	HubOnline   bool
	Exit        int
	Outcome     string
	Notice      string
	Location    *time.Location
	// Next is an optional pre-parsed calendar function owned by the caller.
	// It must be deterministic; it avoids reparsing unchanged declarations.
	Next func(time.Time) time.Time
}

type EffectKind string

const (
	StartTask  EffectKind = "start-task"
	SendNotice EffectKind = "send-notice"
	StopTask   EffectKind = "stop-task"
)

type Effect struct {
	Kind    EffectKind
	Attempt string
}

// Advance consumes one observed event without IO or ambient time. Effects are
// emitted only by explicit durability acknowledgements, never reservation.
func Advance(state TaskState, event Event) (TaskState, []Effect) {
	if event.At.IsZero() {
		return state, nil
	}
	if state.Active != nil {
		copy := *state.Active
		state.Active = &copy
	}
	if state.Last != nil {
		copy := *state.Last
		state.Last = &copy
	}
	if event.Kind == CatalogChanged {
		d := event.Declaration
		if d == nil {
			state.Allowed = false
			state.Refusal = "task removed"
			return cancelActive(state)
		}
		changed := state.Revision != d.Revision || state.Key != d.Key
		state.Key = d.Key
		state.Revision = d.Revision
		state.Spec = d.Spec
		state.Allowed = d.Allowed
		state.Refusal = d.Reason
		if changed {
			state.NextDue = futureDue(state.Spec, event)
			if state.Active != nil {
				return cancelActive(state)
			}
		}
		return state, nil
	}
	if event.Kind == ClockObserved || event.Kind == ManualRequested {
		manual := event.Kind == ManualRequested
		if !ValidTaskName(event.RunID) || state.Revision == "" {
			return state, nil
		}
		due := state.NextDue
		if !manual {
			if due.IsZero() || event.At.Before(due) {
				return state, nil
			}
			state.NextDue = futureDue(state.Spec, event)
		}
		if state.Active != nil {
			return state, nil
		}
		source := "scheduled"
		if manual {
			source = "manual"
			due = time.Time{}
		}
		result := func(outcome, notice string) (TaskState, []Effect) {
			state.Last = &RunSummary{ID: event.RunID, Source: source, DueAt: due, StartedAt: event.At, EndedAt: event.At, Outcome: outcome, Notice: notice}
			return state, nil
		}
		if !state.Allowed {
			return result("refused", state.Refusal)
		}
		if !manual && event.At.Sub(due) >= time.Minute && state.Spec.Missed != "once" {
			return result("skipped", "missed slot")
		}
		if !manual && !state.Spec.ScriptRun && !event.HubOnline {
			return result("skipped", "hub offline, skipped")
		}
		state.Active = &Attempt{ID: event.RunID, Revision: state.Revision, Source: source, Stage: "reserved", DueAt: due, StartedAt: event.At, Deadline: event.At.Add(state.Spec.Timeout)}
		return state, nil
	}
	a := state.Active
	if a == nil || event.Attempt != a.ID || event.Revision != a.Revision {
		return state, nil
	}
	switch event.Kind {
	case ReservationDurable:
		if a.Stage != "reserved" {
			break
		}
		if !event.At.Before(a.Deadline) {
			state.Last = summary(a, event.At, nil, "timeout", "admission deadline elapsed")
			state.Active = nil
			break
		}
		if a.Source == "manual" || state.Spec.ScriptRun {
			a.Stage = "running"
			return state, []Effect{{Kind: StartTask, Attempt: a.ID}}
		}
		if !event.HubOnline {
			state.Last = summary(a, event.At, nil, "skipped", "hub offline, skipped")
			state.Active = nil
			break
		}
		a.Stage = "notifying"
		a.NoticeDeadline = a.Deadline
		a.NoticeMayHaveHappened = true
		state.Last = summary(a, event.At, nil, "completed", "pending")
	case TaskFinished:
		if a.Stage != "running" && a.Stage != "building" && a.Stage != "queued" && a.Stage != "stopping" {
			break
		}
		outcome := event.Outcome
		if outcome == "" {
			outcome = "failed"
			if event.Exit == 0 {
				outcome = "quiet"
			}
			if event.Exit == 10 {
				outcome = "wake"
			}
		}
		exit := event.Exit
		state.Last = summary(a, event.At, &exit, outcome, "")
		if a.Source == "manual" || outcome == "quiet" || a.Revision != state.Revision || a.Stage == "stopping" {
			state.Active = nil
			break
		}
		if !event.HubOnline {
			state.Last.Notice = "dropped: hub offline"
			state.Active = nil
			break
		}
		a.Stage = "notifying"
		a.NoticeDeadline = event.At.Add(time.Minute)
		a.NoticeMayHaveHappened = true
		state.Last.Notice = "pending"
	case NoticeIntentDurable:
		if a.Stage != "notifying" {
			break
		}
		if !event.HubOnline {
			state.Last.Notice = "dropped: hub offline"
			state.Active = nil
			break
		}
		if !event.At.Before(a.NoticeDeadline) {
			state.Last.Notice = "dropped: notice admission expired"
			state.Active = nil
			break
		}
		a.Stage = "delivery"
		return state, []Effect{{Kind: SendNotice, Attempt: a.ID}}
	case NoticeObserved:
		if a.Stage != "delivery" {
			break
		}
		switch event.Notice {
		case "submitted", "expired", "not-dispatched", "uncertain":
			state.Last.Notice = event.Notice
			state.Active = nil
		case "not-admitted":
			if event.HubOnline && event.At.Before(a.NoticeDeadline) {
				a.Stage = "notifying"
			} else {
				state.Last.Notice = "dropped: notice not admitted"
				state.Active = nil
			}
		}
	case OwnerUnknown:
		if a.Stage == "notifying" || a.Stage == "delivery" {
			state.Last.Notice = "uncertain"
			state.Active = nil
			break
		}
		a.Stage = "uncertain"
		state.Last = summary(a, event.At, nil, "unknown", "execution owner unavailable; no replay")
	case CancelRequested:
		return cancelActive(state)
	}
	return state, nil
}

func futureDue(spec TaskSpec, event Event) time.Time {
	if event.Next != nil {
		return event.Next(event.At)
	}
	now, location := event.At, event.Location
	if location == nil {
		location = time.UTC
	}
	calendar, err := NewCalendar(spec.Cron, location)
	if err != nil {
		return time.Time{}
	}
	return calendar.Next(now)
}

func summary(a *Attempt, ended time.Time, exit *int, outcome, notice string) *RunSummary {
	return &RunSummary{ID: a.ID, Source: a.Source, DueAt: a.DueAt, StartedAt: a.StartedAt, EndedAt: ended, Exit: exit, Outcome: outcome, Notice: notice}
}

func cancelActive(state TaskState) (TaskState, []Effect) {
	if state.Active == nil || state.Active.Stage == "uncertain" || state.Active.Stage == "stopping" {
		return state, nil
	}
	if state.Active.Stage == "reserved" || state.Active.Stage == "notifying" {
		state.Active = nil
		return state, nil
	}
	if state.Active.Stage == "delivery" {
		state.Last.Notice = "uncertain"
		state.Active = nil
		return state, nil
	}
	state.Active.Stage = "stopping"
	return state, []Effect{{Kind: StopTask, Attempt: state.Active.ID}}
}
