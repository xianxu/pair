package termcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// THE DECLARED PAINT BUDGET, AS AN ENUMERATION WITH A TEST PER ENTRY.
//
// The plan's ARCH-CONSTRAINTS says: "A repaint happens on tab change, resize,
// and `batch.RowDirty`; **not** per output chunk." ARCH-ORDER adds the
// transitions: host resize → re-`Reserve` and re-`Paint`; process exit →
// `Release` on every exit path.
//
// Those were sentences. Measured by the M3 review, all three of these mutations
// left the whole suite GREEN: making every active-tab chunk paint (the literal
// violation of bullet 1), deleting the resize repaint, and dropping teardown's
// region release. The reason is uniform and worth stating, because it shapes
// what a useful test here looks like: the suite asked "did a repaint happen"
// almost everywhere and "did one happen that should NOT have" in one place, and
// a budget is a bound on the negative direction.
//
// So each row below either counts paints and expects ZERO, or names a transition
// and expects exactly one — and every row was checked by deleting the behaviour
// it pins.
func TestTheDeclaredPaintBudgetHoldsPerEvent(t *testing.T) {
	// paintsAt counts strip repaints in what reached the pane. Every paint
	// positions the reserved row, so that is the marker -- and the row moves with
	// the pane, which is why the height is a parameter rather than a constant.
	paintsAt := func(rows int, s string) int { return strings.Count(s, hostty.MoveTo(rows, 1)) }
	paintsIn := func(s string) int { return paintsAt(24, s) }

	t.Run("ordinary output does NOT paint, however much of it there is", func(t *testing.T) {
		m, rec := stripMux(t)
		defer close(m.done)
		m.drainForTest()
		rec.reset()

		// A shell echoing: many chunks, none of them row-dirty.
		for i := 0; i < 50; i++ {
			m.output <- ptyChunk{id: 2, data: []byte("hello\r\n")}
		}
		m.drainForTest()

		if n := paintsIn(rec.String()); n != 0 {
			t.Fatalf("%d repaint(s) for 50 ordinary chunks; the budget is none per "+
				"output chunk, and this is the keystroke path", n)
		}
	})

	t.Run("a row-dirty batch pays its debt exactly once", func(t *testing.T) {
		m, rec := stripMux(t)
		defer close(m.done)
		m.drainForTest()
		rec.reset()

		m.output <- ptyChunk{id: 2, data: []byte("\x1b[2J"), rowDirty: true}
		for i := 0; i < 10; i++ {
			m.output <- ptyChunk{id: 2, data: []byte("more\r\n")}
		}
		m.drainForTest()

		if n := paintsIn(rec.String()); n != 1 {
			t.Fatalf("%d repaint(s) for one row-dirty batch followed by ten clean "+
				"chunks; the debt is paid once, not held and repaid", n)
		}
	})

	t.Run("a host resize re-Reserves AND repaints", func(t *testing.T) {
		m, rec := stripMux(t)
		defer close(m.done)
		m.drainForTest()
		rec.reset()

		host := hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 100})
		m.inheritSize(host)

		got := rec.String()
		if n := paintsAt(30, got); n != 1 {
			t.Fatalf("resize repainted %d time(s) at the NEW bottom row, want exactly one: %q", n, got)
		}
		// The REGION, not just the row: the pane is a different height now, so a
		// repaint that does not re-Reserve leaves the child scrolling over the
		// strip (ARCH-ORDER's "host resize → re-Reserve, re-Paint").
		if !strings.Contains(got, hostty.SetRegion(1, 29)) {
			t.Fatalf("resize did not re-assert the region for the new height: %q", got)
		}
	})

	t.Run("teardown releases the region", func(t *testing.T) {
		m, rec := stripMux(t)
		defer close(m.done)
		m.drainForTest()
		rec.reset()

		m.restoreTerminal()

		// ARCH-ORDER: "Release on every exit path". Without it the operator's
		// shell is left scrolling inside a box that nothing will remove.
		if !strings.Contains(rec.String(), hostty.ResetRegion) {
			t.Fatalf("teardown wrote %q, without the region reset", rec.String())
		}
	})
}
