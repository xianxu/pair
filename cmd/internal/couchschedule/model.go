package couchschedule

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
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
	ManualIntentDigest string                     `json:"manual_intent_digest,omitempty"`
	Revision           string                     `json:"revision"`
	ID                 string                     `json:"id"`
	Source             string                     `json:"source"`
	DueAt              time.Time                  `json:"due_at"`
	StartedAt          time.Time                  `json:"started_at"`
	EndedAt            time.Time                  `json:"ended_at"`
	Exit               *int                       `json:"exit"`
	Outcome            string                     `json:"outcome"`
	Notice             string                     `json:"notice"`
	Unknown            map[string]json.RawMessage `json:"-"`
}

type Attempt struct {
	ManualIntentDigest string    `json:"manual_intent_digest,omitempty"`
	ID                 string    `json:"id"`
	Revision           string    `json:"revision"`
	Source             string    `json:"source"`
	Stage              string    `json:"stage"`
	DueAt              time.Time `json:"due_at"`
	StartedAt          time.Time `json:"started_at"`
	Deadline           time.Time `json:"deadline"`
	NoticeDeadline     time.Time `json:"notice_deadline"`
	ControllerPID      int       `json:"controller_pid"`
	ControllerStart    string    `json:"controller_start"`
	GroupPID           int       `json:"group_pid"`
	ParentPID          int       `json:"parent_pid"`
	// Frozen before broker admission; retries reuse the same identity/body/binding.
	NoticeBody            string                     `json:"notice_body"`
	NoticeBinding         json.RawMessage            `json:"notice_binding"`
	ReceiptID             string                     `json:"receipt_id"`
	NoticeMayHaveHappened bool                       `json:"notice_may_have_happened"`
	Unknown               map[string]json.RawMessage `json:"-"`
}

// TaskState is the pure transition unit. A ledger composes tasks without
// copying or traversing all other tasks on each event.
type TaskState struct {
	Key       TaskKey                    `json:"key"`
	Revision  string                     `json:"revision"`
	Spec      TaskSpec                   `json:"spec"`
	Allowed   bool                       `json:"allowed"`
	Refusal   string                     `json:"refusal"`
	NextDue   time.Time                  `json:"next_due"`
	RemovedAt time.Time                  `json:"removed_at"`
	Active    *Attempt                   `json:"active"`
	Last      *RunSummary                `json:"last"`
	Unknown   map[string]json.RawMessage `json:"-"`
}

type EventKind string

const (
	CatalogChanged      EventKind = "catalog-changed"
	ClockObserved       EventKind = "clock-observed"
	ManualRequested     EventKind = "manual-requested"
	ReservationDurable  EventKind = "reservation-durable"
	TaskFinished        EventKind = "task-finished"
	NoticeFrozen        EventKind = "notice-frozen"
	NoticeIntentDurable EventKind = "notice-intent-durable"
	NoticeObserved      EventKind = "notice-observed"
	OwnerUnknown        EventKind = "owner-unknown"
	CancelRequested     EventKind = "cancel-requested"
	ExecutionRefused    EventKind = "execution-refused"
	ResultRecovered     EventKind = "result-recovered"
	OwnerRecorded       EventKind = "owner-recorded"
)

type Event struct {
	ManualIntentDigest string
	Kind               EventKind
	At                 time.Time
	Declaration        *Declaration
	Attempt            string
	Revision           string
	RunID              string
	HubOnline          bool
	Exit               int
	Outcome            string
	Notice             string
	ReceiptID          string
	NoticeBody         string
	NoticeBinding      json.RawMessage
	Location           *time.Location
	Recovered          *RunSummary
	// Next is an optional pre-parsed calendar function owned by the caller.
	// It must be deterministic; it avoids reparsing unchanged declarations.
	Next  func(time.Time) time.Time
	Owner *OwnerIdentity
}

