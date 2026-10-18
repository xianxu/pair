package couchtty

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// LifecycleObservation carries completion evidence, never a new lifecycle
// authority. The server enriches/routes it and owns any resulting message.
type LifecycleObservation struct {
	Address                                        couchcore.ThreadAddress
	Target                                         couchcore.ThreadTarget
	Kind, Path, Incarnation, Operation, Initiation string
	Agent, Model, Branch, Issue, Mode              string
	Observation                                    uint64
	Failures                                       int
}

// SetLifecycleObserver is configured before Run. Callbacks run outside Console
// locks and must only copy bounded facts, never perform IO.
func (c *Console) SetLifecycleObserver(fn func(LifecycleObservation)) {
	c.mu.Lock()
	c.lifecycleObserver = fn
	armed := c.menu.Reattach.Phase != ReattachIdle
	c.mu.Unlock()
	if armed {
		c.observeLifecycleArm()
	}
}

func (c *Console) observeLifecycleArm() { c.emitLifecycle(LifecycleObservation{Kind: "restart-armed"}) }

func (c *Console) emitLifecycle(e LifecycleObservation) {
	c.mu.Lock()
	fn := c.lifecycleObserver
	if fn != nil {
		c.lifecycleSequence++
		e.Observation = c.lifecycleSequence
	}
	c.mu.Unlock()
	if fn != nil {
		fn(e)
	}
}

// ObserveNativeExit emits a confirmed native agent exit (pair#657). Couch
// calls it only after verifying the wrapper's report against the current
// launch; the Console adds slot metadata and the per-slot observation order.
// target is the caller's verified slot; it stands in when the Console has no
// row for the slot, so the fact is never dropped for missing metadata.
func (c *Console) ObserveNativeExit(address couchcore.ThreadAddress, target couchcore.ThreadTarget, incarnation string, exitCode int) {
	e := c.lifecycleMetadata(address, target.Slot.WorktreeRoot)
	if e.Target.Kind == "" {
		e.Target = target
	}
	e.Kind, e.Incarnation, e.Mode = "exited", incarnation, fmt.Sprintf("code %d", exitCode)
	c.emitLifecycle(e)
}

func (c *Console) lifecycleMetadata(address couchcore.ThreadAddress, path string) LifecycleObservation {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := LifecycleObservation{Address: address, Path: path}
	for _, row := range menuRows(c.menu) {
		if !(address != (couchcore.ThreadAddress{}) && row.Address == address) && !(path != "" && (row.Target.Slot.WorktreeRoot == path || row.WorkingPath == path)) {
			continue
		}
		e.Target, e.Agent = row.Target, row.Agent
		if e.Path == "" {
			e.Path = row.WorkingPath
		}
		if status, ok := c.menu.SlotGit[row.Target.Slot.WorktreeRoot]; ok {
			e.Branch = status.Branch
			if len(status.Branch) >= 6 {
				if issue, err := strconv.Atoi(status.Branch[:6]); err == nil {
					e.Issue = strconv.Itoa(issue)
				}
			}
		}
		break
	}
	return e
}

