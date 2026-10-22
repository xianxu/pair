package couchmessage

import (
	"errors"
	"time"
)

// Registry decides which wrapper bindings may receive. Membership follows
// the authenticated wrapper connection, closed by death or exec; client panes
// only describe local presentation. It is pure: callers feed events in arrival
// order and execute effects; every interleaving is reproducible.
type Registry struct {
	sessions map[SessionToken]*registrySession
	panes    map[ThreadKey]PaneHandle
}

// SessionToken names one wrapper connection; a larger token is a newer one.
type SessionToken uint64

// PaneHandle names one Console pane incarnation.
type PaneHandle string

type ThreadKey struct{ Scope, Tag string }

func (b Binding) Thread() ThreadKey { return ThreadKey{Scope: b.Scope, Tag: b.Tag} }

// AdmissionPhase is a session's whole state; there are no side flags.
//
//	hello --> Admitting --ok--> Admitted
//	Admitting --fail--> Rejected --retry due--> Admitting   (≤ MaxAdmissionAttempts)
//	Rejected (last attempt) --> Dormant --submit/send--> Admitting
//	any open phase --a newer session for the slot is admitted--> Displaced (final)
//	(a newer session that is never admitted displaces nothing)
type AdmissionPhase int

const (
	AwaitingAdmission AdmissionPhase = iota
	Admitting
	Admitted
	Rejected
	Dormant
	Displaced
)

// RetryDelays is the bounded admission backoff; after the last a session is
// Dormant until an event touches it.
var RetryDelays = []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}

type registrySession struct {
	binding Binding
	// build and settled are the wrapper's self-report (#421): build is nil
	// for a pre-hello-v2 wrapper, settled nil until a hello-v2 session says.
	build          *BuildIdentity
	settled        *bool
	backgroundWork *bool
	// picker is the hello-v3 self-report (pair#465): nil until one says.
	// pickerSince is the wrapper's LastActivity on the frame that opened it.
	picker      *bool
	pickerSince time.Time
	lastWorkAt  time.Time
	submission  uint64
	phase       AdmissionPhase
	pane        PaneHandle // optional client metadata captured when admission began
	attempt     int        // failed admissions since the last event that reset it
}

type RegistryEventKind int

const (
	SessionOpened RegistryEventKind = iota
	SessionClosed
	// PaneChanged carries the thread's current pane ("" = none). A state, not
	// an edge, so a coalescing producer can drop intermediate panes safely.
	PaneChanged
	AdmissionDone
	RetryDue
	SessionActivity
	SessionSubmit
	SendTargeted
	// ConnectFailed reports that executing EffectConnect failed (the broker
	// refused the actor): an effect that can fail reports back as an event,
	// or the registry would believe a binding the broker never registered.
	ConnectFailed
)

type RegistryEvent struct {
	Kind        RegistryEventKind
	Token       SessionToken
	Binding     Binding // SessionOpened
	Slot        string  // SendTargeted: a send names a slot, not a binding
	Thread      ThreadKey
	Pane        PaneHandle // PaneChanged, AdmissionDone, ConnectFailed
	Err         error      // AdmissionDone, ConnectFailed
	Attempt     int        // RetryDue
	Observation Observation
	Build       *BuildIdentity // SessionOpened: non-nil for a hello-v2 session
	Settled     *bool          // SessionActivity/SessionSubmit on a hello-v2 session
	Picker      *bool          // SessionActivity/SessionSubmit on a hello-v3 session
	At          time.Time      // supervisor receipt time; display activity only

	BackgroundWork *bool // SessionActivity/SessionSubmit on a hello-v4 session
}

type RegistryEffectKind int

const (
	// EffectAdmit runs one full wrapper/terminal authority check of Binding.
	EffectAdmit RegistryEffectKind = iota
	EffectConnect
	EffectDisconnect
	EffectScheduleRetry
	// EffectObserve forwards a session's pushed observation to the broker.
	EffectObserve
)

type RegistryEffect struct {
	Kind        RegistryEffectKind
	Token       SessionToken
	Binding     Binding
	Pane        PaneHandle
	Attempt     int
	Delay       time.Duration
	Observation Observation
}

func NewRegistry() *Registry {
	return &Registry{sessions: map[SessionToken]*registrySession{}, panes: map[ThreadKey]PaneHandle{}}
}

var ErrRegistryFull = errors.New("message session capacity reached")

