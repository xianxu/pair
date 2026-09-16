package termcmd

import (
	"context"
	"testing"

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
