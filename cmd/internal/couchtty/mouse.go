package couchtty

import "github.com/xianxu/pair/cmd/internal/mouseinput"

// MouseDisposition is what happens to one decoded mouse report.
//
// Three-way, not a bool. "The child asked for this" and "the child must never
// see this" are the two cases this whole feature exists to separate: mouse
// reporting is a terminal-GLOBAL mode, so couch enabling it for its own row
// means a child that never asked starts receiving reports unless something
// withholds them.
type MouseDisposition uint8

const (
	// MouseSwallow drops the report. The child never asked for mouse reporting,
	// so it must receive nothing -- otherwise pointer activity types SGR bytes
	// into it as typeahead.
	MouseSwallow MouseDisposition = iota
	// MouseForward writes the report to the child verbatim.
	MouseForward
	// MouseCouch acts on couch's own surface.
	MouseCouch
)

// RouteMouseReport decides what happens to one report.
//
// The full table, because the first draft of this had a rule per prose
// paragraph and two of them contradicted each other:
//
//	press button 0 on couch's row                    -> couch
//	any other report on couch's row, child has mouse -> forward
//	any other report on couch's row, child has none  -> swallow
//	press button 0 anywhere, SWITCHER up             -> couch
//	any other report anywhere, SWITCHER up           -> swallow
//	anything elsewhere, child has mouse              -> forward
//	anything elsewhere, child has none               -> swallow
//
// couch's row is the LAST row, which it owns by reservation. When the switcher
// is up couch owns the whole screen, because no child is displayed and a click
// forwarded to one would go somewhere the operator cannot see.
//
// The release rule is NARROWER than termcmd's, deliberately. termcmd forwards
// every release unconditionally (run.go:449-452), which is right where the child
// is already receiving presses. Here a child with no tracking must receive
// NOTHING, and a release it never saw a press for is both an unpaired event and
// a contradiction of that. So a release follows the same childWantsMouse rule as
// everything else.
//
// The wheel is not couch's gesture either: on couch's row it swallows or
// forwards like any other non-zero button. Translating it to a scroll action is
// termcmd's job in termcmd's context, and a second translator would be two
// policies for one gesture.
func RouteMouseReport(event mouseinput.Event, hostRows int, childWantsMouse, couchOwnsScreen bool) MouseDisposition {
	// The RAW button here, deliberately, unlike the wheel predicate below and in
	// termcmd. A MODIFIED click is not couch's gesture: #213 narrowed its change
	// to the wheel precisely because ctrl+click may mean something to a child,
	// and shift/ctrl+click on couch's row therefore forwards or swallows like
	// any other report rather than switching threads. Modifier bits live in the
	// button field, so this is a real distinction, not an oversight.
	press := !event.Release && event.Button == 0
	if hostRows > 0 && event.Y == hostRows && press {
		return MouseCouch
	}
	// When the SWITCHER is up, couch owns the whole screen: no child is
	// displayed, so forwarding a click to one would deliver it to something the
	// operator cannot see. Missing this made every switcher click a swallow --
	// the panel branch below was unreachable, and the test that clicked a row
	// found it.
	if couchOwnsScreen {
		if press {
			return MouseCouch
		}
		return MouseSwallow
	}
	if childWantsMouse {
		return MouseForward
	}
	return MouseSwallow
}

// stripWheelResizeModifier clears the ctrl bit from a WHEEL report, returning
// the rewritten event and bytes. Every other report passes through untouched.
//
// Why couch has to do this at all: zellij 0.44.3 maps ctrl+wheel to a pane
// resize and offers no way to turn it off — its config parser silently ignores
// unknown keys, so setting the newer `mouse_scroll_resize` there is a no-op that
// LOOKS accepted. Ghostty cannot suppress it either: wheel events are not
// bindable triggers, only keys and modifiers are. couch owns the host tty and
// sees these bytes before zellij does, which makes it the only interception
// point above the resize. pair#213 records how each layer was ruled out.
//
// DELETE THIS once zellij is upgraded to a version carrying
// `mouse_scroll_resize`: that option supersedes the filter for both couch and
// standalone pair, and this function should go with it. Without the name here it
// calcifies into a workaround nobody can date or justify.
//
// Strip rather than swallow. Swallowing makes ctrl+scroll do nothing; stripping
// makes it scroll, which is what the operator wants when their hand happens to
// be resting on ctrl.
//
// Narrow deliberately, on three axes:
//   - Wheel only. Ctrl+click may mean something to a child (nvim, a TUI), and
//     taking a modifier away from every button is a far larger behavioural
//     change than this issue asks for.
//   - VERTICAL wheel only. Horizontal wheel (66/67) is left alone because
//     zellij's resize is mapped off the vertical wheel; stripping ctrl there
//     would alter a gesture that is not causing harm.
//   - Ctrl only, so ctrl+shift+wheel still arrives as shift+wheel.
//
// The predicate is the modifier-MASKED button. Modifier bits live in the button
// field, so ctrl+wheel-up is 80 and a `Button == WheelUp` test would match
// nothing at all — the change would compile and silently do nothing.
func stripWheelResizeModifier(event mouseinput.Event, raw []byte) (mouseinput.Event, []byte) {
	base := mouseinput.BaseButton(event.Button)
	if base != mouseinput.WheelUp && base != mouseinput.WheelDown {
		return event, raw
	}
	if event.Button&mouseinput.ModCtrl == 0 {
		return event, raw
	}
	stripped, ok := mouseinput.WithButton(raw, event.Button&^mouseinput.ModCtrl)
	if !ok {
		// raw reached us through Parse, so this cannot fail. If it ever did,
		// forwarding the original beats dropping the report on the floor.
		return event, raw
	}
	event.Button &^= mouseinput.ModCtrl
	return event, stripped
}
