package couchtty

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func openPanel(t *testing.T, f *consoleFixture) {
	t.Helper()
	waitFor(t, "the console to start", func() bool { return len(f.child.Resizes()) > 0 })
	_, _ = f.stdin.Write([]byte("\x00"))
	waitFor(t, "the panel", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.focus.IsPanel() && f.con.panel != nil
	})
}

func TestPanelCtrlSpaceStart(t *testing.T) {
	for _, key := range []string{"\x00", "\x1b[32;5u"} {
		t.Run(fmt.Sprintf("%q", key), func(t *testing.T) {
			f := newFixture(t, 24, 80)
			openPanel(t, f)
			_, _ = f.stdin.Write([]byte(key))
			waitFor(t, "the start prompt", func() bool {
				f.con.mu.Lock()
				defer f.con.mu.Unlock()
				return f.con.prompt == "start in path: "
			})
		})
	}
}

func TestPanelCtrlSpaceIsNoOpInsideStartPrompt(t *testing.T) {
	f := newFixture(t, 24, 80)
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x00../pa"))
	waitFor(t, "the partial path", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.promptArg == "../pa"
	})
	_, _ = f.stdin.Write([]byte("\x00x"))
	waitFor(t, "the prompt to remain unchanged", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.promptArg == "../pax" && f.con.prompt == "start in path: ../pax"
	})
}

func TestPanelColonAndDigitsAreFilterText(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.con.SetResolver(func(string) []couchcore.ThreadAddress { return nil })
	openPanel(t, f)
	before := len(f.child.Writes())
	_, _ = f.stdin.Write([]byte(":2"))
	waitFor(t, "literal filter text", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.query == ":2" && f.con.focus.IsPanel()
	})
	if len(f.child.Writes()) != before {
		t.Fatalf("filter input reached child: %q", f.child.Writes()[before:])
	}
}

func TestPanelEnterOnAlreadyActiveActorForcesClearAndReplay(t *testing.T) {
	f := newFixture(t, 24, 80)
	openPanel(t, f)
	f.host.Reset()
	f.child.Feed([]byte("detached progress"))
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "the actor replay", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return !f.con.focus.IsPanel() && strings.Contains(f.host.Written(), "detached progress")
	})
	got := f.host.Written()
	if !strings.Contains(got, hostty.HomeAndClear) {
		t.Fatalf("same-active Enter did not clear and replay: %q", got)
	}
	if lastClear := strings.LastIndex(got, hostty.HomeAndClear); strings.Contains(got[lastClear:], "couch — actors") {
		t.Fatalf("panel remained after final clear: %q", got[lastClear:])
	}
}

func parkedFixture(t *testing.T) *consoleFixture {
	t.Helper()
	f := newFixture(t, 24, 80)
	f.con.SetSummaries(func() []couchcore.ThreadSummary {
		return []couchcore.ThreadSummary{
			{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
			{Address: panelAddress("parked"), WorkingPath: "/w/parked", Name: "parked"},
		}
	})
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x1b[B"))
	waitFor(t, "parked selection", func() bool {
		row, ok := f.con.selectedRow()
		return ok && row.Tree == "/w/parked"
	})
	return f
}

func TestPanelEnterOnParkedRowStartsItsPath(t *testing.T) {
	f := parkedFixture(t)
	var mu sync.Mutex
	var called map[string]string
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name != "start" {
			t.Fatalf("operation = %q, want start", name)
		}
		mu.Lock()
		called = args
		mu.Unlock()
		return nil, errors.New("stop after dispatch")
	})
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "parked start dispatch", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return called != nil
	})
	mu.Lock()
	defer mu.Unlock()
	if called["path"] != "/w/parked" {
		t.Fatalf("path = %q, want /w/parked", called["path"])
	}
}

