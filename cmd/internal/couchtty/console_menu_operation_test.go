package couchtty

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestConsoleMenuOperationFailureUsesExactReducerOrigin(t *testing.T) {
	f := newFixture(t, 24, 80)
	target := menuAddress("couch-one")
	state := NewMenuState(menuThreads(), target)
	if !appendMenuFrame(&state, MenuFrame{Kind: MenuFrameText, Thread: target, Action: "name", Input: "new-name"}) {
		t.Fatal("could not build name form")
	}
	var effects []MenuEffect
	state, effects = dispatchMenuOperation(state, MenuEffect{
		Operation: "name",
		Args:      map[string]string{"repo-scope": target.RepoScope, "tag": string(target.Tag), "name": "new-name"},
	}, target)
	if len(effects) != 1 {
		t.Fatalf("dispatch effects = %+v", effects)
	}
	want, _ := ReduceMenu(state, MenuEvent{
		Kind: MenuEventOperationResult, Operation: "name", Attempt: effects[0].Attempt,
		Address: target, Error: "metadata unavailable",
	})
	f.con.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) {
		return nil, errors.New("metadata unavailable")
	})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.mu.Unlock()
	f.con.dispatchMenuEffects(effects)

	waitFor(t, "operation failure to enter reducer", func() bool {
		got := f.con.menuSnapshot()
		return got.operationForTest().Operation == "" && got.Notice == want.Notice && reflect.DeepEqual(got.Frames, want.Frames)
	})
}

func TestConsoleMenuMissingDispatcherPaintsLocalError(t *testing.T) {
	f := newFixture(t, 24, 80)
	target := menuAddress("couch-one")
	state := NewMenuState(menuThreads(), target)
	state, effects := dispatchThreadOperation(state, "switch", target)
	f.con.mu.Lock()
	f.con.ops = nil
	f.con.menu, f.con.menuReady = state, true
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()
	f.host.Reset()
	f.con.dispatchMenuEffects(effects)

	got := f.con.menuSnapshot()
	if got.operationForTest().Operation != "" || got.Notice.Level != MenuNoticeError || got.Notice.Text != "no action dispatcher wired" {
		t.Fatalf("missing-dispatcher state = %+v", got)
	}
	if screen := string(ansi.Strip([]byte(f.screenText()))); !strings.Contains(screen, "error: no action dispatcher wired") {
		t.Fatalf("missing dispatcher was not painted locally: %q", screen)
	}
}

func TestConsoleMenuStartAttachesBeforeSuccessfulRestoration(t *testing.T) {
	f := newFixture(t, 24, 80)
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	if !appendMenuFrame(&state, MenuFrame{Kind: MenuFrameStart}) {
		t.Fatal("could not build start form")
	}
	state.Frames[len(state.Frames)-1].PreviewResolution = couchcore.StartResolution{
		CanonicalPath: "/repo", Worktree: "/repo", Fingerprint: "accepted",
	}
	state, effects := dispatchMenuOperation(state, MenuEffect{
		Operation: "start", Args: startCommitArgs("accepted"),
	}, couchcore.ThreadAddress{})
	if len(effects) != 1 {
		t.Fatalf("dispatch effects = %+v", effects)
	}
	created := couchcore.ThreadAddress{RepoScope: "repo", Tag: "couch-created"}
	start, _ := attachStartResult(t, "created-actor", created)
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		if name != "start" {
			return nil, errors.New("unexpected owner operation " + name)
		}
		return start, nil
	})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.mu.Unlock()
	f.con.dispatchMenuEffects(effects)

	waitFor(t, "started terminal attach and reducer completion", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		_, attached := f.con.panes[start.Handle.ID()]
		return attached && f.con.menu.operationForTest().Operation == ""
	})
	if !f.con.menuSnapshot().ProjectionPending {
		t.Fatal("successful start silently presented the pre-start inventory as current")
	}
}

