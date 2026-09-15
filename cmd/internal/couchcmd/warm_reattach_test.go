package couchcmd

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// This crosses the real inventory, keyboard reducer, operation table, resume
// transaction and Console attachment. Only the terminal and external session
// processes are fake; no native binding is supplied anywhere.
func TestSwitcherWarmReattachWithoutNativeBindingReachesTerminal(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.runner = couchcore.NewFakeRunner()
	thread := seedDetachedThread(t, rt, "/repo")
	session := "pair-" + string(thread.Address.Tag)
	rt.artifacts.SetDetachedSession(thread.Address, session)
	rt.runner.AfterAcknowledge = func(string) error {
		rt.artifacts.SetDetachedSession(thread.Address, "")
		rt.artifacts.SetPairSession(thread.Address, session, true)
		return nil
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100})
	reader, input := io.Pipe()
	t.Cleanup(func() { _ = reader.Close(); _ = input.Close() })
	console := couchtty.New(host, reader)
	t.Cleanup(console.Stop)
	initial := ptychild.NewFakeChild(nil)
	console.Attach("initial-console-pane", "initial", initial)
	t.Cleanup(func() { initial.Exit(0) })
	rt.runner.AfterBlockedStart = func(id string) {
		rt.runner.Terminal(id).SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return console.Deliver(ctx, id, batch) })
	}
	wireResolver(console, c)
	rows, err := console.ActionableProvider()(context.Background(), nil)
	if err != nil || len(rows) != 1 || rows[0].State != couchcore.ThreadDetached {
		t.Fatalf("unbound switcher inventory = %+v, %v; want one detached row", rows, err)
	}
	if got := rt.artifacts.BindingResolutions(); got != 0 {
		t.Fatalf("warm inventory resolved native binding %d times", got)
	}
	type dispatchResult struct {
		call  couchcore.OperationCall
		value any
		err   error
	}
	completed := make(chan dispatchResult, 8)
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		value, err := actual(call)
		completed <- dispatchResult{call, value, err}
		return value, err
	})
	done := make(chan int, 1)
	go func() { done <- console.Run() }()
	t.Cleanup(func() {
		console.Stop()
		_ = input.Close()
		_ = reader.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("Console did not stop")
		}
	})
	if _, err := input.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "switcher row", func() bool {
		return strings.Contains(host.Written(), "threads") && strings.Contains(host.Written(), thread.WorkingPath)
	})
	if _, err := input.Write([]byte{'\r'}); err != nil {
		t.Fatal(err)
	}
	var start couchcore.StartResult
	for {
		select {
		case result := <-completed:
			if result.err != nil {
				t.Fatalf("%s failed: %v", result.call.Name, result.err)
			}
			if result.call.Name == "resume" {
				var ok bool
				start, ok = result.value.(couchcore.StartResult)
				if !ok {
					t.Fatalf("resume returned %T", result.value)
				}
				t.Cleanup(func() { rt.runner.SetExited(start.Handle.ID(), 0) })
				if result.call.Args["warm-only"] != "true" {
					t.Fatalf("foreground resume lost warm intent: %+v", result.call)
				}
			}
			if result.call.Name == "attach" {
				goto attached
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("switcher did not attach: %s", host.Written())
		}
	}
attached:
	if start.Handle == nil || start.Record.Thread != thread.Address {
		t.Fatalf("attached wrong thread: %+v", start)
	}
	child := rt.runner.Child(start.Handle.ID())
	if !slices.Equal(child.Argv, []string{"pair", "resume", string(thread.Address.Tag)}) || child.ExecCount != 1 {
		t.Fatalf("warm helper = %+v", child)
	}
	for _, entry := range child.Env {
		if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") || strings.HasPrefix(entry, "PAIR_USE_REPO_DEFAULT=") {
			t.Fatalf("warm helper carried cold-launch configuration: %s", entry)
		}
	}
	terminal := start.Handle.(couchcore.TerminalHandle).Terminal()
	terminal.Feed([]byte("WARM-EXACT-TERMINAL"))
	waitWarmAcceptance(t, "resumed output", func() bool { return strings.Contains(host.Written(), "WARM-EXACT-TERMINAL") })
	if _, err := input.Write([]byte("warm-input")); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "resumed input", func() bool { return bytes.Contains(bytes.Join(terminal.Writes(), nil), []byte("warm-input")) })
	binding, err := rt.artifacts.PairSession(thread.Address)
	if err != nil || !binding.Present || binding.Name != session {
		t.Fatalf("surviving session changed: %+v, %v", binding, err)
	}
	if got := rt.artifacts.BindingResolutions(); got != 0 {
		t.Fatalf("warm acceptance resolved native binding %d times", got)
	}
	if stopped := rt.artifacts.Quiesces(); len(stopped) != 0 {
		t.Fatalf("warm attachment stopped preexisting sessions: %v", stopped)
	}
	native, nativeErr := rt.artifacts.ResolveEstablished(context.Background(), thread.Address.RepoScope, string(thread.Address.Tag), "claude")
	if native.Status != sessioninventory.BindingUnbound || native.NativeID != "" || nativeErr == nil {
		t.Fatalf("warm attachment fabricated native authority: %+v, %v", native, nativeErr)
	}
}

