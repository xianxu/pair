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
	mu               sync.Mutex
	binding          couchmessage.Binding
	now              func() time.Time
	lastActivity     time.Time
	sequence         uint64
	outputSequence   uint64
	submitSequence   uint64
	pasteSequence    uint64
	outputPending    int
	reservation      string
	reservationUntil time.Time
	current          couchmessage.Receipt
	state            couchmessage.PeerDeliveryState
	image            bool
	operatorDraft    bool
	interrupted      bool
	exited           bool
	replies          orientationReplies
	wake             chan struct{}
	submit           chan struct{}
	notices          chan string
}

func newPeerDelivery(binding couchmessage.Binding, now func() time.Time) *peerDelivery {
	return &peerDelivery{binding: binding, now: now, lastActivity: now(), wake: make(chan struct{}, 1), submit: make(chan struct{}, 1), notices: make(chan string, 1)}
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
	d.outputSequence = d.sequence
	d.outputPending++
	d.lastActivity = d.now()
	d.replies.observeQueries(data)
	d.mu.Unlock()
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
func (d *peerDelivery) admitInput(data []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.replies.inFlight++
	if d.replies.operatorData(data) {
		d.operatorDraft = true
		d.sequence++
		d.lastActivity = d.now()
		d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
	}
	if bytes.Contains(data, []byte{0x16}) {
		d.image = true
		d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
	}
	d.signal()
}
func (d *peerDelivery) inputForwarded() { d.mu.Lock(); d.replies.inFlight--; d.mu.Unlock(); d.signal() }
func (d *peerDelivery) admitImage() {
	d.mu.Lock()
	d.image = true
	d.interrupted = d.interrupted || d.current.Status == couchmessage.Delivering
	d.sequence++
	d.lastActivity = d.now()
	d.mu.Unlock()
	d.signal()
}
func (d *peerDelivery) humanSubmit() {
	d.mu.Lock()
	d.image = false
	d.operatorDraft = false
	d.sequence++
	d.submitSequence = d.sequence
	d.lastActivity = d.now()
	d.mu.Unlock()
	select {
	case d.submit <- struct{}{}:
	default:
	}
}
func (d *peerDelivery) reserve(id string, sequence uint64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
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
	if d.current.Message.ID == m.ID {
		return errors.New("delivery already committed")
	}
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

// dispatchPeer is called only by the existing PTY input writer. It checks the
// original deadline before every automatic effect, even after a successful paste.
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
	case !d.now().Before(d.current.Message.Deadline):
		event.Kind = couchmessage.PeerDeadlineElapsed
	case d.image && d.current.Status == couchmessage.Delivering:
		event.Kind = couchmessage.PeerImageInput
	case d.interrupted:
		event.Kind = couchmessage.PeerOperatorInput
	case p.pickerActive.Load() && d.current.Status == couchmessage.Delivering:
		event.Kind = couchmessage.PeerOverlayObserved
	case d.replies.inFlight > 0 || len(d.replies.input) > 0 || d.outputPending > 0:
		return
	default:
		if d.image || p.pickerActive.Load() {
			return
		}
		if p.terminal == nil {
			return
		}
		snapshot := p.terminal.Snapshot()
		if d.current.Status == couchmessage.Queued {
			// Rendered blanks cannot prove that a human-owned composer is
			// empty: input may not have rendered yet, or may be whitespace.
			if d.operatorDraft || (d.submitSequence > 0 && d.outputSequence <= d.submitSequence) {
				return
			}
			event.Ready = p.childAcceptsPaste() && peerComposerState(p.agentBasename, snapshot) == PeerComposerEmpty
		} else {
			event.Kind = couchmessage.PeerRenderObserved
			_, known := peerComposerText(p.agentBasename, snapshot)
			event.Ready = known && d.sequence > d.pasteSequence
			event.Matches = known && peerComposerMatches(p.agentBasename, snapshot, peerEnvelope(d.current.Message))
		}
	}
	p.advancePeer(event, out)
}
func (p *proxy) advancePeer(event couchmessage.PeerDeliveryEvent, out io.Writer) {
	d := p.peer
	state, effect := couchmessage.AdvancePeerDelivery(d.state, event)
	d.state = state
	if (effect == couchmessage.PeerPaste || effect == couchmessage.PeerSubmit) && !d.now().Before(d.current.Message.Deadline) {
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerDeadlineElapsed}, out)
		return
	}
	switch effect {
	case couchmessage.PeerPaste:
		if !p.childAcceptsPaste() {
			p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerOverlayObserved}, out)
			return
		}
		data := []byte(workbenchshortcut.PasteStart + peerEnvelope(d.current.Message) + workbenchshortcut.PasteEnd)
		d.current.Status = couchmessage.Delivering
		d.pasteSequence = d.sequence
		n, err := out.Write(data)
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerPasteCompleted, Written: n, Expected: len(data), Failed: err != nil}, out)
	case couchmessage.PeerSubmit:
		if p.ttyProfile == nil {
			return
		}
		data := p.ttyProfile.keymap.altCR
		n, err := out.Write(data)
		p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerSubmitCompleted, Written: n, Expected: len(data), Failed: err != nil}, out)
	case couchmessage.PeerPublish:
		d.current.Status = state.Outcome()
		d.current.Detail = state.Reason
		select {
		case d.notices <- fmt.Sprintf("Couch message from %s: %s", d.current.Message.From.Slot, d.current.Status):
		default:
		}
	}
}