// Advance applies one event. Unknown or stale references are no-ops: a late
// admission result, a close of a superseded token, a frame from a session that
// has gone. Only SessionOpened can fail (capacity, invalid binding).
func (r *Registry) Advance(e RegistryEvent) ([]RegistryEffect, error) {
	var fx []RegistryEffect
	switch e.Kind {
	case SessionOpened:
		if err := e.Binding.Validate(); err != nil {
			return nil, err
		}
		if _, ok := r.sessions[e.Token]; ok {
			return nil, errors.New("duplicate session token")
		}
		if len(r.sessions) >= MaxActors {
			return nil, ErrRegistryFull
		}
		// The same binding on a new connection (an exec keeps PID, start time,
		// nonce and session) supersedes the old connection outright, so an
		// in-flight delivery at the old incarnation ends Indeterminate.
		for t, s := range r.sessions {
			if t != e.Token && s.binding == e.Binding && s.phase != Displaced {
				fx = r.displace(t, s, fx)
			}
		}
		s := &registrySession{binding: e.Binding, build: e.Build, phase: AwaitingAdmission}
		r.sessions[e.Token] = s
		fx = r.readmit(e.Token, s, fx)
	case SessionClosed:
		s := r.sessions[e.Token]
		if s == nil {
			return nil, nil
		}
		if r.connected(s) {
			fx = append(fx, RegistryEffect{Kind: EffectDisconnect, Token: e.Token, Binding: s.binding})
		}
		delete(r.sessions, e.Token)
	case PaneChanged:
		// Client attachment is presentation state. It cannot disconnect a
		// live wrapper or prevent one from joining after a Couch restart.
		if e.Pane == "" {
			delete(r.panes, e.Thread)
		} else {
			r.panes[e.Thread] = e.Pane
		}
	case AdmissionDone:
		s := r.sessions[e.Token]
		if s == nil || s.phase != Admitting || s.pane != e.Pane {
			return nil, nil
		}
		if e.Err != nil {
			fx = r.fail(e.Token, s, fx)
			break
		}
		if r.newerAdmittedForSlot(e.Token, s.binding) {
			s.phase = Displaced
			break
		}
		// Newest admitted wins: admitting displaces only older sessions. A
		// newer one still being checked (or failing) does not strand this
		// working one; it displaces this one if and when it is admitted.
		for t, other := range r.sessions {
			if t < e.Token && sameSlot(other.binding, s.binding) && other.phase != Displaced {
				fx = r.displace(t, other, fx)
			}
		}
		s.phase, s.attempt = Admitted, 0
		fx = append(fx, RegistryEffect{Kind: EffectConnect, Token: e.Token, Binding: s.binding, Pane: s.pane})
	case ConnectFailed:
		s := r.sessions[e.Token]
		if s == nil || s.phase != Admitted || s.pane != e.Pane {
			return nil, nil
		}
		fx = r.fail(e.Token, s, fx)
	case RetryDue:
		if s := r.sessions[e.Token]; s != nil && s.phase == Rejected && s.attempt == e.Attempt {
			fx = r.readmit(e.Token, s, fx)
		}
	case SessionActivity, SessionSubmit:
		s := r.sessions[e.Token]
		if s == nil {
			return nil, nil
		}
		s.submission = max(s.submission, e.Observation.Submission)
		if s.build != nil && e.Settled != nil {
			v := *e.Settled
			if s.settled == nil && v && !e.At.IsZero() && e.Observation.LastActivity.After(s.lastWorkAt) {
				s.lastWorkAt = e.Observation.LastActivity
				if s.lastWorkAt.After(e.At) {
					s.lastWorkAt = e.At
				}
			}
			wasBusy := s.settled != nil && !*s.settled
			if (!v || wasBusy) && e.At.After(s.lastWorkAt) {
				s.lastWorkAt = e.At
			}
			s.settled = &v
		}
		if s.build != nil {
			wasActive := s.backgroundWork != nil && *s.backgroundWork
			s.backgroundWork = copyBackgroundWork(e.BackgroundWork)
			if s.backgroundWork != nil && (*s.backgroundWork || wasActive) && e.At.After(s.lastWorkAt) {
				s.lastWorkAt = e.At
			}
		}
		if s.build != nil && e.Picker != nil {
			v := *e.Picker
			switch {
			case !v:
				s.pickerSince = time.Time{}
			case s.picker == nil || !*s.picker:
				s.pickerSince = e.Observation.LastActivity
			}
			s.picker = &v
		}
		if s.phase == Admitted {
			fx = append(fx, RegistryEffect{Kind: EffectObserve, Token: e.Token, Binding: s.binding, Observation: e.Observation})
		} else if e.Kind == SessionSubmit && s.phase == Dormant {
			s.attempt = 0
			fx = r.readmit(e.Token, s, fx)
		}
	case SendTargeted:
		for _, t := range r.tokens() {
			if s := r.sessions[t]; s.binding.Slot == e.Slot && s.phase == Dormant {
				s.attempt = 0
				fx = r.readmit(t, s, fx)
			}
		}
	default:
		return nil, errors.New("unknown registry event")
	}
	return fx, nil
}

