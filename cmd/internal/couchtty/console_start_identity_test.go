package couchtty

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// A primary start is admitted before its native address exists. Its ordinary
// inventory row can arrive while adoption is still running; that row must show
// and select the original owner rather than admitting another lifecycle action.
func TestConsoleStartIdentityMaterializesDuringHeldAdoption(t *testing.T) {
	worktree := t.TempDir()
	scope, err := launcher.ResolveRepoScope(worktree)
	if err != nil {
		t.Fatal(err)
	}
	address := couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "created-primary"}
	r := &navigationJobsFixture{
		adoptionEntered: make(chan struct{}), adoptionRelease: make(chan struct{}),
	}
	r.start, _ = attachStartResult(t, "created-primary", address)
	t.Cleanup(func() { _ = r.start.Handle.(couchcore.TerminalHandle).Terminal().Close() })
	var originalRow couchcore.ActionableThreadSummary
	r.f = newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		originalRow = couchcore.ActionableThreadSummary{Address: c.panes["c1"].thread, Name: "original", WorkingPath: "/original", State: couchcore.ThreadLive}
		c.menu = NewMenuState([]couchcore.ActionableThreadSummary{originalRow}, originalRow.Address)
		c.menuReady = true
		c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			switch call.Name {
			case "start":
				r.lifecycleCalls.Add(1)
				return r.start, nil
			case "attach":
				if r.adoptionCalls.Add(1) == 1 {
					close(r.adoptionEntered)
				}
				<-r.adoptionRelease
				return c.ExecuteConsoleOperation(call)
			default:
				r.lifecycleCalls.Add(1)
				return nil, fmt.Errorf("unexpected second operation %s during start", call.Name)
			}
		})
	})
	t.Cleanup(func() { r.releaseAdoption.Do(func() { close(r.adoptionRelease) }) })
	var originalKey menuOperationTarget
	var originalOrigin MenuOperationOrigin
	if err := r.f.con.runTerminalCommand(context.Background(), func() error {
		c := r.f.con
		c.mu.Lock()
		appendMenuFrame(&c.menu, MenuFrame{Kind: MenuFrameStart, PreviewResolution: couchcore.StartResolution{Worktree: couchcore.Worktree(worktree)}})
		state, effects := dispatchMenuOperation(c.menu, MenuEffect{Operation: "start"}, couchcore.ThreadAddress{})
		if len(effects) != 1 {
			c.mu.Unlock()
			return fmt.Errorf("start effects = %+v", effects)
		}
		r.attempt = effects[0].Attempt
		originalKey, _, _ = menuOperationForAttempt(state, r.attempt)
		originalOrigin = menuOriginForAttempt(state, r.attempt)
		c.menu, c.focus = state, FocusPanel()
		c.mu.Unlock()
		c.showMenu()
		c.dispatchMenuEffects(effects)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if originalKey.StartPath != worktree || originalOrigin.RowKey != (couchcore.ThreadRowKey{}) || originalOrigin.Address != (couchcore.ThreadAddress{}) {
		t.Fatalf("fixture must reserve an untyped primary start: key=%+v origin=%+v", originalKey, originalOrigin)
	}
	r.await(t, "start adoption entry", r.adoptionEntered)
	created := couchcore.ActionableThreadSummary{
		Address: address, Name: "created-primary", WorkingPath: worktree,
		Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetOrdinary, Address: address},
		RowKey: couchcore.ThreadRowKey{Kind: couchcore.ThreadTargetOrdinary, Address: address},
		State:  couchcore.ThreadLive,
	}
	if err := r.f.con.runTerminalCommand(context.Background(), func() error {
		r.f.con.reduceMenu(MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{originalRow, created}, Generation: 1})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	state := r.f.con.menuSnapshot()
	job, owned := menuOperationForRow(state, created)
	if !owned || job.Origin.Attempt != r.attempt || job.Phase != MenuOperationRunning {
		t.Fatalf("new primary row lost its running start owner during adoption: owned=%v job=%+v", owned, job)
	}
	rendered := RenderMenuView(state, 100, 24, time.Now(), false)
	lines := strings.Split(string(ansi.Strip([]byte(rendered.Body))), "\r\n")
	visibleProgress := false
	for _, extent := range rendered.Extents {
		if extent.Thread == address {
			visibleProgress = strings.Contains(lines[extent.Start], "start…")
		}
	}
	if !visibleProgress {
		t.Fatalf("created primary row has no visible start progress: %q", rendered.Body)
	}
	if err := r.f.con.runTerminalCommand(context.Background(), func() error {
		c := r.f.con
		c.mu.Lock()
		next, effects := dispatchThreadOperation(c.menu, "detach", address)
		c.menu = next
		c.mu.Unlock()
		if len(effects) != 0 || !strings.Contains(next.Notice.Text, "busy: start") {
			return fmt.Errorf("materialized primary admitted a second operation: effects=%+v notice=%+v", effects, next.Notice)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	r.choose(t, address)
	state = r.f.con.menuSnapshot()
	key, waiting, running := menuOperationForAttempt(state, r.attempt)
	if !running || state.LandingAttempt != r.attempt || key != originalKey || !reflect.DeepEqual(waiting.Origin, originalOrigin) {
		t.Fatalf("busy-row selection changed its owner instead of waiting: key=%+v origin=%+v landing=%d", key, waiting.Origin, state.LandingAttempt)
	}
	r.releaseAdoption.Do(func() { close(r.adoptionRelease) })
	r.completed(t)
	r.input(t, r.start.Handle.ID(), r.start.Handle.(couchcore.TerminalHandle).Terminal(), "n")
	state = r.f.con.menuSnapshot()
	job, owned = menuOperationForRow(state, created)
	if !owned || !reflect.DeepEqual(job.Origin, originalOrigin) {
		t.Fatalf("completion rewrote the start origin: owned=%v origin=%+v want=%+v", owned, job.Origin, originalOrigin)
	}
	if got := r.lifecycleCalls.Load(); got != 1 {
		t.Fatalf("materialized busy row caused %d native lifecycle effects, want one start", got)
	}
	if got := r.adoptionCalls.Load(); got != 1 {
		t.Fatalf("original child adopted %d times, want once", got)
	}
}
