package couchtty

import (
	"context"
	"fmt"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestCollapseCountClickUsesConfirmedScopeWithoutSwitch(t *testing.T) {
	one, two := groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/pair", 2, "two")
	c, input, _, calls := newMouseFixture(t, func(c *Console) {
		c.SetTabStyle(TabStyleCollapse)
		c.mu.Lock()
		defer c.mu.Unlock()
		c.panes["c1"].thread, c.panes["c1"].tree = one.Address, couchcore.Worktree(one.StartingPath)
		c.panes["c2"].thread, c.panes["c2"].tree = two.Address, couchcore.Worktree(two.StartingPath)
		c.menu = NewMenuState([]couchcore.ActionableThreadSummary{one, two}, one.Address)
	})
	waitFor(t, "confirmed count span", func() bool {
		if !c.chromeConfirmed() {
			return false
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		return len(c.statusGroups) == 1
	})
	c.mu.Lock()
	span := c.statusGroups[0]
	c.mu.Unlock()
	// Reuse the exact production decoder, under the Console's command owner.
	if err := c.runTerminalCommand(context.Background(), func() error {
		c.mu.Lock()
		revision := c.statusRevision
		c.statusRevision++
		c.mu.Unlock()
		defer func() { c.mu.Lock(); c.statusRevision = revision; c.mu.Unlock() }()
		var decoder terminal.Decoder
		events, err := decoder.Feed([]byte(fmt.Sprintf("\x1b[<0;%d;24M", span.Start+1)))
		if err != nil {
			return err
		}
		for _, event := range events {
			c.routeInputEvent(event)
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.focus.IsPanel() || c.menu.Frames[0].GroupScope != nil {
			return fmt.Errorf("stale count click opened scope")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	clickAt(t, input, span.Start+1, 24)
	waitFor(t, "count opens exact group", func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.focus.IsPanel() && c.menu.Frames[0].GroupScope != nil && *c.menu.Frames[0].GroupScope == span.Group && c.active == "c1" && len(VisibleMenuThreads(c.menu)) == 2
	})
	select {
	case call := <-calls:
		t.Fatalf("count switched actor: %+v", call.call)
	default:
	}
}
