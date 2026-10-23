package couchcore

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// SlotWrapperLookup resolves a thread's wrapper from Couch's message registry:
// the pid and kernel start identity the wrapper registered with.
type SlotWrapperLookup func(ThreadAddress) (ProcessIdentity, bool)

const slotSessionRole = "zellij session "

// withSlotTree makes a Fresh recovery stop everything that belongs to the slot
// (pair#607): its registered live wrapper and that wrapper's descendants, each
// by exact pid and identity, and its own zellij session by exact recorded name
// when no other live slot names it. Terminal attribution may have failed (a
// legacy slot) or covered only part of this; the registry still knows the
// wrapper. Retirement refuses only when an exit cannot be confirmed.
func (c *Couch) withSlotTree(ctx context.Context, record ThreadRecord, r RecoveryRetirement) (RecoveryRetirement, []string, error) {
	if c.SlotWrapper == nil || c.ProcessTable == nil {
		return r, nil, nil
	}
	wrapper, ok := c.SlotWrapper(record.Address)
	if !ok {
		return r, nil, nil
	}
	switch observeExactProcessOrUnknown(c.Proc, wrapper) {
	case Dead:
		return r, nil, nil
	case Unknown:
		// An unobservable registered wrapper may still be running; starting
		// beside it is the orphan this exists to prevent.
		return r, nil, fmt.Errorf("slot wrapper pid %d cannot be observed, so its stop cannot be confirmed; nothing was stopped", wrapper.PID)
	}
	table, err := c.ProcessTable()
	if err != nil {
		return r, nil, fmt.Errorf("slot process tree unavailable; nothing was stopped: %w", err)
	}
	tree, unidentified := slotWrapperTree(table, wrapper)
	if len(tree) == 0 {
		return r, nil, errors.New("slot wrapper changed while reading its process tree; nothing was stopped")
	}
	listed := map[ProcessIdentity]bool{}
	for _, stop := range r.Stops {
		listed[ProcessIdentity{PID: stop.PID, Identity: stop.Identity}] = true
	}
	var added []RecoveryStopProcess
	for _, stop := range tree {
		if !listed[ProcessIdentity{PID: stop.PID, Identity: stop.Identity}] {
			added = append(added, stop)
		}
	}
	var notes []string
	if unidentified > 0 {
		notes = append(notes, fmt.Sprintf("%d wrapper descendant(s) without a readable identity were not stopped", unidentified))
	}
	session := recordSessionName(record)
	quiescer, canQuiesce := c.Artifacts.(interface {
		QuiesceNamed(context.Context, ThreadAddress, string) error
	})
	if session != "" && canQuiesce {
		if holder, err := c.otherLiveSessionHolder(ctx, record, session); err != nil {
			return r, nil, err
		} else if holder != "" {
			notes = append(notes, fmt.Sprintf("zellij session %s is also named by %s; left running", session, holder))
			session = ""
		} else {
			added = append(added, RecoveryStopProcess{Role: slotSessionRole + session})
		}
	} else {
		session = ""
	}
	if len(added) == 0 {
		return r, notes, nil
	}
	next := r
	next.Stops = append(append([]RecoveryStopProcess(nil), r.Stops...), added...)
	next.Digest = r.Digest + "+slot tree"
	check, retire := r.Check, r.Retire
	next.Check = func(ctx context.Context) error {
		if check != nil {
			if err := check(ctx); err != nil {
				return err
			}
		}
		for _, stop := range added {
			if stop.PID > 0 && observeExactProcessOrUnknown(c.Proc, ProcessIdentity{PID: stop.PID, Identity: stop.Identity}) == Unknown {
				return fmt.Errorf("slot %s pid %d can no longer be observed; refresh the recovery choice", stop.Role, stop.PID)
			}
		}
		return nil
	}
	next.Retire = func(ctx context.Context) error {
		if retire != nil {
			if err := retire(ctx); err != nil {
				return err
			}
		}
		// Live pane commands prove session ownership. Quiesce while the added
		// wrapper tree still carries that evidence; held panes lose it on exit.
		if session != "" {
			if err := quiescer.QuiesceNamed(ctx, record.Address, session); err != nil {
				return fmt.Errorf("slot zellij session %s kill not confirmed: %w", session, err)
			}
		}
		for _, stop := range added {
			if stop.PID > 0 && observeExactProcess(c.Proc, ProcessIdentity{PID: stop.PID, Identity: stop.Identity}) == Live {
				if err := c.Proc.Signal(stop.PID, TermSignal); err != nil && observeExactProcess(c.Proc, ProcessIdentity{PID: stop.PID, Identity: stop.Identity}) == Live {
					return fmt.Errorf("stop slot %s pid %d: %w", stop.Role, stop.PID, err)
				}
			}
		}
		for _, stop := range added {
			if stop.PID > 0 {
				if err := c.awaitExactProcessExit(ctx, ProcessIdentity{PID: stop.PID, Identity: stop.Identity}); err != nil {
					return fmt.Errorf("slot %s kill not confirmed: %w", stop.Role, err)
				}
			}
		}
		return nil
	}
	return next, notes, nil
}

// slotWrapperTree is the wrapper and its descendants in one process-table
// snapshot, each with the identity the table read, and how many live
// descendants had no readable identity; empty when the table no longer shows
// the exact registered wrapper.
func slotWrapperTree(table []procutil.Process, wrapper ProcessIdentity) ([]RecoveryStopProcess, int) {
	children := make(map[int][]procutil.Process, len(table))
	found := false
	for _, p := range table {
		if p.PID == wrapper.PID {
			found = p.Identity == wrapper.Identity
		}
		if p.PPID != p.PID {
			children[p.PPID] = append(children[p.PPID], p)
		}
	}
	if !found {
		return nil, 0
	}
	tree := []RecoveryStopProcess{{PID: wrapper.PID, Identity: wrapper.Identity, Role: "wrapper"}}
	seen := map[int]bool{wrapper.PID: true}
	queue := []int{wrapper.PID}
	unidentified := 0
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, child := range children[pid] {
			if seen[child.PID] || child.Zombie {
				continue
			}
			if child.Identity == "" {
				unidentified++
				continue
			}
			seen[child.PID] = true
			tree = append(tree, RecoveryStopProcess{PID: child.PID, Identity: child.Identity, Role: "wrapper child " + child.Name})
			queue = append(queue, child.PID)
		}
	}
	return tree, unidentified
}

// otherLiveSessionHolder names another positively live slot whose record names
// the same zellij session, or "" when this slot alone holds it.
func (c *Couch) otherLiveSessionHolder(ctx context.Context, record ThreadRecord, session string) (string, error) {
	if c.Threads == nil {
		return "", errors.New("thread registry unavailable")
	}
	snapshot, err := c.Threads.PreviewSnapshotContext(ctx)
	if err != nil {
		return "", err
	}
	for _, other := range snapshot.Records {
		if other.Address == record.Address || recordSessionName(other) != session || !recordPositivelyLive(c.Proc, other) {
			continue
		}
		if slot, err := c.recoverySlotAddress(ctx, other); err == nil {
			return slot, nil
		}
		return string(other.Address.Tag), nil
	}
	return "", nil
}
