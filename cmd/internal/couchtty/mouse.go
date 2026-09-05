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
//	press button 0 on couch's row          -> couch
//	any other press on couch's row, child has mouse  -> forward
//	any other press on couch's row, child has none   -> swallow
//	release on couch's row, child has mouse -> forward
//	release on couch's row, child has none  -> swallow
//	anything elsewhere, child has mouse     -> forward
//	anything elsewhere, child has none      -> swallow
//
// couch's row is the LAST row, which it owns by reservation.
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
func RouteMouseReport(event mouseinput.Event, hostRows int, childWantsMouse bool) MouseDisposition {
	onCouchRow := hostRows > 0 && event.Y == hostRows
	if onCouchRow && !event.Release && event.Button == 0 {
		return MouseCouch
	}
	if childWantsMouse {
		return MouseForward
	}
	return MouseSwallow
}
