package couchtty

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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
		child.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return f.con.Deliver(ctx, id, batch) })
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
	waitFor(t, "three-thread inventory", func() bool {
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
