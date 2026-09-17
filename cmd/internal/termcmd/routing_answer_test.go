package termcmd

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// pair#265, the same defect one layer over: a tab can exist while the presenter
// holds no admitted endpoint, and "I have nothing to deliver this to" is not a
// reason to tear down the mux.
func TestRoutingAnswerDoesNotStopTheMux(t *testing.T) {
	p := terminal.NewPresenter(ttyio.NewFake(), terminal.ChildRequested)
	t.Cleanup(func() { _ = p.Release(context.Background()) })
	// Deliberately NOT admitted: addPresentationTab would call admitTab, which
	// is the opposite of the state this test is about.
	m := &terminalMux{
		tabs:      []*terminalTab{{id: 1, child: ptychild.NewFakeChild(nil)}},
		active:    0,
		presenter: p,
		done:      make(chan struct{}),
	}

	m.writeEvents([]terminal.InputEvent{{Event: uv.KeyPressEvent{Code: 'x', Text: "x"}}})

	if m.failure != nil {
		t.Fatalf("a routing answer latched a mux failure: %v", m.failure)
	}
	select {
	case <-m.done:
		t.Fatal("a routing answer stopped the mux")
	default:
	}
}

// Task 5 originally swept writeEvents only. The same disagreement reaches the
// presenter by REPAINT and by RESIZE, and both guarded on pair term's own tab
// model before escalating (pair#265 BR-2).
func TestRoutingAnswerDoesNotStopTheMuxOnRepaintOrResize(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*terminalMux)
	}{
		{"repaint", func(m *terminalMux) { m.paintStrip() }},
		{"resize", func(m *terminalMux) {
			m.inheritSize(hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 100}))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := terminal.NewPresenter(ttyio.NewFake(), terminal.ChildRequested)
			t.Cleanup(func() { _ = p.Release(context.Background()) })
			m := &terminalMux{
				tabs:      []*terminalTab{{id: 1, child: ptychild.NewFakeChild(nil)}},
				active:    0,
				rows:      24,
				cols:      80,
				presenter: p,
				done:      make(chan struct{}),
			}

			tc.call(m)

			if m.failure != nil {
				t.Fatalf("a %s latched a mux failure: %v", tc.name, m.failure)
			}
			select {
			case <-m.done:
				t.Fatalf("a %s stopped the mux", tc.name)
			default:
			}
		})
	}
}

// Red without BR-17's presenterResized: inheritSize returned on a routing
// answer, skipping the mux's own geometry update AND every tab's resize -- the
// active tab included, since its only resize lives in the refused call's apply
// callback. Same defect as couch's BR-13, in the other consumer.
func TestResizeWithNoEndpointStillResizesEveryTab(t *testing.T) {
	p := terminal.NewPresenter(ttyio.NewFake(), terminal.ChildRequested)
	t.Cleanup(func() { _ = p.Release(context.Background()) })
	active := &terminalTab{id: 1, child: ptychild.NewFakeChild(nil)}
	other := &terminalTab{id: 2, child: ptychild.NewFakeChild(nil)}
	m := &terminalMux{
		tabs: []*terminalTab{active, other}, active: 0,
		rows: 24, cols: 80, presenter: p, done: make(chan struct{}),
	}

	m.inheritSize(hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 100}))

	if m.rows != 30 || m.cols != 100 {
		t.Fatalf("mux geometry = %dx%d, want 30x100", m.rows, m.cols)
	}
	want := m.childSizeLocked()
	for _, tab := range []*terminalTab{active, other} {
		if got := tab.child.Size(); got != want {
			t.Errorf("tab %d size = %+v, want %+v", tab.id, got, want)
		}
	}
}
