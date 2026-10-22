// Package remotesession owns the separate, least-capability remote document
// boundary. Its lifecycle decisions never perform IO or grant authority on disk.
package remotesession

// Lifecycle values are private: public CLI/HTTP action decoders cannot express
// console reset. The supervisor alone translates owned completion events.
type remotePhase uint8

const (
	phaseOff remotePhase = iota
	phasePreparing
	phaseOpening
	phaseReady
	phaseStopping
	phaseKilled
	phaseResetting
	phaseBlocked
)

type remoteState struct {
	phase   remotePhase
	attempt uint64
	epoch   uint64
	joined  bool
}
type eventKind uint8

const (
	eventOn eventKind = iota
	eventOff
	eventKill
	eventConsoleReset
	eventArmed
	eventOpened
	eventJoined
	eventOffStored
	eventResetStored
	eventFailure
	eventPoisoned // uncertain persistent authority or unconfirmed cleanup
	eventTailnetUnavailable // expired, revoked or unknown node authentication
)

type remoteEvent struct {
	kind    eventKind
	attempt uint64
}
type effectKind uint8

const (
	effectArm effectKind = iota
	effectOpen
	effectRevoke
	effectClose
	effectStoreOff
	effectStoreKilled
	effectReset
	effectDispose
)

type remoteEffect struct {
	kind    effectKind
	attempt uint64
	epoch   uint64
}

// ReduceRemote returns a value transition and ordered effect descriptions. Arm
// acknowledgement is the only door to opening. Revocation precedes all teardown
// IO; late successful opens always yield disposal rather than adoption.
func ReduceRemote(s remoteState, e remoteEvent) (remoteState, []remoteEffect) {
	var effects []remoteEffect
	emit := func(k effectKind) { effects = append(effects, remoteEffect{k, s.attempt, s.epoch}) }
	if e.kind == eventOpened && e.attempt == s.attempt && s.phase == phaseReady {
		return s, nil
	}
	if e.kind == eventOpened && (e.attempt != s.attempt || s.phase != phaseOpening) {
		return s, []remoteEffect{{kind: effectDispose, attempt: e.attempt, epoch: s.epoch}}
	}
	switch e.kind {
	case eventOn:
		if s.phase == phaseOff {
			if s.attempt == ^uint64(0) || s.epoch == ^uint64(0) {
				s.phase = phaseBlocked
				break
			}
			s.attempt++
			s.epoch++
			s.joined = false
			s.phase = phasePreparing
			emit(effectArm)
		}
	case eventOff:
		if s.phase == phasePreparing || s.phase == phaseOpening || s.phase == phaseReady {
			s.phase = phaseStopping
			emit(effectRevoke)
			emit(effectClose)
		}
	case eventKill:
		if s.phase == phaseBlocked {
			emit(effectRevoke)
			emit(effectClose)
			break
		}
		if s.phase == phaseKilled {
			break
		}
		s.joined = false // persistent kill publication is part of the join fence
		s.phase = phaseKilled
		emit(effectRevoke)
		emit(effectStoreKilled)
		emit(effectClose)
	case eventConsoleReset:
		if s.phase == phaseKilled && s.joined {
			if s.attempt == ^uint64(0) {
				s.phase = phaseBlocked
				break
			}
			s.attempt++
			s.phase = phaseResetting
			emit(effectReset)
		}
	case eventArmed:
		if e.attempt == s.attempt && s.phase == phasePreparing {
			s.phase = phaseOpening
			emit(effectOpen)
		}
	case eventOpened:
		s.phase = phaseReady
	case eventJoined:
		if e.attempt == s.attempt && (s.phase == phaseStopping || s.phase == phaseKilled || s.phase == phaseBlocked) {
			if s.joined {
				break
			}
			s.joined = true
			if s.phase == phaseStopping {
				emit(effectStoreOff)
			}
		}
	case eventOffStored:
		if e.attempt == s.attempt && s.phase == phaseStopping && s.joined {
			s.phase = phaseOff
		}
	case eventResetStored:
		if e.attempt == s.attempt && s.phase == phaseResetting {
			s.phase = phaseOff
			s.joined = true
		}
	case eventFailure, eventTailnetUnavailable:
		if e.attempt != s.attempt {
			break
		}
		switch s.phase {
		case phasePreparing, phaseResetting, phaseStopping:
			s.phase = phaseBlocked
			emit(effectRevoke)
			emit(effectClose)
		case phaseOpening, phaseReady:
			s.phase = phaseKilled
			s.joined = false
			emit(effectRevoke)
			emit(effectStoreKilled)
			emit(effectClose)
		}
	case eventPoisoned:
		if e.attempt == s.attempt {
			s.phase = phaseBlocked
			s.joined = false
			emit(effectRevoke)
			emit(effectClose)
		}
	}
	return s, effects
}
