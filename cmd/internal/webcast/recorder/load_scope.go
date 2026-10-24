package recorder

import (
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"math"
)

type loadSelection struct{ Session, Target, Frame string }
type loadCell struct {
	id, epoch uint64
	selection loadSelection
	attached  bool
	loader    string
	replySeq  uint64
}
type loadCellChange struct {
	frame, loader      string
	replySeq           uint64
	invalidate, detach bool
}

// nextLoadCell changes one supplied cell only; lookup and lifetime belong to
// loadOwner. No pure transition scans or copies the owner's session map.
func nextLoadCell(cell loadCell, change loadCellChange) loadCell {
	if change.detach {
		cell.attached = false
	}
	if change.invalidate {
		if cell.epoch == math.MaxUint64 {
			cell.attached = false
		} else {
			cell.epoch++
		}
	}
	if change.frame != "" {
		cell.selection.Frame = change.frame
	}
	if change.loader != "" {
		cell.loader = change.loader
	}
	if change.replySeq > cell.replySeq {
		cell.replySeq = change.replySeq
	}
	return cell
}

type snapshotIntent struct {
	serial, activation uint64
	cell               *loadCell
	activationStart    bool
	retired            bool // written only by the owner's typed settlement effect
}
type loadAttempt struct {
	intent            *snapshotIntent
	epoch, generation uint64
	selection         loadSelection
}
type loadEvidence struct {
	Selection loadSelection
	Load      *recording.PageLoad
}
type loadReservation struct {
	attempt  *loadAttempt
	evidence loadEvidence
}
type loadState struct {
	generation                       uint64
	selectedCell, selectedActivation uint64
	intendedLive, closing            bool
	active                           *loadAttempt
	reservation                      *loadReservation
}

func ownedLoadIdentity(load *recording.PageLoad) *recording.PageLoad {
	if load == nil {
		return nil
	}
	return &recording.PageLoad{Target: load.Target, Loader: load.Loader, Root: load.Root}
}
func (r *loadReservation) evidenceCopy() loadEvidence {
	return loadEvidence{Selection: r.evidence.Selection, Load: ownedLoadIdentity(r.evidence.Load)}
}

type loadEvent interface{ loadEvent() }
type loadLive struct{ cellID, activation uint64 }
type loadPause struct{}
type loadChanged struct{ cellID uint64 }
type loadStart struct{ intent *snapshotIntent }
type loadSnapshotReady struct {
	attempt  *loadAttempt
	evidence loadEvidence
}
type loadCommitDone struct {
	reservation *loadReservation
	success     bool
}
type loadSnapshotFailed struct{ attempt *loadAttempt }
type loadClose struct{}

func (loadLive) loadEvent()           {}
func (loadPause) loadEvent()          {}
func (loadChanged) loadEvent()        {}
func (loadStart) loadEvent()          {}
func (loadSnapshotReady) loadEvent()  {}
func (loadCommitDone) loadEvent()     {}
func (loadSnapshotFailed) loadEvent() {}
func (loadClose) loadEvent()          {}

type loadEffect interface{ loadEffect() }
type loadCancelSnapshot struct{ attempt *loadAttempt }
type loadPublishPending struct{ generation uint64 }
type loadPublishConfirmed struct {
	generation uint64
	load       *recording.PageLoad
}
type loadPublishUnavailable struct{ generation uint64 }
type loadBeginSnapshot struct{ attempt *loadAttempt }
type loadCommitSnapshot struct{ reservation *loadReservation }
type loadRetireIntent struct{ intent *snapshotIntent }

func (loadCancelSnapshot) loadEffect()     {}
func (loadPublishPending) loadEffect()     {}
func (loadPublishConfirmed) loadEffect()   {}
func (loadPublishUnavailable) loadEffect() {}
func (loadBeginSnapshot) loadEffect()      {}
func (loadCommitSnapshot) loadEffect()     {}
func (loadRetireIntent) loadEffect()       {}

