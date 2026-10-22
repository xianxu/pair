package couchtty

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestConsoleWrappedStatusReservesTwoRows(t *testing.T) {
	var second *ptychild.Child
	f := newFixtureBeforeRun(t, 24, 18, func(c *Console) {
		second = ptychild.NewFakeChild(nil)
		second.SetSink(func(ctx context.Context, b ptychild.OutputBatch) error { return c.Deliver(ctx, "c2", b) })
		c.Attach("c2", "secondlong", second)
	})
	for _, child := range []*ptychild.Child{f.child, second} {
		if got := child.Size(); got.Rows != 22 || got.Cols != 18 {
			t.Fatalf("wrapped child geometry %+v, want22x18", got)
		}
	}
	f.host.SetSize(ptychild.Size{Rows: 24, Cols: 80})
	waitFor(t, "unwrapped child geometry", func() bool { return f.child.Size().Rows == 23 && second.Size().Rows == 23 && f.child.Size().Cols == 80 })
}

func TestConsoleWrappedPanelSharesChromeReservation(t *testing.T) {
	f := newFixtureBeforeRun(t, 24, 18, func(c *Console) {
		child := ptychild.NewFakeChild(nil)
		child.SetSink(func(ctx context.Context, b ptychild.OutputBatch) error { return c.Deliver(ctx, "c2", b) })
		c.Attach("c2", "secondlong", child)
	})
	if err := f.con.runTerminalCommand(f.con.lifetime, func() error { f.con.showMenu(); return nil }); err != nil {
		t.Fatal(err)
	}
	state, err := f.con.presenter.ChromeState(f.con.lifetime)
	if err != nil || state.Rows != 2 || !state.Confirmed {
		t.Fatalf("panel geometry %+v: %v", state, err)
	}
	if got := f.con.ChildSize(); got.Rows != 22 {
		t.Fatalf("panel child size %+v", got)
	}
	f.host.SetSize(ptychild.Size{Rows: 24, Cols: 80})
	waitFor(t, "panel unwrapped", func() bool {
		state, err := f.con.presenter.ChromeState(f.con.lifetime)
		return err == nil && state.Rows == 1 && state.Confirmed && f.child.Size().Rows == 23
	})
}
