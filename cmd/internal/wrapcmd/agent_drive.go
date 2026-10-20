package wrapcmd

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"io"
	"time"
)

const maxWrapperDriveReceipts = 8

type wrapperDrive struct {
	id, digest string
	start      couchmessage.DriveStart
	state      agentdrive.DriveState
	effects    []agentdrive.DriveEffect
	ctx        context.Context
	cancel     context.CancelFunc
	finished   time.Time
}

// The peer mailbox mutex owns the drive queue and results; only the existing
// stdin writer executes effects. RPC handlers can cancel, never write input.
type driveMailbox struct {
	instance string
	active   *wrapperDrive
	receipts map[string]*wrapperDrive
}

func newDriveMailbox() *driveMailbox {
	return &driveMailbox{instance: rand.Text(), receipts: map[string]*wrapperDrive{}}
}
func (p *proxy) agentDriveScreen() agentdrive.Screen {
	s := agentdrive.Screen{}
	if p.terminal == nil {
		return s
	}
	// Atomic flags may change around the snapshot. Fail closed if either side
	// observes an active turn or picker instead of combining permissive samples.
	turnBefore, pickerBefore := p.turnActive.Load(), p.pickerActive.Load()
	evidence := p.terminal.DriveSnapshot(couchmessage.MaxTailLines)
	s.Epoch, s.History, s.Complete = evidence.Generation, evidence.Tail.Lines, evidence.Complete
	// Picker authority comes from the current viewport. Scrollback is durable
	// outcome evidence, but an old picker there must never authorize a key.
	snap := evidence.Snapshot
	rows := make([][]uv.Cell, snap.Height)
	for y := range rows {
		rows[y] = snap.Cells[y*snap.Width : (y+1)*snap.Width]
	}
	s.Lines = renderTail(rows, couchmessage.MaxTailLines, snap.Cursor.X, snap.Cursor.Y, vt.Cursor{Hidden: !snap.CursorVisible}).Lines
	s.TurnActive = turnBefore || p.turnActive.Load()
	picker := pickerBefore || p.pickerActive.Load()
	s.Idle = wrapperSettled(peerComposerState(p.agentBasename, evidence.Snapshot), s.TurnActive, picker)
	text, known := qualifiedComposerText(p.agentBasename, evidence.Snapshot)
	s.Recoverable = known && s.Complete && !s.TurnActive && !picker
	// A panel replaces the composer, so it is the unknown-composer case that
	// still carries positive evidence: its own Esc footer (pair#691). It has
	// no turn gate: Claude reports progress as working for as long as its
	// panel is open, and the panel proof outranks that inference for a lone
	// esc only. The picker gate stays.
	s.Dismissable = !known && s.Complete && !picker && agentdrive.DismissablePanel(p.agentBasename, s.Lines)
	// Cleanup proof is intentionally one acknowledged native command line. A
	// multiline composer has no captured clearing contract and grants no cleanup.
	if s.Recoverable && text == "/model" {
		s.ComposerText = text
	}
	return s
}

// agentDriveScreenLocked combines display evidence with input ownership. The
// caller holds peer.mu; plain screen readers never inspect writer-owned state.
func (p *proxy) agentDriveScreenLocked() agentdrive.Screen {
	s := p.agentDriveScreen()
	if d := p.peer; d != nil {
		owned := !d.exited && !d.image && !d.inputBuffered && d.replies.inFlight == 0 && len(d.replies.input) == 0 && d.reservation == "" && (d.current.Message.ID == "" || d.current.Status.Terminal())
		if p.automaticInput.phase == automaticInputPeer {
			owned = false
		}
		if p.automaticInput.phase == automaticInputDrive && (d.drive == nil || d.drive.active == nil || !d.drive.active.state.Started()) {
			owned = false
		}
		s.Recoverable = s.Recoverable && owned
		s.Dismissable = s.Dismissable && owned
	}
	if !s.Recoverable {
		s.ComposerText = ""
	}
	return s
}

