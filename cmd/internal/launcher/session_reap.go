package launcher

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Reaping an orphaned server (#399) is a tree operation, and its order is the
// whole design. On 2026-10-06 killing the servers first let their descendants
// -- a `pair wrap` that ignored SIGTERM, `pair title` helpers -- reparent to
// PID 1, where nothing that asks "children of the server" finds them again.
// So: snapshot the tree once while the server still parents it, signal the
// descendants deepest first and the server last, escalate by pid (the snapshot
// remembers a reparented survivor), and re-read every pid's start identity
// before every signal so a recycled pid is never touched.

// ErrServerChanged is a server whose start identity no longer matches the one
// the caller observed: a different process now holds the pid. Nothing is
// signalled.
var ErrServerChanged = errors.New("zellij server changed since it was observed")

// ProcessRow is one process in a snapshot.
type ProcessRow struct {
	PID, PPID int
	Identity  string
}

// ProcessTable is the reaper's view of the host.
type ProcessTable interface {
	Snapshot(ctx context.Context) ([]ProcessRow, error)
	Identity(pid int) string
	Signal(pid int, sig syscall.Signal) error
}

// ReapStep is one process to end, with the identity it must still have.
type ReapStep struct {
	PID      int
	Identity string
	depth    int
}

// PlanReap orders the server's tree for signalling: descendants deepest first,
// the server last. Pure over one snapshot.
func PlanReap(server SessionServerIdentity, rows []ProcessRow) []ReapStep {
	children := map[int][]ProcessRow{}
	for _, row := range rows {
		children[row.PPID] = append(children[row.PPID], row)
	}
	var steps []ReapStep
	seen := map[int]bool{server.PID: true}
	queue := []ReapStep{{PID: server.PID, Identity: server.Identity}}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		steps = append(steps, at)
		for _, child := range children[at.PID] {
			if seen[child.PID] {
				continue
			}
			seen[child.PID] = true
			queue = append(queue, ReapStep{PID: child.PID, Identity: child.Identity, depth: at.depth + 1})
		}
	}
	sort.SliceStable(steps, func(i, j int) bool { return steps[i].depth > steps[j].depth })
	return steps
}

// Reaper signals one planned tree with bounded escalation.
type Reaper struct {
	Table              ProcessTable
	TermWait, KillWait time.Duration
	Poll               time.Duration
	// beforeSignal is a test seam for the window between the identity
	// re-read's snapshot and the signal.
	beforeSignal func(pid int)
}

// Reap ends server and every process under it. It refuses, signalling nothing,
// when the server's identity no longer matches; it returns an error naming any
// process still alive after the SIGKILL bound.
func (r Reaper) Reap(ctx context.Context, server SessionServerIdentity) error {
	rows, err := r.Table.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("snapshot processes: %w", err)
	}
	current := ""
	for _, row := range rows {
		if row.PID == server.PID {
			current = row.Identity
		}
	}
	if current == "" || current != server.Identity {
		return fmt.Errorf("%w (pid %d)", ErrServerChanged, server.PID)
	}
	plan := PlanReap(server, rows)
	r.signalAll(plan, syscall.SIGTERM)
	if survivors := r.await(ctx, plan, r.TermWait); len(survivors) > 0 {
		r.signalAll(survivors, syscall.SIGKILL)
		if left := r.await(ctx, survivors, r.KillWait); len(left) > 0 {
			var pids []string
			for _, step := range left {
				pids = append(pids, strconv.Itoa(step.PID))
			}
			return fmt.Errorf("reap %s: still running after SIGKILL: pid %s", server.Session, strings.Join(pids, ", "))
		}
	}
	return nil
}

// signalAll walks the plan in order, re-reading each identity first.
func (r Reaper) signalAll(steps []ReapStep, sig syscall.Signal) {
	for _, step := range steps {
		if r.beforeSignal != nil {
			r.beforeSignal(step.PID)
		}
		if r.Table.Identity(step.PID) != step.Identity {
			continue // gone, or the pid now belongs to someone else
		}
		_ = r.Table.Signal(step.PID, sig)
	}
}

// await polls until every step is gone (or recycled), bounded by wait.
func (r Reaper) await(ctx context.Context, steps []ReapStep, wait time.Duration) []ReapStep {
	poll := r.Poll
	if poll <= 0 {
		poll = 50 * time.Millisecond
	}
	deadline := time.Now().Add(wait)
	for {
		var alive []ReapStep
		for _, step := range steps {
			if r.Table.Identity(step.PID) == step.Identity {
				alive = append(alive, step)
			}
		}
		if len(alive) == 0 || !time.Now().Before(deadline) || ctx.Err() != nil {
			return alive
		}
		time.Sleep(poll)
	}
}

// OSProcessTable reads the real host.
type OSProcessTable struct{}

func (OSProcessTable) Snapshot(ctx context.Context) ([]ProcessRow, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return nil, err
	}
	var rows []ProcessRow
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, perr := strconv.Atoi(fields[0])
		ppid, qerr := strconv.Atoi(fields[1])
		if perr != nil || qerr != nil || pid <= 0 {
			continue
		}
		rows = append(rows, ProcessRow{PID: pid, PPID: ppid})
	}
	// Identity only for the rows a plan can reach is cheaper, but the plan is
	// built from this snapshot, so every row needs one; a vanished pid reads ""
	// and is never planned against a live identity.
	for i := range rows {
		rows[i].Identity = procutil.Identity(strconv.Itoa(rows[i].PID))
	}
	return rows, nil
}

func (OSProcessTable) Identity(pid int) string { return procutil.Identity(strconv.Itoa(pid)) }

func (OSProcessTable) Signal(pid int, sig syscall.Signal) error { return syscall.Kill(pid, sig) }

// OSOrphanReaper ends an orphaned server on the real host: the whole tree,
// then zellij's leftover session record (an EXITED resurrect row), proven
// absent by the same quiescence loop every session deletion uses.
type OSOrphanReaper struct{}

func (OSOrphanReaper) ReapOrphan(ctx context.Context, server SessionServerIdentity) error {
	reaper := Reaper{Table: OSProcessTable{}, TermWait: 3 * time.Second, KillWait: 2 * time.Second, Poll: 50 * time.Millisecond}
	if err := reaper.Reap(ctx, server); err != nil {
		return err
	}
	return quiesceZellijSession(ctx, server.Session, newOSSessionQuiescenceOps(), zellijQueryTimeout, 25*time.Millisecond)
}
