package wrapcmd

import (
	"math"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// The single stdin writer owns this lease. Only advanceAutomaticInput changes
// its state; input admission independently fences human input and every paste.
type automaticInputPhase uint8

const (
	automaticInputIdle automaticInputPhase = iota
	automaticInputPeer
	automaticInputAwaitingEmpty
	automaticInputAlerted
	automaticInputNeedsOperator
)

type automaticInputTransaction struct {
	phase        automaticInputPhase
	renderFence  uint64
	submitted    bool
	residueCause string
	blockedSince time.Time
	sequence     uint64 // lifetime high-water, retained across successful transactions
}
type automaticInputEventKind uint8

const (
	automaticBegin automaticInputEventKind = iota
	automaticFinish
	automaticObserve
	automaticQuarantine
)

type automaticInputEvent struct {
	kind                      automaticInputEventKind
	owner                     automaticInputPhase
	epoch                     uint64
	confirmedEmpty, submitted bool
	now                       time.Time
	composer                  PeerComposerState
	complete, eligible        bool
	reportCause               string
	wait                      time.Duration
	recoveryAfter             time.Duration // negative disables recovery in root-cause tests
}
type automaticInputDecision struct {
	available bool
	cause     string
	report    *couchmessage.AutomaticInputStall
}

func advanceAutomaticInput(tx automaticInputTransaction, e automaticInputEvent) (automaticInputTransaction, automaticInputDecision) {
	clear := func() { tx = automaticInputTransaction{sequence: tx.sequence} }
	d := automaticInputDecision{}
	switch e.kind {
	case automaticBegin:
		if tx.phase == automaticInputIdle {
			tx.phase = e.owner
		}
	case automaticFinish:
		if tx.phase == e.owner {
			if e.confirmedEmpty && e.submitted {
				clear()
			} else {
				tx = automaticInputTransaction{phase: automaticInputAwaitingEmpty, renderFence: e.epoch, submitted: e.submitted, sequence: tx.sequence}
			}
		}
	case automaticQuarantine:
		if tx.phase == e.owner {
			tx.phase, tx.renderFence, tx.submitted = automaticInputNeedsOperator, e.epoch, false
			tx.residueCause = e.reportCause
			if tx.sequence < math.MaxUint64 {
				tx.sequence++
				d.report = &couchmessage.AutomaticInputStall{Sequence: tx.sequence, Cause: e.reportCause, WaitMillis: min(max(e.wait.Milliseconds(), 0), couchmessage.MaxAutomaticInputStallWaitMillis), Fence: e.epoch, Epoch: e.epoch}
			}
		}
	case automaticObserve:
		if tx.phase == automaticInputAwaitingEmpty || tx.phase == automaticInputAlerted || tx.phase == automaticInputNeedsOperator {
			if e.complete && e.epoch > tx.renderFence && e.composer == PeerComposerEmpty {
				clear()
				break
			}
			d.cause = "no post-completion render"
			if !e.complete {
				d.cause = "incomplete terminal frame"
			} else if e.composer == PeerComposerUnknown {
				d.cause = "composer unrecognized"
			} else if e.composer == PeerComposerOccupied {
				d.cause = "composer occupied"
			}
			if tx.phase == automaticInputNeedsOperator {
				break
			}
			eligible := e.eligible && e.complete && e.composer != PeerComposerOccupied
			if !eligible {
				tx.blockedSince = time.Time{}
				break
			}
			if e.recoveryAfter <= 0 || tx.phase == automaticInputAlerted {
				break
			}
			if tx.blockedSince.IsZero() {
				tx.blockedSince = e.now
				break
			}
			if e.now.Before(tx.blockedSince.Add(e.recoveryAfter)) {
				break
			}
			// Exhaustion never wraps into a duplicate occurrence or releases input.
			if tx.sequence == math.MaxUint64 {
				d.cause = "tripwire sequence exhausted"
				break
			}
			tx.sequence++
			cause := couchmessage.AutomaticInputStallNoRender
			if e.composer == PeerComposerUnknown {
				cause = couchmessage.AutomaticInputStallUnrecognizedComposer
			}
			report := couchmessage.AutomaticInputStall{Sequence: tx.sequence, Cause: cause, WaitMillis: min(e.now.Sub(tx.blockedSince).Milliseconds(), couchmessage.MaxAutomaticInputStallWaitMillis), Fence: tx.renderFence, Epoch: e.epoch, Released: tx.submitted && e.composer == PeerComposerEmpty}
			d.report = &report
			if report.Released {
				clear()
			} else {
				tx.phase = automaticInputAlerted
			}
		}
	}
	d.available = tx.phase == automaticInputIdle || tx.phase == e.owner
	return tx, d
}

const automaticInputRecoveryAfter = 30 * time.Second

// Called with inputAdmission and peer.mu held. Only captures evidence and
// enqueues exceptional work; diagnostic and socket IO run in the runtime worker.
func (p *proxy) automaticInputAvailable(owner automaticInputPhase) bool {
	if p.automaticInput.phase != automaticInputAwaitingEmpty && p.automaticInput.phase != automaticInputAlerted && p.automaticInput.phase != automaticInputNeedsOperator {
		return p.automaticInput.phase == automaticInputIdle || p.automaticInput.phase == owner
	}
	d := p.peer
	e := automaticInputEvent{kind: automaticObserve, owner: owner, epoch: p.automaticRenderEpoch.Load(), now: d.now(), recoveryAfter: p.automaticRecoveryAfter}
	if e.recoveryAfter == 0 {
		e.recoveryAfter = automaticInputRecoveryAfter
	}
	var snapshot terminalSnapshot
	if p.terminal != nil {
		s := p.terminal.Snapshot()
		snapshot = s
		e.composer = peerComposerState(p.agentBasename, s)
		e.complete = s.Complete && (d.outputPending == 0 || p.terminal.FrameComplete())
	}
	e.eligible = !d.exited && !d.interrupted && d.now().Before(d.deadlineLocked()) && p.childAcceptsPaste() && !p.turnActive.Load() && !p.pickerActive.Load() && !d.image && !d.holding && !d.inputBuffered && d.replies.inFlight == 0 && len(d.replies.input) == 0 && (d.lastInput.IsZero() || !d.now().Before(d.lastInput.Add(time.Second)))
	return p.applyAutomaticInputEvent(e, snapshot).available
}
func (p *proxy) beginAutomaticInput(owner automaticInputPhase) {
	p.automaticInput, _ = advanceAutomaticInput(p.automaticInput, automaticInputEvent{kind: automaticBegin, owner: owner})
}
func (p *proxy) finishAutomaticInput(owner automaticInputPhase, confirmedEmpty bool) {
	p.automaticInput, _ = advanceAutomaticInput(p.automaticInput, automaticInputEvent{kind: automaticFinish, owner: owner, epoch: p.automaticRenderEpoch.Load(), confirmedEmpty: confirmedEmpty, submitted: p.peer != nil && p.peer.state.Outcome() == couchmessage.Submitted})
}

func (p *proxy) applyAutomaticInputEvent(e automaticInputEvent, snapshot terminalSnapshot) automaticInputDecision {
	var decision automaticInputDecision
	p.automaticInput, decision = advanceAutomaticInput(p.automaticInput, e)
	p.automaticInputCause = decision.cause
	if decision.report != nil && p.automaticReport != nil {
		p.automaticReport(*decision.report, snapshot)
	}
	return decision
}
