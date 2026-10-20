package wrapcmd

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"io"
	"time"
)

type wrapperDrainInterrupt struct {
	observed couchmessage.DrainObservation
	result   couchmessage.DrainInterruptResult
}

// ReduceDrainInterrupt grants only a single Esc against a positively observed
// original turn. Input and lifecycle fences are independent of output epochs.
func ReduceDrainInterrupt(want, now couchmessage.DrainObservation) (string, string) {
	switch {
	case !want.Active:
		return "skipped", "original observation had no active turn"
	case !want.Complete || !want.ComposerKnown || want.Picker || want.AutomaticInput:
		return "refused", "original observation lacked safe input ownership"
	case want.Instance != now.Instance:
		return "refused", "wrapper instance changed"
	case want.Turn != now.Turn || want.Input != now.Input || !now.Active:
		return "skipped", "original turn ended or changed"
	case !now.Complete || !now.ComposerKnown:
		return "refused", "unsupported or incomplete composer evidence"
	case now.Picker:
		return "refused", "picker active"
	case now.AutomaticInput:
		return "refused", "input lease unavailable"
	default:
		return "interrupted", "single Esc written to original turn"
	}
}

// observeDrain snapshots evidence with inputAdmission then peer.mu, the
// single input-ownership lock order. Observation never advances an epoch.
func (d *peerDelivery) observeDrain() couchmessage.DrainObservation {
	p := d.drainProxy
	if p == nil {
		return couchmessage.DrainObservation{}
	}
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	_ = d.initDrainGenerationLocked()
	return p.drainObservationLocked(d)
}
func (d *peerDelivery) initDrainGenerationLocked() error {
	if d.drainGeneration.View().Namespace != "" {
		return nil
	}
	state, err := couchmessage.NewDrainGeneration(d.drainNamespace, d.drive.instance)
	if err != nil {
		return err
	}
	d.drainGeneration = state
	return nil
}
func (d *peerDelivery) applyDrainGenerationLocked(event couchmessage.DrainGenerationEvent) error {
	if err := d.initDrainGenerationLocked(); err != nil {
		return err
	}
	before := d.drainGeneration.View().Epoch
	next, err := couchmessage.ReduceDrainGeneration(d.drainGeneration, event)
	if err != nil {
		return err
	}
	if next.View().Epoch != before {
		// Generation retirement forgets only its fixed outcomes, never input residue.
		d.drainInterrupt = nil
		d.drainInterrupts = nil
		d.drainRevokes = nil
		if !d.reservationFence.IsZero() {
			d.reservation = ""
			d.reservationFence = couchmessage.DrainFence{}
		}
	}
	d.drainGeneration = next
	return nil
}
func (p *proxy) drainObservationLocked(d *peerDelivery) couchmessage.DrainObservation {
	v := d.drainGeneration.View()
	o := couchmessage.DrainObservation{Protocol: 2, Namespace: d.drainNamespace, Epoch: v.Epoch, Generation: v.Generation, Instance: d.drive.instance, Turn: p.drainTurn, Input: d.drainInput, Active: p.turnActive.Load(), Picker: p.pickerActive.Load()}
	o.AutomaticInput = p.automaticInput.phase != automaticInputIdle || d.exited || d.image || d.inputBuffered || d.holding || d.replies.inFlight != 0 || len(d.replies.input) != 0 || d.reservation != "" || (d.current.Message.ID != "" && !d.current.Status.Terminal()) || d.drive.active != nil
	if p.terminal != nil && p.ttyProfile != nil {
		s := p.terminal.Snapshot()
		o.Complete = s.Complete
		c := peerComposerState(p.agentBasename, s)
		o.ComposerKnown = c != PeerComposerUnknown
		o.ComposerEmpty = c == PeerComposerEmpty
	}
	o.Idle = o.Complete && o.ComposerEmpty && !o.Active && !o.Picker && !o.AutomaticInput
	return o
}
func (d *peerDelivery) lockDrainInput() func() {
	if d.drainProxy != nil {
		d.drainProxy.inputAdmission.Lock()
		return d.drainProxy.inputAdmission.Unlock
	}
	return func() {}
}
func (d *peerDelivery) startDrainInterrupt(id string, fence couchmessage.DrainFence, observed couchmessage.DrainObservation) (couchmessage.DrainInterruptResult, error) {
	unlock := d.lockDrainInput()
	defer unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.drainProxy == nil || fence.Kind != couchmessage.DrainInterruptControl {
		return couchmessage.DrainInterruptResult{}, errors.New("wrapper interrupt capability unavailable")
	}
	if err := d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainAdmitControl, Fence: fence, ID: id}); err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	if old := d.drainInterrupts[id]; old != nil {
		return old.result, nil
	}
	r := couchmessage.DrainInterruptResult{ID: id, Fence: fence, Turn: observed.Turn, Outcome: "pending"}
	if !d.drainGeneration.Allows(fence, id) {
		r.Outcome = "skipped"
		r.Detail = "drain interruption revoked"
	}
	if observed.Protocol != 2 || observed.Namespace != fence.Namespace || observed.Instance != fence.Instance {
		r.Outcome = "refused"
		r.Detail = "wrapper drain observation unsupported or changed"
	}
	item := &wrapperDrainInterrupt{observed: observed, result: r}
	if d.drainInterrupts == nil {
		d.drainInterrupts = map[string]*wrapperDrainInterrupt{}
	}
	d.drainInterrupts[id] = item
	d.drainInterrupt = item
	if r.Outcome != "pending" {
		_ = d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainSettleControl, Fence: fence, ID: id, Disposition: couchmessage.DrainControlNotCommitted})
	}
	d.signal()
	return r, nil
}
func (d *peerDelivery) drainInterruptStatus(id string, fence couchmessage.DrainFence) (couchmessage.DrainInterruptResult, error) {
	unlock := d.lockDrainInput()
	defer unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.drainGeneration.Lookup(fence, id); err != nil {
		return couchmessage.DrainInterruptResult{}, err
	}
	if r := d.drainInterrupts[id]; r != nil && r.result.Fence == fence {
		return r.result, nil
	}
	return couchmessage.DrainInterruptResult{}, errors.New("unknown current drain interruption")
}

