package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"golang.org/x/sys/unix"
)

// The slot renicer (#502): every few seconds, one process-table snapshot,
// and each slot's agent's strict descendants lowered to COUCH_SLOT_NICE.
// The decision is couchcore.PlanRenice's; this is the shell that reads the
// slots, the agent pids and the table, applies the decision and keeps the
// facts `couch --actors` shows.
//
// Ordering. One goroutine runs the passes, so they never overlap; a slow
// pass skips ticks rather than queueing them. Between the snapshot and a
// renice a process can exit (ESRCH: skipped) or its pid be reused (its
// identity is re-read first and must still be the snapshot's). A refused
// renice (EPERM: not ours) is counted and that process is never asked
// again. Facts, refusals and the one notice per slot are kept per binding
// and dropped when the binding goes: nothing outlives a slot.

// defaultReniceInterval is how often the renicer looks: a command an agent
// starts runs at normal priority for at most about this long (#502, the
// operator accepted the delay).
const defaultReniceInterval = 5 * time.Second

// reniceInterval reads COUCH_RENICE_INTERVAL: a Go duration of at least a
// second; unset is defaultReniceInterval.
func reniceInterval(getenv func(string) string) (time.Duration, error) {
	raw := getenv("COUCH_RENICE_INTERVAL")
	if raw == "" {
		return defaultReniceInterval, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Second || d > time.Hour {
		return 0, fmt.Errorf("COUCH_RENICE_INTERVAL must be a duration from 1s to 1h, got %q", raw)
	}
	return d, nil
}

// procSource is the renicer's process seam: the table, one process's start
// identity now ("" when it is gone), and a renice.
type procSource interface {
	Table() ([]procutil.Process, error)
	Identity(pid int) string
	Renice(pid, nice int) error
}

// osProcs is the machine's: one sysctl snapshot, one sysctl per identity,
// setpriority(2).
type osProcs struct{}

func (osProcs) Table() ([]procutil.Process, error) { return procutil.Table() }
func (osProcs) Identity(pid int) string            { return procutil.Identity(strconv.Itoa(pid)) }
func (osProcs) Renice(pid, nice int) error {
	return unix.Setpriority(unix.PRIO_PROCESS, pid, nice)
}

type renicer struct {
	procs    procSource
	slots    func() []couchmessage.Binding
	agentPID func(couchmessage.Binding) (int, error)
	level    int
	self     int
	notify   func(string)

	mu      sync.Mutex
	facts   map[couchmessage.Binding]couchmessage.NiceFacts
	noticed map[couchmessage.Binding]bool
	refused map[couchmessage.Binding]map[string]bool // "pid@identity"
}

func newRenicer(procs procSource, slots func() []couchmessage.Binding, agentPID func(couchmessage.Binding) (int, error), level, self int, notify func(string)) *renicer {
	return &renicer{procs: procs, slots: slots, agentPID: agentPID, level: level, self: self, notify: notify,
		facts: map[couchmessage.Binding]couchmessage.NiceFacts{}, noticed: map[couchmessage.Binding]bool{},
		refused: map[couchmessage.Binding]map[string]bool{}}
}

// view is the broker's nice view: a slot's facts from the last pass.
func (r *renicer) view(b couchmessage.Binding) (couchmessage.NiceFacts, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.facts[b]
	return f, ok
}

// run passes every interval until ctx ends.
func (r *renicer) run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		r.pass()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// pass is one look: the slots, their agents, one snapshot, the plan, and
// each renice behind its identity re-check.
func (r *renicer) pass() {
	bindings := r.slots()
	roots := make([]couchcore.ReniceRoot, len(bindings))
	for i, b := range bindings {
		pid, err := r.agentPID(b)
		if err != nil {
			pid = 0 // unread: the slot stays unconfirmed
		}
		roots[i] = couchcore.ReniceRoot{Slot: b.Slot, WrapperPID: b.PID, WrapperStart: b.Start, AgentPID: pid}
	}
	table, err := r.procs.Table()
	if err != nil {
		return // no snapshot, no decision; the facts stand until the next pass
	}
	plan := couchcore.PlanRenice(table, roots, r.level, r.self)
	own := 0 // Couch's own nice: slots launch at it, so only above it is low
	for _, p := range table {
		if p.PID == r.self {
			own = p.Nice
		}
	}
	facts := make(map[couchmessage.Binding]couchmessage.NiceFacts, len(bindings))
	refused := make(map[couchmessage.Binding]map[string]bool, len(bindings))
	r.mu.Lock()
	for _, b := range bindings {
		refused[b] = r.refused[b]
		if refused[b] == nil {
			refused[b] = map[string]bool{}
		}
	}
	r.mu.Unlock()
	var low []couchmessage.Binding // slots whose agent runs below Couch's priority
	for i, b := range bindings {
		s := plan[i]
		f := couchmessage.NiceFacts{Confirmed: s.Confirmed, AgentNice: s.Agent.Nice, AtLevel: s.AtLevel}
		for _, p := range s.Lower {
			key := strconv.Itoa(p.PID) + "@" + p.Identity
			if refused[b][key] {
				f.Denied++
				continue
			}
			if r.procs.Identity(p.PID) != p.Identity {
				continue // exited, or the pid is another process's now
			}
			switch err := r.procs.Renice(p.PID, r.level); {
			case err == nil:
				f.Lowered++
			case errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES):
				refused[b][key] = true
				f.Denied++
			}
		}
		facts[b] = f
		if s.Confirmed && s.Agent.Nice > own {
			low = append(low, b)
		}
	}
	// Facts, refusals and notices are kept for the slots there are now.
	r.mu.Lock()
	noticed := make(map[couchmessage.Binding]bool, len(bindings))
	for _, b := range bindings {
		noticed[b] = r.noticed[b]
	}
	var say []string
	for _, b := range low {
		if !noticed[b] {
			noticed[b] = true
			say = append(say, fmt.Sprintf("%s's agent runs at nice %d (lowered at launch); relaunch the slot to restore its priority.", b.Slot, facts[b].AgentNice))
		}
	}
	r.facts, r.refused, r.noticed = facts, refused, noticed
	r.mu.Unlock()
	if r.notify != nil {
		for _, s := range say {
			r.notify(s)
		}
	}
}
