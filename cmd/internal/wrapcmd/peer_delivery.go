package wrapcmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// peerDelivery is the receiver's observation mailbox. Only translateStdinFrom
// drives its reducer or writes to the PTY. The admission mutex fences reader
// observations against automatic writes, including observations not rendered yet.
type peerDelivery struct {
	drainProxy      *proxy
	drainInput      uint64
	drainInterrupt  *wrapperDrainInterrupt
	drainInterrupts map[string]*wrapperDrainInterrupt
	drainRevokes    map[string]couchmessage.DrainRevokeResult
	drive           *driveMailbox
	recordBatch     func([]couchmessage.Message)
	batch           *peerDeliveryBatch
	recordTurn      func(couchmessage.Message) // submitted turns into the Pair log (pair#618); nil without a log path
	driveProbe      func() agentdrive.Screen
	mu              sync.Mutex
	binding         couchmessage.Binding
	now             func() time.Time
	lastActivity    time.Time
	sequence        uint64
	lastInput       time.Time
	inputBuffered   bool
	submissions     uint64
	pasteSequence   uint64
	// The post-paste half of a delivery (pair#427). deliverBy is the paste
	// time plus DeliveryTimeout; Message.Deadline bounds the initial commit.
	pastedAt                time.Time
	submittedAt             time.Time
	pasteEpoch, submitEpoch uint64
	deliverBy               time.Time
	sawOccupied             bool   // the composer read occupied after the paste
	submitSequence          uint64 // output sequence at the submit write
	turnAtSubmit            bool   // a turn was already open when we submitted
	outputPending           int
	reservation             string
	reservationUntil        time.Time
	current                 couchmessage.Receipt
	state                   couchmessage.PeerDeliveryState
	image                   bool
	interrupted             bool
	released                bool // broker cancellation, never operator input
	// What held this half of the delivery, and for how long (pair#438):
	// each dispatch charges the time since waitAt to the standing Detail.
	waits  map[string]time.Duration
	waitAt time.Time
	// A draft-pane paste held while a delivery is in flight (pair#428).
	holding   bool
	holdUntil time.Time
	// The operator's view of the pane (pair#428, peer_viewport.go);
	// viewProbe is nil outside zellij-observable setups and then never gates.
	view      peerView
	viewProbe func() bool
	exited    bool
	replies   terminalReplies
	wake      chan struct{}
	notices   chan string
	// session receives every observation change (#365); nil outside Couch.
	// Both methods only record state and wake a sender: no IO on these paths.
	session peerSessionSink
	// recent remembers the last deliveries after current moves on, so a
	// status query after a broker restart still has an answer and a reused
	// ID is refused rather than pasted again (#365).
	recent couchmessage.RecentDeliveries
	// Settle state (#421, peer_settle.go). settleProbe is nil when the
	// wrapper cannot judge (no terminal model), and then nothing settles.
	settled     bool
	settleGen   uint64 // advanced by every source transition (armSettleLocked)
	settleProbe func() wrapperSettleState
	settleTimer settleTimer
	afterFunc   func(time.Duration, func()) settleTimer
	// tailProbe renders the agent pane's tail (pair#425); nil without a
	// terminal model, and then a tail request says so.
	tailProbe func(lines int) couchmessage.Tail
	// pickerTitle is the title of the picker holding the current message
	// (pair#465), read once for pickerTitleEpisode; "" when the screen showed
	// none.
	pickerTitle        string
	pickerTitleEpisode uint64
}

type peerSessionSink interface {
	Update(couchmessage.Observation)
	Submit(couchmessage.Observation)
	Settle(bool)
	BackgroundWork(*bool)
}

// observationLocked is the wrapper's current evidence; d.mu must be held.
func (d *peerDelivery) observationLocked() couchmessage.Observation {
	return couchmessage.Observation{LastActivity: d.lastActivity, Sequence: d.sequence, Submission: d.submissions}
}

