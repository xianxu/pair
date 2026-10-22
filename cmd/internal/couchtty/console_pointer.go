package couchtty

import (
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
)

// pointerPhase is the console's side of a broadcast's pointer link (#412).
// Transitions, all on the Run loop, all only while the broadcast is live:
//
//	none --click 👆-->                 on: link minted and copied, watch armed
//	on   --click 👆-->                 off: marks cleared, link stays view-only
//	off  --click 👆-->                 on: the same link, re-copied
//	on   --👆 hidden past the grace--> off: marks cleared, notice
//	any  --right-click 👆-->           the link minted/copied, admission unchanged
//	any  --broadcast ends-->           none: marks cleared, overlay removed
type pointerPhase uint8

const (
	pointerNone pointerPhase = iota
	pointerOn
	pointerOff
)

// pointerMarks is the marks state the overlay reads on the Presenter
// goroutine. mu is a leaf lock: it guards marks and nothing else, and is
// never held across a call to the Presenter, the session or the console.
type pointerMarks struct {
	mu    sync.Mutex
	marks *broadcast.Marks
	timer *time.Timer
}

// screenOverlay draws what the collaboration layer adds to a public frame:
// sticky notes, marks over them, and the notes menu over everything. A
// private frame (the switcher) gets none of it. It runs on the Presenter
// goroutine.
func (c *Console) screenOverlay(f terminal.Frame, class terminal.FrameClass) terminal.Frame {
	return c.menuOverlay(c.markOverlay(c.noteOverlay(f, class), class), class)
}

// overlayUser is who needs the collaboration overlay on the screen: the
// screen broadcast (its marks, notes and menu) or the web cast (the notes
// menu for its notes, #470).
type overlayUser uint8

const (
	overlayScreen overlayUser = 1 << iota
	overlayWeb
)

// syncOverlay records whether user needs the overlay and installs it while
// anyone does, removing it when nobody does. Every caller runs on the
// terminal loop (broadcast attach and detach, web cast start and end), so
// the SetOverlay after unlocking follows the record it was computed from.
func (c *Console) syncOverlay(user overlayUser, on bool) {
	c.mu.Lock()
	if on {
		c.overlayUsers |= user
	} else {
		c.overlayUsers &^= user
	}
	want := c.overlayUsers != 0
	c.mu.Unlock()
	var overlay func(terminal.Frame, terminal.FrameClass) terminal.Frame
	if want {
		overlay = c.screenOverlay
	}
	c.terminalError(c.presenter.SetOverlay(c.lifetime, overlay))
}

// markOverlay draws the marks on a public frame; a private frame (the
// switcher) is never marked. It runs on the Presenter goroutine.
func (c *Console) markOverlay(f terminal.Frame, class terminal.FrameClass) terminal.Frame {
	if class != terminal.FramePublic {
		return f
	}
	c.pmarks.mu.Lock()
	defer c.pmarks.mu.Unlock()
	if c.pmarks.marks == nil {
		return f
	}
	return c.pmarks.marks.Overlay(f, c.now())
}

func (c *Console) pointerCellLocked() PointerCell {
	if c.pointer == pointerOn {
		return PointerOn
	}
	return PointerOff
}

// onPointerClick is a click on 👆: left toggles pointing, right re-copies
// the link.
func (c *Console) onPointerClick(right bool) {
	c.mu.Lock()
	s, phase := c.bcast.session, c.pointer
	live := c.bcast.phase == broadcastLive
	c.mu.Unlock()
	if !live || s == nil {
		return
	}
	action := "enable"
	if phase == pointerOn {
		action = "disable"
	}
	intent := castControlIntent(right, capability.Presentation{Action: action})
	if intent == "copy" {
		if link, err := s.EnsurePointerLink(); err == nil {
			if !c.copyLink(link) {
				return
			}
			c.setNotice("Pointer link copied. Its viewers need its code: hover LIVE to see it.")
		} else {
			c.setNotice("Couldn’t create pointer link: " + err.Error())
		}
		return
	}
	if intent == "disable" {
		s.DisablePointer()
		c.pointerOff("Pointing off.")
		return
	}
	link, err := s.EnablePointer()
	if err != nil {
		c.setNotice("Pointing failed to start.")
		return
	}
	c.mu.Lock()
	c.pointer = pointerOn
	c.mu.Unlock()
	c.resyncNotes(s)
	if !c.copyLink(link) {
		return
	}
	// The notice never contains the link: it would be drawn into the frames
	// being broadcast.
	c.setNotice("Pointing on — link copied. Click 👆 to turn it off. Its viewers need its code: hover LIVE to see it.")
}

