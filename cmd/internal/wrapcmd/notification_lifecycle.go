package wrapcmd

import (
	"fmt"
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
	// ObservationIdleExpired is the always-armed floor under attention (#171):
	// the agent's byte stream went quiet for the idle interval while a turn was
	// still open and unreported. It is an ALERT, not a completion — see the
	// Reduce case for why the turn stays open.
	ObservationIdleExpired
	// ObservationBareReturn is a plain Enter that reached the agent as a bare
	// CR because the composer gate reported inactive or unknown (#171). It is
	// the only submission signal on that path, so it opens a turn when none is
	// open; inside an open turn (answering a menu) it is a no-op.
	ObservationBareReturn
	// observationKindCount is one past the last kind. Enumerations over the
	// kind space derive their bound from it, so a kind added above is covered
	// without editing the consumer (BR-4).
	observationKindCount
)

// defaultIdleMessage is used when the caller supplies no interval-formatted
// text. It states what was observed — silence — and claims nothing about
// whether the agent finished, since the floor cannot know that.
const defaultIdleMessage = "no agent output"

// idleAlertMessage renders the floor's interval honestly at any scale — the
// production interval is seconds, but tests run it in milliseconds and
// "%.0fs" reported those as "0s".
func idleAlertMessage(interval time.Duration) string {
	if interval < time.Second {
		return fmt.Sprintf("%s for %dms", defaultIdleMessage, interval.Milliseconds())
	}
	return fmt.Sprintf("%s for %.0fs", defaultIdleMessage, interval.Seconds())
}

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
	// IdleNotified records that this turn already raised the idle floor, so
	// the alert fires at most once per turn rather than every interval while
	// the operator is away. open() clears it, so each new turn is armed again.
	IdleNotified bool
	// IdleToken identifies the turn's idle epoch. It is minted once per turn
	// (not once per chunk): output resets the deadline's duration but not its
	// epoch, so a timer expiry that raced a completion is rejected by token.
	IdleToken uint64
	nextToken uint64
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
		state.IdleNotified = false
		state.IdleToken = nextToken()
	}
	complete := func(message string) {
		state.Active = false
		state.Completed = true
		state.GracePending = false
		state.WatchdogToken = 0
		state.GraceToken = 0
		state.IdleToken = 0
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
	case ObservationIdleExpired:
		// Deliberately NOT complete(). A 60s silence does not know the turn
		// ended; asserting it would set the Completed tombstone and swallow
		// the agent's real end-of-turn, pulling the operator in early and then
		// never telling them it actually finished. So the alert fires, the
		// turn stays open, and a later completion still notifies. This does
		// not double-notify a healthy turn: a working pane is not byte-quiet
		// for the interval, so the timer never expires on one (#171 Log).
		if state.Active && !state.Completed && !state.IdleNotified &&
			observation.Token != 0 && observation.Token == state.IdleToken {
			state.IdleNotified = true
			message := observation.Message
			if message == "" {
				message = defaultIdleMessage
			}
			decision.Notify = true
			decision.Message = message
		}
	case ObservationBareReturn:
		// A bare CR that reached the agent is operator activity aimed at it.
		// With no turn open it opens one. Inside an open turn it must not
		// reset the turn's identity — but it DOES re-arm a floor that already
		// fired: answering a menu starts a new attention window, and without
		// this the turn that alerted is left with no floor at all, while a
		// mid-turn Alt+Enter (which re-opens) would have re-armed (BR-14).
		if !state.Active || state.Completed {
			open("", false)
		} else if state.IdleNotified {
			state.IdleNotified = false
			state.IdleToken = nextToken()
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
	p.syncIdleTimer()
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

// applyIdleExpiry handles one idle-timer expiry. `token` is the epoch that
// actually expired and MUST be read by the caller before this runs: the drain
// below reduces queued observations, and an opener among them mints a new turn
// with a new IdleToken (advancing p.idleTimerToken in lockstep). Reading the
// token after the drain would match that brand-new turn and alert against it
// microseconds after it opened, consuming its floor (BR-2).
//
// The drain itself mirrors the chunk branch: a submission or completion already
// published but not yet reduced must be applied before the expiry is.
func (p *proxy) applyIdleExpiry(token uint64) {
	for {
		select {
		case observation := <-p.lifecycleEvents:
			p.processLifecycleObservation(observation)
			continue
		default:
		}
		break
	}
	// BR-6: the watchdog/grace deadline and this one can be co-ready, and Go
	// picks between select cases at random. Give the lifecycle timer
	// precedence so "agent stopped working" is not displaced by the less
	// informative silence alert (the 0.5s emit limiter would drop the second).
	if p.lifecycleTimer != nil {
		select {
		case <-p.lifecycleTimer.C:
			kind, lifecycleToken := p.lifecycleTimerKind, p.lifecycleTimerToken
			p.processLifecycleObservation(TurnObservation{Kind: kind, Token: lifecycleToken})
		default:
		}
	}
	message := idleAlertMessage(p.idleS)
	p.debug("IDLE", message)
	p.traceWrap("idle", map[string]any{"idle_s": p.idleS.Seconds()})
	p.processLifecycleObservation(TurnObservation{
		Kind: ObservationIdleExpired, Token: token, Message: message,
	})
}

// syncIdleTimer ties the idle floor's arming to lifecycle state, exactly as
// syncLifecycleTimer does for the watchdog and grace deadlines. The floor is
// armed only while a turn is open, unreported, and has not already raised its
// one alert; anything that closes or reports the turn disarms it. This is why
// the floor no longer depends on a `idleFired` latch in the master loop: a new
// turn re-arms it because open() clears IdleNotified and mints a fresh epoch.
func (p *proxy) syncIdleTimer() {
	// idleS is fixed at startup, so a disabled floor never armed the timer and
	// has nothing to stop — return before touching it, keeping the per-chunk
	// cost of the always-on path at zero when the operator has opted out.
	if p.idleTimer == nil || p.idleS <= 0 {
		return
	}
	state := p.notificationLifecycle
	if !(state.Active && !state.Completed && !state.IdleNotified && state.IdleToken != 0) {
		p.stopIdleTimer()
		return
	}
	if p.idleTimerToken != state.IdleToken {
		p.resetIdleTimer() // a new turn: arm the floor on its epoch
	}
}

// bumpIdleDeadline pushes an armed floor's deadline out on agent OUTPUT. Only
// output moves it: the alert claims byte-silence, so a lifecycle record that
// arrives with no pane output must not extend the window it describes (BR-10).
func (p *proxy) bumpIdleDeadline() {
	if p.idleTimer == nil || p.idleS <= 0 || p.idleTimerToken == 0 {
		return
	}
	p.resetIdleTimer()
}

// resetIdleTimer restarts the deadline without changing the epoch: IdleToken is
// minted once per turn by the reducer, so output pushes the deadline out but
// cannot make a racing expiry from this same turn look stale.
func (p *proxy) resetIdleTimer() {
	if p.idleTimer == nil {
		return
	}
	// Stop+drain+reset is safe here because only the master goroutine reads
	// idleTimer.C.
	drainStop(p.idleTimer)
	p.idleTimerToken = p.notificationLifecycle.IdleToken
	p.idleTimer.Reset(p.idleS)
}

func (p *proxy) stopIdleTimer() {
	if p.idleTimer == nil {
		return
	}
	drainStop(p.idleTimer)
	p.idleTimerToken = 0
}

// drainStop stops a timer and clears any expiry already sitting in its channel.
// Safe only on the master goroutine, the sole reader of these channels.
func drainStop(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
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
	drainStop(p.lifecycleTimer)
	p.lifecycleTimerKind = ObservationUnknown
	p.lifecycleTimerToken = 0
}