func TestSwitcherWarmSelectionCannotColdResumeAfterPark(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.runner = couchcore.NewFakeRunner()
	thread := seedDetachedThread(t, rt, "/repo")
	rt.artifacts.SetDetachedSession(thread.Address, "pair-"+string(thread.Address.Tag))
	rt.artifacts.SetNativeBinding(thread.Address, "claude", sessioninventory.BindingEstablished, "native-established")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
	t.Cleanup(console.Stop)
	wireResolver(console, c)
	rows, err := console.ActionableProvider()(context.Background(), nil)
	if err != nil || len(rows) != 1 || rows[0].State != couchcore.ThreadDetached {
		t.Fatalf("inventory = %+v, %v", rows, err)
	}
	state := couchtty.NewMenuState(rows, couchcore.ThreadAddress{})
	_, effects := couchtty.ReduceMenu(state, couchtty.MenuEvent{Kind: couchtty.MenuEventKey, Key: couchtty.PanelKey{Kind: couchtty.KeyEnter}})
	if len(effects) != 1 || effects[0].Operation != "resume" {
		t.Fatalf("Enter effects = %+v", effects)
	}
	if effects[0].Args["warm-only"] != "true" {
		t.Fatalf("Enter lost selected warm intent: %+v", effects[0])
	}
	// Park after selection. Supply a valid native binding so an unrestricted
	// resume would really start a cold replacement instead of failing incidentally.
	current, err := c.Threads.UpdateExistingThread(thread.Address, thread.Revision, func(next *couchcore.ThreadRecord) error {
		next.Incarnations = []couchcore.ThreadIncarnation{{PID: 42, Identity: "parked-helper", State: couchcore.IncarnationLive, RepoIdentity: "/repo/.git", LaunchProfile: next.LatestLaunchProfile}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	identity := couchcore.ParkIdentity{Nonce: "selected-warm-then-parked", Address: thread.Address, PID: 42, ProcessIdentity: "parked-helper"}
	begun, err := c.Threads.BeginPark(thread.Address, current.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	parked, err := c.Threads.FinalizePark(thread.Address, begun.Revision, identity, 1, time.Unix(2, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	rt.artifacts.SetDetachedSession(thread.Address, "")
	rt.artifacts.SetNativeBinding(thread.Address, "claude", sessioninventory.BindingEstablished, "native-established")
	effect := effects[0]
	_, err = console.Ops()(couchcore.OperationCall{Name: effect.Operation, Args: effect.Args, Implicit: true, Context: context.Background()})
	if got := couchcore.ResumeDiagnosticOf(err); got != couchcore.ResumeNotDetached {
		t.Fatalf("stale warm selection = %v (%s), want %s", err, got, couchcore.ResumeNotDetached)
	}
	after, err := c.Threads.GetThread(thread.Address)
	if err != nil || !reflect.DeepEqual(after, parked) {
		t.Fatalf("refusal changed parked record: %+v, %v", after, err)
	}
	if len(rt.runner.Ops) != 0 || rt.artifacts.BindingResolutions() != 0 {
		t.Fatalf("stale selection reached cold path: ops=%v binding resolutions=%d", rt.runner.Ops, rt.artifacts.BindingResolutions())
	}
}

func waitWarmAcceptance(t *testing.T, what string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