func (d *peerDelivery) sweepDrivesLocked() {
	if d.drive == nil {
		return
	}
	for id, r := range d.drive.receipts {
		if !r.finished.IsZero() && d.now().Sub(r.finished) >= couchmessage.ReceiptRetention && r != d.drive.active {
			delete(d.drive.receipts, id)
		}
	}
}
func (d *peerDelivery) driveAvailableLocked() bool {
	return d.drive != nil && d.drive.active == nil && !d.exited && !d.image && !d.inputBuffered && d.replies.inFlight == 0 && len(d.replies.input) == 0 && d.reservation == "" && (d.current.Message.ID == "" || d.current.Status.Terminal())
}
func (d *peerDelivery) driveIdleLocked() bool { return d.driveAvailableLocked() && d.settled }
func (d *peerDelivery) agentDriveObservation() couchmessage.DriveObservation {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.drive == nil {
		return couchmessage.DriveObservation{}
	}
	d.sweepDrivesLocked()
	o := couchmessage.DriveObservation{Instance: d.drive.instance, Sequence: d.sequence}
	if d.driveProbe != nil {
		s := d.driveProbe()
		o.Idle = d.driveIdleLocked() && s.Idle && s.Complete
		o.Recoverable = d.driveAvailableLocked() && s.Recoverable && s.Complete
		o.Dismissable = d.driveAvailableLocked() && s.Dismissable && s.Complete
	}
	return o
}
func (d *peerDelivery) startAgentDrive(id string, start couchmessage.DriveStart) (agentdrive.Result, error) {
	d.mu.Lock()
	defer func() { d.mu.Unlock(); d.signal() }()
	if d.drive == nil || d.driveProbe == nil {
		return agentdrive.Result{}, errors.New("wrapper drive unavailable; relaunch the slot on the new build")
	}
	d.sweepDrivesLocked()
	if old, ok := d.drive.receipts[id]; ok {
		if old.digest != start.Request.IntentDigest() {
			return agentdrive.Result{}, errors.New("drive ID already names different input")
		}
		return driveResult(old), nil
	}
	refuse := func(detail string) (agentdrive.Result, error) {
		return agentdrive.Result{ID: id, Outcome: agentdrive.OutcomeRefused, Detail: detail}, nil
	}
	if err := start.Validate(d.now()); err != nil {
		return refuse(err.Error())
	}
	if start.Instance != d.drive.instance || start.Sequence != d.sequence {
		return refuse("wrapper observation changed; inspect the slot and send a new request")
	}
	if len(d.drive.receipts) >= maxWrapperDriveReceipts {
		return refuse("eight recent drive outcomes are held; wait up to five minutes and try again")
	}
	screen := d.driveProbe()
	available := d.driveAvailableLocked()
	admission := agentdrive.Admission{Idle: d.driveIdleLocked() && screen.Idle, Recoverable: available && screen.Recoverable, Dismissable: available && screen.Dismissable}
	if !start.Request.Admits(admission) || !screen.Complete {
		return refuse(start.Request.Refusal(admission, "agent composer must be empty with no turn or picker; wait until it is idle"))
	}
	state, err := agentdrive.NewDrive(start.Request, d.binding.Agent, start.Deadline)
	if err != nil {
		return refuse(err.Error())
	}
	ctx, cancel := context.WithDeadline(context.Background(), start.Deadline)
	r := &wrapperDrive{id: id, digest: start.Request.IntentDigest(), start: start, state: state, ctx: ctx, cancel: cancel}
	d.drive.active = r
	d.drive.receipts[id] = r
	return driveResult(r), nil
}
func driveResult(r *wrapperDrive) agentdrive.Result {
	result := r.state.Result()
	result.ID = r.id
	return result
}
func (d *peerDelivery) agentDriveResult(id string) (agentdrive.Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sweepDrivesLocked()
	if d.drive != nil {
		if r, ok := d.drive.receipts[id]; ok {
			return driveResult(r), nil
		}
	}
	return agentdrive.Result{}, errors.New("drive outcome unknown; inspect the slot before sending more input")
}
func (d *peerDelivery) cancelDriveLocked(reason string) {
	if d.drive == nil || d.drive.active == nil {
		return
	}
	r := d.drive.active
	r.cancel()
	r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventCancel, At: d.now(), Error: reason})
}
func (d *peerDelivery) cancelAgentDrive(id string) (agentdrive.Result, error) {
	d.mu.Lock()
	if d.drive == nil || d.drive.receipts[id] == nil {
		d.mu.Unlock()
		return agentdrive.Result{}, errors.New("drive outcome unknown; inspect the slot before sending more input")
	}
	r := d.drive.receipts[id]
	if r == d.drive.active {
		d.cancelDriveLocked("canceled by supervisor")
	}
	result := driveResult(r)
	d.mu.Unlock()
	d.signal()
	return result, nil
}
func (p *proxy) finishAgentDriveLocked(r *wrapperDrive, screen agentdrive.Screen) {
	d := p.peer
	if r.finished.IsZero() {
		r.finished = d.now()
		r.cancel()
	}
	if d.drive.active == r {
		d.drive.active = nil
	}
	if p.automaticInput.phase == automaticInputDrive {
		outcome := r.state.Result().Outcome
		confirmed := outcome == agentdrive.OutcomeConfirmed || outcome == agentdrive.OutcomeFailed
		empty := confirmed && screen.Idle && screen.Complete && !screen.TurnActive
		p.automaticInput, _ = advanceAutomaticInput(p.automaticInput, automaticInputEvent{kind: automaticFinish, owner: automaticInputDrive, epoch: p.automaticRenderEpoch.Load(), submitted: confirmed, confirmedEmpty: empty})
	}
	d.armSettleLocked()
}