// nextLoadScope grants capture and historical commit authority. A later live
// selection fences UI publication, not earlier ordered activation history.
// The relevant cell is an immutable value supplied by the owner lookup.
func nextLoadScope(state loadState, cell loadCell, event loadEvent) (loadState, []loadEffect) {
	pending := func() (loadState, []loadEffect) {
		if state.closing {
			return state, nil
		}
		if state.generation == math.MaxUint64 {
			state.closing = true
			return state, []loadEffect{loadPublishUnavailable{state.generation}}
		}
		state.generation++
		return state, []loadEffect{loadPublishPending{state.generation}}
	}
	selected := func(a *loadAttempt) bool {
		return a != nil && a.generation != 0 && a.generation == state.generation && a.intent.activation == state.selectedActivation && a.intent.cell.id == state.selectedCell && !state.closing
	}
	switch e := event.(type) {
	case loadLive:
		if state.closing {
			return state, nil
		}
		state.selectedCell = e.cellID
		state.selectedActivation = e.activation
		state.intendedLive = true
		return pending()
	case loadPause:
		state.intendedLive = false
	case loadChanged:
		var effects []loadEffect
		if state.active != nil && state.active.intent.cell.id == e.cellID {
			effects = append(effects, loadCancelSnapshot{state.active})
		}
		if e.cellID == state.selectedCell {
			var p []loadEffect
			state, p = pending()
			effects = append(effects, p...)
		}
		return state, effects
	case loadStart:
		if e.intent == nil || e.intent.retired || !cell.attached || state.active != nil || state.reservation != nil {
			return state, nil
		}
		generation := uint64(0)
		if e.intent.activation == state.selectedActivation && cell.id == state.selectedCell && !state.closing {
			generation = state.generation
		}
		attempt := &loadAttempt{intent: e.intent, epoch: cell.epoch, generation: generation, selection: cell.selection}
		state.active = attempt
		return state, []loadEffect{loadBeginSnapshot{attempt}}
	case loadSnapshotReady:
		a := e.attempt
		if a == nil || state.active != a {
			return state, nil
		}
		state.active = nil
		if !cell.attached || cell.epoch != a.epoch {
			return state, nil
		}
		facts := e.evidence.Selection
		if facts.Session != a.selection.Session || facts.Target != a.selection.Target || facts.Frame == "" || (a.selection.Frame != "" && facts.Frame != a.selection.Frame) || e.evidence.Load.Validate() != nil || e.evidence.Load.Target != facts.Target {
			effects := []loadEffect{loadRetireIntent{a.intent}}
			if selected(a) {
				effects = append(effects, loadPublishUnavailable{state.generation})
			}
			return state, effects
		}
		reservation := &loadReservation{attempt: a, evidence: loadEvidence{Selection: facts, Load: ownedLoadIdentity(e.evidence.Load)}}
		state.reservation = reservation
		return state, []loadEffect{loadCommitSnapshot{reservation}}
	case loadSnapshotFailed:
		a := e.attempt
		if a == nil || state.active != a {
			return state, nil
		}
		state.active = nil
		if cell.attached && cell.epoch != a.epoch {
			return state, nil
		}
		effects := []loadEffect{loadRetireIntent{a.intent}}
		if selected(a) {
			effects = append(effects, loadPublishUnavailable{state.generation})
		}
		return state, effects
	case loadCommitDone:
		r := e.reservation
		if r == nil || state.reservation != r {
			return state, nil
		}
		state.reservation = nil
		effects := []loadEffect{loadRetireIntent{r.attempt.intent}}
		if selected(r.attempt) && cell.attached && cell.epoch == r.attempt.epoch {
			if e.success {
				effects = append(effects, loadPublishConfirmed{state.generation, ownedLoadIdentity(r.evidence.Load)})
			} else {
				effects = append(effects, loadPublishUnavailable{state.generation})
			}
		}
		return state, effects
	case loadClose:
		if !state.closing {
			state.closing = true
			return state, []loadEffect{loadPublishUnavailable{state.generation}}
		}
	}
	return state, nil
}
