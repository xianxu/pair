package couchcmd

import "fmt"

type profilePhase uint8

const (
	profileWarmup profilePhase = iota
	profileAcquiring
	profileCapturing
	profileReleasingUnowned
	profileReleasingOwned
	profileFinished
)

type profileEventKind uint8

const (
	profileWarmupElapsed profileEventKind = iota
	profileCancel
	profileAcquired
	profileAcquiredCancelled
	profileAcquireFailed
	profileCaptureElapsed
	profileReleased
)

type profileEvent struct {
	kind profileEventKind
	err  error
}
type profileEffect uint8

const (
	profileNoEffect profileEffect = iota
	profileAcquire
	profileReleaseUnowned
	profileReleaseOwned
	profileComplete
)

// ProfileLifecycle is the pure ownership state of one finite diagnostic window.
// Its zero value is armed; only StepProfile changes authoritative state.
type ProfileLifecycle struct {
	phase     profilePhase
	cancelled bool
	failure   error
}

// StepProfile returns the only admitted transition and its synchronous effect.
// In particular, failed acquisition never authorizes stopping another profiler.
func StepProfile(s ProfileLifecycle, e profileEvent) (ProfileLifecycle, profileEffect, error) {
	if s.phase == profileFinished {
		return s, profileNoEffect, nil
	}
	next := s
	effect := profileNoEffect
	switch {
	case s.phase == profileWarmup && e.kind == profileWarmupElapsed:
		next.phase, effect = profileAcquiring, profileAcquire
	case (s.phase == profileWarmup || s.phase == profileAcquiring) && e.kind == profileCancel:
		next.phase, next.cancelled, effect = profileReleasingUnowned, true, profileReleaseUnowned
	case s.phase == profileAcquiring && e.kind == profileAcquired:
		next.phase = profileCapturing
	case s.phase == profileAcquiring && e.kind == profileAcquiredCancelled:
		next.phase, next.cancelled, effect = profileReleasingOwned, true, profileReleaseOwned
	case s.phase == profileAcquiring && e.kind == profileAcquireFailed && e.err != nil:
		next.phase, next.failure, effect = profileReleasingUnowned, e.err, profileReleaseUnowned
	case s.phase == profileCapturing && (e.kind == profileCaptureElapsed || e.kind == profileCancel):
		next.phase, next.cancelled, effect = profileReleasingOwned, e.kind == profileCancel, profileReleaseOwned
	case (s.phase == profileReleasingOwned || s.phase == profileReleasingUnowned) && e.kind == profileReleased:
		next.phase, effect = profileFinished, profileComplete
		if next.failure == nil {
			next.failure = e.err
		}
	default:
		return s, profileNoEffect, fmt.Errorf("profile: invalid event %d in state %d", e.kind, s.phase)
	}
	return next, effect, nil
}
