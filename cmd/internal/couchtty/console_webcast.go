package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/castlock"

	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/webcast"
)

// The cast Chrome (#454): one per Couch.
//
//	off      --key-->                    starting
//	starting --Chrome up, adopted-->     running (cell Chrome ▶ / ⏸)
//	starting --failed-->                 off + notice
//	starting --key-->                    starting + notice (Chrome is coming)
//	running  --key / click-->            running, live toggled
//	running  --Ctrl+click-->             ending
//	running  --Chrome closed by hand-->  off + notice
//	ending   --cast done-->              off
//
// Couch shutdown ends the cast, bounded, beside the screen broadcast.
type webcastPhase uint8

const (
	webcastOff webcastPhase = iota
	webcastStarting
	webcastRunning
	webcastEnding
)

// webcastState is guarded by Console.mu and mutated on the Run loop.
type webcastState struct {
	phase   webcastPhase
	attempt uint64
	cancel  context.CancelFunc
	cast    *webcast.Cast
	// update is the cast's latest, as last delivered to the loop.
	update webcast.Update
}

// webcastShutdownWait bounds how long Couch's exit waits for Chrome and the
// cast's links to close. A variable so tests can shorten it.
var webcastShutdownWait = 6 * time.Second

// SetWebcast enables the cast Chrome with this configuration. Without it the
// key only explains why nothing happens.
func (c *Console) SetWebcast(cfg webcast.CastConfig) {
	// Generic recording reports include healthy checks and recovered resyncs.
	// Only explicitly actionable sources raise attention. Both callbacks post
	// asynchronously so Chrome never waits on the console.
	cfg.Logf = func(format string, args ...any) {
		msg := "Chrome cast: " + fmt.Sprintf(format, args...)
		c.GoTracked(func() { c.setNotice(msg) })
	}
	cfg.Warnf = func(format string, args ...any) {
		msg := "Chrome cast: " + fmt.Sprintf(format, args...)
		c.GoTracked(func() { c.setWarning(msg) })
	}
	c.webcastCfg = &cfg
}

func (c *Console) webcastCellLocked() WebcastCell {
	if c.wcast.phase != webcastRunning && c.wcast.phase != webcastEnding {
		return WebcastOff
	}
	if c.wcast.update.State.Phase == webcast.PhaseLive {
		return WebcastLive
	}
	return WebcastIdle
}

// toggleWebcast is Shift+Ctrl+Alt+b and a click on the Chrome cell.
func (c *Console) toggleWebcast() {
	c.mu.Lock()
	cfg, state := c.webcastCfg, c.wcast
	c.mu.Unlock()
	switch state.phase {
	case webcastOff:
		if cfg == nil {
			c.setNotice("Web casting is not configured in this Couch.")
			return
		}
		c.startWebcast(*cfg)
	case webcastStarting:
		c.setNotice("Chrome is starting…")
	case webcastRunning:
		if err := state.cast.ToggleLive(); err != nil {
			c.setNotice("Chrome cast: " + err.Error())
		}
	case webcastEnding:
		c.setNotice("Chrome is closing.")
	}
}

func (c *Console) startWebcast(cfg webcast.CastConfig) {
	ctx, cancel := context.WithCancel(c.lifetime)
	c.mu.Lock()
	if c.submissions != nil {
		cfg.ObservePublication = c.submissions.observePublication
	}
	c.wcast = webcastState{phase: webcastStarting, attempt: c.wcast.attempt + 1, cancel: cancel}
	attempt := c.wcast.attempt
	c.mu.Unlock()
	// Its links' codes (pair#495), bound to this start.
	cfg.OnCodeRotated = func(link string, r castlock.Rotation) { c.onCodeRotated(castWeb, attempt, link, r) }
	c.setNotice("Starting Chrome…")
	c.GoTracked(func() {
		cast, err := webcast.StartCast(ctx, cfg)
		cancel()
		claim := &startClaim{}
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.webcastStarted(attempt, cast, err, claim)
			return nil
		})
		if cast != nil && claim.abandon() {
			cast.End()
		}
	})
}

