package couchtty

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// broadcastPhase is the console's side of a broadcast (#395). Transitions,
// all on the Run loop:
//
//	off      --toggle-->                   starting (Start runs off the loop)
//	starting --toggle-->                   off (start cancelled; a late session is stopped, not adopted)
//	starting --start failed-->             off + notice
//	starting --start succeeded-->          live: tap installed, chrome repainted, watch activated, link copied
//	live     --toggle or click-->          stopping: tap removed, viewers told; teardown off the loop
//	live     --session ended on its own--> off + notice with the reason
//	stopping --teardown done-->            off
//	stopping --toggle-->                   refused with a notice
//
// Couch shutdown stops any session before the presenter is released.
type broadcastPhase uint8

const (
	broadcastOff broadcastPhase = iota
	broadcastStarting
	broadcastLive
	broadcastStopping
)

// broadcastState is guarded by Console.mu and mutated on the Run loop.
type broadcastState struct {
	phase broadcastPhase
	// attempt names the current start, so a start that completes after it was
	// cancelled (or superseded) is recognised and stopped.
	attempt uint64
	cancel  context.CancelFunc
	session *broadcast.Session
}

// broadcastShutdownWait bounds how long Couch's exit waits for a broadcast's
// listener and tunnel to close. A variable so tests can shorten it.
var broadcastShutdownWait = 6 * time.Second

// startClaim decides, exactly once, who owns a session whose start finished:
// the loop adopting it, or the start goroutine abandoning (stopping) it. The
// goroutine can stop waiting for the loop and the loop can still run the
// handover later, so neither side may infer the other's decision from its
// own view of the handover; the one successful CAS is the decision.
type startClaim struct{ state atomic.Uint32 }

const (
	claimOpen uint32 = iota
	claimAdopted
	claimAbandoned
)

func (c *startClaim) adopt() bool   { return c.state.CompareAndSwap(claimOpen, claimAdopted) }
func (c *startClaim) abandon() bool { return c.state.CompareAndSwap(claimOpen, claimAbandoned) }

// awaitDown waits for a stopped session to finish tearing down, or for the
// console to stop; teardown's own bounded wait covers the rest.
func (c *Console) awaitDown(s *broadcast.Session) {
	select {
	case <-s.Done():
	case <-c.stop:
	}
}

// SetBroadcast enables the broadcast control with this configuration. Without
// it the tab bar never shows the cell and Ctrl+Alt+b only explains why.
func (c *Console) SetBroadcast(cfg broadcast.Config) { c.broadcastCfg = &cfg }

// BroadcastStatus is the broadcast as `couch --broadcast-list` reports it
// (pair#413); false when none is running. Never the link: the status type has
// no field for it. The viewer count is read after c.mu is released, so the
// console lock never waits on the hub's loop.
func (c *Console) BroadcastStatus() (couchmessage.BroadcastStatus, bool) {
	c.mu.Lock()
	phase, session := c.bcast.phase, c.bcast.session
	c.mu.Unlock()
	var state string
	switch phase {
	case broadcastStarting:
		state = "starting"
	case broadcastLive:
		state = "live"
	case broadcastStopping:
		state = "stopping"
	default:
		return couchmessage.BroadcastStatus{}, false
	}
	status := couchmessage.BroadcastStatus{State: state}
	if session != nil {
		described := session.Status()
		status.StartedAt, status.Mode, status.Viewers = described.StartedAt, described.Mode, described.Viewers
	}
	return status, true
}

// broadcastCellLocked is what the status row draws for the current phase.
func (c *Console) broadcastCellLocked() BroadcastCell {
	switch c.bcast.phase {
	case broadcastStarting:
		return BroadcastStarting
	case broadcastLive:
		return BroadcastLive
	}
	return BroadcastOff
}

// toggleBroadcast is Ctrl+Alt+b and a click on the LIVE cell.
func (c *Console) toggleBroadcast() {
	c.mu.Lock()
	cfg, state := c.broadcastCfg, c.bcast
	c.mu.Unlock()
	switch state.phase {
	case broadcastOff:
		if cfg == nil {
			c.setNotice("Broadcasting is not configured in this Couch.")
			return
		}
		c.startBroadcast(*cfg)
	case broadcastStarting:
		c.mu.Lock()
		c.bcast.cancel()
		c.bcast = broadcastState{attempt: c.bcast.attempt + 1}
		c.mu.Unlock()
		c.setNotice("Broadcast cancelled.")
	case broadcastLive:
		c.stopBroadcast()
	case broadcastStopping:
		c.setNotice("The previous broadcast is still stopping.")
	}
}

