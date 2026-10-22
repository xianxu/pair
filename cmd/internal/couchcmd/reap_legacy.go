package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// legacyReapPlan owns one immutable forest census. A process belongs to one
// root even when the two legacy families overlap; no effect discovers children.
type legacyReapPlan struct {
	targets []couchcore.LegacyOrphanTarget
	nvim    []orphanReapOutcome
	left    []leftRunningReapOutcome
	owners  map[couchcore.ResourceProcessKey]couchcore.LegacyOrphanTarget
	visits  int
}

func freezeLegacyReap(ctx context.Context, reaper launcher.Reaper, nvims []couchcore.OrphanNvim, left []couchcore.LeftRunningOrphan) legacyReapPlan {
	p := legacyReapPlan{owners: map[couchcore.ResourceProcessKey]couchcore.LegacyOrphanTarget{}}
	var roots []couchcore.LegacyOrphanTarget
	for _, v := range nvims {
		p.nvim = append(p.nvim, orphanReapOutcome{Row: v, Status: orphanSkipped, Detail: "legacy root absent or identity changed before freeze"})
		roots = append(roots, couchcore.LegacyOrphanTarget{Family: "orphan-nvim", Root: v.Key, Key: v.Key})
	}
	for _, v := range left {
		k := couchcore.ResourceProcessKey{PID: v.Process.PID, Identity: v.Process.Identity}
		p.left = append(p.left, leftRunningReapOutcome{Entry: v, Status: orphanSkipped, Detail: "legacy root absent or identity changed before freeze"})
		roots = append(roots, couchcore.LegacyOrphanTarget{Family: "left-running", Root: k, Key: k})
	}
	if len(roots) == 0 {
		return p
	}
	refuse := func(status, detail string) legacyReapPlan {
		for i := range p.nvim {
			p.nvim[i].Status = status
			p.nvim[i].Detail = detail
		}
		for i := range p.left {
			p.left[i].Status = status
			p.left[i].Detail = detail
		}
		return p
	}
	for _, root := range roots {
		if root.Root.PID <= 0 || root.Root.Identity == "" {
			return refuse(orphanRefused, "legacy root identity unavailable")
		}
	}
	if reaper.Table == nil {
		return refuse(orphanFailed, "legacy census unavailable")
	}
	rows, err := reaper.Table.Snapshot(ctx)
	if err != nil {
		return refuse(orphanFailed, "legacy census: "+err.Error())
	}
	if ctx.Err() != nil {
		return refuse(orphanFailed, ctx.Err().Error())
	}
	if len(rows) > couchcore.ResourceMaxTableRows || len(roots) > couchcore.ResourceMaxTableRows {
		return refuse(orphanRefused, "legacy census exceeds bound")
	}
	byPID := map[int]launcher.ProcessRow{}
	children := map[int][]int{}
	for _, v := range rows {
		p.visits++
		if v.PID <= 0 || v.Identity == "" {
			return refuse(orphanRefused, "legacy census has unknown process identity")
		}
		if _, ok := byPID[v.PID]; ok {
			return refuse(orphanRefused, "legacy census has duplicate PID")
		}
		byPID[v.PID] = v
		children[v.PPID] = append(children[v.PPID], v.PID)
	}
	sort.Slice(roots, func(i, j int) bool {
		a, b := roots[i], roots[j]
		if a.Root.PID != b.Root.PID {
			return a.Root.PID < b.Root.PID
		}
		if a.Root.Identity != b.Root.Identity {
			return a.Root.Identity < b.Root.Identity
		}
		return a.Family < b.Family
	})
	type pending struct {
		pid   int
		owner couchcore.LegacyOrphanTarget
	}
	queue := make([]pending, 0, len(rows))
	assigned := map[int]bool{}
	for _, root := range roots {
		row, ok := byPID[root.Root.PID]
		if !ok || row.Identity != root.Root.Identity || root.Root.Identity == "" {
			continue
		}
		if assigned[row.PID] {
			continue
		}
		assigned[row.PID] = true
		p.owners[root.Root] = root
		queue = append(queue, pending{row.PID, root})
	}
	for at := 0; at < len(queue); at++ {
		item := queue[at]
		p.visits++
		row := byPID[item.pid]
		target := item.owner
		target.Key = couchcore.ResourceProcessKey{PID: row.PID, Identity: row.Identity}
		p.targets = append(p.targets, target)
		for _, pid := range children[item.pid] {
			if assigned[pid] {
				continue
			}
			assigned[pid] = true
			queue = append(queue, pending{pid, item.owner})
		}
	}
	// Reverse breadth-first order ends children before their assigned roots.
	for i, j := 0, len(p.targets)-1; i < j; i, j = i+1, j-1 {
		p.targets[i], p.targets[j] = p.targets[j], p.targets[i]
	}
	return p
}

