package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/mouseinput"
)

// Every row of the disposition table, one case each, so a row added later
// without a case is visible. Two of these contradict a naive reading and are the
// reason the table exists at all.
func TestRouteMouseReportCoversEveryRow(t *testing.T) {
	const rows = 24
	for _, tc := range []struct {
		name  string
		event mouseinput.Event
		child bool
		want  MouseDisposition
	}{
		{"button 0 press on couch's row acts", mouseinput.Event{Button: 0, Y: rows}, false, MouseCouch},
		{"...even when the child wants mouse", mouseinput.Event{Button: 0, Y: rows}, true, MouseCouch},
		{"another button on couch's row, no child mouse", mouseinput.Event{Button: 2, Y: rows}, false, MouseSwallow},
		{"another button on couch's row, child wants mouse", mouseinput.Event{Button: 2, Y: rows}, true, MouseForward},
		// The wheel is not couch's gesture; it follows the ordinary rule.
		{"wheel on couch's row, no child mouse", mouseinput.Event{Button: mouseinput.WheelUp, Y: rows}, false, MouseSwallow},
		{"wheel on couch's row, child wants mouse", mouseinput.Event{Button: mouseinput.WheelUp, Y: rows}, true, MouseForward},
		// A release to a child that never saw the press is an unpaired event AND
		// a contradiction of the zero-bytes rule -- narrower than termcmd's.
		{"release on couch's row, no child mouse", mouseinput.Event{Button: 0, Y: rows, Release: true}, false, MouseSwallow},
		{"release on couch's row, child wants mouse", mouseinput.Event{Button: 0, Y: rows, Release: true}, true, MouseForward},
		{"press elsewhere, no child mouse", mouseinput.Event{Button: 0, Y: 3}, false, MouseSwallow},
		{"press elsewhere, child wants mouse", mouseinput.Event{Button: 0, Y: 3}, true, MouseForward},
		{"release elsewhere, no child mouse", mouseinput.Event{Button: 0, Y: 3, Release: true}, false, MouseSwallow},
		{"release elsewhere, child wants mouse", mouseinput.Event{Button: 0, Y: 3, Release: true}, true, MouseForward},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RouteMouseReport(tc.event, rows, tc.child); got != tc.want {
				t.Errorf("RouteMouseReport(%+v, child=%v) = %v, want %v", tc.event, tc.child, got, tc.want)
			}
		})
	}
}

// A child with no mouse mode must receive NOTHING, which is the Done-when's
// "asserted, not assumed" stated as a property over the whole event space rather
// than as one case.
func TestNoReportReachesAChildThatNeverAskedForMouse(t *testing.T) {
	const rows = 24
	for _, button := range []int{0, 1, 2, mouseinput.WheelUp, mouseinput.WheelDown} {
		for _, release := range []bool{false, true} {
			for y := 1; y <= rows; y++ {
				event := mouseinput.Event{Button: button, X: 5, Y: y, Release: release}
				if got := RouteMouseReport(event, rows, false); got == MouseForward {
					t.Fatalf("%+v was forwarded to a child that never enabled mouse tracking", event)
				}
			}
		}
	}
}
