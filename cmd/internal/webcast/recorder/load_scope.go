package recorder

import (
	"math"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type loadPhase uint8

const (
	loadUnknown loadPhase = iota
	loadPending
	loadCommitting
	loadConfirmed
	loadUnavailable
	loadClosed
)

// Selection contains synchronized ingress facts, never actor-owned map reads.
// Frame may be unknown until the initial snapshot supplies its top-frame proof.
type loadSelection struct{ Session, Target, Frame string }

type loadEvidence struct {
	Selection loadSelection
	Load      *recording.PageLoad
}

type loadState struct {
	phase       loadPhase
	generation  uint64
	selection   loadSelection
	reservation *loadReservation
	load        *recording.PageLoad
}

// Only the Pending -> Committing transition grants this token. Identity of
// the token, rather than equality of its contents, authorizes completion.
type loadReservation struct {
	generation uint64
	evidence   loadEvidence
}

func ownedLoadIdentity(load *recording.PageLoad) *recording.PageLoad {
	if load == nil {
		return nil
	}
	return &recording.PageLoad{Target: load.Target, Loader: load.Loader, Root: load.Root}
}

func (reservation *loadReservation) evidenceCopy() loadEvidence {
	if reservation == nil {
		return loadEvidence{}
	}
	return loadEvidence{Selection: reservation.evidence.Selection, Load: ownedLoadIdentity(reservation.evidence.Load)}
}

type loadEvent interface{ loadEvent() }
type loadInvalidate struct{ Selection loadSelection }
type loadSnapshotReady struct {
	Generation uint64
	Evidence   loadEvidence
}
type loadCommitDone struct {
	Reservation *loadReservation
	Success     bool
}
type loadSnapshotFailed struct{ Generation uint64 }
type loadClose struct{}

func (loadInvalidate) loadEvent()     {}
func (loadSnapshotReady) loadEvent()  {}
func (loadCommitDone) loadEvent()     {}
func (loadSnapshotFailed) loadEvent() {}
func (loadClose) loadEvent()          {}

type loadEffect interface{ loadEffect() }
type loadCancelSnapshot struct{ Generation uint64 }
type loadPublishPending struct{ Generation uint64 }
type loadCommitSnapshot struct{ Reservation *loadReservation }
type loadPublishConfirmed struct {
	Generation uint64
	Load       *recording.PageLoad
}
type loadPublishUnavailable struct{ Generation uint64 }

func (loadCancelSnapshot) loadEffect()     {}
func (loadPublishPending) loadEffect()     {}
func (loadCommitSnapshot) loadEffect()     {}
func (loadPublishConfirmed) loadEffect()   {}
func (loadPublishUnavailable) loadEffect() {}

// nextLoadScope owns authority transitions only. The owner applies publication
// effects under its mutex; the sole recorder actor executes reserved history
// writes outside that mutex. Superseding a reservation never retracts history.
func nextLoadScope(state loadState, event loadEvent) (loadState, []loadEffect) {
	if state.phase == loadClosed {
		return state, nil
	}
	closeScope := func() (loadState, []loadEffect) {
		next := state
		next.phase, next.load, next.reservation = loadClosed, nil, nil
		return next, []loadEffect{loadCancelSnapshot{Generation: state.generation}, loadPublishUnavailable{Generation: state.generation}}
	}
	unavailable := func() (loadState, []loadEffect) {
		next := state
		next.phase, next.load, next.reservation = loadUnavailable, nil, nil
		return next, []loadEffect{loadPublishUnavailable{Generation: state.generation}}
	}
	switch event := event.(type) {
	case loadInvalidate:
		// Never wrap into a generation that stale evidence could already own.
		if state.generation == math.MaxUint64 {
			return closeScope()
		}
		next := loadState{phase: loadPending, generation: state.generation + 1, selection: event.Selection}
		return next, []loadEffect{loadCancelSnapshot{Generation: state.generation}, loadPublishPending{Generation: next.generation}}
	case loadSnapshotReady:
		if state.phase != loadPending || event.Generation != state.generation {
			return state, nil
		}
		evidence := event.Evidence
		facts := evidence.Selection
		if facts.Session == "" || !utf8.ValidString(facts.Session) || facts.Frame == "" || !utf8.ValidString(facts.Frame) ||
			facts.Session != state.selection.Session || facts.Target != state.selection.Target ||
			(state.selection.Frame != "" && facts.Frame != state.selection.Frame) ||
			evidence.Load.Validate() != nil || evidence.Load.Target != facts.Target {
			return unavailable()
		}
		reservation := &loadReservation{generation: state.generation, evidence: loadEvidence{Selection: facts, Load: ownedLoadIdentity(evidence.Load)}}
		next := state
		next.phase, next.selection, next.reservation = loadCommitting, facts, reservation
		return next, []loadEffect{loadCommitSnapshot{Reservation: reservation}}
	case loadCommitDone:
		if state.phase != loadCommitting || event.Reservation == nil || event.Reservation != state.reservation || event.Reservation.generation != state.generation {
			return state, nil
		}
		if !event.Success {
			return unavailable()
		}
		next := state
		next.phase, next.reservation, next.load = loadConfirmed, nil, ownedLoadIdentity(state.reservation.evidence.Load)
		return next, []loadEffect{loadPublishConfirmed{Generation: state.generation, Load: ownedLoadIdentity(next.load)}}
	case loadSnapshotFailed:
		if state.phase == loadPending && event.Generation == state.generation {
			return unavailable()
		}
	case loadClose:
		return closeScope()
	}
	return state, nil
}
