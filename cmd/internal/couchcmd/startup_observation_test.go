package couchcmd

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestInteractiveHubStartupOpensEmptySwitcherWithoutLaunching(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.runner = couchcore.NewFakeRunner()
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	before, err := c.Threads.PreviewSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	runObservedInteractiveStartup(t, rt, func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, _ io.Writer) int {
		if console == nil || c == nil || start.Handle != nil || start.Record.Thread != (couchcore.ThreadAddress{}) || len(rt.runner.Ops) != 0 {
			t.Fatalf("empty startup launched instead of opening switcher: %+v ops=%v", start, rt.runner.Ops)
		}
		wireResolver(console, c)
		if err := beginConsole(console, start); err != nil {
			t.Fatalf("empty console initialization: %v", err)
		}
		if !console.ReattachPassArmed() {
			t.Fatal("empty launch hub left surviving-worker warm pass unarmed")
		}
		rows, err := console.ActionableProvider()(context.Background(), nil)
		if err != nil || len(rows) != 0 {
			t.Fatalf("empty switcher inventory: %+v %v", rows, err)
		}
		if _, err := console.Ops()(couchcore.OperationCall{Name: "list"}); err != nil {
			t.Fatalf("empty switcher dispatcher unusable: %v", err)
		}
		after, err := c.Threads.PreviewSnapshot()
		if err != nil || !reflect.DeepEqual(before, after) || len(rt.runner.Ops) != 0 || rt.artifacts.BindingResolutions() != 0 || len(rt.registryRecords(t)) != 0 {
			t.Fatalf("empty startup had lifecycle effects: snapshot=%+v err=%v ops=%v", after, err, rt.runner.Ops)
		}
		return 0
	})
}

func TestInteractiveHubStartupAttachesSeededSurvivingSession(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.runner = couchcore.NewFakeRunner()
	retained := seedDetachedThread(t, rt, "/repo")
	name := "pair-" + retained.Address.RepoScope + "-" + string(retained.Address.Tag)
	rt.artifacts.SetDetachedSession(retained.Address, name)
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	before, err := c.Threads.PreviewSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	runObservedInteractiveStartup(t, rt, func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, _ io.Writer) int {
		if console == nil || c == nil || start.Handle == nil || start.Record.Thread != retained.Address || start.Record.Shape != couchcore.StartWarmReattach || len(rt.runner.Ops) != 2 {
			t.Fatalf("surviving hub was not warm-attached: %+v ops=%v", start, rt.runner.Ops)
		}
		t.Cleanup(func() { rt.runner.SetExited(start.Handle.ID(), 0) })
		child := rt.runner.Child(start.Handle.ID())
		want := []string{"pair", launcher.CouchSessionFlag, "resume", string(retained.Address.Tag)}
		if !reflect.DeepEqual(child.Argv, want) {
			t.Fatalf("surviving startup attempted native launch: argv=%v", child.Argv)
		}
		wireResolver(console, c)
		if err := beginConsole(console, start); err != nil {
			t.Fatalf("surviving hub adoption: %v", err)
		}
		after, err := c.Threads.PreviewSnapshot()
		if err != nil || !reflect.DeepEqual(before, after) || rt.artifacts.BindingResolutions() != 0 || rt.artifacts.DetachedQueries() != 0 {
			t.Fatalf("warm startup changed retained records or read cold proof: snapshot=%+v err=%v", after, err)
		}
		return 0
	})
}

func runObservedInteractiveStartup(t *testing.T, rt testRT, inspect consoleFinisher) {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	if err := pty.Setsize(slave, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	finished := false
	finish := func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, out io.Writer) int {
		finished = true
		if console != nil {
			t.Cleanup(console.Stop)
		}
		return inspect(console, c, start, out)
	}
	op, _ := Resolve("start")
	var stdout, stderr bytes.Buffer
	code := runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"}, true, "", slave, slave, slave, &stdout, &stderr, rt, finish)
	if code != 0 || !finished {
		t.Fatalf("console did not open: code=%d finished=%v err=%s", code, finished, stderr.String())
	}
}
