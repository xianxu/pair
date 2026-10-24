package couchcmd

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// fakeProcs is a stateful process table: Renice changes a process's nice;
// a test spawns, exits and reuses pids between calls, and refuses some.
type fakeProcs struct {
	mu      sync.Mutex
	procs   map[int]procutil.Process
	refuse  map[int]bool
	renices map[int]int // pid -> times asked
	// afterTable runs once the snapshot is taken: the world moves on.
	afterTable func(*fakeProcs)
}

func newFakeProcs(ps ...procutil.Process) *fakeProcs {
	f := &fakeProcs{procs: map[int]procutil.Process{}, refuse: map[int]bool{}, renices: map[int]int{}}
	for _, p := range ps {
		f.procs[p.PID] = p
	}
	return f
}

func (f *fakeProcs) Table() ([]procutil.Process, error) {
	f.mu.Lock()
	out := make([]procutil.Process, 0, len(f.procs))
	for _, p := range f.procs {
		out = append(out, p)
	}
	after := f.afterTable
	f.afterTable = nil
	f.mu.Unlock()
	if after != nil {
		after(f)
	}
	return out, nil
}

func (f *fakeProcs) Identity(pid int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.procs[pid].Identity
}

func (f *fakeProcs) Renice(pid, nice int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.renices[pid]++
	p, ok := f.procs[pid]
	switch {
	case !ok:
		return syscall.ESRCH
	case f.refuse[pid]:
		return syscall.EPERM
	}
	p.Nice = nice
	f.procs[pid] = p
	return nil
}

func (f *fakeProcs) nice(pid int) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.procs[pid].Nice
}

func (f *fakeProcs) set(p procutil.Process) {
	f.mu.Lock()
	f.procs[p.PID] = p
	f.mu.Unlock()
}

func (f *fakeProcs) exit(pid int) {
	f.mu.Lock()
	delete(f.procs, pid)
	f.mu.Unlock()
}

func pr(pid, ppid, nice int, id string) procutil.Process {
	return procutil.Process{PID: pid, PPID: ppid, Nice: nice, Identity: id}
}

// One slot: couch 100 › zellij 120 › wrapper 200 › agent 300 › bash 310 ›
// make 320; the draft's nvim 210 is the wrapper's other child; a decoy
// 900 "named" like the agent sits under launchd.
var slot3 = couchmessage.Binding{Slot: "pair:3", Scope: "s", Tag: "3", PID: 200, Start: "wrap@1"}

func slotMachine() *fakeProcs {
	return newFakeProcs(pr(1, 0, 0, "launchd"), pr(100, 1, 0, "couch"), pr(120, 100, 0, "zellij"), pr(200, 120, 0, "wrap@1"),
		pr(210, 200, 0, "nvim"), pr(300, 200, 0, "agent@1"), pr(310, 300, 0, "bash"), pr(320, 310, 0, "make"), pr(900, 1, 0, "agent-decoy"))
}

func newTestRenicer(procs procSource, bindings []couchmessage.Binding, agent map[couchmessage.Binding]int) (*renicer, *[]string) {
	var notices []string
	r := newRenicer(procs, func() []couchmessage.Binding { return bindings }, func(b couchmessage.Binding) (int, error) {
		if pid, ok := agent[b]; ok {
			return pid, nil
		}
		return 0, os.ErrNotExist
	}, 10, 100, func(_ couchmessage.Binding, s string) { notices = append(notices, s) })
	return r, &notices
}