// publishUnsettled tells the session it is no longer settled, before the
// activity frame that caused it.
func (d *peerDelivery) publishUnsettled(sink peerSessionSink, changed bool) {
	if changed && sink != nil {
		sink.Settle(false)
	}
}

// publish hands the session an observation taken under d.mu, after release.
func (d *peerDelivery) publish(sink peerSessionSink, o couchmessage.Observation, submit bool) {
	switch {
	case sink == nil:
	case submit:
		sink.Submit(o)
	default:
		sink.Update(o)
	}
}

func newPeerDelivery(binding couchmessage.Binding, now func() time.Time) *peerDelivery {
	return &peerDelivery{drive: newDriveMailbox(), binding: binding, now: now, lastActivity: now(), wake: make(chan struct{}, 1), notices: make(chan string, 1)}
}

// PeerSubmitDelay is the fixed pause between the paste and the submit key,
// the draft pane's rule (nvim/draft_send.lua) with margin. Nothing reads how
// the agent rendered the paste: pair renders, never classifies (pair#427).
const PeerSubmitDelay = agentdrive.TextSubmitDelay

// peerPastedPoll paces a delivery past its paste, so the submit delay and the
// confirmation are not quantized to the queued poll's second.
const peerPastedPoll = 50 * time.Millisecond

// Owned by the input writer. Idle wrappers allocate no polling timer.
type peerDeliveryPoll struct {
	ticker *time.Ticker
	every  time.Duration
}

func (p *peerDeliveryPoll) stop() {
	if p.ticker != nil {
		p.ticker.Stop()
		p.ticker = nil
	}
}

func (p *peerDeliveryPoll) update(d *peerDelivery) <-chan time.Time {
	active, pasted := false, false
	if d != nil {
		d.mu.Lock()
		active = d.current.Message.ID != "" && !d.current.Status.Terminal()
		if d.drive != nil && d.drive.active != nil {
			active = true
			pasted = true
		}
		pasted = pasted || d.current.Status == couchmessage.Delivering
		d.mu.Unlock()
	}
	if !active {
		p.stop()
		return nil
	}
	every := time.Second
	if pasted {
		every = peerPastedPoll
	}
	switch {
	case p.ticker == nil:
		p.ticker = time.NewTicker(every)
	case p.every != every:
		p.ticker.Reset(every) // the same channel: a caller's select stays valid
	}
	p.every = every
	return p.ticker.C
}
func (d *peerDelivery) signal() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}
func (d *peerDelivery) observeOutput(data []byte) {
	d.mu.Lock()
	d.sequence++
	d.outputPending++
	d.lastActivity = d.now()
	d.replies.observeQueries(data)
	o, sink, unsettled := d.observationLocked(), d.session, d.unsettleLocked()
	d.mu.Unlock()
	d.publishUnsettled(sink, unsettled)
	d.publish(sink, o, false)
	d.signal()
}
func (d *peerDelivery) outputForwarded() {
	d.mu.Lock()
	if d.outputPending > 0 {
		d.outputPending--
	}
	d.mu.Unlock()
	d.signal()
}

// admitInput observes one stdin chunk before it is forwarded. A draft-pane
// send that arrives while a delivery is between its paste and confirmation
// is held instead (pair#428): held is true, and the input writer forwards
// the chunk once releaseHold allows, through admitReleased.
func (d *peerDelivery) admitInput(data []byte) (human, held bool) {
	d.mu.Lock()
	held = d.holdLocked(data)
	d.mu.Unlock()
	if held {
		d.signal()
		return false, true
	}
	return d.admitReleased(data), false
}

// PeerDraftHold bounds how long an operator paste waits for a delivery in
// flight: about the draft pane's own paste-to-submit rhythm, with margin.
const PeerDraftHold = time.Second

func (d *peerDelivery) holdLocked(data []byte) bool {
	if d.holding {
		return true // everything after a held chunk queues behind it
	}
	for bytes.HasPrefix(data, []byte("\x1b[I")) || bytes.HasPrefix(data, []byte("\x1b[O")) {
		data = data[3:]
	}
	if d.current.Status != couchmessage.Delivering || !bytes.HasPrefix(data, []byte(workbenchshortcut.PasteStart)) {
		return false
	}
	d.holding, d.holdUntil = true, d.now().Add(PeerDraftHold)
	return true
}