func (c *Console) observeLifecycleCompletion(completed operationCompletion, address couchcore.ThreadAddress, err error) {
	var e LifecycleObservation
	if started, ok := completed.value.(couchcore.StartedChild); ok && err == nil {
		if child, exists := started.Started(); exists && child.Record.Thread != (couchcore.ThreadAddress{}) {
			r := child.Record
			e = c.lifecycleMetadata(r.Thread, string(r.Args.Worktree))
			if r.Args.Stack != "" {
				e.Agent = r.Args.Stack
			}
			e.Incarnation = fmt.Sprintf("%d:%s", r.PID, r.Identity)
			if r.Args.Issue != "" {
				e.Issue = r.Args.Issue
			}
			e.Kind = "started"
			switch completed.origin.Operation {
			case "relaunch":
				e.Kind = "relaunched"
			case "resume":
				e.Kind = "resumed"
			case "continue-thread":
				e.Kind, e.Mode = "recovered", "Continue"
			case "reboot":
				e.Kind, e.Mode = "recovered", "Fresh"
			}
			if completed.origin.Background {
				e.Kind, e.Mode = "recovered", "Continue"
			}
			if recovery, ok := completed.value.(couchcore.RecoveryResumeResult); ok {
				e.Kind, e.Mode = "recovered", ""
				switch recovery.Mode {
				case couchcore.Continue:
					e.Mode = "Continue"
				case couchcore.ResumeDisk:
					e.Mode = "Resume"
				case couchcore.Fresh:
					e.Mode = "Fresh"
				}
			}
			// A confirmed replacement is a new usability episode even if it
			// becomes unusable before the next live inventory reaches Console.
			c.mu.Lock()
			delete(c.lifecycleNeeds, lifecycleEvidenceKey(e.Address, e.Target))
			c.mu.Unlock()
		}
	}
	if parked, ok := completed.value.(couchcore.ParkResult); ok && err == nil && parked.Thread.VerifiedPark != nil {
		r := parked.Thread
		e = c.lifecycleMetadata(r.Address, r.WorkingPath)
		e.Kind = "parked"
		e.Incarnation = fmt.Sprintf("park:%d", r.Revision)
		if r.LatestLaunchProfile != nil {
			e.Agent = r.LatestLaunchProfile.Agent
		}
	}
	if result, ok := completed.value.(couchcore.RelaunchResult); ok && result.Outcome == couchcore.ParkedNotResumed {
		path := ""
		if completed.remote != nil {
			path = completed.remote.call.Args["path"]
		}
		e = c.lifecycleMetadata(address, path)
		e.Kind, e.Incarnation = "parked", "relaunch-parked"
	}
	if e.Kind == "" {
		return
	}
	e.Operation = completed.origin.Operation
	e.Initiation = "unknown"
	switch {
	case completed.origin.Background:
		e.Initiation = "automatic"
	case completed.remote != nil:
		e.Initiation = "peer"
	case completed.origin.Attempt != 0:
		e.Initiation = "operator"
	}
	c.emitLifecycle(e)
}

func (c *Console) observeLifecyclePass() {
	c.mu.Lock()
	if c.lifecycleObserver == nil || c.lifecyclePassDone || c.menu.Reattach.Phase != ReattachDone {
		c.mu.Unlock()
		return
	}
	c.lifecyclePassDone = true
	failures := len(c.menu.Reattach.Failed)
	c.mu.Unlock()
	c.emitLifecycle(LifecycleObservation{Kind: "restart-drained", Failures: failures})
}

func (c *Console) observeLifecycleInventory(rows []couchcore.ActionableThreadSummary) {
	for _, row := range rows {
		key := lifecycleEvidenceKey(row.Address, row.Target)
		c.mu.Lock()
		if c.lifecycleNeeds == nil {
			c.lifecycleNeeds = map[string]bool{}
		}
		hosted := false
		for _, pane := range c.panes {
			if pane.thread == row.Address {
				hosted = true
				break
			}
		}
		needs := !hosted && row.Reason == couchcore.ReasonNeedsRelaunch
		if _, known := c.lifecycleNeeds[key]; !known && len(c.lifecycleNeeds) >= 128 {
			c.mu.Unlock()
			continue
		}
		previous := c.lifecycleNeeds[key]
		// Unknown evidence never clears a confirmed observation.
		if needs || row.State == couchcore.ThreadLive || row.State == couchcore.ThreadParked {
			c.lifecycleNeeds[key] = needs
		}
		c.mu.Unlock()
		if !needs || previous {
			continue
		}
		e := c.lifecycleMetadata(row.Address, row.WorkingPath)
		e.Target, e.Agent, e.Kind, e.Initiation = row.Target, row.Agent, "needs-relaunch", "unknown"
		e.Incarnation = strings.Join([]string{row.Address.RepoScope, string(row.Address.Tag), "needs-relaunch"}, ":")
		c.emitLifecycle(e)
	}
}

func lifecycleEvidenceKey(address couchcore.ThreadAddress, target couchcore.ThreadTarget) string {
	if target.Kind == couchcore.ThreadTargetSlot {
		return target.Slot.WorktreeRoot
	}
	return address.RepoScope + ":" + string(address.Tag)
}
