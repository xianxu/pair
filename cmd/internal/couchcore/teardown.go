package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/lifetimelease"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Teardown's bounds (pair#695 P1, ARCH-CONSTRAINTS): the whole teardown, the
// sweep rounds, and how long one exact stop waits after SIGTERM and SIGKILL.
var (
	teardownBudget    = 10 * time.Second
	teardownRounds    = 3
	teardownTermWait  = 3 * time.Second
	teardownKillWait  = 2 * time.Second
	teardownPollEvery = 20 * time.Millisecond
)

// AgentTerminal is the agent process the wrapper published as ready, and its
// controlling terminal ("" from a wrapper launched outside Couch or before
// the field).
type AgentTerminal struct {
	Process ProcessIdentity
	TTY     string
}

// AgentTerminalReader reads a thread's published agent; ok is false when
// none is published.
type AgentTerminalReader interface {
	AgentTerminal(ctx context.Context, address ThreadAddress, agent string) (AgentTerminal, bool, error)
}

// LeftRunningOrphan is a process the agent started that had left the agent's
// terminal when the slot was torn down: named and registered, never killed by
// teardown, so it stays attributable after the thread is gone.
type LeftRunningOrphan struct {
	Origin    string          `json:"origin"` // the lifecycle address key
	Thread    ThreadAddress   `json:"thread"`
	Process   ProcessIdentity `json:"process"`
	Command   string          `json:"command"`
	LeftAt    time.Time       `json:"left_at"`
	Operation string          `json:"operation,omitempty"`
	// Unknown and processUnknown keep a newer build's members through a
	// rewrite of the orphan registry.
	Unknown        map[string]json.RawMessage `json:"-"`
	processUnknown map[string]json.RawMessage
}

// TeardownProof says the thread's tree is gone: the lease is free, nothing
// holds the agent's terminal, the published agent is dead, and the session is
// absent. It is the RetireAuthority for exactly the record teardown left
// (retire.go), so the fields that make it one are unexported: only teardown
// mints a proof, and a fixture uses TestOnlyTeardownProof.
type TeardownProof struct {
	address  LifecycleAddress
	thread   ThreadAddress
	revision uint64
	provenAt time.Time

	AgentTTY    string
	Sessions    []string
	LeftRunning []LeftRunningOrphan
	Notes       []string
}

// Address is the lifecycle address the proof covers.
func (p TeardownProof) Address() LifecycleAddress { return p.address }

// Thread is the record the proof covers.
func (p TeardownProof) Thread() ThreadAddress { return p.thread }

// Revision is the record's revision as teardown left it.
func (p TeardownProof) Revision() uint64 { return p.revision }

// TestOnlyTeardownProof mints a proof for a record a test arranged through
// the store directly, whose tree was never up. Production never calls it
// (TestOnlyTeardownProofHasNoProductionCaller).
func TestOnlyTeardownProof(record ThreadRecord, at time.Time) (TeardownProof, error) {
	address, err := threadLifecycleAddress(record)
	if err != nil {
		return TeardownProof{}, err
	}
	return TeardownProof{address: address, thread: record.Address, revision: record.Revision, provenAt: at}, nil
}

// teardownMode is what one teardown does: stop and fold (Teardown), prove
// only (ProveDown), or prove and fold without stopping (Fresh recovery,
// whose stops the operator already confirmed).
type teardownMode uint8

const (
	teardownStopAndFold teardownMode = iota
	teardownProveOnly
	teardownProveAndFold
)

func (m teardownMode) stops() bool { return m == teardownStopAndFold }
func (m teardownMode) folds() bool { return m != teardownProveOnly }

// ErrTeardownUnproven: a step could not reach its proof. Nothing after it
// ran. Holders are pair processes and processes on the agent's terminal only;
// an agent's detached daemon never appears here.
type ErrTeardownUnproven struct {
	Phase   string
	Holders []ProcessIdentityLine
	Cause   error
}