// releaseHold reports whether held input may go: the delivery left its
// paste-to-confirm half, or the hold reached its bound.
func (d *peerDelivery) releaseHold() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.holding || d.current.Status != couchmessage.Delivering || !d.now().Before(d.holdUntil)
}

// endHold is called by the input writer once every held chunk is forwarded.
func (d *peerDelivery) endHold() {
	d.mu.Lock()
	d.holding = false
	d.mu.Unlock()
}

func (d *peerDelivery) admitReleased(data []byte) bool {
	d.mu.Lock()
	var sink peerSessionSink
	var o couchmessage.Observation
	unsettled := false
	defer func() {
		d.mu.Unlock()
		d.publishUnsettled(sink, unsettled)
		d.publish(sink, o, false)
	}()
	d.replies.inFlight++
	human := d.replies.operatorKeystrokes(data)
	if human {
		o, sink, unsettled = d.operatorInputLocked()
	}
	if bytes.Contains(data, []byte{0x16}) {
		d.image = true
		d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
	}
	d.signal()
	return human
}

// operatorInputLocked records a keystroke (d.mu held).
func (d *peerDelivery) operatorInputLocked() (couchmessage.Observation, peerSessionSink, bool) {
	d.drainInput++
	d.cancelDriveLocked("operator input")
	d.lastInput = d.now()
	d.sequence++
	d.lastActivity = d.now()
	d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
	return d.observationLocked(), d.session, d.unsettleLocked()
}

// flushPartialInput resolves a frame left unfinished when the input writer's
// flush interval passed with no continuation: a lone ESC is a keystroke, and
// must not hold a delivery at "incomplete terminal input" until the next byte.
func (d *peerDelivery) flushPartialInput() {
	d.mu.Lock()
	if !d.replies.flushPartial() {
		d.mu.Unlock()
		return
	}
	o, sink, unsettled := d.operatorInputLocked()
	d.mu.Unlock()
	d.publishUnsettled(sink, unsettled)
	d.publish(sink, o, false)
	d.signal()
}

// inputForwarded reports whether a partial input frame is still pending.
func (d *peerDelivery) inputForwarded(human, buffered bool) bool {
	d.mu.Lock()
	d.replies.inFlight--
	d.inputBuffered = buffered
	if human {
		d.lastInput = d.now()
	}
	partial := len(d.replies.input) > 0
	d.mu.Unlock()
	d.signal()
	return partial
}
func (d *peerDelivery) admitImage() {
	d.mu.Lock()
	d.cancelDriveLocked("operator image input")
	d.image = true
	d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
	d.sequence++
	d.lastActivity = d.now()
	o, sink, unsettled := d.observationLocked(), d.session, d.unsettleLocked()
	d.mu.Unlock()
	d.publishUnsettled(sink, unsettled)
	d.publish(sink, o, false)
	d.signal()
}

// humanSubmit is a genuine operator submission: the broker replenishes the
// inbound allowance once per submission generation.
func (d *peerDelivery) humanSubmit() {
	d.mu.Lock()
	d.image = false
	d.sequence++
	d.submissions++
	d.lastActivity = d.now()
	o, sink, unsettled := d.observationLocked(), d.session, d.unsettleLocked()
	d.mu.Unlock()
	d.publishUnsettled(sink, unsettled)
	d.publish(sink, o, true)
}

// retained is the receipt this wrapper holds for id: current or recent.
func (d *peerDelivery) retained(id string) (couchmessage.Receipt, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.retainedLocked(id)
}
func (d *peerDelivery) retainedLocked(id string) (couchmessage.Receipt, bool) {
	if id != "" {
		for _, m := range d.membersLocked() {
			if m.ID == id {
				r := d.current
				r.Message = m
				return r, true
			}
		}
	}
	return d.recent.Get(id)
}

