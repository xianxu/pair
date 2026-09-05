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
