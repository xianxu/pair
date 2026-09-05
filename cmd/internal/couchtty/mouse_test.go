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
		panel bool
		want  MouseDisposition
	}{
		{"button 0 press on couch's row acts", mouseinput.Event{Button: 0, Y: rows}, false, false, MouseCouch},
		{"...even when the child wants mouse", mouseinput.Event{Button: 0, Y: rows}, true, false, MouseCouch},
		{"another button on couch's row, no child mouse", mouseinput.Event{Button: 2, Y: rows}, false, false, MouseSwallow},
		{"another button on couch's row, child wants mouse", mouseinput.Event{Button: 2, Y: rows}, true, false, MouseForward},
		// The wheel is not couch's gesture; it follows the ordinary rule.
		{"wheel on couch's row, no child mouse", mouseinput.Event{Button: mouseinput.WheelUp, Y: rows}, false, false, MouseSwallow},
		{"wheel on couch's row, child wants mouse", mouseinput.Event{Button: mouseinput.WheelUp, Y: rows}, true, false, MouseForward},
		// A release to a child that never saw the press is an unpaired event AND
		// a contradiction of the zero-bytes rule -- narrower than termcmd's.
		{"release on couch's row, no child mouse", mouseinput.Event{Button: 0, Y: rows, Release: true}, false, false, MouseSwallow},
		{"release on couch's row, child wants mouse", mouseinput.Event{Button: 0, Y: rows, Release: true}, true, false, MouseForward},
		{"press elsewhere, no child mouse", mouseinput.Event{Button: 0, Y: 3}, false, false, MouseSwallow},
		{"press elsewhere, child wants mouse", mouseinput.Event{Button: 0, Y: 3}, true, false, MouseForward},
		{"release elsewhere, no child mouse", mouseinput.Event{Button: 0, Y: 3, Release: true}, false, false, MouseSwallow},
		{"release elsewhere, child wants mouse", mouseinput.Event{Button: 0, Y: 3, Release: true}, true, false, MouseForward},
		// With the SWITCHER up couch owns the screen: no child is displayed, so
		// a forward would deliver the click somewhere invisible.
		{"press in the switcher", mouseinput.Event{Button: 0, Y: 5}, false, true, MouseCouch},
		{"press in the switcher, child wants mouse", mouseinput.Event{Button: 0, Y: 5}, true, true, MouseCouch},
		{"release in the switcher", mouseinput.Event{Button: 0, Y: 5, Release: true}, true, true, MouseSwallow},
		{"wheel in the switcher", mouseinput.Event{Button: mouseinput.WheelUp, Y: 5}, true, true, MouseSwallow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := RouteMouseReport(tc.event, rows, tc.child, tc.panel); got != tc.want {
				t.Errorf("RouteMouseReport(%+v, child=%v, panel=%v) = %v, want %v", tc.event, tc.child, tc.panel, got, tc.want)
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
				if got := RouteMouseReport(event, rows, false, false); got == MouseForward {
					t.Fatalf("%+v was forwarded to a child that never enabled mouse tracking", event)
				}
			}
		}
	}
}