func (d *peerDelivery) reserve(id string, sequence uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, revoked := d.drainRevokes[id]; revoked {
		return errors.New("drain control revoked")
	}
	if r, ok := d.retainedLocked(id); ok {
		return &couchmessage.AlreadyCommittedError{Receipt: r}
	}
	if d.reservation != "" && !d.now().Before(d.reservationUntil) && (d.current.Message.ID == "" || d.current.Status.Terminal()) {
		d.reservation = ""
	}
	if id == "" || d.exited || d.sequence != sequence || d.reservation != "" || (d.current.Message.ID != "" && !d.current.Status.Terminal()) {
		return couchmessage.ErrRecipientBusy
	}
	d.reservation = id
	d.reservationUntil = d.now().Add(couchmessage.AdmissionTimeout)
	return nil
}
func (d *peerDelivery) enqueue(m couchmessage.Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.enqueueLocked(m)
}
func (d *peerDelivery) enqueueLocked(m couchmessage.Message) error {
	if err := d.validateDrainMessageLocked(m); err != nil {
		return err
	}
	if _, revoked := d.drainRevokes[m.ID]; revoked {
		return errors.New("drain control revoked")
	}
	if m.To != d.binding || m.ID != d.reservation || !d.now().Before(d.reservationUntil) || !d.now().Before(m.Deadline) {
		return errors.New("peer reservation expired or identity changed")
	}
	if err := couchmessage.ValidateBody(m.Body); err != nil {
		return err
	}
	if _, ok := d.retainedLocked(m.ID); ok {
		return errors.New("delivery already committed")
	}
	for _, member := range d.membersLocked() {
		r := d.current
		r.Message = member
		d.recent.Add(r)
	}
	d.batch = nil
	d.current = couchmessage.Receipt{Message: m, Status: couchmessage.Queued}
	d.state = couchmessage.PeerDeliveryState{}
	d.interrupted = false
	d.released = false
	d.waits, d.waitAt = nil, time.Time{}
	d.signal()
	return nil
}
func (d *peerDelivery) receipt() couchmessage.Receipt {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.current
}
func peerEnvelope(m couchmessage.Message) string {
	return couchmessage.Envelope(m)
}

func (p *proxy) peerComposerSubmission() bool {
	if p.peer == nil || p.terminal == nil || p.pickerActive.Load() {
		return false
	}
	_, known := peerComposerText(p.agentBasename, p.terminal.Snapshot())
	return known
}

