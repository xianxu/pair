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
// through the same recognizer peer delivery trusts. Pure.
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

type settleTimer interface{ Stop() bool }

// armSettleLocked (d.mu held) cancels any pending check and arms one for the
// current sequence; any activity before it fires makes it a no-op.
func (d *peerDelivery) armSettleLocked() {
	if d.settleProbe == nil {
		return
	}
	if d.settleTimer != nil {
		d.settleTimer.Stop()
	}
	seq := d.sequence
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
	d.armSettleLocked()
	return changed
}

func (d *peerDelivery) settleFired(seq uint64) {
	d.mu.Lock()
	if seq != d.sequence || d.settleProbe == nil {
		d.mu.Unlock()
		return
	}
	pending := d.image || d.inputBuffered || d.replies.inFlight > 0 || d.outputPending > 0
	probe := d.settleProbe
	d.mu.Unlock()
	settled := !pending && probe()
	d.mu.Lock()
	if seq != d.sequence || d.settled == settled {
		d.mu.Unlock()
		return
	}
	d.settled = settled
	sink := d.session
	d.mu.Unlock()
	if sink != nil {
		sink.Settle(settled)
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