// NewRecoveryEvent requires an observation time at every producer. Recovery
// uses the same timestamped transition contract as all other model events.
func NewRecoveryEvent(at time.Time, attempt, revision string, canonical RunSummary) Event {
	return Event{Kind: ResultRecovered, At: at, Attempt: attempt, Revision: revision, Recovered: cloneRunSummary(&canonical)}
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
	if validateTaskPhase(state) != nil {
		return state, nil
	}
	next, effects := advance(state, event)
	if validateTaskPhase(next) != nil {
		return state, nil
	}
	// Known facts may be replaced from canonical history, but additive summary
	// metadata belongs to the persisted occurrence. Keep it on every same-ID
	// rewrite; metadata from another schema (such as a run record) stays there.
	if state.Last != nil && next.Last != nil && state.Last != next.Last && state.Last.ID == next.Last.ID {
		if state.Last.ManualIntentDigest != "" {
			if next.Last.ManualIntentDigest != "" && next.Last.ManualIntentDigest != state.Last.ManualIntentDigest {
				return state, nil
			}
			next.Last.ManualIntentDigest = state.Last.ManualIntentDigest
		}
		next.Last.Unknown = strictjson.MergeRaw(state.Last.Unknown, next.Last.Unknown)
	}
	return next, effects
}

func advance(state TaskState, event Event) (TaskState, []Effect) {
	if event.At.IsZero() {
		return state, nil
	}
	if state.Active != nil {
		state.Active = cloneAttempt(state.Active)
	}
	if state.Last != nil {
		state.Last = cloneRunSummary(state.Last)
	}
	if event.Kind == CatalogChanged {
		d := event.Declaration
		if d == nil {
			state.Allowed = false
			state.Refusal = "task removed"
			state.NextDue = time.Time{}
			if state.RemovedAt.IsZero() {
				state.RemovedAt = event.At
			}
			return cancelActive(state)
		}
		changed := state.Revision != d.Revision || state.Key != d.Key || !state.RemovedAt.IsZero()
		state.RemovedAt = time.Time{}
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
		digest := ""
		if manual {
			digest = event.ManualIntentDigest
			source = "manual"
			due = time.Time{}
		}
		result := func(outcome, notice string) (TaskState, []Effect) {
			state.Last = &RunSummary{ManualIntentDigest: digest, Revision: state.Revision, ID: event.RunID, Source: source, DueAt: due, StartedAt: event.At, EndedAt: event.At, Outcome: outcome, Notice: notice}
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
		state.Active = &Attempt{ManualIntentDigest: digest, ID: event.RunID, Revision: state.Revision, Source: source, Stage: "reserved", DueAt: due, StartedAt: event.At, Deadline: event.At.Add(state.Spec.Timeout)}
		return state, nil
	}
	// Passive outcomes have no Active tuple, but their persisted occurrence ID
	// still owns canonical terminal history. Pending/unknown cannot manufacture
	// missing authority through this branch.
	if event.Kind == ResultRecovered && state.Active == nil && state.Last != nil && event.Recovered != nil {
		canonical := event.Recovered
		if canonical.ID == state.Last.ID && canonical.Notice != "pending" && canonical.Outcome != "unknown" && validRecoveredSummary(*canonical) {
			state.Last = cloneRunSummary(canonical)
		}
		return state, nil
	}
	a := state.Active
	if a == nil || event.Attempt != a.ID || event.Revision != a.Revision {
		return state, nil
	}
	switch event.Kind {
	case OwnerRecorded:
		owner := event.Owner
		if a.Stage != "reserved" || owner == nil || owner.PID <= 0 || owner.Start == "" || (a.ControllerPID != 0 && (a.ControllerPID != owner.PID || a.ControllerStart != owner.Start)) {
			break
		}
		if a.Source == "scheduled" && !state.Spec.ScriptRun {
			break
		}
		a.ControllerPID = owner.PID
		a.ControllerStart = owner.Start
	case ResultRecovered:
		if event.Recovered != nil {
			canonical := event.Recovered
			if canonical.ID != a.ID || !validRecoveredSummary(*canonical) || (a.ManualIntentDigest != "" && canonical.ManualIntentDigest != a.ManualIntentDigest) {
				break
			}
			state.Last = cloneRunSummary(canonical)
			if canonical.Outcome == "unknown" || canonical.Notice == "pending" {
				a.ManualIntentDigest = canonical.ManualIntentDigest
				a.Revision, a.Source = canonical.Revision, canonical.Source
				a.DueAt, a.StartedAt = canonical.DueAt, canonical.StartedAt
				if canonical.Outcome == "unknown" {
					a.Stage = "uncertain"
					a.NoticeDeadline = time.Time{}
					a.NoticeMayHaveHappened = false
					a.ReceiptID, a.NoticeBody, a.NoticeBinding = "", "", nil
				} else {
					if a.Stage != "delivery" {
						a.Stage = "notifying"
					}
					deadline := canonical.EndedAt.Add(time.Minute)
					if !state.Spec.ScriptRun {
						if a.Deadline.Before(deadline) {
							deadline = a.Deadline
						}
						if !a.NoticeDeadline.IsZero() && a.NoticeDeadline.Before(deadline) {
							deadline = a.NoticeDeadline
						}
					}
					a.NoticeDeadline = deadline
					a.NoticeMayHaveHappened = true
				}
			} else {
				state.Active = nil
			}
			break
		}
		noticePhase := a.Source == "scheduled" && (a.Stage == "notifying" || a.Stage == "delivery")
		// Pending history does not erase a previously frozen payload, admission
		// deadline or receipt observation authority. Terminal history is truth
		// after a crash between finalizing it and committing the ledger.
		if noticePhase && event.Notice == "pending" {
			break
		}
		if !noticePhase && a.Stage != "reserved" && a.Stage != "running" && a.Stage != "stopping" && a.Stage != "uncertain" {
			break
		}
		var exit *int
		if event.Exit >= 0 {
			code := event.Exit
			exit = &code
		}
		state.Last = summary(a, event.At, exit, event.Outcome, event.Notice)
		if a.Source == "scheduled" && event.Notice == "pending" {
			a.Stage = "notifying"
			a.NoticeDeadline = event.At.Add(time.Minute)
			a.NoticeMayHaveHappened = true
		} else {
			state.Active = nil
		}
	case ExecutionRefused:
		// A positively known pre-authorization refusal or expired reservation
		// records an execution result without authorizing another execution.
		if a.Stage != "reserved" {
			break
		}
		outcome := "refused"
		if event.Outcome == "timeout" {
			outcome = "timeout"
		}
		state.Last = summary(a, event.At, nil, outcome, event.Notice)
		if a.Source == "scheduled" && state.Spec.ScriptRun && a.Revision == state.Revision {
			a.Stage = "notifying"
			a.NoticeDeadline = event.At.Add(time.Minute)
			a.NoticeMayHaveHappened = true
			state.Last.Notice = "pending"
		} else {
			state.Active = nil
		}
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
		a.NoticeDeadline = event.At.Add(time.Minute)
		if a.Deadline.Before(a.NoticeDeadline) {
			a.NoticeDeadline = a.Deadline
		}
		a.NoticeMayHaveHappened = true
		state.Last = summary(a, event.At, nil, "completed", "pending")
	case TaskFinished:
		if a.Stage != "running" && a.Stage != "stopping" {
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
		var exitStatus *int
		if exit >= 0 {
			exitStatus = &exit
		}
		state.Last = summary(a, event.At, exitStatus, outcome, "")
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
	case NoticeFrozen:
		if a.Stage != "notifying" || !validFrozenNotice(event.ReceiptID, event.NoticeBody, event.NoticeBinding) {
			break
		}
		if a.ReceiptID != "" && (a.ReceiptID != event.ReceiptID || a.NoticeBody != event.NoticeBody || !bytes.Equal(a.NoticeBinding, event.NoticeBinding)) {
			break
		}
		a.ReceiptID, a.NoticeBody = event.ReceiptID, event.NoticeBody
		a.NoticeBinding = append(json.RawMessage(nil), event.NoticeBinding...)
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
		if !validFrozenNotice(a.ReceiptID, a.NoticeBody, a.NoticeBinding) {
			break
		}
		a.Stage = "delivery"
		a.NoticeMayHaveHappened = true
		return state, []Effect{{Kind: SendNotice, Attempt: a.ID}}
	case NoticeObserved:
		if a.Stage != "delivery" {
			break
		}
		if strings.HasPrefix(event.Notice, "dropped: ") && len(event.Notice) > len("dropped: ") && len(event.Notice) <= 4096 && utf8.ValidString(event.Notice) {
			state.Last.Notice = event.Notice
			state.Active = nil
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
		if a.Source == "manual" && a.Stage == "reserved" {
			state.Last = summary(a, event.At, nil, "cancelled", "manual")
			state.Active = nil
			return state, nil
		}
		return cancelActive(state)
	}
	return state, nil
}

// validateTaskPhase is the one source/stage/companion contract for transitions
// and both ledger codecs. Empty bootstrap state is valid; malformed authority
// emits no effects and is refused at the persistence boundary.
func validateTaskPhase(state TaskState) error {
	if last := state.Last; last != nil {
		if last.ManualIntentDigest != "" && last.Source != "manual" {
			return fmt.Errorf("manual intent digest requires manual summary")
		}
		if last.Source != "manual" && last.Source != "scheduled" {
			return fmt.Errorf("invalid summary source")
		}
		switch last.Outcome {
		case "quiet", "wake", "failed", "timeout", "cancelled", "skipped", "refused", "completed", "unknown":
		default:
			return fmt.Errorf("invalid summary outcome")
		}
	}
	a := state.Active
	if last := state.Last; last != nil {
		if last.Notice == "pending" && (a == nil || (a.Stage != "notifying" && a.Stage != "delivery")) {
			return fmt.Errorf("pending summary requires its active notice attempt")
		}
		if last.Outcome == "unknown" && (a == nil || a.Stage != "uncertain") {
			return fmt.Errorf("unknown summary requires retained execution authority")
		}
	}
	if a == nil {
		return nil
	}
	if a.ManualIntentDigest != "" && a.Source != "manual" {
		return fmt.Errorf("manual intent digest requires manual attempt")
	}
	if a.Source != "manual" && a.Source != "scheduled" {
		return fmt.Errorf("invalid attempt source")
	}
	if (a.ControllerPID == 0) != (a.ControllerStart == "") || a.ControllerPID < 0 {
		return fmt.Errorf("execution owner PID and start token must be paired")
	}
	notice := a.Stage == "notifying" || a.Stage == "delivery"
	matchingLast := state.Last != nil && state.Last.ID == a.ID && state.Last.Source == a.Source && state.Last.DueAt.Equal(a.DueAt) && state.Last.StartedAt.Equal(a.StartedAt)
	switch a.Stage {
	case "reserved", "running", "stopping", "uncertain":
		if !a.NoticeDeadline.IsZero() || a.NoticeMayHaveHappened || a.ReceiptID != "" || a.NoticeBody != "" || len(a.NoticeBinding) != 0 {
			return fmt.Errorf("execution phase carries notice authority")
		}
		if a.Source == "scheduled" && a.Revision == state.Revision && !state.Spec.ScriptRun && a.Stage != "reserved" && a.Stage != "uncertain" {
			return fmt.Errorf("notify-only task has an execution phase")
		}
		if a.Stage == "uncertain" && (!matchingLast || state.Last.Outcome != "unknown") {
			return fmt.Errorf("uncertain execution requires its unknown summary")
		}
	case "notifying", "delivery":
		frozen := a.ReceiptID != "" || a.NoticeBody != "" || len(a.NoticeBinding) != 0
		if (frozen || a.Stage == "delivery") && !validFrozenNotice(a.ReceiptID, a.NoticeBody, a.NoticeBinding) {
			return fmt.Errorf("notice requires complete frozen identity and payload")
		}
		if a.Source != "scheduled" || !matchingLast || a.NoticeDeadline.IsZero() || !a.NoticeMayHaveHappened {
			return fmt.Errorf("notice phase requires matching scheduled summary and intent")
		}
		switch state.Last.Outcome {
		case "completed", "wake", "failed", "timeout", "cancelled", "refused":
		default:
			return fmt.Errorf("summary outcome cannot produce a notice")
		}
	default:
		return fmt.Errorf("invalid attempt stage")
	}
	if notice && state.Last.Notice != "pending" {
		return fmt.Errorf("active notice requires pending summary delivery")
	}
	return nil
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
	return &RunSummary{ManualIntentDigest: a.ManualIntentDigest, Revision: a.Revision, ID: a.ID, Source: a.Source, DueAt: a.DueAt, StartedAt: a.StartedAt, EndedAt: ended, Exit: exit, Outcome: outcome, Notice: notice}
}

func cancelActive(state TaskState) (TaskState, []Effect) {
	if state.Active == nil || state.Active.Stage == "uncertain" || state.Active.Stage == "stopping" {
		return state, nil
	}
	if state.Active.Stage == "reserved" {
		state.Active = nil
		return state, nil
	}
	if state.Active.Stage == "notifying" {
		state.Last.Notice = "dropped: task cancelled"
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

// Binding is opaque to the core; the broker adapter validates its full identity.
func validFrozenNotice(id, body string, binding json.RawMessage) bool {
	if !ValidTaskName(id) || body == "" || len(body) > 8192 || !utf8.ValidString(body) || len(binding) == 0 || len(binding) > 4096 || !utf8.Valid(binding) {
		return false
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	_, fields, err := strictjson.SplitObject(binding, nil, nil)
	return err == nil && len(fields) > 0
}

func cloneAttempt(source *Attempt) *Attempt {
	result := *source
	result.NoticeBinding = append(json.RawMessage(nil), source.NoticeBinding...)
	result.Unknown = strictjson.CloneRaw(source.Unknown)
	return &result
}

func cloneRunSummary(source *RunSummary) *RunSummary {
	result := *source
	if source.Exit != nil {
		code := *source.Exit
		result.Exit = &code
	}
	result.Unknown = strictjson.CloneRaw(source.Unknown)
	return &result
}

func validRecoveredSummary(s RunSummary) bool {
	if s.ManualIntentDigest != "" && s.Source != "manual" {
		return false
	}
	if s.Revision == "" || len(s.Revision) > 4096 || !utf8.ValidString(s.Revision) || strings.IndexFunc(s.Revision, unicode.IsControl) >= 0 || !ValidTaskName(s.ID) || (s.Source != "manual" && s.Source != "scheduled") || (s.Exit != nil && (*s.Exit < 0 || *s.Exit > 255)) || s.StartedAt.IsZero() || s.EndedAt.IsZero() {
		return false
	}
	switch s.Outcome {
	case "quiet", "wake", "failed", "timeout", "cancelled", "skipped", "refused", "completed", "unknown":
	default:
		return false
	}
	if len(s.Notice) > 4096 || !utf8.ValidString(s.Notice) {
		return false
	}
	if s.Notice == "pending" {
		return s.Source == "scheduled" && (s.Outcome == "completed" || s.Outcome == "wake" || s.Outcome == "failed" || s.Outcome == "timeout" || s.Outcome == "cancelled" || s.Outcome == "refused")
	}
	switch s.Notice {
	case "", "manual", "none", "submitted", "expired", "uncertain":
		return true
	}
	if strings.HasPrefix(s.Notice, "dropped: ") && len(s.Notice) > len("dropped: ") {
		return true
	}
	// These outcomes carry an explanatory refusal/skip/owner reason, rather
	// than a delivery state. The operational record already owns its identity.
	return s.Outcome == "skipped" || s.Outcome == "refused" || s.Outcome == "unknown"
}
