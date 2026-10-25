package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// BenchRetirement is what bench-retire left: the archived thread, or a slot
// that was already vacant.
type BenchRetirement struct {
	Archived ThreadAddress
	Vacant   bool
}

// RetireBenchSlot retires a bench slot's thread (pair#847): a live agent is
// parked, the tree is torn down and proven gone, and the record is archived
// only when no process any of the slot's agents started survives. The worktree
// stays. A refusal leaves the session stopped and the record kept, and a retry
// refuses again while a survivor lives.
func (c *Couch) RetireBenchSlot(ctx context.Context, t RebootTarget) (BenchRetirement, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.Threads == nil {
		return BenchRetirement{}, errors.New("bench-retire requires a thread store")
	}
	if (t.Path == "") == (t.Address == ThreadAddress{}) {
		return BenchRetirement{}, errors.New("bench-retire needs exactly one slot path or thread address")
	}
	address, vacant, err := c.parkBenchSlot(ctx, t, "bench-retire")
	if err != nil {
		return BenchRetirement{}, err
	}
	if vacant {
		scope, err := launcher.ResolveRepoScope(t.Path)
		if err != nil {
			return BenchRetirement{}, err
		}
		if err := c.refuseBenchSurvivors(scope.Key, nil); err != nil {
			return BenchRetirement{}, err
		}
		return BenchRetirement{Vacant: true}, nil
	}
	r, err := c.prepareRetirement(ctx, address)
	if err != nil {
		return BenchRetirement{}, err
	}
	if r.Unreadable {
		return BenchRetirement{}, fmt.Errorf("bench-retire %s: the record is unreadable; archive it from the switcher", address.Tag)
	}
	var proof TeardownProof
	if !r.RolledBack {
		if proof, err = c.Teardown(ctx, r.Record, "bench-retire"); err != nil {
			return BenchRetirement{}, fmt.Errorf("bench-retire %s: %w", address.Tag, err)
		}
	}
	if err := c.refuseBenchSurvivors(address.RepoScope, proof.LeftRunning); err != nil {
		return BenchRetirement{}, err
	}
	if !r.RolledBack {
		if err := c.Threads.ArchiveThreadExpected(address, r.Revision, proof); err != nil {
			return BenchRetirement{}, err
		}
	}
	return BenchRetirement{Archived: address}, nil
}

// parkBenchSlot resolves a bench target to its thread and parks a live agent,
// the graceful stop both bench verbs share; vacant reports a slot path with
// no thread. A parked thread then needs no stop set from Reboot or archive.
func (c *Couch) parkBenchSlot(ctx context.Context, t RebootTarget, verb string) (ThreadAddress, bool, error) {
	address := t.Address
	if t.Path != "" {
		_, record, err := c.recoveryTarget(ctx, ResumeTarget{Path: t.Path})
		if errors.Is(err, ErrThreadNotFound) {
			return ThreadAddress{}, true, nil
		}
		if err != nil {
			return ThreadAddress{}, false, err
		}
		address = record.Address
	}
	state, _, err := c.classifyForAction(ctx, address)
	if err != nil {
		return address, false, fmt.Errorf("%s %s: %w", verb, address.Tag, err)
	}
	if state == ThreadLive {
		if _, err := c.Park(ctx, address, "normal"); err != nil {
			return address, false, fmt.Errorf("%s %s: park: %w", verb, address.Tag, err)
		}
	}
	return address, false, nil
}

// benchSurvivors are the processes a slot's agents started that may still
// run: this teardown's LeftRunning and every registered orphan of the slot's
// scope, from any generation. Teardown names LeftRunning only while the agent
// is still listed, so the registry is the record once it is gone. Only a
// process proven dead is excluded; unknown liveness counts.
func (c *Couch) benchSurvivors(scope string, left []LeftRunningOrphan) ([]LeftRunningOrphan, error) {
	registered, err := NewLeftRunningRegistry(c.Namespace).Read()
	if err != nil {
		return nil, err
	}
	seen := map[ProcessIdentity]bool{}
	var out []LeftRunningOrphan
	for _, orphan := range append(append([]LeftRunningOrphan{}, left...), registered...) {
		if orphan.Thread.RepoScope != scope || seen[orphan.Process] {
			continue
		}
		seen[orphan.Process] = true
		if observeExactProcessOrUnknown(c.Proc, orphan.Process) != Dead {
			out = append(out, orphan)
		}
	}
	return out, nil
}

func (c *Couch) refuseBenchSurvivors(scope string, left []LeftRunningOrphan) error {
	survivors, err := c.benchSurvivors(scope, left)
	if err != nil {
		return fmt.Errorf("bench-retire: left-running processes could not be read: %w", err)
	}
	if len(survivors) == 0 {
		return nil
	}
	named := make([]string, 0, len(survivors))
	for _, s := range survivors {
		named = append(named, fmt.Sprintf("pid %d (%s)", s.Process.PID, s.Command))
	}
	return &SlotOperationError{Code: SlotOpBenchSurvivors, Detail: "still running: " + strings.Join(named, ", ") + "; stop them, then retry"}
}
