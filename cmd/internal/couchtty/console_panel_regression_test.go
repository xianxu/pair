package couchtty

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
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

func TestRootAltXLeaveUIEventOrdering(t *testing.T) {
	for _, key := range workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltX) {
		t.Run(fmt.Sprintf("%q", key), func(t *testing.T) {
			f := newFixture(t, 24, 80)
			started := make(chan struct{})
			release := make(chan struct{})
			setTestOps(f.con, func(name string, args map[string]string) (any, error) {
				if name != "leave" || len(args) != 0 {
					t.Fatalf("leave dispatch = %q %+v", name, args)
				}
				close(started)
				<-release
				return nil, nil
			})

			_, _ = f.stdin.Write(key)
			waitFor(t, "immediate park confirmation", func() bool {
				f.con.mu.Lock()
				defer f.con.mu.Unlock()
				return strings.HasPrefix(f.con.prompt, "leave couch")
			})
			select {
			case <-started:
				t.Fatal("leave lifecycle started before confirmation")
			default:
			}

			_, _ = f.stdin.Write([]byte("yes\r"))
			waitFor(t, "leave lifecycle start", func() bool {
				select {
				case <-started:
					return true
				default:
					return false
				}
			})
			waitFor(t, "leaving status", func() bool { return strings.Contains(f.host.Written(), "leaving…") })

			_, _ = f.stdin.Write([]byte("z"))
			waitFor(t, "input while park is blocked", func() bool {
				f.con.mu.Lock()
				defer f.con.mu.Unlock()
				return f.con.query == "z"
			})
			close(release)
			waitFor(t, "console stop after verified leave", func() bool {
				select {
				case <-f.con.stop:
					return true
				default:
					return false
				}
			})
		})
	}
}

func TestNonRootAltXParksOnlySelectedActor(t *testing.T) {
	f := newFixture(t, 24, 80)
	otherAddress := panelAddress("other")
	other := ptychild.NewFakeChild(nil)
	f.con.attachThreadActor("other", "other", otherAddress, "/w/other", "other", other)
	f.con.mu.Lock()
	f.con.active = "other"
	f.con.focus = FocusActor("other")
	f.con.mu.Unlock()
	called := make(chan map[string]string, 1)
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name != "park" {
			t.Fatalf("operation = %q, want park", name)
		}
		called <- args
		return nil, nil
	})

	f.con.onParkHotkey()
	f.con.mu.Lock()
	prompt := f.con.prompt
	confirm := f.con.promptFn
	f.con.mu.Unlock()
	if !strings.HasPrefix(prompt, "park ") || confirm == nil {
		t.Fatalf("prompt = %q", prompt)
	}
	confirm("yes")
	select {
	case args := <-called:
		if args["repo-scope"] != otherAddress.RepoScope || args["tag"] != string(otherAddress.Tag) {
			t.Fatalf("park args = %+v", args)
		}
	case <-time.After(time.Second):
		t.Fatal("park was not dispatched")
	}
}

func TestLastActorExitWhilePanelFocusedKeepsConsoleForResume(t *testing.T) {
	address := panelAddress("parked")
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	child := ptychild.NewFakeChild(nil)
	con.attachThreadActor("only", "actor", address, "/w/parked", "parked", child)
	con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
		return []couchcore.ThreadSummary{{Address: address, WorkingPath: "/w/parked", Name: "parked"}}, nil
	})
	con.mu.Lock()
	con.focus = FocusPanel()
	con.mu.Unlock()

	if exitConsole := con.onExit(childExit{id: "only", code: 0}); exitConsole {
		t.Fatal("last actor exit closed the panel before its parked row could be resumed")
	}
	con.mu.Lock()
	rows := append([]PanelRow(nil), con.panel.Shown()...)
	con.mu.Unlock()
	if len(rows) != 1 || rows[0].Live || rows[0].Target != "" || rows[0].Address != address {
		t.Fatalf("post-park panel rows = %+v, want one exact parked row", rows)
	}
}

func TestEscapeFromPanelWithNoActorStopsConsole(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	con.onPanelKey(PanelKey{Kind: KeyEscape})
	select {
	case <-con.stop:
	default:
		t.Fatal("Escape with no actor left the operator trapped in the panel")
	}
}