func TestConsoleMenuResumeLandsOnExactReturnedHandle(t *testing.T) {
	copied := noticeClipboard(t)
	f := newFixture(t, 24, 80)
	target := menuAddress("couch-two")
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Frames[0].SelectedAddress = target
	state, effects := dispatchThreadOperation(state, "resume", target)
	if len(effects) != 1 {
		t.Fatalf("resume effects = %+v", effects)
	}
	started, _ := attachStartResult(t, "resumed-actor", target)
	started.Record.AttachmentNotice = "BORROWED ATTACH: lifecycle claim is open; retained client state was left unchanged."
	started.Handle.(couchcore.TerminalHandle).Terminal().Feed([]byte("RESUMED-EXACT-SCREEN"))
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		if name != "resume" {
			return nil, errors.New("unexpected owner operation " + name)
		}
		return started, nil
	})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()
	f.host.Reset()
	f.con.dispatchMenuEffects(effects)

	waitFor(t, "exact resumed handle landing", func() bool {
		f.con.mu.Lock()
		active, focus, inflight := f.con.active, f.con.focus, f.con.menu.operationForTest().Operation
		f.con.mu.Unlock()
		return active == started.Handle.ID() && focus == FocusActor(started.Handle.ID()) && inflight == "" && strings.Contains(f.host.Written(), "RESUMED-EXACT-SCREEN")
	})
	if got := f.con.menuSnapshot().Notice.Text; got != started.Warning() {
		t.Fatalf("borrowed attachment notice lost after landing: %q", got)
	}
	if got := f.notice(); got != started.Warning() {
		t.Fatalf("borrowed warning lost from feed: %q", got)
	}
	if !strings.Contains(f.lastRow(), "BORROWED ATTACH: lifecycle claim is ope…") {
		t.Fatalf("borrowed attachment reason was not painted: %q", f.host.Written())
	}
	if strings.Contains(f.screenText(), "threads") {
		t.Fatalf("successful resume repainted the switcher: %q", f.host.Written())
	}
	if !f.con.menuSnapshot().ProjectionPending {
		t.Fatal("successful resume silently presented the pre-resume inventory as current")
	}
	f.con.copyLastNotice()
	if got := copied(); got != started.Warning() {
		t.Fatalf("borrowed warning copied %q; want full %q", got, started.Warning())
	}
}

func TestConsoleSuccessfulOperationsApplyExhaustiveProjectionPolicy(t *testing.T) {
	for _, test := range []struct {
		operation string
		pending   bool
	}{
		{operation: "park", pending: true},
		{operation: "name", pending: true},
		{operation: "describe", pending: true},
		{operation: "switch", pending: false},
	} {
		t.Run(test.operation, func(t *testing.T) {
			con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(con.Stop)
			target := menuAddress("couch-one")
			state := NewMenuState(menuThreads(), target)
			setTestMenuOperation(&state, MenuOperationOrigin{
				Operation: test.operation, Attempt: 1, Address: target,
				FrameInstance: state.Frames[0].Instance, FrameKind: MenuFrameRoot, Depth: 1,
			})
			con.mu.Lock()
			con.menu, con.menuReady = state, true
			con.mu.Unlock()

			con.finishPreparedOperationForTest(operationCompletion{name: test.operation, origin: state.operationForTest()})
			if got := con.menuSnapshot().ProjectionPending; got != test.pending {
				t.Fatalf("ProjectionPending = %v, want %v", got, test.pending)
			}
		})
	}
}

func TestConsoleRefreshFailureKeepsCommittedMutationVisiblyPending(t *testing.T) {
	f := newFixture(t, 24, 80)
	target := menuAddress("couch-one")
	state := NewMenuState(menuThreads(), target)
	setTestMenuOperation(&state, MenuOperationOrigin{
		Operation: "park", Attempt: 1, Address: target,
		FrameInstance: state.Frames[0].Instance, FrameKind: MenuFrameRoot, Depth: 1,
	})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady, f.con.focus = state, true, FocusPanel()
	f.con.refreshSchedule = RefreshSchedule{Sequence: 1, Running: 1}
	f.con.mu.Unlock()
	f.con.finishPreparedOperationForTest(operationCompletion{name: "park", origin: state.operationForTest()})
	f.host.Reset()
	f.con.refreshResults <- menuRefreshResult{generation: 1, err: errors.New("store unavailable")}

	waitFor(t, "failed refresh pending banner", func() bool {
		state := f.con.menuSnapshot()
		screen := string(ansi.Strip([]byte(f.screenText())))
		return state.ProjectionPending && strings.Contains(screen, "error: thread inventory unavailable: store unavailable; refresh pending")
	})
}

