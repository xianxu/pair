package couchcore

import (
	"context"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// OrphanReaper ends one orphaned server's whole process tree and its zellij
// session record (#399). The launcher owns the real one; tests use a fake.
type OrphanReaper interface {
	ReapOrphan(ctx context.Context, server launcher.SessionServerIdentity, scope, tag string) error
}

// ReapTarget names the thread to reap: a slot by its host checkout, or a
// thread by its exact address -- the same two forms reboot takes.
type ReapTarget struct {
	Path    string
	Address ThreadAddress
}

// ReapResult is the thread whose orphaned server was ended.
type ReapResult struct {
	Address ThreadAddress
	Server  launcher.SessionServerIdentity
}

// ReapRefusal is a reap that did nothing, and why.
type ReapRefusal struct{ Detail string }

func (r *ReapRefusal) Error() string { return "reap refused: " + r.Detail }

// reapConfirmInterval separates the two observations a reap requires. One
// snapshot is provisional: a server that is starting is in `ps` before its
// socket exists, and reads orphaned for that moment (#399 M1 review).
const reapConfirmInterval = time.Second

// Reap ends an orphaned thread's server tree so the thread can be resumed.
//
// ORDER (ARCH-ORDER): everything that can refuse runs before the first signal --
// the row must be orphaned now, and must STILL be orphaned, with the same server
// identity, after reapConfirmInterval. Only then does the reaper signal, and it
// re-reads every pid's identity before each signal itself.
func (c *Couch) Reap(ctx context.Context, target ReapTarget) (ReapResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return ReapResult{}, err
	}
	row, ok := reapRow(rows, target)
	if !ok {
		return ReapResult{}, &ReapRefusal{Detail: "no Couch thread stands for that target"}
	}
	// Any row carrying an orphan, a live one included: Couch may still host
	// the client of a server nothing else can reach (#399).
	if row.Orphan == nil {
		return ReapResult{}, &ReapRefusal{Detail: fmt.Sprintf("thread %s is %s, not an orphaned server", row.Address.Tag, rowStateWord(row))}
	}
	server := *row.Orphan
	sleep := c.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	sleep(reapConfirmInterval)
	presence, ok := c.Artifacts.(SessionPresenceResolver)
	if !ok {
		return ReapResult{}, &ReapRefusal{Detail: "session presence cannot be observed"}
	}
	again, err := presence.SessionPresence(ctx, []ThreadAddress{row.Address})
	if err != nil {
		return ReapResult{}, &ReapRefusal{Detail: "the orphan could not be observed again: " + err.Error()}
	}
	if o := again[row.Address]; o.State != SessionOrphaned || o.Orphan == nil || *o.Orphan != server {
		return ReapResult{}, &ReapRefusal{Detail: fmt.Sprintf("server PID %d did not stay orphaned (now %s); read the report again", server.PID, o.State)}
	}
	if err := c.reaper().ReapOrphan(ctx, server, row.Address.RepoScope, string(row.Address.Tag)); err != nil {
		return ReapResult{}, err
	}
	return ReapResult{Address: row.Address, Server: server}, nil
}

func (c *Couch) reaper() OrphanReaper {
	if c.Reaper != nil {
		return c.Reaper
	}
	// The thread's helpers live under the Pair data directory the artifacts
	// controller already knows; without it reap refuses rather than leave them.
	if dir, ok := c.Artifacts.(interface{ PairLifecycleDataDir() string }); ok {
		return launcher.OSOrphanReaper{DataDir: dir.PairLifecycleDataDir()}
	}
	return launcher.OSOrphanReaper{}
}

func reapRow(rows []ActionableThreadSummary, target ReapTarget) (ActionableThreadSummary, bool) {
	for _, row := range rows {
		if target.Path != "" && row.Target.Kind == ThreadTargetSlot && row.Target.Slot.WorktreeRoot == target.Path {
			return row, true
		}
		if target.Path == "" && row.Address == target.Address {
			return row, true
		}
	}
	return ActionableThreadSummary{}, false
}

func rowStateWord(row ActionableThreadSummary) string {
	if row.Reason != "" {
		return string(row.State) + "/" + string(row.Reason)
	}
	return string(row.State)
}
