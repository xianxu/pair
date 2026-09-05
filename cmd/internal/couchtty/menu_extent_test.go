package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func extentFixture(withNotice bool) MenuState {
	one, two := menuAddress("one"), menuAddress("two")
	state := NewMenuState([]couchcore.ActionableThreadSummary{
		{Address: one, Name: "one", WorkingPath: "/w/one", State: couchcore.ThreadLive},
		{Address: two, Name: "two", WorkingPath: "/w/two", State: couchcore.ThreadLive},
	}, one)
	// `one` renders across THREE lines: its own, plus two attention messages.
	state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{
		one: {{Text: "first message"}, {Text: "second message"}},
	}
	if withNotice {
		state.Notice = errorMenuNotice("something to say")
	}
	return state
}

// An actor occupies a VARIABLE number of rows, so the map is point→actor, never
// point→line: a click on a notification line belongs to the actor above it.
//
// The notice case is not decoration. RenderMenuView inserts the notice at index
// 1, shifting every actor row down one; an extent computed before that shift is
// right by one line and wrong by one, which is the least visible way to be wrong.
func TestPointToActorSpansEveryLineOfAnActor(t *testing.T) {
	for _, withNotice := range []bool{false, true} {
		name := "no notice"
		if withNotice {
			name = "with a notice, which shifts every actor row down"
		}
		t.Run(name, func(t *testing.T) {
			state := extentFixture(withNotice)
			view := RenderMenuView(state, 60, 14, time.Unix(1_700_000_000, 0), false)
			lines := strings.Split(view.Body, "\r\n")

			one, two := menuAddress("one"), menuAddress("two")
			// Every line whose text names an actor must map to that actor, and
			// the attention lines must map to the actor they belong to.
			var oneRows, twoRows []int
			for row, line := range lines {
				switch {
				case strings.Contains(line, "/w/one"), strings.Contains(line, "message"):
					oneRows = append(oneRows, row)
				case strings.Contains(line, "/w/two"):
					twoRows = append(twoRows, row)
				}
			}
			if len(oneRows) < 3 {
				t.Fatalf("fixture drew %d rows for `one`, want its line plus two messages: %q", len(oneRows), lines)
			}
			for _, row := range oneRows {
				got, ok := view.PointToActor(row, 4)
				if !ok || got != one {
					t.Errorf("row %d (%q) mapped to (%+v,%v), want `one`", row, lines[row], got, ok)
				}
			}
			for _, row := range twoRows {
				got, ok := view.PointToActor(row, 4)
				if !ok || got != two {
					t.Errorf("row %d (%q) mapped to (%+v,%v), want `two`", row, lines[row], got, ok)
				}
			}
			// Below the last actor is nobody.
			if _, ok := view.PointToActor(len(lines)+5, 4); ok {
				t.Error("a row past the drawn menu mapped to an actor")
			}
			// The breadcrumb is nobody.
			if _, ok := view.PointToActor(0, 4); ok {
				t.Error("the breadcrumb row mapped to an actor")
			}
		})
	}
}