func TestPanelEnterOnRemoteLiveRowExplainsAttachmentIsDeferred(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.con.SetSummaries(func() []couchcore.ThreadSummary {
		return []couchcore.ThreadSummary{
			{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
			{Address: panelAddress("remote"), WorkingPath: "/w/remote", Name: "remote", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
		}
	})
	called := false
	setTestOps(f.con, func(string, map[string]string) (any, error) {
		called = true
		return nil, nil
	})
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x1b[B\r"))
	waitFor(t, "remote attachment notice", func() bool {
		return strings.Contains(f.host.Written(), "live in another couch")
	})
	if called {
		t.Fatal("remote live row dispatched start as if it were parked")
	}
	row, ok := f.con.selectedRow()
	if !ok || row.Tree != "/w/remote" || !row.Live || row.Target != "" {
		t.Fatalf("remote row state = %+v, %v", row, ok)
	}
}

func TestPanelStartFailurePreservesListState(t *testing.T) {
	f := parkedFixture(t)
	f.con.SetResolver(func(string) []couchcore.ThreadAddress { return []couchcore.ThreadAddress{panelAddress("parked")} })
	_, _ = f.stdin.Write([]byte("p"))
	waitFor(t, "the retained filter", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.query == "p"
	})
	setTestOps(f.con, func(string, map[string]string) (any, error) { return nil, errors.New("boom") })
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "the failure notice", func() bool { return strings.Contains(f.host.Written(), "start: boom") })
	f.con.mu.Lock()
	query := f.con.query
	focus := f.con.focus
	f.con.mu.Unlock()
	row, ok := f.con.selectedRow()
	if query != "p" || !focus.IsPanel() || !ok || row.Tree != "/w/parked" {
		t.Fatalf("failure changed state: query=%q focus=%+v row=%+v ok=%v", query, focus, row, ok)
	}
}

func TestPanelStartSuccessAttachesAndSelectsReturnedTree(t *testing.T) {
	f := newFixture(t, 24, 80)
	runner := couchcore.NewFakeRunner()
	h, err := runner.Start("/w/parked", []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	setTestOps(f.con, func(string, map[string]string) (any, error) {
		return couchcore.StartResult{
			Record: couchcore.ActorRecord{
				ID:     "actor-parked",
				Thread: panelAddress("parked"),
				Args:   couchcore.StartArgs{Worktree: "/w/parked"},
			},
			Handle: h,
		}, nil
	})
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x00/w/parked\r"))
	waitFor(t, "the started child and selected row", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		row, ok := f.con.panel.Selected()
		return len(f.con.panes) == 2 && f.con.focus.IsPanel() && ok && row.Tree == "/w/parked"
	})
	if !strings.Contains(f.host.Written(), "parked") {
		t.Fatalf("final panel omitted started row: %q", f.host.Written())
	}
}

func TestPanelStartPromptCancelPreservesListState(t *testing.T) {
	f := parkedFixture(t)
	f.con.SetResolver(func(string) []couchcore.ThreadAddress { return []couchcore.ThreadAddress{panelAddress("parked")} })
	_, _ = f.stdin.Write([]byte("park"))
	waitFor(t, "the filtered parked row", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.query == "park"
	})
	_, _ = f.stdin.Write([]byte("\x00/tmp/nope\x1b"))
	waitFor(t, "prompt cancellation", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.promptFn == nil
	})
	row, ok := f.con.selectedRow()
	f.con.mu.Lock()
	query := f.con.query
	f.con.mu.Unlock()
	if query != "park" || !ok || row.Tree != "/w/parked" {
		t.Fatalf("cancel changed list state: query=%q row=%+v ok=%v", query, row, ok)
	}
}

func TestPanelEmptyStartUsesOperationDotDefault(t *testing.T) {
	f := newFixture(t, 24, 80)
	var mu sync.Mutex
	got := "not called"
	setTestOps(f.con, func(_ string, args map[string]string) (any, error) {
		mu.Lock()
		got = args["path"]
		mu.Unlock()
		return nil, nil
	})
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x00\r"))
	waitFor(t, "empty path dispatch", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return got != "not called"
	})
	mu.Lock()
	defer mu.Unlock()
	if got != "" {
		t.Fatalf("console translated empty path to %q", got)
	}
}

func TestPanelBackspaceRemovesLastDecodedCharacter(t *testing.T) {
	f := newFixture(t, 24, 80)
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("路径\x7f"))
	waitFor(t, "rune-safe filter backspace", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.query == "路"
	})
	_, _ = f.stdin.Write([]byte("\x1b\x00/tmp/路径\x7f"))
	waitFor(t, "rune-safe prompt backspace", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.promptArg == "/tmp/路"
	})
}

func TestPanelEnterWithNoMatchReportsNoSelection(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.con.SetResolver(func(string) []couchcore.ThreadAddress { return nil })
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("nothing\r"))
	waitFor(t, "the no-selection notice", func() bool {
		return strings.Contains(f.host.Written(), "no selection")
	})
}

func TestPanelRefreshPreservesOrFallsBackSelection(t *testing.T) {
	f := parkedFixture(t)
	f.con.SetSummaries(func() []couchcore.ThreadSummary {
		return []couchcore.ThreadSummary{
			{Address: panelAddress("parked"), WorkingPath: "/w/parked", Name: "parked"},
			{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
		}
	})
	f.con.rebuildPanel()
	row, ok := f.con.selectedRow()
	if !ok || row.Tree != "/w/parked" {
		t.Fatalf("refresh lost visible selection: %+v, %v", row, ok)
	}
	f.con.SetSummaries(func() []couchcore.ThreadSummary {
		return []couchcore.ThreadSummary{{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}}}
	})
	f.con.rebuildPanel()
	row, ok = f.con.selectedRow()
	if !ok || row.Tree != "c1" {
		t.Fatalf("refresh did not fall back to first row: %+v, %v", row, ok)
	}
}

func TestPanelRefreshesWhenInactiveChildExitsWhileOpen(t *testing.T) {
	f := newFixture(t, 24, 80)
	other := ptychild.NewFakeChild(nil)
	other.SetSink(func(chunk []byte) { f.con.Deliver("c2", chunk) })
	f.con.Attach("c2", "ariadne", other)
	f.con.SetForget(func(couchcore.Worktree, couchcore.ActorID) error { return nil })
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x1b[B"))
	waitFor(t, "second-row selection", func() bool {
		row, ok := f.con.selectedRow()
		return ok && row.Tree == "c2"
	})
	f.child.Exit(0)
	waitFor(t, "dead row removal", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		rows := f.con.panel.Rows()
		row, ok := f.con.panel.Selected()
		return len(rows) == 1 && ok && row.Tree == "c2"
	})
}

func TestPanelStartWithoutOpsSaysSo(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.con.mu.Lock()
	f.con.ops = nil
	f.con.mu.Unlock()
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x00\r"))
	waitFor(t, "the dispatcher refusal", func() bool {
		return strings.Contains(f.host.Written(), "no action dispatcher wired")
	})
}
