package wrapcmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// peerDelivery is the receiver's observation mailbox. Only translateStdinFrom
// drives its reducer or writes to the PTY. The admission mutex fences reader
// observations against automatic writes, including observations not rendered yet.
type peerDelivery struct {
	mu            sync.Mutex
	binding       couchmessage.Binding
	now           func() time.Time
	lastActivity  time.Time
	sequence      uint64
	lastInput     time.Time
	inputBuffered bool
	submissions   uint64
	pasteSequence uint64
	// The post-paste half of a delivery (pair#427). deliverBy is the paste
	// time plus DeliveryTimeout; Message.Deadline only bounds the paste.
	pastedAt         time.Time
	deliverBy        time.Time
	sawOccupied      bool   // the composer read occupied after the paste
	submitSequence   uint64 // output sequence at the submit write
	turnAtSubmit     bool   // a turn was already open when we submitted
	outputPending    int
	reservation      string
	reservationUntil time.Time
	current          couchmessage.Receipt
	state            couchmessage.PeerDeliveryState
	image            bool
	interrupted      bool
	exited           bool
	replies          orientationReplies
	wake             chan struct{}
	notices          chan string
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
	settleProbe func() bool
	settleTimer settleTimer
	afterFunc   func(time.Duration, func()) settleTimer
}

type peerSessionSink interface {
	Update(couchmessage.Observation)
	Submit(couchmessage.Observation)
	Settle(bool)
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
	return &peerDelivery{binding: binding, now: now, lastActivity: now(), wake: make(chan struct{}, 1), notices: make(chan string, 1)}
}

// PeerSubmitDelay is the fixed pause between the paste and the submit key,
// the draft pane's rule (nvim/draft_send.lua) with margin. Nothing reads how
// the agent rendered the paste: pair renders, never classifies (pair#427).
const PeerSubmitDelay = 150 * time.Millisecond

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
		pasted = d.current.Status == couchmessage.Delivering
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
func (d *peerDelivery) admitInput(data []byte) bool {
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
	human := d.replies.operatorDataWithoutFocus(data)
	if human {
		d.lastInput = d.now()
		d.sequence++
		d.lastActivity = d.now()
		d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
		o, sink, unsettled = d.observationLocked(), d.session, d.unsettleLocked()
	}
	if bytes.Contains(data, []byte{0x16}) {
		d.image = true
		d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
	}
	d.signal()
	return human
}
func (d *peerDelivery) inputForwarded(human, buffered bool) {
	d.mu.Lock()
	d.replies.inFlight--
	d.inputBuffered = buffered
	if human {
		d.lastInput = d.now()
	}
	d.mu.Unlock()
	d.signal()
}
func (d *peerDelivery) admitImage() {
	d.mu.Lock()
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
	if d.current.Message.ID == id && id != "" {
		return d.current, true
	}
	return d.recent.Get(id)
}

func (d *peerDelivery) reserve(id string, sequence uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
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
	if m.To != d.binding || m.ID != d.reservation || !d.now().Before(d.reservationUntil) || !d.now().Before(m.Deadline) {
		return errors.New("peer reservation expired or identity changed")
	}
	if err := couchmessage.ValidateBody(m.Body); err != nil {
		return err
	}
	if _, ok := d.retainedLocked(m.ID); ok {
		return errors.New("delivery already committed")
	}
	d.recent.Add(d.current)
	d.current = couchmessage.Receipt{Message: m, Status: couchmessage.Queued}
	d.state = couchmessage.PeerDeliveryState{}
	d.interrupted = false
	d.signal()
	return nil
}
func (d *peerDelivery) receipt() couchmessage.Receipt {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.current
}
func peerEnvelope(m couchmessage.Message) string {
	return fmt.Sprintf("[Couch peer from %s; delivery %s]\n%s", m.From.Slot, m.ID, m.Body)
}

func (p *proxy) peerComposerSubmission() bool {
	if p.peer == nil || p.terminal == nil || p.pickerActive.Load() {
		return false
	}
	_, known := peerComposerText(p.agentBasename, p.terminal.Snapshot())
	return known
}

