package launcher

import (
	"context"
	"errors"
	"fmt"
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
	Name      string
}

// ProcessTable is the reaper's view of the host. Command reports whether argv
// could be read at all: a caller that attributes a process by its argv must not
// mistake an unreadable one for an empty one.
type ProcessTable interface {
	Snapshot(ctx context.Context) ([]ProcessRow, error)
	Identity(pid int) string
	Command(pid int) (args []string, ok bool)
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

// TreeReaper is the escalation cadence #399 measured: SIGTERM, three seconds for
// a clean exit, SIGKILL, two more for the kernel to collect the process. Every
// reaper of a process tree builds through it, so the cadence has one home and a
// second reaper cannot quietly invent a shorter fuse.
func TreeReaper(table ProcessTable) Reaper {
	return Reaper{Table: table, TermWait: 3 * time.Second, KillWait: 2 * time.Second, Poll: 50 * time.Millisecond}
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

// OSProcessTable reads the real host. Source is the process IO it reads
// through; the zero value reads natively, and an injected one lets a test own
// the host (#583's reaper runs on this seam).
type OSProcessTable struct{ Source procutil.Source }

// Snapshot reads pid, parent and start identity together (procutil.Table: one
// kern.proc.all sysctl on macOS), so a planned row's parent and identity
// describe the same process at the same moment (#399 close review BR-15).
func (t OSProcessTable) Snapshot(ctx context.Context) ([]ProcessRow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	procs, err := t.Source.Table()
	if err != nil {
		return nil, err
	}
	rows := make([]ProcessRow, len(procs))
	for i, p := range procs {
		rows[i] = ProcessRow{PID: p.PID, PPID: p.PPID, Identity: p.Identity, Name: p.Name}
	}
	return rows, nil
}

func (t OSProcessTable) Identity(pid int) string { return t.Source.Identity(pid) }

func (t OSProcessTable) Command(pid int) ([]string, bool) { return t.Source.Command(pid) }

func (t OSProcessTable) Signal(pid int, sig syscall.Signal) error { return syscall.Kill(pid, sig) }
