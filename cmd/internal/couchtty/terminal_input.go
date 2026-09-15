package couchtty

import (
	"bytes"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func (c *Console) routeInputEvent(event terminal.InputEvent) {
	if event.Reply {
		return
	}
	switch event.Event.(type) {
	case uv.MouseClickEvent, uv.MouseReleaseEvent, uv.MouseMotionEvent, uv.MouseWheelEvent:
		c.routeMouseEvent(event)
		return
	}
	c.mu.Lock()
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	route := func(raw []byte) {
		if len(raw) == 0 {
			return
		}
		if panel {
			if bytes.Equal(raw, []byte{27}) {
				c.onMenuKey(PanelKey{Kind: KeyEscape})
			} else {
				c.onMenuInput(raw)
			}
		} else {
			c.terminalError(c.presenter.Input(c.lifetime, event.Event))
		}
	}
	switch event.Event.(type) {
	case uv.PasteEvent:
		route(event.Raw)
		return
	case uv.KeyReleaseEvent, uv.FocusEvent, uv.BlurEvent:
		c.terminalError(c.presenter.Input(c.lifetime, event.Event))
		return
	}
	if bytes.Equal(event.Canonical, []byte{27}) {
		route(event.Canonical)
		return
	}
	// Framing already completed in Decoder. Interceptor is now only a product
	// chord recognizer; retaining any prefix would lose its associated UV event.
	var policy Interceptor
	before, hit, _ := policy.FeedHit(productKey(event))
	if hit == HitNone {
		route(event.Canonical)
		return
	}
	c.dispatchInputCandidate(before, hit, policy.RawHit(), route)
}

func (c *Console) routeMouseEvent(event terminal.InputEvent) {
	if wheel, ok := event.Event.(uv.MouseWheelEvent); ok {
		m := uv.Mouse(wheel)
		if m.Button == uv.MouseWheelUp || m.Button == uv.MouseWheelDown {
			m.Mod &^= uv.ModCtrl
			event.Event = uv.MouseWheelEvent(m)
		}
	}
	// Release/motion always visit the presenter, including over panels and
	// chrome, so its gesture owner can cancel or clip them consistently.
	if _, ok := event.Event.(uv.MouseReleaseEvent); ok {
		c.terminalError(c.presenter.Input(c.lifetime, event.Event))
		return
	}
	hit, _, _, ok := mouseinput.ParsePrefix(event.Raw)
	if ok && !hit.Release && hit.Button == 0 {
		c.mu.Lock()
		rows, panel := int(c.size.Rows), c.focus.IsPanel()
		chips, extents := c.statusChips, c.menuExtents
		c.mu.Unlock()
		if hit.Y == rows {
			if thread, ok := (RenderedStatusRow{Chips: chips}).ColumnToActor(hit.X - 1); ok {
				c.switchToThread(thread)
			}
			return
		}
		if panel {
			if thread, ok := (RenderedMenu{Extents: extents}).PointToActor(hit.Y-1, hit.X-1); ok {
				c.switchToThread(thread)
			}
			return
		}
	}
	c.terminalError(c.presenter.Input(c.lifetime, event.Event))
}

// Some product chords intentionally exist only in enhanced encoding (Alt+d
// and Alt+n), while their legacy ESC-letter bytes remain ordinary input.
func productKey(event terminal.InputEvent) []byte {
	if bytes.Equal(event.Raw, []byte{8}) {
		return event.Raw
	}
	if key, ok := event.Event.(uv.KeyPressEvent); ok && bytes.HasPrefix(event.Raw, []byte("\x1b[")) && bytes.HasSuffix(event.Raw, []byte("u")) {
		mod := uv.Key(key).Mod &^ (uv.ModCapsLock | uv.ModNumLock | uv.ModScrollLock)
		return []byte(fmt.Sprintf("\x1b[%d;%du", uv.Key(key).Code, int(mod)+1))
	}
	return event.Canonical
}