func reapFrozenLegacy(ctx context.Context, socket string, caller couchmessage.Request, call messageCall, reaper launcher.Reaper, p legacyReapPlan) ([]orphanReapOutcome, []leftRunningReapOutcome) {
	steps := make([]launcher.ReapStep, 0, len(p.targets))
	byKey := map[couchcore.ResourceProcessKey]couchcore.LegacyOrphanTarget{}
	for _, target := range p.targets {
		steps = append(steps, launcher.ReapStep{PID: target.Key.PID, Identity: target.Key.Identity})
		byKey[target.Key] = target
	}
	signal := func(ctx context.Context, step launcher.ReapStep, sig syscall.Signal) error {
		target := byKey[couchcore.ResourceProcessKey{PID: step.PID, Identity: step.Identity}]
		req := couchmessage.Request{Op: "reap-legacy-signal", Confirmed: true, Scope: caller.Scope, Tag: caller.Tag, Session: caller.Session, Nonce: caller.Nonce, LegacyTarget: &target, OrphanSignal: "TERM"}
		if sig == syscall.SIGKILL {
			req.OrphanSignal = "KILL"
		}
		var reply couchmessage.Response
		if err := call(ctx, socket, req, &reply); err != nil {
			return fmt.Errorf("legacy signal outcome uncertain: %w", err)
		}
		if reply.Code != "ok" || reply.LegacySignal == nil {
			return errors.New("legacy signal unsupported or unavailable")
		}
		if !reflect.DeepEqual(reply.LegacySignal.Target, target) {
			return errors.New("legacy signal witness differs")
		}
		switch reply.LegacySignal.Status {
		case "sent":
			return nil
		case "skipped":
			return fmt.Errorf("%w: %s", launcher.ErrServerChanged, reply.LegacySignal.Detail)
		case "refused":
			return fmt.Errorf("%w: %s", launcher.ErrExactReapRefused, reply.LegacySignal.Detail)
		default:
			return fmt.Errorf("legacy signal outcome uncertain: %s", reply.LegacySignal.Detail)
		}
	}
	type aggregate struct{ status, detail string }
	grouped := map[couchcore.ResourceProcessKey]aggregate{}
	rank := func(status string) int {
		switch status {
		case orphanFailed:
			return 4
		case orphanRefused:
			return 3
		case orphanReaped:
			return 2
		default:
			return 1
		}
	}
	for _, outcome := range reaper.ReapExact(ctx, steps, signal) {
		target := byKey[couchcore.ResourceProcessKey{PID: outcome.Step.PID, Identity: outcome.Step.Identity}]
		old := grouped[target.Root]
		if rank(outcome.Status) > rank(old.status) || old.status == "" {
			grouped[target.Root] = aggregate{outcome.Status, outcome.Detail}
		}
	}
	for i := range p.nvim {
		if owner, ok := p.owners[p.nvim[i].Row.Key]; ok {
			if a, ok := grouped[owner.Root]; ok {
				p.nvim[i].Status, p.nvim[i].Detail = a.status, a.detail
			}
		}
	}
	for i := range p.left {
		v := p.left[i].Entry.Process
		k := couchcore.ResourceProcessKey{PID: v.PID, Identity: v.Identity}
		if owner, ok := p.owners[k]; ok {
			if a, ok := grouped[owner.Root]; ok {
				p.left[i].Status, p.left[i].Detail = a.status, a.detail
			}
		}
	}
	return p.nvim, p.left
}
