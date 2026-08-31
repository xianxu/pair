package couchtty

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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

	waitUpTo(t, 250*time.Millisecond, "operation failure to enter reducer", func() bool {
		got := f.con.menuSnapshot()
		return got.InFlight.Operation == "" && got.Notice == want.Notice && reflect.DeepEqual(got.Frames, want.Frames)
	})
}

func TestConsoleMenuStartAttachesBeforeSuccessfulRestoration(t *testing.T) {
	f := newFixture(t, 24, 80)
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	if !appendMenuFrame(&state, MenuFrame{Kind: MenuFrameStart}) {
		t.Fatal("could not build start form")
	}
	state, effects := dispatchMenuOperation(state, MenuEffect{
		Operation: "start", Args: map[string]string{"path": "/repo", "token": "accepted"},
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

	waitUpTo(t, 250*time.Millisecond, "started terminal attach and reducer completion", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		_, attached := f.con.panes[start.Handle.ID()]
		return attached && f.con.menu.InFlight.Operation == ""
	})
}
