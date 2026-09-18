package checkpoint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Phase string

const (
	Pending  Phase = "pending"
	Running  Phase = "running"
	Failed   Phase = "failed"
	Complete Phase = "complete"
)
const MaxFailureBytes = 4096

// AllPhases is the request vocabulary in lifecycle order, so callers and tests
// enumerate it rather than restating it.
func AllPhases() []Phase { return []Phase{Pending, Running, Failed, Complete} }

// Exits names what an operator can do about an unfinished request: the ONE
// wording for every refusal a retained request causes, in couchcore and in
// pair's CLI. A FAILED request has two exits and every refusal offers both --
// when retry was the only one named, a thread whose failed handoff the operator
// had already taken over could not be relaunched without re-delivering it
// (pair#280). A non-empty tag adds the forms that work after Couch exits.
func Exits(phase Phase, tag string) string {
	if phase == Failed {
		s := "Retry continuation re-delivers it and Dismiss continuation drops it"
		if tag != "" {
			s += fmt.Sprintf("; after Couch exits, `couch --internal retry-continuation %s` or `couch --internal dismiss-continuation %s`", tag, tag)
		}
		return s
	}
	s := "Retry continuation reconciles it"
	if tag != "" {
		s += fmt.Sprintf("; after Couch exits, `couch --internal retry-continuation %s`", tag)
	}
	return s
}

// CheckDismissible is the rule for retiring a request by deleting it: only the
// exact FAILED request, where an empty id means the retained one. Pure, so it
// is tested on literal requests; couchcore's ThreadStore.DismissFailedContinuation
// applies it inside the store's revision CAS (pair#280).
func CheckDismissible(r *Request, id string) error {
	if r == nil {
		return errors.New("thread has no continuation request")
	}
	if id != "" && r.ID != id {
		return errors.New("obsolete continuation request")
	}
	if r.Phase != Failed {
		return fmt.Errorf("continuation %s is %s; only a failed continuation can be dismissed", r.ID, r.Phase)
	}
	return nil
}

type Process struct {
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
}

// SourceAbsence is a durable observation, distinct from a successful park.
// The store binds RecordRevision to the record inspected by recovery.
type SourceAbsence struct {
	Session        string    `json:"session"`
	LaunchOrdinal  uint64    `json:"launch_ordinal"`
	ObservedAt     time.Time `json:"observed_at"`
	RecordRevision uint64    `json:"record_revision"`
}

// TargetGeneration joins a launch ledger generation to an exact request attempt.
// It survives helper reattachment and, as the previous witness, target death.
type TargetGeneration struct {
	Agent         string `json:"agent"`
	Session       string `json:"session"`
	Attempt       string `json:"attempt"`
	LaunchOrdinal uint64 `json:"launch_ordinal"`
}

type Target struct {
	Process
	ObservedAt time.Time         `json:"observed_at"`
	Generation *TargetGeneration `json:"generation,omitempty"`
}
type Source struct {
	Agent         string  `json:"agent"`
	Session       string  `json:"session"`
	LaunchOrdinal uint64  `json:"launch_ordinal"`
	Helper        Process `json:"helper"`
}
type Request struct {
	Version                  int               `json:"version"`
	ID                       string            `json:"id"`
	Checkpoint               Checkpoint        `json:"checkpoint"`
	Source                   Source            `json:"source"`
	CreatedAt                time.Time         `json:"created_at"`
	Phase                    Phase             `json:"phase"`
	Attempt                  string            `json:"attempt,omitempty"`
	SourcePark               string            `json:"source_park,omitempty"`
	SourceAbsence            *SourceAbsence    `json:"source_absence,omitempty"`
	PreviousTargetGeneration *TargetGeneration `json:"previous_target_generation,omitempty"`
	Target                   *Target           `json:"target,omitempty"`
	Failure                  string            `json:"failure,omitempty"`
}

