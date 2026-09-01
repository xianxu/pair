package wrapcmd

import (
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

const (
	lifecycleWatchdogAfter = 60 * time.Second
	lifecycleGraceAfter    = 250 * time.Millisecond
)

type ObservationKind uint8

const (
	ObservationUnknown ObservationKind = iota
	ObservationUserSubmission
	ObservationWorking
	ObservationStopped
	ObservationNativeCompletion
	ObservationMarkerCompletion
	ObservationTranscriptStarted
	ObservationTranscriptCompletion
	ObservationTranscriptAbort
	ObservationWatchdogExpired
	ObservationGraceExpired
)

type TurnObservation struct {
	Kind    ObservationKind
	TurnID  string
	Message string
	Token   uint64
}

type NotificationLifecycle struct {
	Generation    uint64
	TurnID        string
	Active        bool
	Completed     bool
	ActivitySeen  bool
	GracePending  bool
	WatchdogToken uint64
	GraceToken    uint64
	nextToken     uint64
}

type LifecycleDecision struct {
	Notify        bool
	Message       string
	WatchdogToken uint64
	GraceToken    uint64
}

func Reduce(state NotificationLifecycle, observation TurnObservation) (NotificationLifecycle, LifecycleDecision) {
	var decision LifecycleDecision
	nextToken := func() uint64 {
		state.nextToken++
		return state.nextToken
	}
	open := func(turnID string, activity bool) {
		state.Generation++
		state.TurnID = turnID
		state.Active = true
		state.Completed = false
		state.ActivitySeen = activity
		state.GracePending = false
		state.WatchdogToken = 0
		state.GraceToken = 0
	}
	complete := func(message string) {
		state.Active = false
		state.Completed = true
		state.GracePending = false
		state.WatchdogToken = 0
		state.GraceToken = 0
		decision.Notify = true
		decision.Message = message
	}

	switch observation.Kind {
	case ObservationUserSubmission:
		open("", false)
	case ObservationWorking:
		if !state.Active || state.Completed {
			open("", true)
		} else {
			state.ActivitySeen = true
		}
		state.GracePending = false
		state.GraceToken = 0
		state.WatchdogToken = nextToken()
		decision.WatchdogToken = state.WatchdogToken
	case ObservationTranscriptStarted:
		if observation.TurnID == "" {
			break
		}
		if state.Active && !state.Completed && state.TurnID == "" {
			state.TurnID = observation.TurnID
		} else if state.TurnID != observation.TurnID {
			open(observation.TurnID, true)
		}
		state.ActivitySeen = true
		state.GracePending = false
		state.GraceToken = 0
		state.WatchdogToken = nextToken()
		decision.WatchdogToken = state.WatchdogToken
	case ObservationStopped:
		if state.Active && state.ActivitySeen {
			state.GracePending = true
			state.GraceToken = nextToken()
			decision.GraceToken = state.GraceToken
		}
	case ObservationNativeCompletion, ObservationMarkerCompletion:
		if state.Active && !state.Completed {
			message := observation.Message
			if message == "" {
				message = "agent finished working"
			}
			complete(message)
		}
	case ObservationTranscriptCompletion:
		if state.Active && !state.Completed && observation.TurnID != "" && observation.TurnID == state.TurnID {
			message := observation.Message
			if message == "" {
				message = "agent finished working"
			}
			complete(message)
		}
	case ObservationTranscriptAbort:
		if state.Active && !state.Completed && observation.TurnID != "" && observation.TurnID == state.TurnID {
			message := observation.Message
			if message == "" {
				message = "agent stopped with an error"
			}
			complete(message)
		}
	case ObservationWatchdogExpired:
		if state.Active && state.ActivitySeen && observation.Token != 0 && observation.Token == state.WatchdogToken {
			complete("agent stopped working")
		}
	case ObservationGraceExpired:
		if state.Active && state.GracePending && observation.Token != 0 && observation.Token == state.GraceToken {
			complete("agent stopped working")
		}
	}
	return state, decision
}

func (p *proxy) publishLifecycleObservation(observation TurnObservation) {
	if p.lifecycleEvents == nil {
		return
	}
	select {
	case p.lifecycleEvents <- observation:
	default:
		p.debug("LIFECYCLE-drop", "event channel full")
	}
}

// processLifecycleObservation is called only by the master proxy loop (or by
// direct single-threaded tests). The loop owns the sole lifecycle timer and
// feeds its immutable expiry token back through this reducer.
func (p *proxy) processLifecycleObservation(observation TurnObservation) {
	state, decision := Reduce(p.notificationLifecycle, observation)
	p.notificationLifecycle = state
	if decision.Notify {
		p.emitOuter(decision.Message)
	}
	p.syncLifecycleTimer()
}

func (p *proxy) processLifecycleRecord(record sessionwatch.LifecycleRecord) {
	observation := TurnObservation{TurnID: record.TurnID, Message: record.Message}
	switch record.Outcome {
	case "started":
		observation.Kind = ObservationTranscriptStarted
	case "completed":
		observation.Kind = ObservationTranscriptCompletion
	case "aborted", "error":
		observation.Kind = ObservationTranscriptAbort
	default:
		p.debug("LIFECYCLE-skip", "unknown transcript outcome "+record.Outcome)
		return
	}
	p.processLifecycleObservation(observation)
}

func (p *proxy) syncLifecycleTimer() {
	if p.lifecycleTimer == nil {
		return
	}
	state := p.notificationLifecycle
	switch {
	case state.Active && state.GracePending && state.GraceToken != 0:
		p.resetLifecycleTimer(ObservationGraceExpired, state.GraceToken, lifecycleGraceAfter)
	case state.Active && state.ActivitySeen && state.WatchdogToken != 0:
		p.resetLifecycleTimer(ObservationWatchdogExpired, state.WatchdogToken, lifecycleWatchdogAfter)
	default:
		p.stopLifecycleTimer()
	}
}

func (p *proxy) resetLifecycleTimer(kind ObservationKind, token uint64, after time.Duration) {
	p.stopLifecycleTimer()
	p.lifecycleTimerKind = kind
	p.lifecycleTimerToken = token
	p.lifecycleTimer.Reset(after)
}

func (p *proxy) stopLifecycleTimer() {
	if p.lifecycleTimer == nil {
		return
	}
	if !p.lifecycleTimer.Stop() {
		select {
		case <-p.lifecycleTimer.C:
		default:
		}
	}
	p.lifecycleTimerKind = ObservationUnknown
	p.lifecycleTimerToken = 0
}