// executeDrainInterrupt runs only on the existing stdin writer. Retained objects
// keep their original fence; even a delayed callback rechecks the watermark.
func (p *proxy) executeDrainInterrupt(out io.Writer) {
	d := p.peer
	if d == nil {
		return
	}
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	p.executeDrainInterruptLocked(d.drainInterrupt, out)
}
func (p *proxy) executeDrainInterruptLocked(r *wrapperDrainInterrupt, out io.Writer) {
	d := p.peer
	if r == nil || r.result.Outcome != "pending" {
		return
	}
	if !d.drainGeneration.Allows(r.result.Fence, r.result.ID) {
		r.result.Outcome = "skipped"
		r.result.Detail = "retired or revoked drain control"
		return
	}
	p.overlayMu.Lock()
	defer p.overlayMu.Unlock()
	r.result.Outcome, r.result.Detail = ReduceDrainInterrupt(r.observed, p.drainObservationLocked(d))
	if r.result.Outcome != "interrupted" {
		_ = d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainSettleControl, Fence: r.result.Fence, ID: r.result.ID, Disposition: couchmessage.DrainControlNotCommitted})
		return
	}
	if err := d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainBeginInput, Fence: r.result.Fence, ID: r.result.ID}); err != nil {
		r.result.Outcome = "refused"
		r.result.Detail = err.Error()
		return
	}
	r.result.Outcome = "unknown"
	r.result.Detail = "Esc handoff uncertain"
	var n int
	var err error
	if writer, ok := out.(ttyio.Writer); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		n, err = writer.WriteContext(ctx, []byte{0x1b})
		cancel()
	} else {
		n, err = out.Write([]byte{0x1b})
	}
	if err == nil && n == 1 {
		r.result.Outcome = "interrupted"
		r.result.Detail = "single Esc written to original turn"
		_ = d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainSettleControl, Fence: r.result.Fence, ID: r.result.ID, Disposition: couchmessage.DrainControlCommitted})
	}
}

