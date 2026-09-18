package couchtty

import (
	"bytes"
	"fmt"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// deliverPresenterInput is the ONE door from the console to Presenter.Input.
//
// It exists because the two are separate state machines that can legitimately
// disagree: the console tracks Focus, the presenter tracks View, and "I have no
// endpoint for this" is a correct answer from the second, not a failure of it.
// terminalError means "terminal ownership is lost"; routing a domain answer
// into it is what exited couch on a keystroke (pair#265). Every call site goes
// through here, and TestConsoleReachesPresenterInputOnlyThroughItsDoor pins it.
func (c *Console) deliverPresenterInput(event uv.Event) {
	err := c.presenter.Input(c.lifetime, event)
	if terminal.IsRoutingAnswer(err) {
		c.traceDropped("input", err)
		return
	}
	c.terminalError(err)
}

// deliverChildInput routes an event that only means something to a child.
//
// On the panel there is no child, and a key release, focus or blur has no panel
// meaning, so it is dropped.
//
// A no-destination answer with an actor focused is ignored too, deliberately.
// The obvious move is to surface it as a notice -- and that is a trap. setNotice
// is publishNotice (console.go), documented as "push and paint are therefore one
// operation", so it repaints through paintNow, which hands any UpdateChrome
// error to terminalError. That is the very exit this issue exists to remove,
// reached by a longer path.
func (c *Console) deliverChildInput(event uv.Event) {
	c.mu.Lock()
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	if panel {
		c.traceDropped("panel", nil)
		return
	}
	c.deliverPresenterInput(event)
}

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
			c.deliverChildInput(event.Event)
		}
	}
	switch event.Event.(type) {
	case uv.PasteEvent:
		route(event.Raw)
		return
	// These three reach the console because couch ASKS for them: the first mode
	// delta writes \x1b[?1004h (focus reporting) and \x1b[>3u (kitty flags 1|2,
	// whose flag 2 is "report event types", i.e. key release) on the very first
	// paint -- panel or not. They carry no panel meaning, so the panel check
	// governs them exactly as it governs printable keys and paste (pair#265).
	case uv.KeyReleaseEvent, uv.FocusEvent, uv.BlurEvent:
		c.deliverChildInput(event.Event)
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
		c.deliverPresenterInput(event.Event)
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
	c.deliverPresenterInput(event.Event)
}

// Some product chords intentionally exist only in enhanced encoding (Alt+d
// and Alt+n), while their legacy ESC-letter bytes remain ordinary input. They
// reach Couch only because the presenter keeps the parent disambiguated on
// every screen it presents on (#279).
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
