package couchcmd

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

type wakeRig struct {
	*serviceRig
	defs    []couchcore.HubDefinition
	callers []couchmessage.Binding
	worker  couchmessage.Binding

	mu      sync.Mutex
	asleep  []couchcore.ActionableThreadSummary
	pending map[string]bool
	resumed []string
	guards  []error
}

// newWakeRig is three hubs (ops is main) and a worker, each connected; the
// queue holds every admitted resume pending, as a slow resume would.
func newWakeRig(t *testing.T) *wakeRig {
	h, defs, _ := moveFixture(t)
	x := &wakeRig{defs: defs, pending: map[string]bool{}}
	r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}}
	r.slotOps = newSlotOperations(nil, func(context.Context) ([]couchcore.RepositoryName, error) {
		names := []couchcore.RepositoryName{{Key: filepath.Join(defs[0].Workspace, "pair"), Dir: "pair"}}
		for _, def := range defs {
			names = append(names, couchcore.RepositoryName{Key: def.Root, Dir: filepath.Base(def.Root)})
		}
		return names, nil
	}, time.Now)
	r.slotOps.guardedRun = func(key, op, target string, _ couchcore.LiveRestartOptions, guard func(context.Context) error, _ func(), _ couchcore.OperationProgress, _ func(any, error)) error {
		x.mu.Lock()
		defer x.mu.Unlock()
		if x.pending[key] {
			return couchtty.ErrRemotePending
		}
		x.pending[key] = true
		x.resumed = append(x.resumed, op+" "+target)
		x.guards = append(x.guards, guard(context.Background()))
		return nil
	}
	r.decorateAuthority = func(a messageAuthority) messageAuthority {
		a.hubs, a.hub = h, h.snapshot().Main()
		a.asleep = func(context.Context) ([]couchcore.ActionableThreadSummary, error) {
			x.mu.Lock()
			defer x.mu.Unlock()
			return append([]couchcore.ActionableThreadSummary(nil), x.asleep...), nil
		}
		return a
	}
	r.init()
	x.serviceRig = r
	for i, def := range defs {
		b := r.world.add(i)
		r.world.set(b, func(s *worldSlot) {
			n := 0
			s.root = def.Root
			s.workspace.WorktreeRoot = def.Root
			s.workspace.PrimaryRoot = def.Slot.Repository
			s.workspace.Slot = &n
			s.workspace.FleetRoot = def.Workspace
		})
		r.attach(b, couchmessage.PaneHandle(fmt.Sprintf("hub-%d", i)))
		r.wrapper(b)
		r.waitConnected(b, true)
		r.waitWorkspace(b)
		x.callers = append(x.callers, b)
	}
	x.worker = r.connect(7)
	return x
}

func (x *wakeRig) sleep(repo string, number int, root string) {
	at := time.Date(2026, 10, 23, 1, 0, 0, 0, time.UTC)
	x.mu.Lock()
	defer x.mu.Unlock()
	x.asleep = append(x.asleep, couchcore.ActionableThreadSummary{
		AsleepAt: &at,
		Target:   couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: repo, Number: number, WorktreeRoot: root}},
	})
}

func (x *wakeRig) wake(caller couchmessage.Binding, id string) couchmessage.Response {
	req := callerRequest(caller, "wake")
	req.ID = id
	return switchCall(x.t, x.serviceRig, req)
}

func wakeTargets(report *couchmessage.WakeReport) (admitted, refused []string) {
	for _, r := range report.Admitted {
		admitted = append(admitted, r.Op+" "+r.Target)
	}
	for _, r := range report.Refused {
		refused = append(refused, r.Target+" "+r.Code)
	}
	return admitted, refused
}

// pair#830: the main hub's wake resumes exactly the other asleep hubs; the
// main hub itself, asleep workers and plain parked slots stay put, and a
// repeat starts nothing twice.
func TestWakeResumesExactlyTheOtherAsleepHubs(t *testing.T) {
	x := newWakeRig(t)
	x.sleep("third", 0, x.defs[2].Root)
	x.sleep("bench", 0, x.defs[1].Root)
	x.sleep("ops", 0, x.defs[0].Root)
	x.sleep("pair", 3, filepath.Join(x.defs[0].Workspace, "pair-slot3"))

	for name, caller := range map[string]couchmessage.Binding{"worker": x.worker, "extra hub": x.callers[1]} {
		got := x.wake(caller, "wake-"+strings.ReplaceAll(name, " ", "-"))
		if got.Code != "refused" || !strings.Contains(got.Error, "only the main hub") {
			t.Fatalf("%s wake = %+v", name, got)
		}
	}
	got := x.wake(x.callers[0], "wake-1")
	if got.Code != "ok" || got.Wake == nil {
		t.Fatalf("main hub wake = %+v", got)
	}
	admitted, refused := wakeTargets(got.Wake)
	if strings.Join(admitted, ",") != "resume bench:0,resume third:0" || len(refused) != 0 {
		t.Fatalf("admitted %v refused %v", admitted, refused)
	}
	if strings.Join(x.resumed, ",") != "resume bench:0,resume third:0" {
		t.Fatalf("queued %v", x.resumed)
	}
	for i, err := range x.guards {
		if err != nil {
			t.Fatalf("effect gate %d refused the main hub: %v", i, err)
		}
	}
	for _, receipt := range got.Wake.Admitted {
		if status := switchCall(t, x.serviceRig, statusRequest(x.callers[0], receipt.ID)); status.Operation == nil || status.Operation.Target != receipt.Target {
			t.Fatalf("receipt %s not held for the caller: %+v", receipt.ID, status)
		}
	}

	// Still resuming: a repeat is refused busy and queues nothing.
	again := x.wake(x.callers[0], "wake-2")
	admitted, refused = wakeTargets(again.Wake)
	if again.Code != "ok" || len(admitted) != 0 || strings.Join(refused, ",") != "bench:0 busy,third:0 busy" || len(x.resumed) != 2 {
		t.Fatalf("repeat admitted %v refused %v queued %v", admitted, refused, x.resumed)
	}

	// Woken: nothing is asleep, so a repeat is an empty report.
	x.mu.Lock()
	x.asleep = nil
	x.mu.Unlock()
	done := x.wake(x.callers[0], "wake-3")
	if done.Code != "ok" || done.Wake == nil || len(done.Wake.Admitted)+len(done.Wake.Refused) != 0 {
		t.Fatalf("wake with nothing asleep = %+v", done)
	}
}