// revokeDrain never writes input. It refreshes uncertain retained outcomes from
// current evidence while the same lease prevents a final input race.
func (d *peerDelivery) revokeDrain(id string, fence couchmessage.DrainFence) couchmessage.DrainRevokeResult {
	unlock := d.lockDrainInput()
	defer unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	r := couchmessage.DrainRevokeResult{ID: id, Fence: fence, Outcome: "unknown"}
	if d.drainProxy == nil {
		return r
	}
	if err := d.initDrainGenerationLocked(); err != nil {
		return r
	}
	disposition := couchmessage.DrainControlNotCommitted
	r.Outcome = "not-committed"
	var pending *wrapperDrainInterrupt
	if fence.Kind == couchmessage.DrainInterruptControl {
		if interrupt := d.drainInterrupts[id]; interrupt != nil && interrupt.result.Fence == fence {
			pending = interrupt
			result := interrupt.result
			switch result.Outcome {
			case "pending":
				result.Outcome = "skipped"
				result.Detail = "drain interruption revoked before input handoff"
			case "interrupted":
				r.Outcome = "already-committed"
				disposition = couchmessage.DrainControlCommitted
			case "unknown":
				r.Outcome = "unknown"
				disposition = couchmessage.DrainControlUnknown
			}
			r.Interrupt = &result
		} else if control, err := d.drainGeneration.Lookup(fence, id); err == nil && control.Begun {
			r.Outcome = "unknown"
			disposition = couchmessage.DrainControlUnknown
		}
	} else if fence.Kind == couchmessage.DrainNoticeControl {
		if receipt, ok := d.retainedLocked(id); ok {
			if receipt.Message.DrainFence != fence {
				return couchmessage.DrainRevokeResult{ID: id, Fence: fence, Outcome: "unknown"}
			}
			r.Receipt = &receipt
			if receipt.Status == couchmessage.Submitted {
				r.Outcome = "already-committed"
				disposition = couchmessage.DrainControlCommitted
			} else if receipt.Status == couchmessage.Delivering || receipt.Status == couchmessage.Indeterminate || d.current.Message.ID == id && d.state.BodyWritten {
				r.Outcome = "unknown"
				disposition = couchmessage.DrainControlUnknown
			}
		} else if control, err := d.drainGeneration.Lookup(fence, id); err == nil && control.Begun {
			r.Outcome = "unknown"
			disposition = couchmessage.DrainControlUnknown
		}
	} else {
		return couchmessage.DrainRevokeResult{ID: id, Fence: fence, Outcome: "unknown"}
	}
	if err := d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainRevokeControl, Fence: fence, ID: id, Disposition: disposition}); err != nil {
		return couchmessage.DrainRevokeResult{ID: id, Fence: fence, Outcome: "unknown"}
	}
	if pending != nil && pending.result.Outcome == "pending" {
		pending.result = *r.Interrupt
	}
	if fence.Kind == couchmessage.DrainNoticeControl && disposition == couchmessage.DrainControlNotCommitted && d.current.Message.ID == id && !d.current.Status.Terminal() {
		d.current.Status = couchmessage.Cancelled
		d.current.Detail = "drain notice revoked before input handoff"
		d.released = true
	}
	if d.reservation == id && d.reservationFence == fence {
		d.reservation = ""
		d.reservationFence = couchmessage.DrainFence{}
	}
	if d.drainRevokes == nil {
		d.drainRevokes = map[string]couchmessage.DrainRevokeResult{}
	}
	d.drainRevokes[id] = r
	d.signal()
	return r
}