func (c *Console) copyLink(link string) bool {
	if err := c.presenter.Copy(c.lifetime, []byte(link)); err != nil {
		c.terminalError(err)
		c.setNotice("Couldn’t copy link; right-click to retry.")
		return false
	}
	return true
}

// copyViewLink is a right-click on LIVE ⏸: the view-only link again.
func (c *Console) copyViewLink() {
	c.mu.Lock()
	s, live := c.bcast.session, c.bcast.phase == broadcastLive
	c.mu.Unlock()
	if !live || s == nil {
		return
	}
	c.copyLink(s.Link())
	c.setNotice("Broadcast link copied. Its viewers need its code: hover LIVE to see it.")
}

// pointerOff ends pointing on the console's side: phase off, marks cleared
// and repainted away, with a notice.
func (c *Console) pointerOff(notice string) {
	c.mu.Lock()
	if c.pointer == pointerOn {
		c.pointer = pointerOff
	}
	s := c.bcast.session
	c.mu.Unlock()
	// The session already emptied its board; take that, notes and version.
	c.resyncNotes(s)
	c.clearMarks()
	c.setNotice(notice)
}

// pointerOffByWatch is the session reporting that the active 👆 stayed off
// the operator's screen; it reaches the Run loop like any input.
func (c *Console) pointerOffByWatch() {
	_ = c.runTerminalCommand(c.lifetime, func() error {
		c.mu.Lock()
		on := c.pointer == pointerOn
		c.mu.Unlock()
		if on {
			c.pointerOff("Pointing off: 👆 wasn't visible.")
		}
		return nil
	})
}

// onPoints is the session handing over a batch that passed its checks. It
// runs on a request goroutine; the batch is applied on the Run loop, where
// the phase and the screen are checked again: a batch in flight when pointing
// turned off or the switcher opened is dropped (#412 security review, L1).
func (c *Console) onPoints(b broadcast.PointBatch) {
	_ = c.runTerminalCommand(c.lifetime, func() error {
		c.applyPoints(b)
		return nil
	})
}

func (c *Console) applyPoints(b broadcast.PointBatch) {
	c.mu.Lock()
	ok := c.pointer == pointerOn && c.bcast.phase == broadcastLive && !c.focus.IsPanel()
	palette := c.menu.Palette
	c.mu.Unlock()
	state, confirmed := c.confirmedChromeState()
	if !ok || !confirmed || state.Host.Cols != b.Cols || state.Host.Rows != b.Rows || state.Rows != b.ChromeRows {
		return
	}
	now := c.now()
	c.pmarks.mu.Lock()
	if c.pmarks.marks == nil {
		c.pmarks.marks = broadcast.NewMarks()
	}
	// Fade into the operator's real background when the terminal can show
	// it (truecolor, and the background reported by OSC 11).
	c.pmarks.marks.SetBlend(palette.BG, blendable(palette))
	c.pmarks.marks.Add(b.Points, b.Cols, b.Rows, b.ChromeRows, b.Color, now)
	c.pmarks.mu.Unlock()
	c.refreshMarks()
}

// refreshMarks repaints the marks and schedules the next fade step while any
// remain.
func (c *Console) refreshMarks() {
	c.terminalError(c.presenter.Refresh(c.lifetime))
	c.pmarks.mu.Lock()
	defer c.pmarks.mu.Unlock()
	if c.pmarks.timer != nil {
		c.pmarks.timer.Stop()
		c.pmarks.timer = nil
	}
	if c.pmarks.marks == nil {
		return
	}
	d, ok := c.pmarks.marks.NextChange(c.now())
	if !ok {
		return
	}
	c.pmarks.timer = time.AfterFunc(d, func() {
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.refreshMarks()
			return nil
		})
	})
}

// clearMarks removes every mark and repaints them away.
func (c *Console) clearMarks() {
	c.pmarks.mu.Lock()
	had := c.pmarks.marks != nil
	c.pmarks.marks = nil
	if c.pmarks.timer != nil {
		c.pmarks.timer.Stop()
		c.pmarks.timer = nil
	}
	c.pmarks.mu.Unlock()
	if had {
		c.terminalError(c.presenter.Refresh(c.lifetime))
	}
}

// resetPointer forgets the pointer when the broadcast ends.
func (c *Console) resetPointer() {
	c.mu.Lock()
	c.pointer = pointerNone
	c.mu.Unlock()
	c.pmarks.mu.Lock()
	c.pmarks.marks = nil
	if c.pmarks.timer != nil {
		c.pmarks.timer.Stop()
		c.pmarks.timer = nil
	}
	c.pmarks.mu.Unlock()
	c.forgetNotes()
}