func TestWakeCLIParsesAndSummarizes(t *testing.T) {
	for _, args := range [][]string{{"--wake"}, {"--wake", "--json"}} {
		inv, err := ParseCLI(args, couchcore.Operations())
		if err != nil || inv.kind != cliMessage || inv.messageOp != "wake" || inv.jsonOutput != (len(args) == 2) {
			t.Fatalf("%v: %+v %v", args, inv, err)
		}
	}
	if _, err := ParseCLI([]string{"--wake", "bench:0"}, couchcore.Operations()); err == nil {
		t.Fatal("--wake took a target")
	}
	for _, tc := range []struct {
		outcomes []wakeOutcome
		want     string
	}{
		{nil, "no asleep hubs to wake"},
		{[]wakeOutcome{{Target: "bench:0", Status: "woken"}, {Target: "third:0", Status: "woken"}}, "woke 2 of 2 asleep hubs: bench:0, third:0"},
		{[]wakeOutcome{{Target: "bench:0", Status: "woken"}, {Target: "third:0", Status: "not admitted", Detail: "busy: pending"}},
			"woke 1 of 2 asleep hubs: bench:0\n  third:0 not admitted: busy: pending\nrun couch --wake again once the cause is fixed; it wakes only what is still asleep"},
	} {
		if got := wakeSummary(tc.outcomes); got != tc.want {
			t.Errorf("summary = %q\nwant %q", got, tc.want)
		}
	}
}

// The CLI polls each admitted resume to its end and prints one summary; a hub
// that did not come back fails the run.
func TestWakeCLIPollsEachResumeToOneSummary(t *testing.T) {
	now := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	clock := slotPollClock{now: func() time.Time { return now }, sleep: func(d time.Duration) { now = now.Add(d) }}
	finished := map[string]couchmessage.OperationReceipt{
		"r-bench": {ID: "r-bench", Op: "resume", Target: "bench:0", Status: couchmessage.ReceiptSucceeded},
		"r-third": {ID: "r-third", Op: "resume", Target: "third:0", Status: couchmessage.ReceiptFailed, Detail: "launch failed"},
	}
	var ops []string
	call := func(_ context.Context, _ string, request any, response any) error {
		req := request.(couchmessage.Request)
		ops = append(ops, req.Op+" "+req.ID)
		var result couchmessage.Response
		switch req.Op {
		case "wake":
			result = couchmessage.Response{Code: "ok", Wake: &couchmessage.WakeReport{
				Admitted: []couchmessage.OperationReceipt{
					{ID: "r-bench", Op: "resume", Target: "bench:0", Status: couchmessage.ReceiptQueued},
					{ID: "r-third", Op: "resume", Target: "third:0", Status: couchmessage.ReceiptQueued},
				},
				Refused: []couchmessage.WakeRefusal{{Target: "docs:0", Code: "busy", Error: "pending"}},
			}}
		case "operation-status":
			receipt := finished[req.ID]
			result = couchmessage.Response{Code: "ok", Operation: &receipt}
		}
		*response.(*couchmessage.Response) = result
		return nil
	}
	var out, errout bytes.Buffer
	code := runWakeCLI(cliInvocation{kind: cliMessage, messageOp: "wake"}, messageRuntime(), &out, &errout, call, clock)
	want := "woke 1 of 3 asleep hubs: bench:0\n  third:0 failed: launch failed\n  docs:0 not admitted: busy: pending\n"
	if code != 1 || !strings.HasPrefix(out.String(), want) || len(ops) != 3 || ops[1] != "operation-status r-bench" || ops[2] != "operation-status r-third" {
		t.Fatalf("code %d ops %v stdout %q stderr %q", code, ops, out.String(), errout.String())
	}
}