func (c *Console) startBroadcast(cfg broadcast.Config) {
	if cfg.Theme == nil {
		cfg.Theme = c.broadcastTheme
	}
	// The pointer link's points and watch reports (#412).
	if cfg.OnPoints == nil {
		cfg.OnPoints = c.onPoints
	}
	if cfg.OnPointerOff == nil {
		cfg.OnPointerOff = c.pointerOffByWatch
	}
	ctx, cancel := context.WithCancel(c.lifetime)
	c.mu.Lock()
	c.bcast = broadcastState{phase: broadcastStarting, attempt: c.bcast.attempt + 1, cancel: cancel}
	attempt := c.bcast.attempt
	c.mu.Unlock()
	c.setNotice("Starting broadcast…")
	c.GoTracked(func() {
		s, err := broadcast.Start(ctx, cfg)
		// The start is over. A session never depends on its start context
		// (broadcast.Tunnel's contract), so release it now.
		cancel()
		claim := &startClaim{}
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.broadcastStarted(attempt, s, err, claim)
			return nil
		})
		if s != nil && claim.abandon() {
			s.Stop(nil)
			c.awaitDown(s)
		}
	})
}

// broadcastStarted runs on the loop when a start finishes. It adopts the
// session only by winning claim; a session it doesn't adopt, the start
// goroutine stops.
func (c *Console) broadcastStarted(attempt uint64, s *broadcast.Session, err error, claim *startClaim) {
	c.mu.Lock()
	current := c.bcast.attempt == attempt && c.bcast.phase == broadcastStarting
	if !current {
		c.mu.Unlock()
		return
	}
	if err != nil || !claim.adopt() {
		c.bcast = broadcastState{attempt: attempt}
		c.mu.Unlock()
		if err != nil {
			c.setNotice("Broadcast failed to start: " + err.Error())
		}
		return
	}
	c.mu.Unlock()
	// The tap goes in before the chrome shows LIVE, so the first LIVE frame
	// reaches viewers; Activate comes after, so the grace watch starts with
	// the indicator already on screen.
	// The marks overlay goes in with the tap, so viewers see marks too.
	if err := c.presenter.SetOverlay(c.lifetime, c.markOverlay); err != nil {
		c.terminalError(err)
	}
	if err := c.presenter.SetTap(c.lifetime, s.Offer); err != nil {
		// Adopted, so ours to stop.
		s.Stop(nil)
		c.mu.Lock()
		c.bcast = broadcastState{attempt: attempt}
		c.mu.Unlock()
		c.setNotice("Broadcast failed to start: " + err.Error())
		return
	}
	c.mu.Lock()
	c.bcast.phase, c.bcast.session = broadcastLive, s
	c.mu.Unlock()
	if err := c.presenter.Copy(c.lifetime, []byte(s.Link())); err != nil {
		c.terminalError(err)
	}
	// The notice never contains the link: it would be drawn into the very
	// frames being broadcast.
	c.setNotice("Broadcast live — link copied. Click LIVE or press Ctrl+Alt+b to stop.")
	s.Activate()
	c.GoTracked(func() {
		select {
		case <-s.Done():
		case <-c.stop:
			// Shutdown stops the session itself, with a bounded wait.
			return
		}
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.broadcastEnded(s)
			return nil
		})
	})
}

// stopBroadcast is the operator's stop. Viewers are told at once; the
// listener and tunnel close off the loop, and broadcastEnded finishes.
func (c *Console) stopBroadcast() {
	c.mu.Lock()
	s := c.bcast.session
	c.bcast.phase = broadcastStopping
	c.mu.Unlock()
	c.detachBroadcastScreen()
	s.Stop(nil)
	c.setNotice("Broadcast stopped.")
}

// broadcastEnded runs on the loop once a session is fully down, whether the
// operator stopped it or it ended on its own.
func (c *Console) broadcastEnded(s *broadcast.Session) {
	c.mu.Lock()
	if c.bcast.session != s {
		c.mu.Unlock()
		return
	}
	byOperator := c.bcast.phase == broadcastStopping
	c.bcast = broadcastState{attempt: c.bcast.attempt}
	c.mu.Unlock()
	if !byOperator {
		c.detachBroadcastScreen()
		c.setNotice("Broadcast ended: " + broadcast.EndReason(s.Err()) + ".")
		return
	}
	c.repaint()
}

// endBroadcastForShutdown stops a running or starting broadcast before the
// presenter is released, and waits, bounded, for it to come down.
func (c *Console) endBroadcastForShutdown() {
	c.mu.Lock()
	state := c.bcast
	c.bcast = broadcastState{attempt: state.attempt + 1}
	c.mu.Unlock()
	if state.cancel != nil {
		state.cancel()
	}
	if s := state.session; s != nil {
		s.Stop(nil)
		select {
		case <-s.Done():
		case <-time.After(broadcastShutdownWait):
		}
	}
}

// detachBroadcastScreen removes the broadcast from the screen: no more
// frames tapped, no marks drawn, the pointer forgotten (#412).
func (c *Console) detachBroadcastScreen() {
	c.terminalError(c.presenter.SetTap(c.lifetime, nil))
	c.terminalError(c.presenter.SetOverlay(c.lifetime, nil))
	c.resetPointer()
}
