package couchtty

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// pagingFixture is a live console hosting three actors, all projected as
// actionable so a switcher Return resolves through the production dispatcher,
// with the operator working in c1.
//
// Three rather than two because every wrong answer has to be a DIFFERENT actor.
// With the pages the tests raise, the newest pager is c3, the first paging actor
// in pane order is c2, and the switcher's first row is c1: a jump that picked by
// order, or fell back to a browser's default, lands somewhere a correct one does
// not, instead of coincidentally agreeing with it.
func pagingFixture(t *testing.T) (*consoleFixture, [3]couchcore.ThreadAddress) {
	t.Helper()
	f := newFixture(t, 24, 100)
	for _, id := range []string{"c2", "c3"} {
		child := ptychild.NewFakeChild([]byte(id + " screen"))
		child.SetSink(func(batch ptychild.OutputBatch) { f.con.Deliver(id, batch) })
		f.con.Attach(id, id, child)
	}

	var threads [3]couchcore.ThreadAddress
	f.con.mu.Lock()
	for i, id := range []string{"c1", "c2", "c3"} {
		threads[i] = f.con.panes[id].thread
		f.con.panes[id].process = couchcore.ProcessIdentity{PID: 41 + i, Identity: id + "-start"}
	}
	f.con.menu.ActiveAddress = threads[0]
	f.con.mu.Unlock()

	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		inventory := make([]couchcore.ActionableThreadSummary, len(threads))
		for i, address := range threads {
			inventory[i] = couchcore.ActionableThreadSummary{
				Address: address, WorkingPath: "/repo/" + string(address.Tag), Name: string(address.Tag),
				State: couchcore.ThreadLive, LastActiveAt: time.Now(),
			}
		}
		return inventory, nil
	})
	waitUpTo(t, 250*time.Millisecond, "three-thread inventory", func() bool {
		menu := f.con.menuSnapshot()
		return menu.InventoryReady && len(menu.Inventory) == 3
	})

	f.con.switchTo("c1", true, arrivalOrdinary)
	return f, threads
}

// page raises attention the way onChunk does for an unfocused actor.
func page(f *consoleFixture, address couchcore.ThreadAddress, text string) {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	f.con.attention.Mark(address, text)
	f.con.syncAttentionLocked()
}

// landing is everything a landing is owed: where the operator is, what
// ctrl+backspace will do next, and which bells are still lit.
type landing struct {
	active    string
	focus     Focus
	tracker   SwitchTracker
	attention [3][]string
}

func landingOf(f *consoleFixture, threads [3]couchcore.ThreadAddress) landing {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	l := landing{active: f.con.active, focus: f.con.focus, tracker: f.con.tracker}
	for i, address := range threads {
		l.attention[i] = attentionTexts(f.con.attention.Projection(address))
	}
	return l
}

func activeOf(f *consoleFixture) string {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.active
}

func noticeOf(f *consoleFixture) string {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.feed.Row().Body
}

// The Done-when's headline: ctrl+return lands EXACTLY where ctrl-space then
// Return would have, from one attention state, with both driven through the
// production input path -- the real interceptor and handler table, and for the
// two-gesture path the real menu dispatcher and its arrival derivation.
//
// The whole SwitchTracker is compared, not just the landed actor, because
// arrivalNotification versus arrivalOrdinary changes ONLY the tracker: a jump
// that landed on the right actor but pinned it would break ctrl+backspace and
// nothing else, which is how it would ship.
func TestNewestPageLandsWhereCtrlSpaceThenReturnWould(t *testing.T) {
	arrange := func() (*consoleFixture, [3]couchcore.ThreadAddress) {
		f, threads := pagingFixture(t)
		page(f, threads[1], "tests need approval")
		page(f, threads[2], "review ready")
		return f, threads
	}

	twoGesture, twoThreads := arrange()
	_, _ = twoGesture.stdin.Write([]byte("\x00"))
	waitUpTo(t, 250*time.Millisecond, "the switcher to open on the newest pager", func() bool {
		twoGesture.con.mu.Lock()
		defer twoGesture.con.mu.Unlock()
		return twoGesture.con.focus.IsPanel() && twoGesture.con.menu.CurrentFrame().SelectedAddress == twoThreads[2]
	})
	_, _ = twoGesture.stdin.Write([]byte("\r"))
	waitUpTo(t, time.Second, "ctrl-space + Return to land", func() bool { return activeOf(twoGesture) == "c3" })

	oneGesture, oneThreads := arrange()
	oneGesture.host.Reset()
	_, _ = oneGesture.stdin.Write([]byte(newestPageSequence))
	waitUpTo(t, time.Second, "ctrl+return to land", func() bool { return activeOf(oneGesture) == "c3" })

	if got, want := landingOf(oneGesture, oneThreads), landingOf(twoGesture, twoThreads); !reflect.DeepEqual(got, want) {
		t.Fatalf("ctrl+return landed as\n  %+v\nctrl-space + Return as\n  %+v", got, want)
	}
	if written := oneGesture.host.Written(); strings.Contains(written, "threads") {
		t.Fatalf("ctrl+return drew the switcher on its way: %q", written)
	}

	// What that buys the operator, as README states it: the landing cleared c3's
	// bell, so pressing again answers the next page, and neither hop spent the
	// previous slot, so ctrl+backspace goes back to where they were working.
	_, _ = oneGesture.stdin.Write([]byte(newestPageSequence))
	waitUpTo(t, time.Second, "a second ctrl+return to answer the older page", func() bool { return activeOf(oneGesture) == "c2" })
	_, _ = oneGesture.stdin.Write([]byte("\x08"))
	waitUpTo(t, time.Second, "ctrl+backspace to return home past both pages", func() bool { return activeOf(oneGesture) == "c1" })
}

