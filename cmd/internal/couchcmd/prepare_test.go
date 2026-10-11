package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// prepareCommands is the command seam with a weave that succeeds and a clean
// checkout: what the CLI test checks is the conversation with Couch.
type prepareCommands struct{ weave []string }

func (p *prepareCommands) Run(_ context.Context, c couchcore.ProvisionCommand) ([]byte, error) {
	if c.Program == "weave" {
		p.weave = append(p.weave, c.Dir)
		return nil, nil
	}
	if len(c.Args) > 0 && c.Args[0] == "rev-parse" {
		return []byte("0123456789abcdef\n"), nil
	}
	return nil, nil // status: clean
}

// prepareScript answers the admission and the reload, each admitted then
// finished at the first poll, with the given terminal receipts.
func prepareScript(admit, reload couchmessage.OperationReceipt) slotScript {
	return func(n int, r couchmessage.Request) (couchmessage.Response, error) {
		receipt := admit
		if n >= 2 {
			receipt = reload
		}
		if n%2 == 0 {
			queued := receipt
			queued.Status = couchmessage.ReceiptQueued
			return couchmessage.Response{Code: "accepted", Operation: &queued}, nil
		}
		return couchmessage.Response{Code: "ok", Operation: &receipt}, nil
	}
}

func runPrepareScript(t *testing.T, inv cliInvocation, script slotScript) (slotRun, *prepareCommands) {
	t.Helper()
	var run slotRun
	var out, errout bytes.Buffer
	now := time.Unix(5000, 0)
	clock := slotPollClock{now: func() time.Time { return now }, sleep: func(d time.Duration) { now = now.Add(d) }}
	call := func(_ context.Context, _ string, request any, response any) error {
		r := request.(couchmessage.Request)
		n := len(run.calls)
		run.calls = append(run.calls, r)
		resp, err := script(n, r)
		*(response.(*couchmessage.Response)) = resp
		return err
	}
	commands := &prepareCommands{}
	run.code = runPrepareCLI(inv, messageRuntime(), &out, &errout, call, clock, commands)
	run.stdout, run.stderr = out.String(), errout.String()
	return run, commands
}

func TestPrepareCLI(t *testing.T) {
	inv := cliInvocation{kind: cliMessage, messageOp: couchcore.OpPrepare, ref: "pair:1", confirmed: true, forceUnknown: true}
	admitted := couchmessage.OperationReceipt{ID: "id", Op: couchcore.OpPrepare, Target: "pair:1", Status: couchmessage.ReceiptSucceeded, Tag: "t1", Path: t.TempDir()}
	reloaded := couchmessage.OperationReceipt{ID: "id", Op: couchcore.OpReloadContext, Target: "pair:1", Status: couchmessage.ReceiptSucceeded, Tag: "t1"}

	t.Run("admitted: refresh in the admitted checkout, then a confirmed reload", func(t *testing.T) {
		run, commands := runPrepareScript(t, inv, prepareScript(admitted, reloaded))
		if run.code != 0 || len(commands.weave) != 1 || commands.weave[0] != admitted.Path {
			t.Fatalf("exit %d weave %v\n%s%s", run.code, commands.weave, run.stdout, run.stderr)
		}
		if len(run.calls) != 4 || run.calls[0].Op != couchcore.OpPrepare || run.calls[2].Op != couchcore.OpReloadContext {
			t.Fatalf("requests %+v", run.calls)
		}
		for _, i := range []int{0, 2} {
			if !run.calls[i].Confirmed || !run.calls[i].ForceUnknown || run.calls[i].Target != "pair:1" {
				t.Fatalf("request %d lost the caller's flags: %+v", i, run.calls[i])
			}
		}
		if !strings.Contains(run.stdout, "reload   ok") || !strings.Contains(run.stdout, "0123456789ab") {
			t.Fatalf("report:\n%s", run.stdout)
		}
	})

	t.Run("a refused admission sends nothing else and runs no weave", func(t *testing.T) {
		busy := admitted
		busy.Status, busy.Code, busy.Detail, busy.Path = couchmessage.ReceiptRefused, couchcore.LiveRestartBusy, "pair:1: a turn is open", ""
		run, commands := runPrepareScript(t, inv, prepareScript(busy, reloaded))
		if run.code != 1 || len(run.calls) != 2 || len(commands.weave) != 0 || !strings.Contains(run.stdout, "admit    refused  (busy) pair:1: a turn is open") {
			t.Fatalf("exit %d calls %d weave %v\n%s", run.code, len(run.calls), commands.weave, run.stdout)
		}
	})

	t.Run("--json prints one report", func(t *testing.T) {
		asJSON := inv
		asJSON.jsonOutput = true
		run, _ := runPrepareScript(t, asJSON, prepareScript(admitted, reloaded))
		var report couchcore.PrepareReport
		if err := json.Unmarshal([]byte(run.stdout), &report); err != nil || !report.OK() || report.Path != admitted.Path {
			t.Fatalf("exit %d %v: %q", run.code, err, run.stdout)
		}
	})

	t.Run("an older Couch stops before any step", func(t *testing.T) {
		run, commands := runPrepareScript(t, inv, func(int, couchmessage.Request) (couchmessage.Response, error) {
			return couchmessage.Response{Code: "invalid-request", Error: "unknown message operation"}, nil
		})
		if run.code != 1 || len(commands.weave) != 0 || run.stdout != "" || !strings.Contains(run.stderr, "predates `couch --prepare`") {
			t.Fatalf("exit %d\nstdout %q\nstderr %q", run.code, run.stdout, run.stderr)
		}
	})
}