// dispatchAgentDrive is called only by the single stdin writer. It shares the
// admission fence and automatic-input owner with peer delivery, but not peer
// envelope/submission policy. One callback executes at most one named input.
func (p *proxy) dispatchAgentDrive(out io.Writer) {
	d := p.peer
	if d == nil {
		return
	}
	p.inputAdmission.Lock()
	d.mu.Lock()
	if d.drive == nil || d.drive.active == nil {
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	r := d.drive.active
	screen := p.agentDriveScreenLocked()
	if d.exited {
		d.cancelDriveLocked("wrapper exited")
	}
	if r.state.Terminal() {
		p.finishAgentDriveLocked(r, screen)
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	if r.ctx.Err() != nil || !d.now().Before(r.start.Deadline) {
		r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventTick, At: d.now(), Screen: screen})
		if !r.state.Terminal() {
			d.cancelDriveLocked("drive context ended")
		}
	}
	if !r.state.Started() && !r.state.Terminal() {
		recovery := r.start.Request.RecoveryOnly()
		residual := p.automaticInput.phase == automaticInputAwaitingEmpty || p.automaticInput.phase == automaticInputAlerted
		lease := p.automaticInput.phase == automaticInputIdle || residual && (recovery || p.automaticInputAvailable(automaticInputDrive))
		common := !d.exited && !d.image && d.sequence == r.start.Sequence && !d.inputBuffered && d.replies.inFlight == 0 && len(d.replies.input) == 0 && d.reservation == "" && (d.current.Message.ID == "" || d.current.Status.Terminal()) && lease
		screen.Idle = screen.Idle && d.settled && common
		screen.Recoverable = screen.Recoverable && recovery && common
		screen.Dismissable = screen.Dismissable && recovery && common
		if screen.Idle || screen.Recoverable || screen.Dismissable {
			p.automaticInput, _ = advanceAutomaticInput(p.automaticInput, automaticInputEvent{kind: automaticBegin, owner: automaticInputDrive, eligible: recovery})
		}
		r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventStart, At: d.now(), Screen: screen})
	} else if (len(r.effects) == 0 || r.state.RecoveryDue(d.now())) && !r.state.Terminal() {
		r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventTick, At: d.now(), Screen: screen})
		if len(r.effects) == 0 && !r.state.Terminal() {
			r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventScreen, At: d.now(), Screen: screen})
		}
	}
	if r.state.Terminal() {
		p.finishAgentDriveLocked(r, screen)
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	if screen.TurnActive && !r.state.Dismissal() {
		r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventScreen, At: d.now(), Screen: screen})
		if r.state.Terminal() {
			p.finishAgentDriveLocked(r, screen)
		}
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	if len(r.effects) == 0 {
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	effect := r.effects[0]
	if effect.Kind == agentdrive.EffectWrite && d.now().Before(effect.NotBefore) {
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	if effect.Kind == agentdrive.EffectWrite && effect.ScreenEpoch != 0 && !effect.MatchesScreen(p.agentBasename, screen) {
		d.cancelDriveLocked("screen changed before input; cached picker action refused")
		p.finishAgentDriveLocked(r, screen)
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	if effect.Kind == agentdrive.EffectWrite && !screen.Complete {
		// Initial commands and raw steps wait for a coherent frame within the
		// same absolute budget; no bytes enter a partial redraw.
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	if effect.Kind == agentdrive.EffectWrite && r.start.Request.RecoveryOnly() && !r.start.Request.Admits(agentdrive.Admission{Recoverable: screen.Recoverable, Dismissable: screen.Dismissable}) {
		d.cancelDriveLocked("recovery composer proof changed before input")
		p.finishAgentDriveLocked(r, screen)
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	r.effects = r.effects[1:]
	if effect.Kind != agentdrive.EffectWrite {
		// Observation/wait effects are serviced by the existing active input poll;
		// their next event reads a complete, newly rendered terminal snapshot.
		if effect.Kind == agentdrive.EffectObserve {
			r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventScreen, At: d.now(), Screen: screen})
		}
		if r.state.Terminal() {
			p.finishAgentDriveLocked(r, screen)
		}
		d.mu.Unlock()
		p.inputAdmission.Unlock()
		return
	}
	// A complete named ESC has its own boundary. A leftover prefix is emitted
	// now, never joined to a later Return into an accidental Alt+Return.
	d.mu.Unlock()
	p.driveControl.Store(effect.Control)
	p.peerHumanSubmit = false
	var data, left []byte
	var paste bool
	if p.hasReturnRemap() {
		data, left, paste = p.translateChunk(effect.Key.InputBytes(), false)
	} else {
		data, left, paste = p.passThroughChunk(effect.Key.InputBytes(), false)
	}
	p.driveControl.Store(false)
	data = append(data, left...)
	if paste {
		p.inputAdmission.Unlock()
		d.mu.Lock()
		r.state, r.effects = agentdrive.ReduceDrive(r.state, agentdrive.DriveEvent{Kind: agentdrive.EventWriteFailed, At: d.now(), Error: "unexpected paste state"})
		d.mu.Unlock()
		return
	}
	// Human admission may cancel a blocked context-aware write. Its bytes still
	// reach the one writer afterward; cancellation never adds cleanup input.
	p.inputAdmission.Unlock()
	var n int
	var err error
	if writer, ok := out.(ttyio.Writer); ok {
		n, err = writer.WriteContext(r.ctx, data)
	} else {
		if err = r.ctx.Err(); err == nil {
			n, err = out.Write(data)
		}
	}
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err == nil && p.peerHumanSubmit && !effect.Control {
		d.humanSubmit()
	}
	d.mu.Lock()
	event := agentdrive.DriveEvent{Kind: agentdrive.EventWritten, At: d.now(), Screen: screen}
	if err != nil {
		event.Kind = agentdrive.EventWriteFailed
		event.Error = fmt.Sprintf("accepted %d of %d bytes: %v", n, len(data), err)
	}
	r.state, r.effects = agentdrive.ReduceDrive(r.state, event)
	if r.state.Terminal() {
		p.finishAgentDriveLocked(r, p.agentDriveScreenLocked())
	}
	d.mu.Unlock()
}