// dispatchPeer is called only by the existing PTY input writer. It checks the
// phase's deadline before every automatic effect: paste-by before the paste,
// the paste-relative window after it (pair#427).
func (p *proxy) dispatchPeer(out io.Writer) {
	d := p.peer
	if d == nil {
		return
	}
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.current.Message.ID == "" || d.current.Status.Terminal() {
		return
	}
	event := couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerComposerObserved}
	switch {
	case d.exited:
		event.Kind = couchmessage.PeerChildExited
	case !d.now().Before(d.deadlineLocked()):
		event.Kind = couchmessage.PeerDeadlineElapsed
	case d.image && d.current.Status == couchmessage.Delivering:
		event.Kind = couchmessage.PeerImageInput
	case d.interrupted:
		event.Kind = couchmessage.PeerOperatorInput
	case p.pickerActive.Load() && d.current.Status == couchmessage.Delivering:
		event.Kind = couchmessage.PeerOverlayObserved
	case !p.automaticInputAvailable(automaticInputPeer):
		d.current.Detail = "waiting for previous automatic input to clear"
		return
	case d.replies.inFlight > 0:
		d.current.Detail = "waiting for forwarded input"
		return
	case len(d.replies.input) > 0:
		d.current.Detail = "waiting for incomplete terminal input"
		return
	case d.outputPending > 0:
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
		if p.pickerActive.Load() {
			d.current.Detail = "waiting for picker to close"
			return
		}
		if p.terminal == nil {
			d.current.Detail = "waiting for terminal state"
			return
		}
		snapshot := p.terminal.Snapshot()
		if d.current.Status == couchmessage.Queued {
			// Let just-forwarded keystrokes repaint before reading the source.
			// This bounded settle interval cannot retain stale draft ownership.
			if !d.lastInput.IsZero() && d.now().Before(d.lastInput.Add(time.Second)) {
				d.current.Detail = "waiting for input to settle"
				return
			}
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
			default:
				d.current.Detail = ""
			}
		} else {
			// Agent-agnostic evidence only: whether the composer holds
			// anything, and whether a turn opened. Never what it holds.
			composer := peerComposerState(p.agentBasename, snapshot)
			if composer == PeerComposerOccupied && d.sequence > d.pasteSequence {
				d.sawOccupied = true
			}
			if d.state.Phase == couchmessage.PeerDeliveryConfirming {
				event.Kind = couchmessage.PeerConfirmObserved
				turn := p.turnActive.Load()
				cleared := d.sawOccupied && composer == PeerComposerEmpty && d.sequence > d.submitSequence
				event.Ready = cleared || (turn && !d.turnAtSubmit)
				d.current.Detail = peerConfirmEvidence(composer, d.sawOccupied, turn, d.turnAtSubmit)
			} else {
				event.Kind = couchmessage.PeerRenderObserved
				event.Ready = !d.now().Before(d.pastedAt.Add(PeerSubmitDelay))
				d.current.Detail = "pasted; submit pending"
			}
		}
	}
	p.advancePeer(event, out)
}
func (p *proxy) advancePeer(event couchmessage.PeerDeliveryEvent, out io.Writer) {
	d := p.peer
	state, effect := couchmessage.AdvancePeerDelivery(d.state, event)
	d.state = state
	if (effect == couchmessage.PeerPaste || effect == couchmessage.PeerSubmit) && !d.now().Before(d.deadlineLocked()) {
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerDeadlineElapsed}, out)
		return
	}
	switch effect {
	case couchmessage.PeerPaste:
		p.automaticInput.phase = automaticInputPeer
		if !p.childAcceptsPaste() {
			p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerOverlayObserved}, out)
			return
		}
		data := []byte(workbenchshortcut.PasteStart + peerEnvelope(d.current.Message) + workbenchshortcut.PasteEnd)
		d.current.Status = couchmessage.Delivering
		d.pasteSequence = d.sequence
		d.pastedAt, d.sawOccupied = d.now(), false
		d.deliverBy = d.pastedAt.Add(couchmessage.DeliveryTimeout)
		n, err := out.Write(data)
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerPasteCompleted, Written: n, Expected: len(data), Failed: err != nil}, out)
	case couchmessage.PeerSubmit:
		if p.ttyProfile == nil {
			return
		}
		data := p.ttyProfile.keymap.altCR
		d.submitSequence, d.turnAtSubmit = d.sequence, p.turnActive.Load()
		n, err := out.Write(data)
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerSubmitCompleted, Written: n, Expected: len(data), Failed: err != nil}, out)
	case couchmessage.PeerPublish:
		p.finishAutomaticInput(automaticInputPeer)
		d.current.Status = state.Outcome()
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

// deadlineLocked (d.mu held) is the bound for the delivery's current half.
func (d *peerDelivery) deadlineLocked() time.Time {
	if d.current.Status == couchmessage.Delivering {
		return d.deliverBy
	}
	return d.current.Message.Deadline
}

// peerConfirmEvidence is what an uncertain receipt reports: the composer and
// turn as the wrapper last saw them after the submit.
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