// validateDrainMessageLocked checks current authority without granting input or
// advancing the epoch. Revoked begun input may still produce read-only proof.
func (d *peerDelivery) validateDrainMessageLocked(m couchmessage.Message) error {
	if err := couchmessage.ValidateDrainInstance(m); err != nil {
		return err
	}
	if m.DrainFence.IsZero() {
		return nil
	}
	if m.DrainFence.Instance != d.drive.instance {
		return errors.New("drain wrapper instance changed")
	}
	if m.DrainFence.Kind != couchmessage.DrainNoticeControl {
		return errors.New("invalid notice control kind")
	}
	if _, err := d.drainGeneration.Lookup(m.DrainFence, m.ID); err != nil {
		return err
	}
	return nil
}
func (p *proxy) rejectStaleDrainLocked() bool {
	d := p.peer
	for _, m := range d.membersLocked() {
		if err := d.validateDrainMessageLocked(m); err != nil {
			p.rejectDrainInputLocked(err.Error())
			return true
		}
	}
	return false
}
func (p *proxy) rejectDrainInputLocked(reason string) {
	d := p.peer
	status := couchmessage.Refused
	if d.state.BodyWritten || d.current.Status == couchmessage.Delivering {
		status = couchmessage.Indeterminate
	}
	d.current.Status = status
	d.current.Detail = reason
	if p.automaticInput.phase == automaticInputPeer {
		p.finishAutomaticInput(automaticInputPeer, false)
	}
}

// authorizeDrainWriteLocked is the final Esc-independent paste/submit fence.
func (p *proxy) authorizeDrainWriteLocked(effect couchmessage.PeerDeliveryEffect) bool {
	d := p.peer
	m := d.current.Message
	if m.DrainFence.IsZero() {
		return true
	}
	if !d.drainGeneration.Allows(m.DrainFence, m.ID) {
		p.rejectDrainInputLocked("retired or revoked drain input")
		return false
	}
	if effect == couchmessage.PeerPaste {
		if err := d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainBeginInput, Fence: m.DrainFence, ID: m.ID}); err != nil {
			p.rejectDrainInputLocked(err.Error())
			return false
		}
	}
	return true
}
func (d *peerDelivery) settleDrainNoticeLocked() {
	m := d.current.Message
	if m.DrainFence.IsZero() {
		return
	}
	disposition := couchmessage.DrainControlUnknown
	if d.current.Status == couchmessage.Submitted {
		disposition = couchmessage.DrainControlCommitted
	} else if !d.state.BodyWritten && d.current.Status.Terminal() {
		disposition = couchmessage.DrainControlNotCommitted
	}
	_ = d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainSettleControl, Fence: m.DrainFence, ID: m.ID, Disposition: disposition})
}

func (d *peerDelivery) reserveDrain(id string, sequence uint64, fence couchmessage.DrainFence) error {
	unlock := d.lockDrainInput()
	defer unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.drainProxy == nil || fence.Kind != couchmessage.DrainNoticeControl {
		return errors.New("drain reservation requires notice capability")
	}
	if err := d.applyDrainGenerationLocked(couchmessage.DrainGenerationEvent{Kind: couchmessage.DrainAdmitControl, Fence: fence, ID: id}); err != nil {
		return err
	}
	if !d.drainGeneration.Allows(fence, id) {
		return errors.New("drain reservation revoked")
	}
	if err := d.reserveLocked(id, sequence); err != nil {
		return err
	}
	d.reservationFence = fence
	return nil
}
func (d *peerDelivery) releaseDrain(id string, fence couchmessage.DrainFence) {
	unlock := d.lockDrainInput()
	defer unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, err := d.drainGeneration.Lookup(fence, id); err != nil {
		return
	}
	if d.reservation == id && d.reservationFence == fence {
		d.reservation = ""
		d.reservationFence = couchmessage.DrainFence{}
	}
}