// webcastStarted runs on the loop when a start finishes; a cast it doesn't
// adopt, the start goroutine ends.
func (c *Console) webcastStarted(attempt uint64, cast *webcast.Cast, err error, claim *startClaim) {
	c.mu.Lock()
	current := c.wcast.attempt == attempt && c.wcast.phase == webcastStarting
	if !current {
		c.mu.Unlock()
		return
	}
	if err != nil || !claim.adopt() {
		c.wcast = webcastState{attempt: attempt}
		c.mu.Unlock()
		if err != nil {
			c.setWarning("Chrome failed to start: " + err.Error())
		}
		return
	}
	c.wcast.phase, c.wcast.cast, c.wcast.update = webcastRunning, cast, cast.Current()
	c.mu.Unlock()
	if c.submissions != nil {
		c.submissions.startSubmissionPreview(cast)
	}
	c.setNotice("Chrome is open. Press Shift+Ctrl+Alt+b or click Chrome ▶ to go live; right-click the cell for the view-only link, 👆/👽 enable annotation/driving and copy their links; right-click either to copy while off; ↗ brings Chrome to the front.")
	c.syncOverlay(overlayWeb, true) // the notes menu draws in it
	notes, stopNotes := cast.Notes()
	c.GoTracked(func() {
		defer stopNotes()
		for {
			select {
			case live, open := <-notes:
				if !open {
					return
				}
				_ = c.runTerminalCommand(c.lifetime, func() error {
					c.applyWebNotes(cast, live)
					return nil
				})
			case <-c.stop:
				return
			}
		}
	})
	if ctrl := cast.Drive(); ctrl != nil {
		// Remote driving (#467): who drives and who asks, for the tab bar
		// and the drive menu. The channel closes with the cast.
		states, stopStates := ctrl.Subscribe()
		c.GoTracked(func() {
			defer stopStates()
			for {
				select {
				case st, open := <-states:
					if !open {
						return
					}
					_ = c.runTerminalCommand(c.lifetime, func() error {
						c.applyDrive(cast, st)
						return nil
					})
				case <-c.stop:
					return
				}
			}
		})
	}
	updates, unsubscribe := cast.Subscribe()
	c.GoTracked(func() {
		defer unsubscribe()
		for {
			select {
			case u, open := <-updates:
				if !open {
					<-cast.Done()
					_ = c.runTerminalCommand(c.lifetime, func() error {
						c.webcastEnded(cast)
						return nil
					})
					return
				}
				_ = c.runTerminalCommand(c.lifetime, func() error {
					c.webcastUpdated(cast, u)
					return nil
				})
			case <-c.stop:
				// Shutdown ends the cast itself, with a bounded wait.
				return
			}
		}
	})
}

func (c *Console) webcastUpdated(cast *webcast.Cast, u webcast.Update) {
	c.mu.Lock()
	if c.wcast.cast != cast {
		c.mu.Unlock()
		return
	}
	oldRecovery := c.wcast.update.State.Recovery
	c.wcast.update = u
	c.mu.Unlock()
	if u.State.Recovery != "" && u.State.Recovery != oldRecovery {
		c.setWarning(u.State.Recovery)
	}
	c.repaint()
}

// endWebcast is a Ctrl+click on the Chrome cell: end the cast, close Chrome.
func (c *Console) endWebcast() {
	c.mu.Lock()
	cast := c.wcast.cast
	running := c.wcast.phase == webcastRunning
	if running {
		c.wcast.phase = webcastEnding
	}
	c.mu.Unlock()
	if !running {
		return
	}
	c.setNotice("Closing Chrome…")
	c.GoTracked(cast.End)
}

// webcastEnded runs on the loop once a cast is fully down.
func (c *Console) webcastEnded(cast *webcast.Cast) {
	c.mu.Lock()
	if c.wcast.cast != cast {
		c.mu.Unlock()
		return
	}
	byOperator := c.wcast.phase == webcastEnding
	c.wcast = webcastState{attempt: c.wcast.attempt}
	c.mu.Unlock()
	c.forgetCastCodes(castWeb)
	c.forgetWebNotes()
	c.forgetDrive()
	c.syncOverlay(overlayWeb, false)
	switch err := cast.Err(); {
	case byOperator || err == nil:
		c.setNotice("Chrome cast ended.")
	case errors.Is(err, webcast.ErrClosedByOperator):
		c.setNotice("Chrome closed — cast ended.")
	default:
		c.setWarning("Chrome cast ended: " + err.Error())
	}
}

// focusWebcast is a click on the ↗ button (#501): the operator's own
// action, so bringing Chrome in front of Couch is the point.
func (c *Console) focusWebcast() {
	c.mu.Lock()
	cast, running := c.wcast.cast, c.wcast.phase == webcastRunning
	c.mu.Unlock()
	if !running {
		return
	}
	c.GoTracked(func() {
		if err := cast.Focus(); err != nil {
			c.setNotice("Chrome cast: " + err.Error())
		}
	})
}

// webcastLink is one of the cast's three links as Couch copies it
// (pair#506): which it is, what its viewers may do, and why a cast whose
// view-only link answered may still have none of this kind.
type webcastLink struct {
	name   string
	kind   string // the link's label, broadcast.LinkView/LinkPointer/LinkDrive
	grants string
	// missing says why the cast has no such link; "" means it isn't ready.
	missing func(*webcast.Cast) string
}