func TestRenicerLowersTheAgentsWorkOnly(t *testing.T) {
	m := slotMachine()
	r, notices := newTestRenicer(m, []couchmessage.Binding{slot3}, map[couchmessage.Binding]int{slot3: 300})
	r.pass()
	for pid, want := range map[int]int{100: 0, 120: 0, 200: 0, 210: 0, 300: 0, 900: 0, 310: 10, 320: 10} {
		if got := m.nice(pid); got != want {
			t.Errorf("pid %d at nice %d, want %d", pid, got, want)
		}
	}
	if f, ok := r.view(slot3); !ok || !f.Confirmed || f.Lowered != 2 || f.AgentNice != 0 {
		t.Errorf("facts %+v %v", f, ok)
	}
	// A command the agent starts later is lowered on the next pass; the
	// ones already low aren't asked again.
	m.set(pr(330, 300, 0, "pytest"))
	r.pass()
	if m.nice(330) != 10 || m.renices[310] != 1 {
		t.Errorf("later command nice %d; bash reniced %d times", m.nice(330), m.renices[310])
	}
	if f, _ := r.view(slot3); f.Lowered != 1 || f.AtLevel != 2 {
		t.Errorf("second pass facts %+v", f)
	}
	if len(*notices) != 0 {
		t.Errorf("notices %q", *notices)
	}
}

// The decoy: a slot whose agent-pid file names a process outside its
// wrapper (stale, or a decoy) is unconfirmed and nothing is touched.
func TestRenicerNeverTouchesAnUnconfirmedTree(t *testing.T) {
	m := slotMachine()
	m.set(pr(910, 900, 0, "decoy-child"))
	r, _ := newTestRenicer(m, []couchmessage.Binding{slot3}, map[couchmessage.Binding]int{slot3: 900})
	r.pass()
	for pid := range m.procs {
		if m.renices[pid] != 0 {
			t.Errorf("pid %d was reniced", pid)
		}
	}
	if f, ok := r.view(slot3); !ok || f.Confirmed {
		t.Errorf("facts %+v", f)
	}
}

// Between the snapshot and the renice a process exits, or its pid is
// reused by another process: neither is lowered.
func TestRenicerRechecksIdentity(t *testing.T) {
	m := slotMachine()
	m.afterTable = func(f *fakeProcs) {
		f.exit(310)
		f.set(pr(320, 1, 0, "someone-elses")) // 320 reused
	}
	r, _ := newTestRenicer(m, []couchmessage.Binding{slot3}, map[couchmessage.Binding]int{slot3: 300})
	r.pass()
	if m.renices[310] != 0 || m.renices[320] != 0 || m.nice(320) != 0 {
		t.Errorf("reniced a gone or reused pid: %v, 320 at %d", m.renices, m.nice(320))
	}
}

// A refused renice (not ours) is counted and never asked again.
func TestRenicerRefusalIsCountedNotLooped(t *testing.T) {
	m := slotMachine()
	m.refuse[320] = true
	r, _ := newTestRenicer(m, []couchmessage.Binding{slot3}, map[couchmessage.Binding]int{slot3: 300})
	for range 3 {
		r.pass()
	}
	if m.renices[320] != 1 {
		t.Errorf("a refused pid was asked %d times", m.renices[320])
	}
	if f, _ := r.view(slot3); f.Denied != 1 {
		t.Errorf("facts %+v, want one denied", f)
	}
}

// An agent found below normal (launched under #498) gets one notice that
// says the fix; a slot that goes takes its facts and notice with it.
func TestRenicerNoticesALowAgentOnceAndForgetsGoneSlots(t *testing.T) {
	m := slotMachine()
	m.set(pr(300, 200, 10, "agent@1"))
	bindings := []couchmessage.Binding{slot3}
	var notices []string
	r := newRenicer(m, func() []couchmessage.Binding { return bindings }, func(couchmessage.Binding) (int, error) { return 300, nil }, 10, 100,
		func(_ couchmessage.Binding, s string) { notices = append(notices, s) })
	r.pass()
	r.pass()
	if len(notices) != 1 || !strings.Contains(notices[0], "pair:3") || !strings.Contains(notices[0], "relaunch the slot to restore its priority") {
		t.Fatalf("notices %q", notices)
	}
	if m.nice(300) != 10 || m.renices[300] != 0 {
		t.Error("the agent itself was reniced")
	}
	bindings = nil
	r.pass()
	if _, ok := r.view(slot3); ok || len(r.noticed) != 0 || len(r.refused) != 0 {
		t.Error("a gone slot's state outlived it")
	}
	bindings = []couchmessage.Binding{slot3}
	r.pass()
	if len(notices) != 2 {
		t.Errorf("a slot that comes back is noticed again: %q", notices)
	}
}