// Nothing paging: no switch, and the operator is told why. ctrl-space's fallback
// to the thread being left is deliberately NOT inherited -- for a jump it means
// "go where you are", a no-op that reads as a dropped key.
func TestNewestPageWithNothingPagingSaysSo(t *testing.T) {
	f, threads := pagingFixture(t)
	before := landingOf(f, threads)
	f.host.Reset()

	_, _ = f.stdin.Write([]byte(newestPageSequence))
	waitUpTo(t, time.Second, "the refusal", func() bool { return strings.Contains(noticeOf(f), "nothing is paging") })

	if after := landingOf(f, threads); !reflect.DeepEqual(after, before) {
		t.Fatalf("with nothing paging the landing moved:\n  before %+v\n  after  %+v", before, after)
	}
	if written := f.host.Written(); strings.Contains(written, hostty.HomeAndClear) {
		t.Fatalf("with nothing paging the screen was taken over: %q", written)
	}
}

// The newest pager is the actor the operator is already in. Reachable only
// through the focusedAtDelivery race -- a page delivered while the switcher was
// up and reduced after the operator came back -- so the state is set directly.
// Acknowledge, stay, and say so: no takeover of a screen that did not change,
// no tracker movement, and the older pager left lit for the next press.
func TestNewestPageOnTheCurrentActorAcknowledgesAndStays(t *testing.T) {
	f, threads := pagingFixture(t)
	page(f, threads[1], "older page")
	page(f, threads[0], "newest page, on the actor in use")
	before := landingOf(f, threads)
	f.host.Reset()

	_, _ = f.stdin.Write([]byte(newestPageSequence))
	waitUpTo(t, time.Second, "the stay notice", func() bool {
		return strings.Contains(noticeOf(f), "already on the paging thread")
	})

	after := landingOf(f, threads)
	if after.active != "c1" || after.focus != before.focus || after.tracker != before.tracker {
		t.Fatalf("staying moved the operator:\n  before %+v\n  after  %+v", before, after)
	}
	if len(after.attention[0]) != 0 {
		t.Fatalf("the actor in use is still paging: %v", after.attention[0])
	}
	if !reflect.DeepEqual(after.attention[1], before.attention[1]) {
		t.Fatalf("another actor's page changed: %v -> %v", before.attention[1], after.attention[1])
	}
	if written := f.host.Written(); strings.Contains(written, hostty.HomeAndClear) {
		t.Fatalf("staying took over an unchanged screen: %q", written)
	}
}

// The paging thread is durable but its child is done and the exit not yet
// reduced -- the one state where NewestActor names a thread with no live pane,
// since onExit drops the attention together with the pane. Driven on a console
// that is not running, because that is what holds the exit unreduced.
func TestNewestPageToAnExitedChildSaysSo(t *testing.T) {
	con, _ := notificationConsole(t)
	con.switchTo("c1", true, arrivalOrdinary)
	con.mu.Lock()
	two := con.panes["c2"].thread
	con.attention.Mark(two, "paged, then exited")
	exited := con.panes["c2"].child
	con.mu.Unlock()
	exited.Exit(0)

	con.onNewestPageHotkey()

	if got := con.feed.Row().Body; !stringsContains(got, "no longer attached") {
		t.Fatalf("status = %q, want the not-attached notice", got)
	}
	con.mu.Lock()
	active := con.active
	con.mu.Unlock()
	if active != "c1" {
		t.Fatalf("active = %q, want the operator left where they were", active)
	}
}

// In the switcher the chord is not claimed: it does whatever the panel's decoder
// makes of it, which is Return, so it acts on the SELECTED row. The selection is
// moved off the newest pager first -- otherwise "Return" and "jump" land on the
// same actor and this test could not tell them apart.
func TestNewestPageInTheSwitcherIsItsReturn(t *testing.T) {
	f, threads := pagingFixture(t)
	page(f, threads[1], "older page")
	page(f, threads[2], "newest page")

	_, _ = f.stdin.Write([]byte("\x00"))
	waitUpTo(t, 250*time.Millisecond, "the switcher to open on the newest pager", func() bool {
		return f.con.menuSnapshot().CurrentFrame().SelectedAddress == threads[2]
	})
	_, _ = f.stdin.Write([]byte("\x1b[A"))
	waitUpTo(t, 250*time.Millisecond, "the selection to move off it", func() bool {
		return f.con.menuSnapshot().CurrentFrame().SelectedAddress == threads[1]
	})

	_, _ = f.stdin.Write([]byte(newestPageSequence))
	waitUpTo(t, time.Second, "Return on the selected row", func() bool { return activeOf(f) == "c2" })
}
