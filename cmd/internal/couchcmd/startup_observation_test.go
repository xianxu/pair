package couchcmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

func TestInteractiveStartupOpensConsoleWithoutLaunching(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.runner = couchcore.NewFakeRunner()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	finished := false
	finish := func(console *couchtty.Console, c *couchcore.Couch, start couchcore.StartResult, _ io.Writer) int {
		finished = true
		if console == nil || c == nil || start.Handle != nil || len(rt.runner.Ops) != 0 {
			t.Fatalf("startup launched before choice: %+v ops=%v", start, rt.runner.Ops)
		}
		wireResolver(console, c)
		if err := beginConsole(console, start, true); err != nil {
			t.Fatalf("empty console initialization: %v", err)
		}
		return 0
	}
	op, _ := Resolve("start")
	var stdout, stderr bytes.Buffer
	code := runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"}, true, "", slave, slave, slave, &stdout, &stderr, rt, finish)
	if code != 0 || !finished {
		t.Fatalf("console did not open: code=%d finished=%v err=%s", code, finished, stderr.String())
	}
}
