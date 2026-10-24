package couchtty

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// sendStatusTarget belongs to the accepted physical frame, not later focus.
type sendStatusTarget struct {
	pane     *pane
	thread   couchcore.ThreadAddress
	endpoint *terminal.Endpoint
	cast     *webcast.Cast
}

func (c *Console) sendStatusTargetLocked(p *pane) sendStatusTarget {
	if p == nil {
		return sendStatusTarget{}
	}
	return sendStatusTarget{pane: p, thread: p.thread, endpoint: p.child.Endpoint(), cast: c.wcast.cast}
}

// The existing cast watcher calls this only initially and on candidate wakes.
// Board observation is outside Console.mu; a successor cannot accept the result.
func (c *Console) refreshSendFeedback(cast *webcast.Cast) {
	pending, err := cast.HasSubmissionFeedback()
	pending = pending && err == nil
	_ = c.runTerminalCommand(c.lifetime, func() error {
		c.mu.Lock()
		if c.wcast.cast != cast || c.wcast.phase != webcastRunning || c.wcast.sendPending == pending {
			c.mu.Unlock()
			return nil
		}
		c.wcast.sendPending = pending
		c.mu.Unlock()
		c.repaint()
		return nil
	})
}

// routeMouseEvent has already confirmed the physical chrome revision.
func (c *Console) routeSendStatusClick(hit mouseinput.Event) bool {
	c.mu.Lock()
	row := int(c.size.Rows) - c.statusRows + 1
	if hit.Y != row || !c.statusWebSend.Contains(hit.X-1) {
		c.mu.Unlock()
		return false
	}
	// Consume the control's non-click inputs too, without forwarding a chord.
	if hit.Release || hit.Button != 0 {
		c.mu.Unlock()
		return true
	}
	target := c.statusSendTarget
	p := c.panes[c.active]
	valid := c.lifetime.Err() == nil && !c.focus.IsPanel() && p != nil && p == target.pane && p.thread == target.thread && p.child.Endpoint() == target.endpoint && target.endpoint != nil && c.wcast.phase == webcastRunning && c.wcast.cast == target.cast && c.wcast.sendPending
	var err error
	if valid {
		encoded := workbenchshortcut.ChordEncodings(workbenchshortcut.ChordCtrlAltShiftEnter)[0]
		var n int
		n, err = target.endpoint.WriteInput(encoded)
		if err == nil && n != len(encoded) {
			err = fmt.Errorf("send chord input was incomplete")
		}
	} else {
		err = fmt.Errorf("target changed; select the intended live actor and retry")
	}
	c.mu.Unlock()
	if err != nil {
		c.setNotice("Cannot send cast feedback: " + err.Error())
	}
	return true
}
