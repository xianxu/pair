package couchtty

import (
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// pointerPhase is the console's side of a broadcast's pointer link (#412).
// Transitions, all on the Run loop, all only while the broadcast is live:
//
//	none --click 👆-->                 on: link minted and copied, watch armed
//	on   --click 👆-->                 off: marks cleared, link stays view-only
//	off  --click 👆-->                 on: the same link, re-copied
//	on   --👆 hidden past the grace--> off: marks cleared, notice
//	any  --right-click 👆-->           the link re-copied (if one exists)
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

// overlay draws the marks on a public frame; a private frame (the switcher)
// is never marked. It runs on the Presenter goroutine.
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
	if right {
		if link := s.PointerLink(); link != "" {
			c.copyLink(link)
			c.setNotice("Pointer link copied.")
		}
		return
	}
	if phase == pointerOn {
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
	c.copyLink(link)
	// The notice never contains the link: it would be drawn into the frames
	// being broadcast.
	c.setNotice("Pointing on — link copied. Click 👆 to turn it off.")
}

func (c *Console) copyLink(link string) {
	if err := c.presenter.Copy(c.lifetime, []byte(link)); err != nil {
		c.terminalError(err)
	}
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
	c.setNotice("Broadcast link copied.")
}

// onRemoteClick is a click on 👽, which waits for #407.
func (c *Console) onRemoteClick() {
	c.setNotice("Remote control isn't available yet.")
}

// pointerOff ends pointing on the console's side: phase off, marks cleared
// and repainted away, with a notice.
func (c *Console) pointerOff(notice string) {
	c.mu.Lock()
	if c.pointer == pointerOn {
		c.pointer = pointerOff
	}
	c.mu.Unlock()
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
	c.mu.Unlock()
	if !ok {
		return
	}
	now := c.now()
	c.pmarks.mu.Lock()
	if c.pmarks.marks == nil {
		c.pmarks.marks = broadcast.NewMarks()
	}
	c.pmarks.marks.Add(b.Points, b.Cols, b.Rows, now)
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
}
