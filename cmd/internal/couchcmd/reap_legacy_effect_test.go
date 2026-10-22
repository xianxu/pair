package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

func lateLegacyWorld(t *testing.T) (*serviceRig, testRT, *composedReapTable) {
	t.Helper()
	rig, rt, table := composedReapWorld(t)
	sampler := rig.s.resources.Load()
	if _, err := sampler.sweep(t.Context()); err != nil {
		t.Fatal(err)
	}
	selectionEnableLegacyNvim(sampler, table)
	selectionSetWorld(table, []procutil.Process{
		{PID: 11, PPID: 1, Identity: "old-child", Name: "codex"},
		{PID: 31, PPID: 1, Identity: "foreign-child", Name: "codex"},
		{PID: 20, PPID: 1, Identity: "current-own", Name: "zellij"},
		{PID: 40, PPID: 1, Identity: "current-foreign", Name: "zellij"},
		{PID: 52, PPID: 1, Identity: "clean-runner", Name: "codex"},
		selectionNvimRow(),
	})
	if err := sampler.leftRunning.Add(t.Context(), []couchcore.LeftRunningOrphan{{Origin: "slot:/repo", Process: couchcore.ProcessIdentity{PID: 52, Identity: "clean-runner"}, Command: "codex", LeftAt: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	return rig, rt, table
}

// Loss is acknowledged by the real shared worker before resuming the already
// admitted CLI. A later healthy collection cannot revoke the sticky refusal.
func acknowledgeLegacyLoss(t *testing.T, ctx context.Context, sampler *resourceSampler, reader *samplerReader) {
	t.Helper()
	reader.mu.Lock()
	reader.tableErr = errors.New("late incomplete census")
	reader.mu.Unlock()
	if _, err := sampler.proveReap(ctx); err != nil {
		t.Fatal(err)
	}
	reader.mu.Lock()
	reader.tableErr = nil
	reader.mu.Unlock()
}

func TestLegacySignalsRecheckLossAfterHealthyAdmission(t *testing.T) {
	for _, stage := range []string{"after-admission", "during-exact-dispatch", "after-exact-signal", "during-exact-wait"} {
		t.Run(stage, func(t *testing.T) {
			rig, rt, table := lateLegacyWorld(t)
			admitted, injected := false, false
			armed := false
			native := &gatedLegacyProcesses{composedReapTable: table}
			native.beforeSnapshot = func() {
				if stage == "during-exact-wait" && armed && !injected {
					acknowledgeLegacyLoss(t, t.Context(), rig.s.resources.Load(), table.reader)
					injected = true
				}
			}
			rig.s.provenOrphanProcesses = native
			call := func(ctx context.Context, _ string, raw, out any) error {
				req := raw.(couchmessage.Request)
				if stage == "during-exact-dispatch" && req.Op == "reap-orphan-signal" && !injected {
					acknowledgeLegacyLoss(t, ctx, rig.s.resources.Load(), table.reader)
					injected = true
				}
				response := rig.s.handle(ctx, req)
				if stage == "during-exact-wait" && req.Op == "reap-orphan-signal" && response.OrphanSignal != nil && response.OrphanSignal.Status == "sent" {
					armed = true
				}

				if req.Op == "reap-legacy-signal" && response.LegacySignal == nil {
					t.Fatalf("legacy effect did not reach supervisor gate: %+v", response)
				}
				if req.Op == orphanReapOp {
					if response.Code != "ok" || response.Resources == nil || len(response.Resources.OrphanNvim) != 1 || len(response.Resources.LeftRunning) != 1 {
						t.Fatalf("not a healthy admitted legacy plan: %+v", response)
					}
					admitted = true
					if stage == "after-admission" {
						acknowledgeLegacyLoss(t, ctx, rig.s.resources.Load(), table.reader)
						injected = true
					}
				}
				if stage == "after-exact-signal" && req.Op == "reap-orphan-signal" && !injected && response.OrphanSignal != nil && response.OrphanSignal.Status == "sent" {
					acknowledgeLegacyLoss(t, ctx, rig.s.resources.Load(), table.reader)
					injected = true
				}
				*out.(*couchmessage.Response) = response
				return nil
			}
			var stdout, stderr bytes.Buffer
			code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true, jsonOutput: true}, rt, &stdout, &stderr, call, fastOrphanReaper(native))
			if !admitted || !injected {
				t.Fatalf("scenario did not execute admission/loss: admitted=%v injected=%v", admitted, injected)
			}
			for _, pid := range []int{51, 52} {
				if len(table.signals[pid]) != 0 {
					t.Errorf("%s signalled legacy PID%d after acknowledged loss: %v", stage, pid, table.signals[pid])
				}
			}
			if code != 1 {
				t.Fatalf("missing refusal exit%d stdout%s stderr%s", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestLegacySignalsRecheckLossBetweenTermAndKill(t *testing.T) {
	for _, family := range []string{"orphan-nvim", "left-running"} {
		t.Run(family, func(t *testing.T) {
			rig, rt, table := lateLegacyWorld(t)
			selectionSetWorld(table, []procutil.Process{{PID: 20, PPID: 1, Identity: "current-own", Name: "zellij"}, {PID: 40, PPID: 1, Identity: "current-foreign", Name: "zellij"}, {PID: 52, PPID: 1, Identity: "clean-runner", Name: "codex"}, selectionNvimRow()})
			table.diesOn = nil // both roots survive TERM, requiring new guarded KILLs
			injected := false
			counts := map[int]int{}
			call := func(ctx context.Context, _ string, raw, out any) error {
				req := raw.(couchmessage.Request)
				response := rig.s.handle(ctx, req)
				if req.Op == "reap-legacy-signal" && response.LegacySignal == nil {
					t.Fatalf("legacy handler unavailable: %+v", response)
				}
				if req.Op == "reap-legacy-signal" && req.LegacyTarget.Family == family && req.OrphanSignal == "TERM" && !injected && response.LegacySignal.Status == "sent" {
					acknowledgeLegacyLoss(t, ctx, rig.s.resources.Load(), table.reader)
					injected = true
					for _, pid := range []int{51, 52} {
						counts[pid] = len(table.signals[pid])
					}
				}
				*out.(*couchmessage.Response) = response
				return nil
			}
			var stdout, stderr bytes.Buffer
			code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true, jsonOutput: true}, rt, &stdout, &stderr, call, fastOrphanReaper(table))
			if !injected {
				t.Fatalf("did not acknowledge loss after %s TERM: stdout%s stderr%s", family, stdout.String(), stderr.String())
			}
			for _, pid := range []int{51, 52} {
				if len(table.signals[pid]) != counts[pid] {
					t.Errorf("late legacy signal PID%d: before%d after%v", pid, counts[pid], table.signals[pid])
				}
			}
			if code != 1 {
				t.Fatalf("lost partial refusal: exit%d stdout%s stderr%s", code, stdout.String(), stderr.String())
			}
		})
	}
}

type gatedLegacyProcesses struct {
	*composedReapTable
	beforeSignal   func()
	beforeSnapshot func()
	identity       func(int) string
}

func (p *gatedLegacyProcesses) Snapshot(ctx context.Context) ([]launcher.ProcessRow, error) {
	if p.beforeSnapshot != nil {
		p.beforeSnapshot()
	}
	return p.composedReapTable.Snapshot(ctx)
}

func (p *gatedLegacyProcesses) Identity(pid int) string {
	if p.identity != nil {
		return p.identity(pid)
	}
	return p.composedReapTable.Identity(pid)
}
func (p *gatedLegacyProcesses) Signal(pid int, sig syscall.Signal) error {
	if p.beforeSignal != nil {
		p.beforeSignal()
	}
	return p.composedReapTable.Signal(pid, sig)
}

func legacyEffectRequest(rt testRT, family string, root, key couchcore.ResourceProcessKey) couchmessage.Request {
	req := callerIdentity(rt, "reap-legacy-signal")
	req.Confirmed = true
	req.LegacyTarget = &couchcore.LegacyOrphanTarget{Family: family, Root: root, Key: key}
	req.OrphanSignal = "TERM"
	return req
}

func TestLegacySignalGateLinearizesStickyLoss(t *testing.T) {
	rig, rt, table := lateLegacyWorld(t)
	sampler := rig.s.resources.Load()
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	native := &gatedLegacyProcesses{composedReapTable: table, beforeSignal: func() {
		if sampler.legacyGate.TryLock() {
			sampler.legacyGate.Unlock()
			t.Error("legacy sticky transition could commit during native signal")
		}
		if rig.s.authority.hubs.mu.TryLock() {
			rig.s.authority.hubs.mu.Unlock()
			t.Error("hub move could commit during native signal")
		}
		close(entered)
		<-release
	}}
	rig.s.provenOrphanProcesses = native
	req := legacyEffectRequest(rt, "left-running", couchcore.ResourceProcessKey{PID: 52, Identity: "clean-runner"}, couchcore.ResourceProcessKey{PID: 52, Identity: "clean-runner"})
	effectDone := make(chan couchmessage.Response, 1)
	go func() { effectDone <- rig.s.handleLegacyOrphanSignal(t.Context(), req) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("signal never reached native effect")
	}
	tableCollected := make(chan struct{})
	var collectedOnce sync.Once
	table.reader.mu.Lock()
	table.reader.tableErr = errors.New("concurrent ownership loss")
	table.reader.tableHook = func(context.Context) { collectedOnce.Do(func() { close(tableCollected) }) }
	table.reader.mu.Unlock()
	lossDone := make(chan error, 1)
	go func() { _, err := sampler.proveReap(t.Context()); lossDone <- err }()
	select {
	case <-tableCollected:
	case <-time.After(time.Second):
		t.Fatal("concurrent proof did not collect loss")
	}
	select {
	case err := <-lossDone:
		t.Fatalf("sticky loss committed through active signal gate: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case response := <-effectDone:
		if response.LegacySignal == nil || response.LegacySignal.Status != "sent" {
			t.Fatalf("ordered effect %+v", response)
		}
	case <-time.After(time.Second):
		t.Fatal("native effect did not join")
	}
	select {
	case err := <-lossDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("sticky transition did not commit after signal")
	}
	table.reader.mu.Lock()
	table.reader.tableErr = nil
	table.reader.tableHook = nil
	table.reader.mu.Unlock()
	req = legacyEffectRequest(rt, "orphan-nvim", selectionNvimRowKey(), selectionNvimRowKey())
	refused := rig.s.handleLegacyOrphanSignal(t.Context(), req)
	if refused.LegacySignal == nil || refused.LegacySignal.Status != "refused" || len(table.signals[51]) != 0 {
		t.Fatalf("post-loss effect %+v signals%v", refused, table.signals)
	}
}
func selectionNvimRowKey() couchcore.ResourceProcessKey {
	row := selectionNvimRow()
	return couchcore.ResourceProcessKey{PID: row.PID, Identity: row.Identity}
}

func TestLegacySignalRefusesSurvivorAfterRootOrCallerChanges(t *testing.T) {
	for _, change := range []string{"root-gone", "reparented", "caller-incarnation", "native-root-reused"} {
		t.Run(change, func(t *testing.T) {
			rig, rt, table := lateLegacyWorld(t)
			table.reader.mu.Lock()
			rows := append([]procutil.Process(nil), table.reader.rows...)
			table.reader.mu.Unlock()
			rows = append(rows, procutil.Process{PID: 53, PPID: 52, Identity: "surviving-child", Name: "codex"})
			selectionSetWorld(table, rows)
			req := legacyEffectRequest(rt, "left-running", couchcore.ResourceProcessKey{PID: 52, Identity: "clean-runner"}, couchcore.ResourceProcessKey{PID: 53, Identity: "surviving-child"})
			first := rig.s.handleLegacyOrphanSignal(t.Context(), req)
			if first.LegacySignal == nil || first.LegacySignal.Status != "sent" {
				t.Fatalf("initial TERM %+v", first)
			}
			switch change {
			case "root-gone":
				var next []procutil.Process
				for _, row := range rows {
					if row.PID == 52 {
						continue
					}
					if row.PID == 53 {
						row.PPID = 1
					}
					next = append(next, row)
				}
				selectionSetWorld(table, next)
			case "reparented":
				for i := range rows {
					if rows[i].PID == 53 {
						rows[i].PPID = 1
					}
				}
				selectionSetWorld(table, rows)
			case "caller-incarnation":
				caller, err := rig.s.switchCaller(t.Context(), req)
				if err != nil {
					t.Fatal(err)
				}
				rig.world.set(caller, func(s *worldSlot) { s.recorded = false })
			case "native-root-reused":
				rig.s.provenOrphanProcesses = &gatedLegacyProcesses{composedReapTable: table, identity: func(pid int) string {
					if pid == 52 {
						return "new-root"
					}
					return table.Identity(pid)
				}}
			}
			req.OrphanSignal = "KILL"
			second := rig.s.handleLegacyOrphanSignal(t.Context(), req)
			if second.LegacySignal == nil || second.LegacySignal.Status != "refused" || len(table.signals[53]) != 1 || table.signals[53][0] != syscall.SIGTERM {
				t.Fatalf("%s KILL %+v signals%v", change, second, table.signals)
			}
		})
	}
}

func TestLegacySignalLostAcknowledgmentNeverRetriesOrEscalates(t *testing.T) {
	for _, family := range []string{"orphan-nvim", "left-running"} {
		t.Run(family, func(t *testing.T) {
			rig, rt, table := lateLegacyWorld(t)
			table.diesOn = nil // an acknowledged signal cannot be inferred from absence
			calls := 0
			call := func(ctx context.Context, _ string, raw, out any) error {
				req := raw.(couchmessage.Request)
				response := rig.s.handle(ctx, req)
				if req.Op == "reap-legacy-signal" && req.LegacyTarget.Family == family {
					calls++
					if response.LegacySignal == nil || response.LegacySignal.Status != "sent" {
						t.Fatalf("native effect was not performed: %+v", response)
					}
					return errors.New("reply lost after supervisor applied signal")
				}
				*out.(*couchmessage.Response) = response
				return nil
			}
			var stdout, stderr bytes.Buffer
			code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true, jsonOutput: true}, rt, &stdout, &stderr, call, fastOrphanReaper(table))
			pid := 51
			if family == "left-running" {
				pid = 52
			}
			if calls != 1 || len(table.signals[pid]) != 1 || table.signals[pid][0] != syscall.SIGTERM {
				t.Fatalf("uncertain effect retried/escalated calls%d signals%v", calls, table.signals)
			}
			var report orphanReapReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatalf("invalid report %v stdout%s stderr%s", err, stdout.String(), stderr.String())
			}
			failed := false
			if family == "orphan-nvim" {
				for _, outcome := range report.Outcomes {
					if outcome.Row.Key.PID == pid && outcome.Status == orphanFailed {
						failed = true
					}
				}
			} else {
				for _, outcome := range report.LeftRunning {
					if outcome.Entry.Process.PID == pid && outcome.Status == orphanFailed {
						failed = true
					}
				}
			}
			if code != 1 || !failed {
				t.Fatalf("uncertain outcome lost: exit%d report%+v", code, report)
			}
		})
	}
}