func TestConsolePreMutationRefreshCannotAuthorizeCommittedProjection(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	target := menuAddress("couch-one")
	state := NewMenuState(menuThreads(), target)
	setTestMenuOperation(&state, MenuOperationOrigin{
		Operation: "park", Attempt: 1, Address: target,
		FrameInstance: state.Frames[0].Instance, FrameKind: MenuFrameRoot, Depth: 1,
	})
	release := make(chan struct{})
	con.mu.Lock()
	con.actionable = func(ctx context.Context, _ []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		select {
		case <-release:
			return nil, errors.New("follow-up failed")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	con.menu, con.menuReady = state, true
	con.refreshSchedule = RefreshSchedule{Sequence: 1, Running: 1}
	con.mu.Unlock()

	con.finishPreparedOperationForTest(operationCompletion{name: "park", origin: state.operationForTest()})
	select {
	case <-con.refreshRequests:
		con.advanceMenuRefresh(RefreshScheduleEvent{Kind: RefreshRequested})
	case <-testwait.After(t):
		t.Fatal("successful mutation did not request its post-mutation refresh")
	}
	con.finishMenuRefresh(menuRefreshResult{generation: 1, inventory: menuThreads()})
	if got := con.menuSnapshot(); !got.ProjectionPending || got.ProjectionAfterGeneration != 1 || !got.RefreshPending {
		t.Fatalf("pre-mutation refresh authorized committed projection: %+v", got)
	}

	close(release)
	select {
	case result := <-con.refreshResults:
		con.finishMenuRefresh(result)
	case <-testwait.After(t):
		t.Fatal("post-mutation refresh did not finish")
	}
	if got := con.menuSnapshot(); !got.ProjectionPending || got.Notice.Text != "thread inventory unavailable: follow-up failed" {
		t.Fatalf("failed follow-up hid pending projection: %+v", got)
	}
}

func TestConsoleAttachAndSwitchIgnoreDonePaneAwaitingExit(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	address := couchcore.ThreadAddress{RepoScope: "repo", Tag: "couch-resume"}
	old := ptychild.NewFakeChild(nil)
	con.attachThreadActor("old-handle", "old-actor", address, "/repo", "old", old)
	old.Exit(0)
	waitFor(t, "old pane finalization", old.Done)
	if !old.Done() {
		t.Fatal("old pane did not enter done-but-queued state")
	}

	started, _ := attachStartResult(t, "new-actor", address)
	if _, err := con.ExecuteConsoleOperation(attachCall(context.Background(), started)); err != nil {
		t.Fatalf("done pane blocked replacement attach: %v", err)
	}
	con.mu.Lock()
	con.active = "old-handle"
	con.focus = FocusActor("old-handle")
	con.mu.Unlock()
	if _, err := con.ExecuteConsoleOperation(couchcore.OperationCall{
		Name: "switch", Args: map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag)},
	}); err != nil {
		t.Fatal(err)
	}
	con.mu.Lock()
	active := con.active
	con.mu.Unlock()
	if active != started.Handle.ID() {
		t.Fatalf("address switch selected %q, want live replacement %q", active, started.Handle.ID())
	}

	_ = con.onExit(childExit{id: "old-handle", code: 0})
	con.mu.Lock()
	_, replacementPresent := con.panes[started.Handle.ID()]
	active = con.active
	con.mu.Unlock()
	if !replacementPresent || active != started.Handle.ID() {
		t.Fatalf("old queued exit removed or redirected replacement: present=%t active=%q", replacementPresent, active)
	}
}

func TestConsoleMenuAttachRefusalPaintsLocalErrorBanner(t *testing.T) {
	f := newFixture(t, 24, 80)
	f.con.mu.Lock()
	existing := f.con.panes[f.con.order[0]].thread
	f.con.mu.Unlock()
	state := NewMenuState([]couchcore.ActionableThreadSummary{{Address: existing, Name: "root", State: couchcore.ThreadLive}}, existing)
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	state.Frames[len(state.Frames)-1].PreviewResolution = couchcore.StartResolution{
		CanonicalPath: "/repo", Worktree: "/repo", Fingerprint: "accepted",
	}
	state, effects := dispatchMenuOperation(state, MenuEffect{Operation: "start", Args: startCommitArgs("accepted")}, couchcore.ThreadAddress{})
	if len(effects) != 1 {
		t.Fatalf("start dispatch effects = %+v", effects)
	}
	started, _ := attachStartResult(t, "duplicate-actor", existing)
	setTestOps(f.con, func(name string, _ map[string]string) (any, error) {
		if name != "start" {
			return nil, errors.New("unexpected owner operation " + name)
		}
		return started, nil
	})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.focus = FocusPanel()
	f.con.mu.Unlock()
	f.host.Reset()
	f.con.dispatchMenuEffects(effects)

	waitFor(t, "attach refusal completion", func() bool {
		return f.con.menuSnapshot().operationForTest().Operation == ""
	})
	// Reducer completion precedes its paint; wait for the observable result.
	waitFor(t, "attach refusal banner", func() bool {
		screen := string(ansi.Strip([]byte(f.screenText())))
		return strings.Contains(screen, "error: thread ") && strings.Contains(screen, "already attached")
	})
	f.con.mu.Lock()
	focus, inflight := f.con.focus, f.con.menu.operationForTest().Operation
	f.con.mu.Unlock()
	if !focus.IsPanel() || inflight != "" {
		t.Fatalf("attach refusal focus=%+v inflight=%q, want restored switcher", focus, inflight)
	}
	if frame := f.con.menuSnapshot().CurrentFrame(); frame.Kind != MenuFrameRoot {
		t.Fatalf("completed start restored stale form: %+v", frame)
	}
}
