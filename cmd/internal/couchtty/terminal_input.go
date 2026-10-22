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
	broker := c.messageBroker
	var scope, tag string
	if p := c.panes[c.focus.Actor()]; p != nil {
		scope, tag = p.thread.RepoScope, string(p.thread.Tag)
	}
	c.mu.Unlock()
	if panel {
		c.traceDropped("panel", nil)
		return
	}
	if broker != nil && scope != "" && tag != "" {
		switch event.(type) {
		case uv.KeyPressEvent, uv.PasteEvent:
			broker.ObserveInputThread(scope, tag)
		}
	}
	c.deliverPresenterInput(event)
}

func (c *Console) routeInputEvent(event terminal.InputEvent) {
	if event.Reply {
		// A reply is never a child's input. The terminal's colour answers are
		// the one kind couch asked for itself (pair#247).
		if c.capturePalette(event.Event) {
			c.restyleNotes()
			c.repaintVisible()
		}
		return
	}
	c.observeCopyKey(event)
	switch event.Event.(type) {
	case uv.MouseClickEvent, uv.MouseReleaseEvent, uv.MouseMotionEvent, uv.MouseWheelEvent:
		c.routeMouseEvent(event)
		return
	case uv.KeyPressEvent:
		// The open attention list (pair#673), then the Codes menu (pair#495),
		// take their arrows, Enter and Escape.
		if c.routeAttentionKey(event.Raw, event.Canonical) {
			return
		}
		if c.routeCodesMenuKey(event.Raw, event.Canonical) {
			return
		}
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
	// Geometry is usable only when the presenter has confirmed this exact hit map.
	// Gesture cleanup still reaches its owner during an uncertain publication.
	if !c.chromeConfirmed() {
		switch event.Event.(type) {
		case uv.MouseReleaseEvent, uv.MouseMotionEvent:
			c.deliverPresenterInput(event.Event)
		}
		return
	}

	if wheel, ok := event.Event.(uv.MouseWheelEvent); ok {
		m := uv.Mouse(wheel)
		if m.Button == uv.MouseWheelUp || m.Button == uv.MouseWheelDown {
			m.Mod &^= uv.ModCtrl
			event.Event = uv.MouseWheelEvent(m)
		}
	}
	// The open attention list (pair#673) and notes menu (#447 M4) own the
	// mouse over them, before anything reaches the presenter or the child.
	if c.routeAttentionPanel(event.Raw) {
		return
	}
	if c.routeNotesMenu(event.Raw) {
		return
	}
	// Cast link codes (pair#495): the Codes menu, Alt+click on a cast cell,
	// and the hover tooltip (which never takes the event).
	if c.routeCastCodes(event.Raw) {
		return
	}
	// Release/motion always visit the presenter, including over panels and
	// chrome, so its gesture owner can cancel or clip them consistently.
	if _, ok := event.Event.(uv.MouseReleaseEvent); ok {
		c.deliverPresenterInput(event.Event)
		return
	}
	hit, _, _, ok := mouseinput.ParsePrefix(event.Raw)
	// The cast row's rule (pair#495): a left click acts or toggles, a
	// right-click copies that control's link, and Alt+click opens the Codes
	// menu on LIVE ⏸ and Chrome ⏸ only (routeCastCodes, above). The Chrome
	// cell (#454): click toggles live, right-click copies the view-only link,
	// Ctrl with either button ends the cast (a macOS terminal may turn
	// Ctrl+left into a right click, keeping the Ctrl bit).
	if ok && !hit.Release {
		button := mouseinput.BaseButton(hit.Button)
		c.mu.Lock()
		rows, cell, focusSpan, driveSpan := int(c.size.Rows)-c.statusRows+1, c.statusWebcast, c.statusWebFocus, c.statusWebDrive
		pointerLink, driveLink := c.statusWebPointerLink, c.statusWebDriveLink
		c.mu.Unlock()
		// Left clicks toggle capabilities; right clicks only copy stable links.
		// The focus arrow and Chrome cell keep their separate actions.
		if hit.Y == rows && hit.Button&mouseinput.ModCtrl == 0 {
			switch {
			case button == 0 && focusSpan.Contains(hit.X-1):
				c.focusWebcast()
				return
			case (button == 0 || button == 2) && pointerLink.Contains(hit.X-1):
				c.onWebCapabilityClick("annotation", button == 2)
				return
			case (button == 0 || button == 2) && driveLink.Contains(hit.X-1):
				c.onWebCapabilityClick("drive", button == 2)
				return
			}
		}
		// Remote driving's span (#467): a click opens the drive menu;
		// Ctrl with either button takes control back, whatever is open.
		if (button == 0 || button == 2) && hit.Y == rows && driveSpan.Contains(hit.X-1) {
			if hit.Button&mouseinput.ModCtrl != 0 {
				c.revokeDrive()
			} else {
				c.toggleNotesMenu(driveMenu)
			}
			return
		}
		if (button == 0 || button == 2) && hit.Y == rows && cell.Contains(hit.X-1) {
			switch {
			case hit.Button&mouseinput.ModCtrl != 0:
				c.endWebcast()
			case button == 2:
				// The view-only link; 👽 and the Codes menu copy the drive
				// link (pair#495 dropped Alt+right-click for it).
				c.copyWebcastLink()
			default:
				c.toggleWebcast()
			}
			return
		}
	}
	// The pointer controls take left and right clicks on the status row
	// (#412); a right click anywhere else keeps going to the child.
	if ok && !hit.Release && (hit.Button == 0 || hit.Button == 2) {
		c.mu.Lock()
		rows := int(c.size.Rows) - c.statusRows + 1
		pointer, control, notes, webNotesSpan := c.statusPointer, c.statusControl, c.statusNotes, c.statusWebNotes
		attention := c.statusAttention
		c.mu.Unlock()
		// Right-click on LIVE re-copies the view-only link; a left click
		// (below) still stops the broadcast.
		if hit.Button == 2 && hit.Y == rows && control.Contains(hit.X-1) {
			c.copyViewLink()
			return
		}
		if hit.Y == rows && pointer.Contains(hit.X-1) {
			c.onPointerClick(hit.Button == 2)
			return
		}
		if hit.Button == 0 && hit.Y == rows && notes.Contains(hit.X-1) {
			c.toggleNotesMenu(screenNotes)
			return
		}
		if hit.Button == 0 && hit.Y == rows && webNotesSpan.Contains(hit.X-1) {
			c.toggleNotesMenu(webNotes)
			return
		}
		if hit.Button == 0 && hit.Y == rows && attention.Contains(hit.X-1) {
			c.toggleAttention()
			return
		}
	}
	if ok && !hit.Release && hit.Button == 0 {
		c.mu.Lock()
		rows, panel := int(c.size.Rows), c.focus.IsPanel()
		top := rows - c.statusRows + 1
		chips, extents, control := c.statusChips, c.menuExtents, c.statusControl
		groups := c.statusGroups
		c.mu.Unlock()
		if hit.Y >= top && hit.Y <= rows {
			// The broadcast cell is checked first: anywhere in its red span
			// stops the broadcast (#395).
			if hit.Y == top && control.Contains(hit.X-1) {
				c.toggleBroadcast()
				return
			}
			for _, group := range groups {
				if group.Row == hit.Y-top && hit.X-1 >= group.Start && hit.X-1 < group.End {
					c.openGroupSwitcher(group.Group)
					return
				}
			}
			for _, chip := range chips {
				if chip.Row == hit.Y-top && hit.X-1 >= chip.Start && hit.X-1 < chip.End {
					c.switchToThread(chip.Thread)
					break
				}
			}
			return
		}
		if panel {
			if key, thread, ok := (RenderedMenu{Extents: extents}).PointToRow(hit.Y-1, hit.X-1); ok {
				c.reduceMenu(MenuEvent{Kind: MenuEventMouseSwitch, Address: thread, RowKey: key})
			}
			return
		}
	}
	c.deliverPresenterInput(event.Event)
}

// Several product chords intentionally exist only in enhanced encoding -- every
// workbenchshortcut row with no legacy form, Alt+d and Alt+n among them -- while
// their legacy ESC-letter bytes remain ordinary input. They reach Couch only
// because the presenter keeps the parent disambiguated on every screen it
// presents on (#279).
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