func TestActiveNonRootExitFallsBackToRootForPanelActions(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	root := ptychild.NewFakeChild(nil)
	other := ptychild.NewFakeChild(nil)
	con.attachThreadActor("root", "root-actor", panelAddress("root"), "/w/root", "root", root)
	con.attachThreadActor("other", "other-actor", panelAddress("other"), "/w/other", "other", other)
	con.mu.Lock()
	con.active = "other"
	con.focus = FocusPanel()
	con.mu.Unlock()

	if exitConsole := con.onExit(childExit{id: "other", code: 0}); exitConsole {
		t.Fatal("non-root exit closed a console whose root remained live")
	}
	con.mu.Lock()
	active := con.active
	con.mu.Unlock()
	if active != "root" {
		t.Fatalf("active target after non-root exit = %q, want root", active)
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
	f.con.SetResolver(func(string) ([]couchcore.ThreadAddress, error) { return nil, nil })
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
	f.con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
		return []couchcore.ThreadSummary{
			{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
			{Address: panelAddress("parked"), WorkingPath: "/w/parked", Name: "parked"},
		}, nil
	})
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("\x1b[B"))
	waitFor(t, "parked selection", func() bool {
		row, ok := f.con.selectedRow()
		return ok && row.Tree == "/w/parked"
	})
	return f
}

func TestPanelEnterOnParkedRowResumesExactThread(t *testing.T) {
	f := parkedFixture(t)
	var mu sync.Mutex
	var called map[string]string
	setTestOps(f.con, func(name string, args map[string]string) (any, error) {
		if name != "resume" {
			t.Fatalf("operation = %q, want resume", name)
		}
		mu.Lock()
		called = args
		mu.Unlock()
		return nil, errors.New("stop after dispatch")
	})
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "parked resume dispatch", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return called != nil
	})
	mu.Lock()
	defer mu.Unlock()
	if called["repo-scope"] != panelAddress("parked").RepoScope || called["tag"] != string(panelAddress("parked").Tag) {
		t.Fatalf("resume address args = %+v", called)
	}
}

func TestPanelEnterOnRemoteLiveRowExplainsAttachmentIsDeferred(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
		return []couchcore.ThreadSummary{
			{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
			{Address: panelAddress("remote"), WorkingPath: "/w/remote", Name: "remote", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
		}, nil
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

func TestPanelResumeFailurePreservesListState(t *testing.T) {
	f := parkedFixture(t)
	f.con.SetResolver(func(string) ([]couchcore.ThreadAddress, error) {
		return []couchcore.ThreadAddress{panelAddress("parked")}, nil
	})
	_, _ = f.stdin.Write([]byte("p"))
	waitFor(t, "the retained filter", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.query == "p"
	})
	setTestOps(f.con, func(string, map[string]string) (any, error) { return nil, errors.New("boom") })
	_, _ = f.stdin.Write([]byte("\r"))
	waitFor(t, "the failure notice", func() bool { return strings.Contains(f.host.Written(), "resume: boom") })
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
	f.con.SetResolver(func(string) ([]couchcore.ThreadAddress, error) {
		return []couchcore.ThreadAddress{panelAddress("parked")}, nil
	})
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
	f.con.SetResolver(func(string) ([]couchcore.ThreadAddress, error) { return nil, nil })
	openPanel(t, f)
	_, _ = f.stdin.Write([]byte("nothing\r"))
	waitFor(t, "the no-selection notice", func() bool {
		return strings.Contains(f.host.Written(), "no selection")
	})
}

func TestPanelRefreshPreservesOrFallsBackSelection(t *testing.T) {
	f := parkedFixture(t)
	f.con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
		return []couchcore.ThreadSummary{
			{Address: panelAddress("parked"), WorkingPath: "/w/parked", Name: "parked"},
			{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
		}, nil
	})
	f.con.rebuildPanel()
	row, ok := f.con.selectedRow()
	if !ok || row.Tree != "/w/parked" {
		t.Fatalf("refresh lost visible selection: %+v, %v", row, ok)
	}
	f.con.SetSummaries(func() ([]couchcore.ThreadSummary, error) {
		return []couchcore.ThreadSummary{{Address: panelAddress("c1"), WorkingPath: "c1", Name: "brain", Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}}}, nil
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