func RequestID(scope, tag string, ordinal uint64, digest string) string {
	raw, _ := json.Marshal([]any{scope, tag, ordinal, digest})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (r Request) Clone() Request {
	if r.Target != nil {
		p := *r.Target
		r.Target = &p
		if p.Generation != nil {
			generation := *p.Generation
			r.Target.Generation = &generation
		}
	}
	if r.SourceAbsence != nil {
		absence := *r.SourceAbsence
		r.SourceAbsence = &absence
	}
	if r.PreviousTargetGeneration != nil {
		generation := *r.PreviousTargetGeneration
		r.PreviousTargetGeneration = &generation
	}
	return r
}
func boundedIdentity(s string) bool {
	return s != "" && len(s) <= 1024 && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func validProcess(p Process) bool { return p.PID > 0 && boundedIdentity(p.Identity) }
func (r Request) Validate() error {
	if r.Version != Version {
		return errors.New("unsupported continuation request version")
	}
	id, err := hex.DecodeString(r.ID)
	if err != nil || len(id) != sha256.Size || strings.ToLower(r.ID) != r.ID {
		return errors.New("invalid continuation request ID")
	}
	if err := r.Checkpoint.Validate(); err != nil {
		return err
	}
	if r.Source.Agent != r.Checkpoint.Agent() || !boundedIdentity(r.Source.Agent) || !boundedIdentity(r.Source.Session) || r.Source.LaunchOrdinal == 0 || r.CreatedAt.IsZero() {
		return errors.New("invalid continuation source identity")
	}
	if !validProcess(r.Source.Helper) && (r.SourceAbsence == nil || r.Source.Helper != (Process{})) {
		return errors.New("invalid continuation source helper")
	}
	if absent := r.SourceAbsence; absent != nil {
		if r.SourcePark != "" || absent.Session != r.Source.Session || absent.LaunchOrdinal != r.Source.LaunchOrdinal || absent.ObservedAt.IsZero() || absent.RecordRevision == 0 {
			return errors.New("invalid continuation source absence")
		}
	}
	if r.SourcePark != "" && !boundedIdentity(r.SourcePark) {
		return errors.New("invalid continuation source park receipt")
	}
	if r.Target != nil && ((r.SourcePark == "" && r.SourceAbsence == nil) || (!validProcess(r.Target.Process) || r.Target.ObservedAt.IsZero())) {
		return errors.New("invalid continuation target identity")
	}
	if g := r.PreviousTargetGeneration; g != nil {
		if (r.SourcePark == "" && r.SourceAbsence == nil) || !r.validTargetGeneration(*g) || g.Attempt == r.Attempt {
			return errors.New("invalid previous continuation target generation")
		}
	}
	if r.Target != nil && r.Target.Generation != nil {
		g := r.Target.Generation
		if !r.validTargetGeneration(*g) || g.Attempt != r.Attempt {
			return errors.New("invalid continuation target generation")
		}
		if previous := r.PreviousTargetGeneration; previous != nil && g.LaunchOrdinal <= previous.LaunchOrdinal {
			return errors.New("continuation target generation did not advance")
		}
	}
	if len(r.Failure) > MaxFailureBytes || !utf8.ValidString(r.Failure) || strings.ContainsRune(r.Failure, 0) {
		return errors.New("continuation diagnostic exceeds limit")
	}
	switch r.Phase {
	case Pending:
		if r.Attempt != "" || r.Target != nil || r.Failure != "" || r.SourcePark != "" || r.PreviousTargetGeneration != nil {
			return errors.New("pending continuation has attempt state")
		}
	case Running:
		if !boundedIdentity(r.Attempt) || r.Failure != "" {
			return errors.New("invalid running continuation")
		}
	case Failed:
		if !boundedIdentity(r.Attempt) || r.Failure == "" {
			return errors.New("invalid failed continuation")
		}
	case Complete:
		if !boundedIdentity(r.Attempt) || r.Target == nil || r.Failure != "" {
			return errors.New("complete continuation lacks exact target")
		}
	default:
		return fmt.Errorf("unknown continuation phase %q", r.Phase)
	}
	return nil
}

func (r Request) validTargetGeneration(g TargetGeneration) bool {
	return g.Agent == r.Source.Agent && g.Session == r.Source.Session && boundedIdentity(g.Attempt) && g.LaunchOrdinal > r.Source.LaunchOrdinal
}

type EventKind string

const (
	Begin         EventKind = "begin"
	Registered    EventKind = "registered"
	Submitted     EventKind = "submitted"
	Fail          EventKind = "fail"
	RetryAbsent   EventKind = "retry-absent"
	RetryObserve  EventKind = "retry-observe"
	RefreshSource EventKind = "refresh-source"
	SourceParked  EventKind = "source-parked"
	SourceAbsent  EventKind = "source-absent"
)

type Event struct {
	At               time.Time
	ParkNonce        string
	Kind             EventKind
	RequestID        string
	Attempt          string
	Target           *Process
	Helper           Process
	Failure          string
	SourceAbsence    *SourceAbsence
	TargetGeneration *TargetGeneration
}

// Advance is the request's lifecycle. Failed leaves it only through retry
// (RetryAbsent/RetryObserve) here; the other exit is couchcore's
// ThreadStore.DismissFailedContinuation, which DELETES the request rather than
// adding a phase, so strict decoders of an older binary still read the record.
func Advance(r Request, e Event) (Request, error) {
	if err := r.Validate(); err != nil {
		return Request{}, err
	}
	if e.RequestID != r.ID {
		return Request{}, errors.New("obsolete continuation request")
	}
	next := r.Clone()
	switch e.Kind {
	case SourceAbsent:
		if r.Phase == Complete || e.Attempt != r.Attempt || r.Target != nil || r.SourcePark != "" || e.SourceAbsence == nil {
			return Request{}, errors.New("source absence requires an exact unresolved source")
		}
		if r.SourceAbsence != nil && *r.SourceAbsence != *e.SourceAbsence {
			return Request{}, errors.New("continuation source absence is immutable")
		}
		absence := *e.SourceAbsence
		next.SourceAbsence = &absence
	case Begin:
		if r.Phase != Pending || !boundedIdentity(e.Attempt) {
			return Request{}, errors.New("continuation is not pending")
		}
		next.Phase = Running
		next.Attempt = e.Attempt
	case RetryAbsent:
		if r.Phase != Failed || !boundedIdentity(e.Attempt) || e.Attempt == r.Attempt {
			return Request{}, errors.New("retry requires a new attempt and proved absence")
		}
		next.Phase = Running
		next.Attempt = e.Attempt
		if next.Target != nil && next.Target.Generation != nil {
			generation := *next.Target.Generation
			next.PreviousTargetGeneration = &generation
		}
		if e.TargetGeneration != nil {
			generation := *e.TargetGeneration
			if generation.Attempt != r.Attempt || !r.validTargetGeneration(generation) {
				return Request{}, errors.New("retry target receipt belongs to another generation")
			}
			if r.Target != nil && r.Target.Generation != nil && *r.Target.Generation != generation {
				return Request{}, errors.New("retry target receipt changed generation")
			}
			next.PreviousTargetGeneration = &generation
		}
		next.Target = nil
		next.Failure = ""
	case RetryObserve:
		if r.Phase != Failed || e.Attempt != r.Attempt {
			return Request{}, errors.New("retry observation has obsolete attempt")
		}
		next.Phase = Running
		next.Failure = ""
	default:
		if r.Phase != Running || e.Attempt != r.Attempt {
			return Request{}, errors.New("obsolete continuation attempt or phase")
		}
		switch e.Kind {
		case SourceParked:
			if !boundedIdentity(e.ParkNonce) || r.SourcePark != "" && r.SourcePark != e.ParkNonce {
				return Request{}, errors.New("invalid source park receipt")
			}
			next.SourcePark = e.ParkNonce
		case Registered:
			if e.Target == nil {
				return Request{}, errors.New("missing registered target")
			}
			observed := e.At
			if r.Target != nil {
				observed = r.Target.ObservedAt
			}
			next.Target = &Target{Process: *e.Target, ObservedAt: observed}
			if e.TargetGeneration != nil {
				generation := *e.TargetGeneration
				if r.Target != nil && r.Target.Generation != nil && *r.Target.Generation != generation {
					return Request{}, errors.New("registered continuation generation changed")
				}
				next.Target.Generation = &generation
			} else if r.Target != nil && r.Target.Generation != nil {
				generation := *r.Target.Generation
				next.Target.Generation = &generation
			}
		case Submitted:
			next.Phase = Complete
		case Fail:
			next.Phase = Failed
			next.Failure = e.Failure
			if len(next.Failure) > MaxFailureBytes {
				next.Failure = next.Failure[:MaxFailureBytes]
				for !utf8.ValidString(next.Failure) {
					next.Failure = next.Failure[:len(next.Failure)-1]
				}
			}
		case RefreshSource:
			next.Source.Helper = e.Helper
		default:
			return Request{}, errors.New("unknown continuation event")
		}
	}
	if err := next.Validate(); err != nil {
		return Request{}, err
	}
	return next, nil
}
