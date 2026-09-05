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

// A row the height clamp cut is not drawn, so it is not clickable. Untested,
// clampExtents could return extents pointing past the end of Body and a click
// below the visible menu would select whatever the arithmetic landed on.
func TestExtentsNeverPointPastTheDrawnMenu(t *testing.T) {
	// More actors than rows, each with attention lines, so the clamp bites.
	var inventory []couchcore.ActionableThreadSummary
	attention := map[couchcore.ThreadAddress][]AttentionMessage{}
	for i := 0; i < 12; i++ {
		address := menuAddress(string(rune('a' + i)))
		inventory = append(inventory, couchcore.ActionableThreadSummary{
			Address: address, Name: string(rune('a' + i)), WorkingPath: "/w", State: couchcore.ThreadLive,
		})
		attention[address] = []AttentionMessage{{Text: "paging"}}
	}
	state := NewMenuState(inventory, inventory[0].Address)
	state.Attention = attention

	for _, height := range []int{10, 12, 16, 24} {
		view := RenderMenuView(state, 60, height, time.Unix(1_700_000_000, 0), false)
		drawn := len(strings.Split(view.Body, "\r\n"))
		for _, extent := range view.Extents {
			if extent.Start < 0 || extent.End > drawn || extent.Start >= extent.End {
				t.Fatalf("height %d: extent %+v outside the %d drawn rows", height, extent, drawn)
			}
			// And the row it claims must really be that actor's.
			if got, ok := view.PointToActor(extent.Start, 0); !ok || got != extent.Thread {
				t.Fatalf("height %d: extent %+v does not map back to its own actor", height, extent)
			}
		}
		// Nothing past the drawn menu resolves.
		if _, ok := view.PointToActor(drawn, 0); ok {
			t.Fatalf("height %d: row %d is past the drawn menu and still mapped", height, drawn)
		}
	}
}

// The clamp and the run-merge, each constructed rather than hoped for.
//
// TestExtentsNeverPointPastTheDrawnMenu walks heights and asserts what it finds,
// so removing the clamp passes it: with the menu fitting, nothing is out of
// range to catch. And deleting the run-merge passes every extent test, because
// nothing asserts that an actor's SECOND line belongs to the same extent as its
// first rather than to a new one.
func TestClampAndRunMergeAreLoadBearing(t *testing.T) {
	one := menuAddress("one")
	state := extentFixture(false)
	now := time.Unix(1_700_000_000, 0)

	t.Run("an actor's lines are ONE extent, not one each", func(t *testing.T) {
		view := RenderMenuView(state, 60, 14, now, false)
		var found int
		for _, extent := range view.Extents {
			if extent.Thread == one {
				found++
				if got := extent.End - extent.Start; got < 3 {
					t.Fatalf("extent covers %d rows, want the actor's line plus its two messages: %+v", got, extent)
				}
			}
		}
		if found != 1 {
			t.Fatalf("actor `one` has %d extents, want exactly 1 covering all its rows", found)
		}
	})

	// NOT tested: that a height clamp truncates an extent. clampExtents cannot
	// be reached -- renderRootMenuFrame's rowBudget already subtracts the
	// breadcrumb, notice and filter, so RenderMenuView's truncation never fires
	// while extents exist (measured, heights 3..16). Constructing a case would
	// mean building a state the renderer cannot produce, which tests the test.
	// The guard stays for the day those two bounds drift; its comment says so.
}
