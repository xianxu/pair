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
// enumerate it rather than restating it. Failed is read only: a request that
// failed before #522 decodes into the thread's one non-blocking line (Miss).
func AllPhases() []Phase { return []Phase{Pending, Running, Failed, Complete} }

// InFlight is the one wording for every refusal an unfinished request causes:
// it is a compaction in progress, Couch drives it to the end (a new
// conversation, or the one visible line), and nothing is for the operator to
// do but wait. A Couch that exits mid-way picks it up when it starts again.
const InFlight = "a compaction is in progress and Couch finishes it on its own (when Couch next runs, if it has exited)"

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
	return g.Agent == r.Source.Agent && boundedIdentity(g.Session) && boundedIdentity(g.Attempt) && g.LaunchOrdinal > r.Source.LaunchOrdinal
}

type EventKind string

const (
	Begin         EventKind = "begin"
	Registered    EventKind = "registered"
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
	SourceAbsence    *SourceAbsence
	TargetGeneration *TargetGeneration
}

// Advance is the request's lifecycle: Pending, then Running, then Complete
// at the target's registration (#522). A request that doesn't finish leaves
// the lifecycle entirely: couchcore's ThreadStore.MissContinuation deletes it
// and writes the thread's one visible line.
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
			// The prompt is part of the launch (#522): a registered target
			// has its continuation, so there is nothing left to confirm.
			next.Phase = Complete
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
