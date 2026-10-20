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

// inputAdmission then peer.mu is the single input-ownership lock order.
func (d *peerDelivery) observeDrain() couchmessage.DrainObservation {
	p := d.drainProxy
	if p == nil {
		return couchmessage.DrainObservation{}
	}
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	return p.drainObservationLocked(d)
}
func (p *proxy) drainObservationLocked(d *peerDelivery) couchmessage.DrainObservation {
	o := couchmessage.DrainObservation{Instance: d.drive.instance, Turn: p.drainTurn, Input: d.drainInput, Active: p.turnActive.Load(), Picker: p.pickerActive.Load()}
	o.AutomaticInput = p.automaticInput.phase != automaticInputIdle || d.exited || d.image || d.inputBuffered || d.holding || d.replies.inFlight != 0 || len(d.replies.input) != 0 || d.reservation != "" || (d.current.Message.ID != "" && !d.current.Status.Terminal()) || (d.drive.active != nil)
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
func (d *peerDelivery) startDrainInterrupt(id string, observed couchmessage.DrainObservation) (couchmessage.DrainInterruptResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if revoked, ok := d.drainRevokes[id]; ok {
		if revoked.Interrupt != nil {
			return *revoked.Interrupt, nil
		}
		return couchmessage.DrainInterruptResult{ID: id, Instance: d.drive.instance, Turn: observed.Turn, Outcome: "refused", Detail: "drain interruption revoked"}, nil
	}
	if d.drainInterrupts == nil {
		d.drainInterrupts = map[string]*wrapperDrainInterrupt{}
	}
	if old := d.drainInterrupts[id]; old != nil {
		return old.result, nil
	}
	if len(d.drainInterrupts) >= 128 {
		return couchmessage.DrainInterruptResult{}, errors.New("drain interrupt receipt capacity reached; relaunch needed")
	}
	if d.drainInterrupt != nil && d.drainInterrupt.result.Outcome == "pending" {
		return couchmessage.DrainInterruptResult{}, errors.New("another drain interruption pending")
	}
	r := couchmessage.DrainInterruptResult{ID: id, Instance: d.drive.instance, Turn: observed.Turn, Outcome: "pending"}
	if d.drainProxy == nil || observed.Instance != d.drive.instance {
		r.Outcome = "refused"
		r.Detail = "wrapper drain capability unavailable or instance changed"
	}
	item := &wrapperDrainInterrupt{observed: observed, result: r}
	d.drainInterrupts[id] = item
	d.drainInterrupt = item
	d.signal()
	return r, nil
}
func (d *peerDelivery) drainInterruptStatus(id string) (couchmessage.DrainInterruptResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if r := d.drainInterrupts[id]; r != nil {
		return r.result, nil
	}
	return couchmessage.DrainInterruptResult{}, errors.New("unknown drain interruption")
}

// Called only by the existing stdin writer. The result is retained before
// releasing the lease, including short/failed writes; RPC retry never writes.
func (p *proxy) executeDrainInterrupt(out io.Writer) {
	d := p.peer
	if d == nil {
		return
	}
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	r := d.drainInterrupt
	if r == nil || r.result.Outcome != "pending" {
		return
	}
	// Picker transitions share overlayMu, so a newly rendered picker cannot
	// slip between the final predicate and the Esc. Lock order matches input translation.
	p.overlayMu.Lock()
	defer p.overlayMu.Unlock()
	r.result.Outcome, r.result.Detail = ReduceDrainInterrupt(r.observed, p.drainObservationLocked(d))
	if r.result.Outcome != "interrupted" {
		return
	}
	// Claim uncertainty before the only external effect, never retry it.
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
	}
}

// Revoke shares peer.mu with final PeerPaste and the automatic-input lease.
// An accepted but unpasted message is revocable; a PTY handoff is never undone.
func (d *peerDelivery) revokeDrain(id string) couchmessage.DrainRevokeResult {
	p := d.drainProxy
	if p == nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		instance := ""
		if d.drive != nil {
			instance = d.drive.instance
		}
		return couchmessage.DrainRevokeResult{ID: id, Instance: instance, Outcome: "unknown"}
	}
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	d.mu.Lock()
	defer d.mu.Unlock()
	if old, ok := d.drainRevokes[id]; ok {
		return old
	}
	r := couchmessage.DrainRevokeResult{ID: id, Instance: d.drive.instance, Outcome: "not-committed"}
	if d.drainRevokes == nil {
		d.drainRevokes = map[string]couchmessage.DrainRevokeResult{}
	}
	if len(d.drainRevokes) >= 128 {
		return couchmessage.DrainRevokeResult{ID: id, Instance: d.drive.instance, Outcome: "unknown"}
	}
	if interrupt := d.drainInterrupts[id]; interrupt != nil {
		switch interrupt.result.Outcome {
		case "pending":
			interrupt.result.Outcome = "skipped"
			interrupt.result.Detail = "drain interruption revoked before input handoff"
		case "interrupted":
			r.Outcome = "already-committed"
		case "unknown":
			r.Outcome = "unknown"
		}
		result := interrupt.result
		r.Interrupt = &result
		d.drainRevokes[id] = r
		d.signal()
		return r
	}
	if receipt, ok := d.retainedLocked(id); ok {
		if d.current.Message.ID == id && d.state.BodyWritten || receipt.Status == couchmessage.Delivering || receipt.Status == couchmessage.Submitted {
			r.Outcome = "already-committed"
			r.Receipt = &receipt
			if receipt.Status == couchmessage.Indeterminate {
				r.Outcome = "unknown"
			}
		} else if receipt.Status == couchmessage.Indeterminate {
			r.Outcome = "unknown"
			r.Receipt = &receipt
		} else if d.current.Message.ID == id && !d.current.Status.Terminal() {
			d.current.Status = couchmessage.Cancelled
			d.current.Detail = "drain notice revoked before input handoff"
			d.released = true
		}
	}
	if d.reservation == id {
		d.reservation = ""
	}
	d.drainRevokes[id] = r
	d.signal()
	return r
}

// peer.mu owns the wrapper instance and immutable delivery membership.
func (d *peerDelivery) validateDrainMessageLocked(m couchmessage.Message) error {
	if err := couchmessage.ValidateDrainInstance(m); err != nil {
		return err
	}
	if m.DrainInstance != "" && (d.drive == nil || m.DrainInstance != d.drive.instance) {
		return errors.New("drain wrapper instance changed; relaunch needed")
	}
	return nil
}

// rejectStaleDrainLocked is also called immediately before every automatic
// peer input effect while inputAdmission and peer.mu are both held.
func (p *proxy) rejectStaleDrainLocked() bool {
	d := p.peer
	for _, m := range d.membersLocked() {
		if err := d.validateDrainMessageLocked(m); err != nil {
			status := couchmessage.Refused
			if d.state.BodyWritten || d.current.Status == couchmessage.Delivering {
				status = couchmessage.Indeterminate
			}
			d.current.Status = status
			d.current.Detail = err.Error()
			if p.automaticInput.phase == automaticInputPeer {
				p.finishAutomaticInput(automaticInputPeer, false)
			}
			return true
		}
	}
	return false
}