// Couch started from a niced shell launches its agents at its own nice:
// that is not low, and the notice (relaunch) couldn't help. Only an agent
// below Couch's own priority is noticed.
func TestRenicerNoticeComparesAgainstCouchsOwnNice(t *testing.T) {
	m := slotMachine()
	m.set(pr(100, 1, 5, "couch"))
	m.set(pr(300, 200, 5, "agent@1"))
	r, notices := newTestRenicer(m, []couchmessage.Binding{slot3}, map[couchmessage.Binding]int{slot3: 300})
	r.pass()
	if len(*notices) != 0 {
		t.Fatalf("an agent at Couch's nice was noticed: %q", *notices)
	}
	m.set(pr(300, 200, 10, "agent@1"))
	r.pass()
	if len(*notices) != 1 || !strings.Contains((*notices)[0], "nice 10") {
		t.Errorf("an agent below Couch's priority: notices %q", *notices)
	}
}

func TestRenicerLevelZeroOnlyReports(t *testing.T) {
	m := slotMachine()
	r := newRenicer(m, func() []couchmessage.Binding { return []couchmessage.Binding{slot3} }, func(couchmessage.Binding) (int, error) { return 300, nil }, 0, 100, nil)
	r.pass()
	if len(m.renices) != 0 {
		t.Errorf("level 0 reniced %v", m.renices)
	}
	if f, ok := r.view(slot3); !ok || !f.Confirmed || f.Lowered+f.AtLevel != 0 {
		t.Errorf("level 0 still reports, and lowers nothing: %+v", f)
	}
}

// The live check of the real seam, on a process tree of the test's own:
// sh › sleep. The grandchild goes to the level, its parent doesn't.
func TestOSProcsRenicesOnlyWhatItIsTold(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 5 & wait")
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	var child procutil.Process
	for deadline := time.Now().Add(2 * time.Second); child.PID == 0 && time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		table, err := osProcs{}.Table()
		if err != nil {
			t.Skip(err)
		}
		for _, p := range table {
			if p.PPID == cmd.Process.Pid {
				child = p
			}
		}
	}
	if child.PID == 0 {
		t.Fatal("the sleep never showed in the table")
	}
	defer syscall.Kill(child.PID, syscall.SIGKILL)
	var procs procSource = osProcs{}
	if procs.Identity(child.PID) != child.Identity {
		t.Fatal("identity re-read differs from the snapshot's")
	}
	err := procs.Renice(child.PID, child.Nice+5)
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		t.Skipf("lowering priority is refused here (a sandbox): %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	table, _ := procs.Table()
	for _, p := range table {
		switch p.PID {
		case child.PID:
			if p.Nice != min(child.Nice+5, 20) {
				t.Errorf("child at %d, want %d", p.Nice, min(child.Nice+5, 20))
			}
		case cmd.Process.Pid:
			if p.Nice != child.Nice {
				t.Errorf("the parent moved to %d", p.Nice)
			}
		}
	}
}

// BenchmarkReniceScan is one pass's decision on this machine's real process
// table: one sysctl snapshot and the plan (#502's cost; the Log records it).
func BenchmarkReniceScan(b *testing.B) {
	self := os.Getpid()
	roots := []couchcore.ReniceRoot{{Slot: "self", WrapperPID: os.Getppid(), AgentPID: self}}
	n := 0
	for b.Loop() {
		table, err := osProcs{}.Table()
		if err != nil {
			b.Skip(err)
		}
		n = len(table)
		_ = couchcore.PlanRenice(table, roots, 10, 1)
	}
	b.ReportMetric(float64(n), "procs")
}