var (
	webcastViewOnlyLink = webcastLink{name: "view-only", kind: broadcast.LinkView,
		grants: "its viewers watch; 👆 copies the pointer link, 👽 the drive link."}
	webcastPointerLink = webcastLink{name: "pointer", kind: broadcast.LinkPointer,
		grants: "its viewers watch; pointing and notes require 👆 on.",
		missing: func(*webcast.Cast) string {
			return "This cast takes no notes (it isn't recorded), so it has no pointer link."
		}}
	webcastDriveLink = webcastLink{name: "drive", kind: broadcast.LinkDrive,
		grants: "its viewers can ask to drive when 👽 is on; you grant.",
		missing: func(cast *webcast.Cast) string {
			if cast != nil && cast.DriveOff() != "" {
				return "This cast can't be driven: " + cast.DriveOff() + "."
			}
			return ""
		}}
)

// copyWebcastLinkOf copies one of the cast's links. The notice never
// contains the link: a screen broadcast would draw it to viewers.
func (c *Console) copyWebcastLinkOf(l webcastLink) {
	// One snapshot: the cast can end between two reads.
	c.mu.Lock()
	u, running, cast := c.wcast.update, c.wcast.phase == webcastRunning, c.wcast.cast
	c.mu.Unlock()
	missing := ""
	if u.Link != "" && l.missing != nil {
		missing = l.missing(cast)
	}
	switch link := u.LinkFor(l.kind); {
	case !running:
	case link != "":
		if !c.copyLink(link) {
			return
		}
		// Every link is locked by its own code (pair#495).
		c.setNotice("Chrome cast " + l.name + " link copied: " + l.grants + " Its viewers need its code: hover Chrome to see it.")
	case u.LinkErr != nil:
		c.setNotice("Chrome cast " + l.name + " link unavailable: " + u.LinkErr.Error())
	case missing != "":
		c.setNotice(missing)
	default:
		c.setNotice("Chrome cast " + l.name + " link is not ready yet.")
	}
}

// copyWebcastLink is a right-click on the Chrome cell, or the Codes menu's
// copy action on the view row: the view-only link.
func (c *Console) copyWebcastLink() { c.copyWebcastLinkOf(webcastViewOnlyLink) }

// copyWebcastPointerLink is the Codes menu copy action, independent of admission.
func (c *Console) copyWebcastPointerLink() { c.copyWebcastLinkOf(webcastPointerLink) }

// copyWebcastDriveLink is the Codes menu copy action, independent of admission.
func (c *Console) copyWebcastDriveLink() { c.copyWebcastLinkOf(webcastDriveLink) }

// endWebcastForShutdown ends a running or starting cast before the presenter
// is released, and waits, bounded, for Chrome and the link to close.
func (c *Console) endWebcastForShutdown() {
	c.mu.Lock()
	state := c.wcast
	c.wcast = webcastState{attempt: state.attempt + 1}
	c.mu.Unlock()
	if state.cancel != nil {
		state.cancel()
	}
	if cast := state.cast; cast != nil {
		ended := make(chan struct{})
		go func() {
			cast.End()
			close(ended)
		}()
		select {
		case <-ended:
		case <-time.After(webcastShutdownWait):
		}
	}
}

// SetReplayShares gives the console the replay shares (#477): it answers
// replay-share and replay-status with them, and closes them at teardown.
// Call before Run.
//
// A share becoming ready is a notice: a recording is now served through
// the tunnel, maybe at an agent's request. The notice never carries the
// link, which a screen broadcast would draw to its viewers.
func (c *Console) SetReplayShares(s *webcast.ReplayShares) {
	s.OnReady(func(id string, st webcast.ShareStatus) {
		msg := fmt.Sprintf("Replay of %s shared through the broadcast tunnel until Couch exits (%d shared).", id, len(s.Shared()))
		if st.Local {
			msg = fmt.Sprintf("Replay of %s shared on this machine only (the tunnel is off) until Couch exits (%d shared).", id, len(s.Shared()))
		}
		c.GoTracked(func() { c.setNotice(msg) })
	})
	c.replays = s
}

// SharedReplays is the recordings shared for replay, for --broadcast-list.
func (c *Console) SharedReplays() []string {
	if c.replays == nil {
		return nil
	}
	return c.replays.Shared()
}

// ReplayShare answers a replay request: replay-share asks for the
// recording to be shared, replay-status reads where its share is.
func (c *Console) ReplayShare(op, id string) couchmessage.ReplayShare {
	if c.replays == nil {
		return couchmessage.ReplayShare{State: string(webcast.ShareFailed), Error: "this Couch has no recordings store"}
	}
	read := c.replays.Status
	if op == "replay-share" {
		read = c.replays.Share
	}
	st := read(id)
	out := couchmessage.ReplayShare{State: string(st.State), Base: st.Base, Local: st.Local}
	if st.Err != nil {
		out.Error = st.Err.Error()
	}
	return out
}
