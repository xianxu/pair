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
// nothing": no turn is open, no overlay is up, the startup orientation is no
// longer pending (pair#427: a restart reports ready only once its new session
// has been briefed), and the composer reads empty through the same recognizer
// peer delivery trusts. Pure.
func wrapperSettled(composer PeerComposerState, turnActive, picker, orienting bool) bool {
	return !turnActive && !picker && !orienting && composer == PeerComposerEmpty
}

// settledNow evaluates wrapperSettled against live proxy state. It runs on the
// settle timer's goroutine: the terminal model locks internally, and the turn
// and picker flags are atomics mirrored by their owners.
func (p *proxy) settledNow() bool {
	if p.terminal == nil {
		return false
	}
	return wrapperSettled(peerComposerState(p.agentBasename, p.terminal.Snapshot()), p.turnActive.Load(), p.pickerActive.Load(), p.orientationPending())
}

// orientationPending reports a startup orientation not yet finalized. The
// pointer is set before the peer runtime starts; finalized only closes.
func (p *proxy) orientationPending() bool {
	if p.orientation == nil {
		return false
	}
	select {
	case <-p.orientation.finalized:
		return false
	default:
		return true
	}
}

// settleSourceChanged re-arms the settle check for a source that changes
// without output or input: orientation finalizing, possibly at its deadline.
func (d *peerDelivery) settleSourceChanged() {
	d.mu.Lock()
	d.armSettleLocked()
	d.mu.Unlock()
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
	settled := !pending && probe()
	d.mu.Lock()
	if seq != d.settleGen || d.settled == settled {
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