// readmit admits the live wrapper connection. A local client pane is optional
// metadata and never grants authority; the effect performs the full check.
func (r *Registry) readmit(t SessionToken, s *registrySession, fx []RegistryEffect) []RegistryEffect {
	s.phase, s.pane = Admitting, r.panes[s.binding.Thread()]
	return append(fx, RegistryEffect{Kind: EffectAdmit, Token: t, Binding: s.binding, Pane: s.pane})
}

// fail records a failed attempt and schedules the next one on the bounded
// ladder, or leaves the session dormant once the ladder is spent.
func (r *Registry) fail(t SessionToken, s *registrySession, fx []RegistryEffect) []RegistryEffect {
	s.attempt++
	if s.attempt > len(RetryDelays) {
		s.phase = Dormant
		return fx
	}
	s.phase = Rejected
	return append(fx, RegistryEffect{Kind: EffectScheduleRetry, Token: t, Binding: s.binding, Attempt: s.attempt, Delay: RetryDelays[s.attempt-1]})
}

func (r *Registry) displace(t SessionToken, s *registrySession, fx []RegistryEffect) []RegistryEffect {
	if r.connected(s) {
		fx = append(fx, RegistryEffect{Kind: EffectDisconnect, Token: t, Binding: s.binding})
	}
	s.phase, s.pane = Displaced, ""
	return fx
}

func (r *Registry) newerAdmittedForSlot(t SessionToken, b Binding) bool {
	for other, s := range r.sessions {
		if other > t && sameSlot(s.binding, b) && s.phase == Admitted {
			return true
		}
	}
	return false
}

func sameSlot(a, b Binding) bool { return a.Repository == b.Repository && a.Slot == b.Slot }

func (r *Registry) connected(s *registrySession) bool { return s.phase == Admitted }

// tokens lists sessions oldest first, so effects come out in a stable order.
func (r *Registry) tokens() []SessionToken {
	out := make([]SessionToken, 0, len(r.sessions))
	for t := range r.sessions {
		out = append(out, t)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Connected is the set of bindings that may receive now. Callers publish it
// as an immutable snapshot for request goroutines.
func (r *Registry) Connected() map[Binding]bool {
	out := map[Binding]bool{}
	for _, s := range r.sessions {
		if r.connected(s) {
			out[s.binding] = true
		}
	}
	return out
}

// Phase reports a session's phase, for listings and tests.
func (r *Registry) Phase(t SessionToken) (AdmissionPhase, bool) {
	s := r.sessions[t]
	if s == nil {
		return 0, false
	}
	return s.phase, true
}

// SlotLiveness is what a slot's admitted wrapper last said about itself
// (#421). Build is nil for a wrapper that predates hello-v2; Settled is nil
// until a hello-v2 session reports it. Both being nil reads as "unknown",
// which is never "idle".
type SlotLiveness struct {
	Submission uint64
	Binding    Binding
	// LastWorkAt is the last observed work/completion of this session (#625).
	// Settled refreshes cannot advance it; replacement sessions start empty.
	LastWorkAt time.Time
	Build      *BuildIdentity
	Settled    *bool
	// BackgroundWork is display-only hello-v4 evidence; nil is unknown.
	BackgroundWork *bool
	// Picker: an input-holding picker is up (pair#465); nil is unknown.
	Picker *bool
	// PickerSince is when the picker opened, on the wrapper's clock: its
	// LastActivity on the frame that first reported it. Zero while closed.
	PickerSince time.Time
	// Session is the admitted connection. A SIGUSR2 re-exec keeps the binding
	// byte-identical (PID, start time, nonce), so a new token is the evidence
	// that the wrapper restarted (pair#421 reload-context).
	Session SessionToken
}

// Liveness snapshots every admitted session by slot. Admission already keeps
// one admitted session per slot (a new incarnation displaces the old); should
// two ever coexist, the slot is omitted, because a restart must not guess.
func (r *Registry) Liveness() map[string]SlotLiveness {
	out := map[string]SlotLiveness{}
	ambiguous := map[string]bool{}
	for _, t := range r.tokens() {
		s := r.sessions[t]
		if s.phase != Admitted {
			continue
		}
		slot := s.binding.Slot
		if _, seen := out[slot]; seen {
			ambiguous[slot] = true
			continue
		}
		out[slot] = SlotLiveness{Binding: s.binding, Submission: s.submission, LastWorkAt: s.lastWorkAt, Build: s.build, Settled: s.settled, BackgroundWork: copyBackgroundWork(s.backgroundWork), Picker: s.picker, PickerSince: s.pickerSince, Session: t}
	}
	for slot := range ambiguous {
		delete(out, slot)
	}
	return out
}

// copyBackgroundWork keeps nullable display evidence owned by each snapshot.
func copyBackgroundWork(v *bool) *bool {
	if v == nil {
		return nil
	}
	owned := *v
	return &owned
}