// dispatchPeer is called only by the existing PTY input writer. A connected
// receiver may wait before paste; input effects after paste share one bounded
// submit/confirmation window.
func (p *proxy) dispatchPeer(out io.Writer) {
	p.executeDrainInterrupt(out)
	p.dispatchAgentDrive(out)
	d := p.peer
	if d == nil {
		return
	}
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.current.Message.ID == "" {
		return
	}
	if p.rejectStaleDrainLocked() {
		return
	}
	// Normal render wakes reconcile terminal quarantine without idle polling.
	automaticAvailable := p.automaticInputAvailable(automaticInputPeer)
	if d.current.Status.Terminal() {
		return
	}
	d.chargeWaitLocked()
	// Only a full composer reading opens or closes an operator hold; a wait
	// that can't judge it (input settling, forwarded input) leaves it as is.
	judged, operatorHold := false, false
	defer func() {
		if judged {
			d.operatorHoldLocked(operatorHold)
		}
	}()
	event := couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerComposerObserved}
	switch {
	case d.exited:
		event.Kind = couchmessage.PeerChildExited
	case d.current.Status == couchmessage.Delivering && !d.now().Before(d.deliverBy):
		event.Kind = couchmessage.PeerDeadlineElapsed
		d.current.Detail = d.waitSummaryLocked()
	case d.image && d.current.Status == couchmessage.Delivering:
		event.Kind = couchmessage.PeerImageInput
	case d.interrupted:
		event.Kind = couchmessage.PeerOperatorInput
	case d.released:
		event.Kind = couchmessage.PeerBrokerReleased
	case !automaticAvailable && p.automaticInput.phase == automaticInputNeedsOperator:
		// Quarantine is already a terminal recovery decision, not another
		// operator hold. Refuse without touching input even under a picker.
		event.Kind = couchmessage.PeerAutomaticInputBlocked
		event.Ready = p.automaticInput.residueCause == couchmessage.AutomaticInputStallUnsubmittedComposer
	case p.pickerActive.Load() && d.current.Status == couchmessage.Delivering:
		event.Kind = couchmessage.PeerOverlayObserved
	case p.pickerActive.Load():
		// A picker holds the message like the operator's draft (pair#465):
		// held and delivered only when it closes and the composer is safe.
		judged, operatorHold = true, true
		d.current.Detail = d.pickerHoldDetailLocked(p.pickerEpisode.Load(), p.agentTail)
		return
	case !automaticAvailable:
		d.current.Detail = "waiting for previous automatic input: " + p.automaticInputCause
		return
	case d.replies.inFlight > 0:
		d.current.Detail = "waiting for forwarded input"
		return
	case len(d.replies.input) > 0:
		d.current.Detail = "waiting for incomplete terminal input"
		return
	case d.outputPending > 0 && (p.terminal == nil || !p.terminal.FrameComplete()):
		// A streaming agent nearly always has output in flight (pair#438);
		// a whole synchronized frame is as coherent a reading as a quiet one.
		d.current.Detail = "waiting for rendered output"
		return
	case d.inputBuffered:
		d.current.Detail = "waiting for buffered input or unfinished paste"
		return
	default:
		if d.image {
			d.current.Detail = "waiting for image input to clear"
			return
		}
		if p.terminal == nil {
			d.current.Detail = "waiting for terminal state"
			return
		}
		epoch := p.automaticRenderEpoch.Load()
		snapshot := p.terminal.Snapshot()
		if d.current.Status != couchmessage.Delivering {
			// Let just-forwarded keystrokes repaint before reading the source.
			// This bounded settle interval cannot retain stale draft ownership.
			if !d.lastInput.IsZero() && d.now().Before(d.lastInput.Add(time.Second)) {
				d.current.Detail = "waiting for input to settle"
				return
			}
			// Held operator input goes before the next automatic paste
			// (pair#428): it is not admitted yet, so lastInput can't say so.
			if d.holding {
				d.current.Detail = "waiting for held operator input"
				return
			}
			judged = true
			pasteReady := p.childAcceptsPaste()
			composer := peerComposerState(p.agentBasename, snapshot)
			event.Ready = pasteReady && composer == PeerComposerEmpty
			switch {
			case !pasteReady:
				d.current.Detail = "waiting for paste mode"
			case composer == PeerComposerUnknown:
				d.current.Detail = "waiting for recognized composer"
			case !event.Ready:
				d.current.Detail = "waiting for empty composer"
				operatorHold = true
			case !d.viewReadyLocked():
				// Never yank the view from scrollback the operator is reading.
				event.Ready = false
				d.current.Detail = "waiting for the composer to be in view"
				// The first reading in flight is not yet the operator's hold.
				operatorHold = !d.view.at.IsZero() && !d.view.visible
			default:
				d.current.Detail = ""
			}
		} else {
			// Shared confirmation uses composer/turn evidence; Qoder retries
			// additionally require the unchanged complete automatic envelope.
			composer := peerComposerState(p.agentBasename, snapshot)
			owned := false
			if p.agentBasename == "qoder" {
				composer, owned = qoderSubmissionEvidence(snapshot, peerEnvelope(d.current.Message))
			}
			if composer == PeerComposerOccupied && d.sequence > d.pasteSequence && (p.agentBasename != "qoder" || epoch > d.pasteEpoch) {
				d.sawOccupied = true
			}
			if d.state.Phase == couchmessage.PeerDeliveryConfirming {
				event.Kind = couchmessage.PeerConfirmObserved
				turn := p.turnActive.Load()
				cleared := snapshot.Complete && d.sawOccupied && composer == PeerComposerEmpty && d.sequence > d.submitSequence
				event.Ready = cleared || (turn && !d.turnAtSubmit)
				if p.agentBasename == "qoder" {
					cleared = cleared && epoch > d.submitEpoch
					event.Ready = cleared
					if !cleared && owned && epoch > d.pasteEpoch && !d.now().Before(d.submittedAt.Add(500*time.Millisecond)) {
						event.Kind, event.Ready = couchmessage.PeerRetrySubmit, true
					}
				}
				event.EmptyConfirmed = cleared
				d.current.Detail = peerConfirmEvidence(composer, d.sawOccupied, turn, d.turnAtSubmit)
			} else {
				event.Kind = couchmessage.PeerRenderObserved
				event.Ready = !d.now().Before(d.pastedAt.Add(PeerSubmitDelay))
				d.current.Detail = "pasted; submit pending"
			}
		}
	}
	if d.batch != nil && d.batch.phase != couchmessage.BatchFrozen && event.Kind == couchmessage.PeerComposerObserved {
		if event.Ready {
			d.batch.phase, _ = couchmessage.AdvanceBatchPreparation(d.batch.phase, couchmessage.BatchAvailable)
		}
		return // preparation cannot write input; finalization rechecks every guard
	}
	p.advancePeer(event, out)
}
func (p *proxy) advancePeer(event couchmessage.PeerDeliveryEvent, out io.Writer) {
	d := p.peer
	state, effect := couchmessage.AdvancePeerDelivery(d.state, event)
	d.state = state
	if effect == couchmessage.PeerSubmit && !d.now().Before(d.deliverBy) {
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerDeadlineElapsed}, out)
		return
	}
	if (effect == couchmessage.PeerPaste || effect == couchmessage.PeerSubmit) && p.rejectStaleDrainLocked() {
		return
	}
	switch effect {
	case couchmessage.PeerPaste:
		p.beginAutomaticInput(automaticInputPeer)
		if !p.childAcceptsPaste() {
			p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerOverlayObserved}, out)
			return
		}
		data := []byte(workbenchshortcut.PasteStart + couchmessage.BatchEnvelope(d.membersLocked()) + workbenchshortcut.PasteEnd)
		d.current.Status = couchmessage.Delivering
		d.pasteSequence = d.sequence
		d.pasteEpoch = p.automaticRenderEpoch.Load()
		d.pastedAt, d.sawOccupied = d.now(), false
		d.waits, d.waitAt = nil, d.now()
		d.deliverBy = d.pastedAt.Add(couchmessage.DeliveryTimeout)
		n, err := out.Write(data)
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerPasteCompleted, Written: n, Expected: len(data), Failed: err != nil}, out)
	case couchmessage.PeerSubmit:
		if p.ttyProfile == nil {
			return
		}
		data := p.ttyProfile.keymap.altCR
		d.submitSequence, d.turnAtSubmit = d.sequence, p.turnActive.Load()
		d.submitEpoch, d.submittedAt = p.automaticRenderEpoch.Load(), d.now()
		n, err := out.Write(data)
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerSubmitCompleted, Written: n, Expected: len(data), Failed: err != nil}, out)
	case couchmessage.PeerPublish:
		if p.agentBasename == "qoder" && state.BodyWritten && state.Outcome() == couchmessage.Indeterminate && !d.interrupted {
			p.quarantineQoderInput()
		} else {
			p.finishAutomaticInput(automaticInputPeer, state.Outcome() == couchmessage.Submitted && event.EmptyConfirmed)
		}
		d.current.Status = state.Outcome()
		if d.current.Status == couchmessage.Submitted && (d.recordTurn != nil || d.recordBatch != nil) && !d.exited {
			if d.batch != nil && d.recordBatch != nil {
				d.recordBatch(d.membersLocked())
			} else if d.recordTurn != nil {
				d.recordTurn(d.current.Message)
			}
		}
		if d.current.Detail != "" && (event.Kind == couchmessage.PeerDeadlineElapsed || state.Outcome() == couchmessage.Indeterminate) {
			d.current.Detail = state.Reason + ": " + d.current.Detail
		} else {
			d.current.Detail = state.Reason
		}
		select {
		case d.notices <- fmt.Sprintf("Couch message from %s: %s", d.current.Message.From.Slot, d.current.Status):
		default:
		}
	}
}

