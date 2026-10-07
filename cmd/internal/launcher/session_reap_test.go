package launcher

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// fakeProcess is one row of the fake table, with the behaviours 2026-10-06
// showed: a child that ignores SIGTERM, reparenting to PID 1 when a parent
// dies, and (for the bound) a process nothing can kill.
type fakeProcess struct {
	ppid        int
	identity    string
	ignoresTERM bool
	unkillable  bool
}

type fakeProcessTable struct {
	mu      sync.Mutex
	procs   map[int]*fakeProcess
	signals []string // "pid:SIG" in delivery order
}

func newFakeProcessTable() *fakeProcessTable {
	return &fakeProcessTable{procs: map[int]*fakeProcess{}}
}

func (f *fakeProcessTable) add(pid, ppid int, identity string) *fakeProcess {
	p := &fakeProcess{ppid: ppid, identity: identity}
	f.procs[pid] = p
	return p
}

func (f *fakeProcessTable) Snapshot(context.Context) ([]ProcessRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var rows []ProcessRow
	for pid, p := range f.procs {
		rows = append(rows, ProcessRow{PID: pid, PPID: p.ppid, Identity: p.identity})
	}
	return rows, nil
}

func (f *fakeProcessTable) Identity(pid int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.procs[pid]; ok {
		return p.identity
	}
	return ""
}

func (f *fakeProcessTable) Signal(pid int, sig syscall.Signal) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.procs[pid]
	if !ok {
		return syscall.ESRCH
	}
	f.signals = append(f.signals, signalName(pid, sig))
	dies := (sig == syscall.SIGTERM && !p.ignoresTERM) || (sig == syscall.SIGKILL && !p.unkillable)
	if !dies {
		return nil
	}
	delete(f.procs, pid)
	for _, child := range f.procs {
		if child.ppid == pid {
			child.ppid = 1
		}
	}
	return nil
}

func signalName(pid int, sig syscall.Signal) string {
	name := "TERM"
	if sig == syscall.SIGKILL {
		name = "KILL"
	}
	return strconv.Itoa(pid) + ":" + name
}

func fastReaper(table ProcessTable) Reaper {
	return Reaper{Table: table, TermWait: 50 * time.Millisecond, KillWait: 50 * time.Millisecond, Poll: time.Millisecond}
}

// The 2026-10-06 tree: server 10 → wrap 11 (ignores SIGTERM) → title 13, and
// nvim 12. Everything goes, the descendants before the server, and the wrap that
// ignored SIGTERM is still found by pid after it reparents.
func TestReapTakesDownTheWholeTreeDescendantsFirst(t *testing.T) {
	table := newFakeProcessTable()
	table.add(10, 1, "server")
	table.add(11, 10, "wrap").ignoresTERM = true
	table.add(12, 10, "nvim")
	table.add(13, 11, "title")
	table.add(99, 1, "unrelated")
	if err := fastReaper(table).Reap(context.Background(), SessionServerIdentity{PID: 10, Identity: "server", Session: "s"}); err != nil {
		t.Fatal(err)
	}
	for _, pid := range []int{10, 11, 12, 13} {
		if table.Identity(pid) != "" {
			t.Fatalf("pid %d survived the reap; signals %v", pid, table.signals)
		}
	}
	if table.Identity(99) == "" {
		t.Fatal("an unrelated process was killed")
	}
	// The server is signalled only after every descendant has been.
	serverAt := -1
	for i, s := range table.signals {
		if strings.HasPrefix(s, "10:") && serverAt < 0 {
			serverAt = i
		}
	}
	for i, s := range table.signals {
		if i > serverAt && strings.HasPrefix(s, "13:") || i > serverAt && strings.HasPrefix(s, "12:") {
			if !strings.HasSuffix(s, ":KILL") {
				t.Fatalf("a descendant was first signalled after the server: %v", table.signals)
			}
		}
	}
	if !slices.Contains(table.signals, "11:TERM") || !slices.Contains(table.signals, "11:KILL") {
		t.Fatalf("the TERM-ignoring wrap was not escalated: %v", table.signals)
	}
}

// A pid recycled between snapshot and signal is never signalled as ours.
func TestReapNeverSignalsARecycledPid(t *testing.T) {
	table := newFakeProcessTable()
	table.add(10, 1, "server")
	child := table.add(12, 10, "nvim")
	reaper := fastReaper(table)
	reaper.beforeSignal = func(pid int) {
		if pid == 12 {
			child.identity = "recycled"
		}
	}
	_ = reaper.Reap(context.Background(), SessionServerIdentity{PID: 10, Identity: "server"})
	if slices.Contains(table.signals, "12:TERM") || slices.Contains(table.signals, "12:KILL") {
		t.Fatalf("a recycled pid was signalled: %v", table.signals)
	}
}

// The server's identity moved before the first signal: nothing is signalled.
func TestReapRefusesAChangedServer(t *testing.T) {
	table := newFakeProcessTable()
	table.add(10, 1, "a-new-server")
	table.add(11, 10, "wrap")
	err := fastReaper(table).Reap(context.Background(), SessionServerIdentity{PID: 10, Identity: "server"})
	if !errors.Is(err, ErrServerChanged) || len(table.signals) != 0 {
		t.Fatalf("err %v, signals %v", err, table.signals)
	}
}

// Waits are bounded: a process that survives SIGKILL is reported, not waited on.
func TestReapIsBoundedAndNamesSurvivors(t *testing.T) {
	table := newFakeProcessTable()
	table.add(10, 1, "server")
	stuck := table.add(11, 10, "stuck")
	stuck.ignoresTERM, stuck.unkillable = true, true
	start := time.Now()
	err := fastReaper(table).Reap(context.Background(), SessionServerIdentity{PID: 10, Identity: "server"})
	if err == nil || !strings.Contains(err.Error(), "11") {
		t.Fatalf("err = %v, want one naming pid 11", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("reap waited without a bound")
	}
}

func TestPlanReapOrdersDeepestFirstServerLast(t *testing.T) {
	rows := []ProcessRow{{PID: 10, PPID: 1, Identity: "s"}, {PID: 11, PPID: 10, Identity: "w"}, {PID: 13, PPID: 11, Identity: "t"}, {PID: 12, PPID: 10, Identity: "n"}, {PID: 50, PPID: 1, Identity: "x"}}
	plan := PlanReap(SessionServerIdentity{PID: 10, Identity: "s"}, rows)
	var order []int
	for _, step := range plan {
		order = append(order, step.PID)
	}
	if len(order) != 4 || order[0] != 13 || order[3] != 10 {
		t.Fatalf("plan order %v: want the grandchild first and the server last", order)
	}
}
