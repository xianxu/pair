package wrapcmd

import (
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// SettleInterval is how long output and input must be quiet before the wrapper
// asks whether it is settled (#421). Streaming output keeps re-arming it, so a
// working agent never pays for the check.
const SettleInterval = 3 * time.Second

// wrapperSettled is the one rule for "an automatic restart would interrupt
// nothing": no turn is open, no overlay is up, and the composer reads empty
// through the same recognizer peer delivery trusts. Pure. (A new session's
// first message is its launch prompt since #522, so its turn is open as soon
// as the agent takes it; there is no separate briefing to wait for.)
func wrapperSettled(composer PeerComposerState, turnActive, picker bool) bool {
	return !turnActive && !picker && composer == PeerComposerEmpty
}

// settledNow evaluates wrapperSettled against live proxy state. It runs on the
// settle timer's goroutine: the terminal model locks internally, and the turn
// and picker flags are atomics mirrored by their owners.
func (p *proxy) settledNow() bool {
	if p.terminal == nil {
		return false
	}
	return wrapperSettled(peerComposerState(p.agentBasename, p.terminal.Snapshot()), p.turnActive.Load(), p.pickerActive.Load())
}

type wrapperSettleState struct {
	Settled        bool
	BackgroundWork *bool
}

func (p *proxy) settleState() wrapperSettleState {
	if p.terminal == nil {
		return wrapperSettleState{}
	}
	snapshot := p.terminal.Snapshot()
	return wrapperSettleState{
		Settled:        wrapperSettled(peerComposerState(p.agentBasename, snapshot), p.turnActive.Load(), p.pickerActive.Load()),
		BackgroundWork: backgroundWorkState(p.agentBasename, snapshot),
	}
}

type settleTimer interface{ Stop() bool }

// armSettleLocked (d.mu held) is the ONE entry for every source transition of
// Settled (output, input, image, submit, a silent turn change). It advances the
// settle generation, which invalidates any check already armed or in flight,
// then arms a fresh one. A check publishes only if no source moved since.
func (d *peerDelivery) armSettleLocked() {
	d.settleGen++
	if d.settleProbe == nil {
		return
	}
	if d.settleTimer != nil {
		d.settleTimer.Stop()
	}
	seq := d.settleGen
	after := d.afterFunc
	if after == nil {
		after = func(dur time.Duration, f func()) settleTimer { return time.AfterFunc(dur, f) }
	}
	d.settleTimer = after(SettleInterval, func() { d.settleFired(seq) })
}

// unsettleLocked (d.mu held) records activity: unsettled at once, re-checked
// after the interval. It reports whether the session must hear about it.
func (d *peerDelivery) unsettleLocked() bool {
	changed := d.settled
	d.settled = false
	if d.session != nil {
		d.session.BackgroundWork(nil)
	}
	d.armSettleLocked()
	return changed
}

func (d *peerDelivery) settleFired(seq uint64) {
	d.mu.Lock()
	if seq != d.settleGen || d.settleProbe == nil {
		d.mu.Unlock()
		return
	}
	pending := d.image || d.inputBuffered || d.replies.inFlight > 0 || d.outputPending > 0
	probe := d.settleProbe
	d.mu.Unlock()
	state := wrapperSettleState{}
	if !pending {
		state = probe()
	}
	d.mu.Lock()
	if seq != d.settleGen {
		d.mu.Unlock()
		return
	}
	changed := d.settled != state.Settled
	d.settled = state.Settled
	sink := d.session
	// This setter only copies state and wakes the existing sender. Publish
	// under the source lock so invalidation cannot overtake this generation.
	if sink != nil {
		sink.BackgroundWork(state.BackgroundWork)
	}
	d.mu.Unlock()
	if changed && sink != nil {
		sink.Settle(state.Settled)
	}
}

// selfBuildIdentity hashes the running executable. Called once, first thing in
// run, so the path still names the image this process loaded.
func selfBuildIdentity() (*couchmessage.BuildIdentity, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	b, err := couchmessage.BuildIdentityOfFile(path)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// lifecycleTurnChanged keeps Settled true to the turn state when the turn
// opens or closes WITHOUT any output: a watchdog, grace expiry or transcript
// record changes Active silently (#421 M1 review). Opening unsettles at once;
// closing re-arms the check, which reads the new state through turnActive.
func (d *peerDelivery) lifecycleTurnChanged(active bool) {
	d.mu.Lock()
	var unsettled bool
	if active {
		unsettled = d.settled
		d.settled = false
	}
	d.armSettleLocked()
	sink := d.session
	d.mu.Unlock()
	d.publishUnsettled(sink, unsettled)
}