// chargeWaitLocked (d.mu held) accounts only for the bounded post-paste
// window. Pre-paste receipts retain the standing reason without a history.
func (d *peerDelivery) chargeWaitLocked() {
	if d.current.Status != couchmessage.Delivering {
		return
	}
	now := d.now()
	if !d.waitAt.IsZero() && d.current.Detail != "" {
		if d.waits == nil {
			d.waits = map[string]time.Duration{}
		}
		d.waits[d.current.Detail] += now.Sub(d.waitAt)
	}
	d.waitAt = now
}

// waitSummaryLocked (d.mu held) names what held this half of the delivery,
// longest first, rather than whichever check ran last (pair#438): at most
// three conditions of a second or more, else the standing Detail.
func (d *peerDelivery) waitSummaryLocked() string {
	details := make([]string, 0, len(d.waits))
	for detail, held := range d.waits {
		if held >= time.Second {
			details = append(details, detail)
		}
	}
	if len(details) == 0 {
		return d.current.Detail
	}
	sort.Slice(details, func(i, j int) bool {
		a, b := d.waits[details[i]], d.waits[details[j]]
		return a > b || (a == b && details[i] < details[j])
	})
	if len(details) > 3 {
		details = details[:3]
	}
	for i, detail := range details {
		details[i] = fmt.Sprintf("%s (%s)", detail, d.waits[detail].Round(time.Second))
	}
	return strings.Join(details, " | ")
}

