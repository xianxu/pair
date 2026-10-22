package couchtty

import (
	"context"
	"fmt"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestCollapseCountClickUsesConfirmedScopeWithoutSwitch(t *testing.T) {
	for _, count := range []int{4, 12} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			one := groupedRow("/workspace/pair", 1, "one")
			c, _, _, calls := newMouseFixture(t, func(c *Console) {
				c.SetTabStyle(TabStyleCollapse)
				rows := []couchcore.ActionableThreadSummary{one}
				for i := 0; i < count; i++ {
					row := groupedRow("/workspace/pair", i+2, fmt.Sprintf("quiet%d", i))
					rows = append(rows, row)
					if i == 0 {
						c.panes["c2"].thread, c.panes["c2"].tree = row.Address, couchcore.Worktree(row.StartingPath)
					} else {
						child := ptychild.NewFakeChild(nil)
						t.Cleanup(func() { _ = child.Close() })
						c.attachThreadActor(fmt.Sprintf("q%d", i), "quiet", row.Address, couchcore.Worktree(row.StartingPath), "quiet", child)
					}
				}
				for i := range rows {
					rows[i].RepositoryAlias = "🛋️pair"
				}
				c.mu.Lock()
				defer c.mu.Unlock()
				c.panes["c1"].thread, c.panes["c1"].tree = one.Address, couchcore.Worktree(one.StartingPath)
				c.menu = NewMenuState(rows, one.Address)
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
			// Obtain actual drawn columns from terminal cells, independently of the hitbox math.
			c.mu.Lock()
			size, model := c.size, c.statusModelLocked()
			c.mu.Unlock()
			_, cells, err := renderChrome(size, model)
			if err != nil {
				t.Fatal(err)
			}
			var columns []int
			for x, cell := range cells {
				if cell.Content == "+" {
					columns = append(columns, x)
					for at := x + 1; at < len(cells) && cells[at].Content >= "0" && cells[at].Content <= "9"; at++ {
						columns = append(columns, at)
					}
					break
				}
			}
			if len(columns) != len(fmt.Sprintf("+%d", count)) {
				t.Fatalf("missing physical count: %+v", columns)
			}
			for _, column := range columns {
				if err := c.runTerminalCommand(context.Background(), func() error {
					c.mu.Lock()
					c.focus = FocusActor("c1")
					c.menu.Frames[0].GroupScope = nil
					c.mu.Unlock()
					c.paintNow()
					var decoder terminal.Decoder
					events, err := decoder.Feed([]byte(fmt.Sprintf("\x1b[<0;%d;24M", column+1)))
					if err != nil {
						return err
					}
					for _, event := range events {
						c.routeInputEvent(event)
					}
					c.mu.Lock()
					defer c.mu.Unlock()
					if !c.focus.IsPanel() || c.menu.Frames[0].GroupScope == nil || *c.menu.Frames[0].GroupScope != span.Group || c.active != "c1" || len(VisibleMenuThreads(c.menu)) != count+1 {
						return fmt.Errorf("physical column%d did not open exact count scope: span%+v", column, span)
					}
					return nil
				}); err != nil {
					t.Error(err)
				}
			}
			select {
			case call := <-calls:
				t.Fatalf("count switched actor: %+v", call.call)
			default:
			}
		})
	}
}