func (e *ErrTeardownUnproven) Error() string {
	msg := "teardown not proven at " + e.Phase
	if len(e.Holders) > 0 {
		msg += fmt.Sprintf(" (%d holder(s), first pid %d)", len(e.Holders), e.Holders[0].Process.PID)
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}
func (e *ErrTeardownUnproven) Unwrap() error                         { return e.Cause }
func (e *ErrTeardownUnproven) FailurePhase() string                  { return e.Phase }
func (e *ErrTeardownUnproven) FailureHolders() []ProcessIdentityLine { return e.Holders }
func (e *ErrTeardownUnproven) FailureOutcome() string                { return "unproven" }

var _ LifecycleFailureDetails = (*ErrTeardownUnproven)(nil)

func unproven(phase string, cause error, holders ...ProcessIdentityLine) error {
	return &ErrTeardownUnproven{Phase: phase, Holders: holders, Cause: cause}
}

// teardownTarget is what Teardown stops and proves for one thread's tree.
type teardownTarget struct {
	address LifecycleAddress
	thread  ThreadRecord
	lease   string
	session string
	agent   *AgentTerminal
}

// Teardown stops everything of the thread's slot tree except the transcript
// and proves it gone (pair#695 P1). It is idempotent: on a slot already down
// it only proves. Agent-started processes that left the agent's terminal are
// named in LeftRunning, never killed or waited on.
func (c *Couch) Teardown(ctx context.Context, thread ThreadRecord, operation string) (TeardownProof, error) {
	return c.teardown(ctx, thread, operation, teardownStopAndFold)
}

// ProveDown proves the slot tree gone without stopping anything: a caller
// whose contract is "start only on a down slot" names a survivor instead of
// killing it.
func (c *Couch) ProveDown(ctx context.Context, thread ThreadRecord) (TeardownProof, error) {
	return c.teardown(ctx, thread, "", teardownProveOnly)
}

// proveDownAndFold proves the tree gone without stopping anything and then
// folds the bookkeeping its proven-dead owners left: what a caller whose
// stops were already confirmed (Fresh recovery) retires with.
func (c *Couch) proveDownAndFold(ctx context.Context, thread ThreadRecord, operation string) (TeardownProof, error) {
	return c.teardown(ctx, thread, operation, teardownProveAndFold)
}

func (c *Couch) teardown(ctx context.Context, thread ThreadRecord, operation string, mode teardownMode) (TeardownProof, error) {
	if c.Proc == nil || c.Clock == nil {
		return TeardownProof{}, unproven("observe", errors.New("teardown requires process and clock seams"))
	}
	ctx, cancel := context.WithTimeout(ctx, teardownBudget)
	defer cancel()
	target, err := c.teardownTarget(ctx, thread)
	if err != nil {
		return TeardownProof{}, unproven("observe", err)
	}
	table, err := c.teardownTable()
	if err != nil {
		return TeardownProof{}, unproven("observe", err)
	}
	tty := ""
	var left []LeftRunningOrphan
	var notes []string
	if target.agent != nil {
		// The published terminal is evidence only while the tree that
		// published it is up. Once the agent is gone and the lease free,
		// the device may already belong to an unrelated terminal, and
		// sweeping it would kill processes no thread owns.
		free, err := lifetimelease.Free(target.lease)
		if err != nil {
			return TeardownProof{}, unproven("observe", err)
		}
		if agentListed(table, target.agent.Process) || !free {
			tty = target.agent.TTY
			left = leftRunning(table, target, c.Clock.Now(), operation)
		} else if target.agent.TTY != "" {
			notes = append(notes, "the agent's terminal was not checked: the tree was already down and the device may be reused")
		}
	}
	if mode.stops() {
		if err := c.teardownStop(ctx, target); err != nil {
			return TeardownProof{}, err
		}
		for round := 0; round < teardownRounds; round++ {
			holders, err := c.teardownHolders(ctx, target, tty)
			if err != nil {
				return TeardownProof{}, unproven("sweep", err)
			}
			if len(holders) == 0 {
				break
			}
			for _, holder := range holders {
				if err := c.stopExact(ctx, holder.Process); err != nil {
					return TeardownProof{}, unproven("sweep", err, c.survivors(holders)...)
				}
			}
		}
	}
	proof, err := c.proveDown(ctx, target, tty)
	if err != nil {
		return TeardownProof{}, err
	}
	if mode.folds() {
		// Every owner is proven gone, which is what makes the record's
		// unfinished bookkeeping foldable.
		if target.thread, err = c.teardownPreserve(target.thread); err != nil {
			return TeardownProof{}, unproven("preserve", err)
		}
	}
	proof.thread, proof.revision = target.thread.Address, target.thread.Revision
	proof.LeftRunning = left
	if len(left) > 0 {
		// Registered so they stay attributable once the thread is gone;
		// the proof stands either way, the note says what was not kept.
		if err := NewLeftRunningRegistry(c.Namespace).Add(ctx, left); err != nil {
			notes = append(notes, "left-running processes not registered: "+err.Error())
		}
	}
	proof.Notes = append(notes, proof.Notes...)
	// The lease is proven free, and every caller retires the record on this
	// proof, so its file goes now, in every mode; a launch recreates it.
	if removed, err := lifetimelease.RemoveFree(target.lease); err != nil {
		proof.Notes = append(proof.Notes, "lifetime lease not removed: "+err.Error())
	} else if !removed {
		proof.Notes = append(proof.Notes, "lifetime lease held again before its removal; left in place")
	}
	return proof, nil
}

// teardownPreserve folds the record's unfinished bookkeeping once its owners
// are proven gone: a park whose helper is now dead is abandoned and a driverless
// start claim rolled back, by the same rules resume and archive use
// (clearLifecycleDebris). A record with nothing unfinished is returned as is,
// so the proof's revision is the one the caller already holds.
func (c *Couch) teardownPreserve(thread ThreadRecord) (ThreadRecord, error) {
	if archivableRecord(thread) == nil {
		return thread, nil
	}
	cleared, err := c.clearLifecycleDebris(thread)
	if err != nil {
		return ThreadRecord{}, err
	}
	if cleared == nil {
		return thread, nil
	}
	if err := archivableRecord(*cleared); err != nil {
		return ThreadRecord{}, err
	}
	return *cleared, nil
}

func (c *Couch) teardownTarget(ctx context.Context, thread ThreadRecord) (teardownTarget, error) {
	address, err := threadLifecycleAddress(thread)
	if err != nil {
		return teardownTarget{}, err
	}
	lease, err := c.lifetimeLease(address, thread.Address)
	if err != nil {
		return teardownTarget{}, err
	}
	target := teardownTarget{address: address, thread: thread, lease: lease, session: recordSessionName(thread)}
	if reader, ok := c.Artifacts.(AgentTerminalReader); ok && thread.LatestLaunchProfile != nil {
		agent, found, err := reader.AgentTerminal(ctx, thread.Address, thread.LatestLaunchProfile.Agent)
		if err != nil {
			return teardownTarget{}, fmt.Errorf("read the published agent: %w", err)
		}
		if found {
			target.agent = &agent
		}
	}
	return target, nil
}

func (c *Couch) teardownTable() ([]procutil.Process, error) {
	if c.ProcessTable == nil {
		return nil, errors.New("process table unavailable")
	}
	return c.ProcessTable()
}

// teardownStop quits the session (its server takes the wrapper and the
// agent's terminal down with it) and stops the registered wrapper.
func (c *Couch) teardownStop(ctx context.Context, target teardownTarget) error {
	if target.session != "" {
		quiescer, ok := c.Artifacts.(interface {
			QuiesceNamed(context.Context, ThreadAddress, string) error
		})
		if !ok {
			return unproven("stop", errors.New("session quiescence unavailable"))
		}
		if err := quiescer.QuiesceNamed(ctx, target.thread.Address, target.session); err != nil {
			// A listing that omits a live server is unknown, never absence
			// (#692): refuse, never retry.
			return unproven("stop", err)
		}
	}
	if c.SlotWrapper != nil {
		if wrapper, ok := c.SlotWrapper(target.thread.Address); ok {
			if err := c.stopExact(ctx, wrapper); err != nil {
				return unproven("stop", err, ProcessIdentityLine{Process: wrapper, Kind: "pair-wrap"})
			}
		}
	}
	return nil
}

// teardownHolders is what still holds the slot: the lease's holders and the
// processes on the agent's terminal, each by the identity one table read
// gives it.
func (c *Couch) teardownHolders(ctx context.Context, target teardownTarget, tty string) ([]ProcessIdentityLine, error) {
	table, err := c.teardownTable()
	if err != nil {
		return nil, err
	}
	rows := make(map[int]procutil.Process, len(table))
	for _, row := range table {
		rows[row.PID] = row
	}
	seen := map[int]bool{}
	var holders []ProcessIdentityLine
	add := func(row procutil.Process, kind string) {
		if seen[row.PID] || row.Zombie {
			return
		}
		seen[row.PID] = true
		holders = append(holders, ProcessIdentityLine{Process: ProcessIdentity{PID: row.PID, Identity: row.Identity}, Kind: kind})
	}
	if c.LeaseHolders != nil {
		pids, err := c.LeaseHolders(ctx, target.lease)
		if err != nil {
			return nil, err
		}
		for _, pid := range pids {
			row, ok := rows[pid]
			if !ok || row.Identity == "" {
				return nil, fmt.Errorf("lease holder pid %d has no readable identity", pid)
			}
			add(row, "pair")
		}
	}
	// The published agent is a witness by its exact identity, whatever its
	// terminal: a wrapper from before the tty field, or one whose table read
	// failed, published none, and the tty sweep alone would never see it.
	if target.agent != nil {
		if row, ok := rows[target.agent.Process.PID]; ok && row.Identity == target.agent.Process.Identity {
			add(row, "agent")
		}
	}
	for _, row := range procutil.TTYProcesses(table, tty) {
		if row.Identity == "" {
			return nil, fmt.Errorf("pid %d on the agent's terminal has no readable identity", row.PID)
		}
		add(row, "process")
	}
	sort.Slice(holders, func(i, j int) bool { return holders[i].Process.PID < holders[j].Process.PID })
	return holders, nil
}

// survivors is the holders not proven dead: what an unproven sweep names.
func (c *Couch) survivors(holders []ProcessIdentityLine) []ProcessIdentityLine {
	var out []ProcessIdentityLine
	for _, holder := range holders {
		if observeExactProcess(c.Proc, holder.Process) != Dead {
			out = append(out, holder)
		}
	}
	return out
}

// stopExact stops one process by exact identity: SIGTERM, then SIGKILL, each
// with its bounded wait. A reused pid never matches, so it is never signalled.
func (c *Couch) stopExact(ctx context.Context, process ProcessIdentity) error {
	for _, step := range []struct {
		signal os.Signal
		wait   time.Duration
	}{{TermSignal, teardownTermWait}, {syscall.SIGKILL, teardownKillWait}} {
		switch observeExactProcessOrUnknown(c.Proc, process) {
		case Dead:
			return nil
		case Unknown:
			return fmt.Errorf("pid %d cannot be observed", process.PID)
		}
		if err := c.Proc.Signal(process.PID, step.signal); err != nil && observeExactProcess(c.Proc, process) == Live {
			return fmt.Errorf("signal pid %d: %w", process.PID, err)
		}
		// Real waits on real signals: the wall clock, not c.Clock, which a
		// test may freeze.
		deadline := time.Now().Add(step.wait)
		for time.Now().Before(deadline) {
			if observeExactProcess(c.Proc, process) == Dead {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(teardownPollEvery):
			}
		}
	}
	if observeExactProcessOrUnknown(c.Proc, process) == Dead {
		return nil
	}
	return fmt.Errorf("pid %d survived SIGKILL", process.PID)
}

// proveDown checks every witness: the lease is free, nothing is on the
// agent's terminal, and the session is absent (quiescence observes it absent
// twice). Any failure is unproven with what still holds the slot.
func (c *Couch) proveDown(ctx context.Context, target teardownTarget, tty string) (TeardownProof, error) {
	free, err := lifetimelease.Free(target.lease)
	if err != nil {
		return TeardownProof{}, unproven("prove", err)
	}
	holders, err := c.teardownHolders(ctx, target, tty)
	if err != nil {
		return TeardownProof{}, unproven("prove", err)
	}
	if !free || len(holders) > 0 {
		cause := errors.New("the slot is still held")
		if !free && len(holders) == 0 {
			cause = errors.New("the lifetime lease is held by a process lsof cannot name")
		}
		return TeardownProof{}, unproven("prove", cause, holders...)
	}
	proof := TeardownProof{address: target.address, AgentTTY: tty}
	if target.session != "" {
		if err := c.sessionAbsent(ctx, target); err != nil {
			return TeardownProof{}, unproven("prove", err)
		}
		proof.Sessions = []string{target.session}
	}
	proof.provenAt = c.Clock.Now()
	return proof, nil
}

// sessionAbsent proves the recorded session gone. Pair-session observation
// answers present/absent; a listing that omits a live server (#692) is never
// read as absence.
func (c *Couch) sessionAbsent(ctx context.Context, target teardownTarget) error {
	observer, ok := c.Artifacts.(interface {
		PairSessionContext(context.Context, ThreadAddress) (PairSessionBinding, error)
	})
	if !ok {
		return errors.New("session observation unavailable")
	}
	for i := 0; i < 2; i++ {
		binding, err := observer.PairSessionContext(ctx, target.thread.Address)
		if errors.Is(err, launcher.ErrListingOmittedLiveServer) {
			return err
		}
		if err != nil && !errors.Is(err, ErrPairSessionBindingAbsent) {
			return err
		}
		if err == nil && binding.Present && binding.Name == target.session {
			return fmt.Errorf("session %s is still present", target.session)
		}
	}
	return nil
}

func agentListed(table []procutil.Process, agent ProcessIdentity) bool {
	for _, row := range table {
		if row.PID == agent.PID && row.Identity == agent.Identity && !row.Zombie {
			return true
		}
	}
	return false
}

// leftRunning is the agent's descendants that are off its terminal: a
// setsid'd runner, a daemon, a detached language server.
func leftRunning(table []procutil.Process, target teardownTarget, now time.Time, operation string) []LeftRunningOrphan {
	agent := target.agent
	children := map[int][]procutil.Process{}
	present := false
	for _, row := range table {
		if row.PID == agent.Process.PID && row.Identity == agent.Process.Identity {
			present = true
		}
		if row.PPID != row.PID {
			children[row.PPID] = append(children[row.PPID], row)
		}
	}
	if !present {
		return nil
	}
	var out []LeftRunningOrphan
	queue := []int{agent.Process.PID}
	seen := map[int]bool{agent.Process.PID: true}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		for _, child := range children[pid] {
			if seen[child.PID] || child.Zombie {
				continue
			}
			seen[child.PID] = true
			queue = append(queue, child.PID)
			if child.Identity == "" || (agent.TTY != "" && child.TTY == agent.TTY) {
				continue
			}
			out = append(out, LeftRunningOrphan{
				Origin: target.address.Key(), Thread: target.thread.Address,
				Process: ProcessIdentity{PID: child.PID, Identity: child.Identity},
				Command: child.Name, LeftAt: now, Operation: operation,
			})
		}
	}
	return out
}