// pickerTailLines is enough screen for a picker and its title.
const pickerTailLines = 60

// pickerHoldDetailLocked (d.mu held) names the picker holding the message by
// its title, read once per picker episode: a title the screen didn't show
// stays unread until the next picker, and the next picker is read afresh.
func (d *peerDelivery) pickerHoldDetailLocked(episode uint64, tail func(int) couchmessage.Tail) string {
	if d.pickerTitleEpisode != episode {
		d.pickerTitleEpisode = episode
		d.pickerTitle = couchmessage.ReadPicker(tail(pickerTailLines).Lines).Title
	}
	return pickerHoldDetail(d.pickerTitle)
}

func pickerHoldDetail(title string) string {
	if title == "" {
		return "blocked by a picker"
	}
	return "blocked by a picker: " + title
}

// operatorHoldLocked (d.mu held) opens or closes the operator hold as this dispatch
// observed it, and reports it on the receipt as Held.
func (d *peerDelivery) operatorHoldLocked(held bool) {
	switch {
	case held && d.current.Status == couchmessage.Queued:
		d.current.Status = couchmessage.Held
	case !held && d.current.Status == couchmessage.Held:
		d.current.Status = couchmessage.Queued
	}
}

// peerConfirmEvidence is the confirm phase's standing Detail: the composer
// and turn as the wrapper last saw them after the submit. An expiry ranks
// these readings with the other waits (waitSummaryLocked).
func peerConfirmEvidence(composer PeerComposerState, sawOccupied, turn, turnAtSubmit bool) string {
	c := map[PeerComposerState]string{PeerComposerUnknown: "unrecognized", PeerComposerOccupied: "still holds text", PeerComposerEmpty: "empty"}[composer]
	if composer == PeerComposerEmpty && !sawOccupied {
		c = "empty but never showed the paste"
	}
	t := "no turn started"
	switch {
	case turn && turnAtSubmit:
		t = "a turn was already running"
	case turn:
		t = "a turn started"
	}
	return "submitted; composer " + c + "; " + t
}
