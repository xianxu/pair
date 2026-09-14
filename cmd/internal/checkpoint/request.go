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

type Process struct {
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
}
type Target struct {
	Process
	ObservedAt time.Time `json:"observed_at"`
}
type Source struct {
	Agent         string  `json:"agent"`
	Session       string  `json:"session"`
	LaunchOrdinal uint64  `json:"launch_ordinal"`
	Helper        Process `json:"helper"`
}
type Request struct {
	Version    int        `json:"version"`
	ID         string     `json:"id"`
	Checkpoint Checkpoint `json:"checkpoint"`
	Source     Source     `json:"source"`
	CreatedAt  time.Time  `json:"created_at"`
	Phase      Phase      `json:"phase"`
	Attempt    string     `json:"attempt,omitempty"`
	SourcePark string     `json:"source_park,omitempty"`
	Target     *Target    `json:"target,omitempty"`
	Failure    string     `json:"failure,omitempty"`
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
	if r.Source.Agent != r.Checkpoint.Agent() || !boundedIdentity(r.Source.Agent) || !boundedIdentity(r.Source.Session) || r.Source.LaunchOrdinal == 0 || !validProcess(r.Source.Helper) || r.CreatedAt.IsZero() {
		return errors.New("invalid continuation source identity")
	}
	if r.SourcePark != "" && !boundedIdentity(r.SourcePark) {
		return errors.New("invalid continuation source park receipt")
	}
	if r.Target != nil && (r.SourcePark == "" || (!validProcess(r.Target.Process) || r.Target.ObservedAt.IsZero())) {
		return errors.New("invalid continuation target identity")
	}
	if len(r.Failure) > MaxFailureBytes || !utf8.ValidString(r.Failure) || strings.ContainsRune(r.Failure, 0) {
		return errors.New("continuation diagnostic exceeds limit")
	}
	switch r.Phase {
	case Pending:
		if r.Attempt != "" || r.Target != nil || r.Failure != "" || r.SourcePark != "" {
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
)

type Event struct {
	At        time.Time
	ParkNonce string
	Kind      EventKind
	RequestID string
	Attempt   string
	Target    *Process
	Helper    Process
	Failure   string
}

func Advance(r Request, e Event) (Request, error) {
	if err := r.Validate(); err != nil {
		return Request{}, err
	}
	if e.RequestID != r.ID {
		return Request{}, errors.New("obsolete continuation request")
	}
	next := r.Clone()
	switch e.Kind {
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
