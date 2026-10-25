package couchtty

import (
	"errors"
	"strconv"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestLifecycleCompletionRequiresFinalAdoption(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "adopted", true: "failed-adoption"}[fail], func(t *testing.T) {
			c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(c.Stop)
			var events []LifecycleObservation
			c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
			c.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				if call.Name == "attach" && fail {
					return nil, errors.New("adoption refused")
				}
				return nil, nil
			})
			address := menuAddress("new")
			c.finishPreparedOperationForTest(operationCompletion{origin: MenuOperationOrigin{Operation: "start", Attempt: 1}, value: couchcore.StartResult{
				Record: couchcore.ActorRecord{Thread: address, PID: 12, Identity: "one", Args: couchcore.StartArgs{Worktree: "/repo"}},
				Handle: stubHandle{id: "one"},
			}})
			if fail && len(events) != 0 {
				t.Fatalf("provisional success: %+v", events)
			}
			if !fail && (len(events) != 1 || events[0].Kind != "started" || events[0].Address != address || events[0].Initiation != "operator") {
				t.Fatalf("adopted completion: %+v", events)
			}
		})
	}
}

func TestLifecycleAttachExitDoesNotEmitAgentExit(t *testing.T) {
	f := newFixture(t, 24, 80)
	var events []LifecycleObservation
	f.con.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
	f.con.onExit(childExit{id: "c1", code: 0})
	if len(events) != 0 {
		t.Fatalf("attachment exit claimed agent exit: %+v", events)
	}
}

func TestLifecycleRestartDrainsAfterLastCompletion(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	var events []LifecycleObservation
	c.ArmReattachPass(menuAddress("root"))
	c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
	if len(events) != 1 || events[0].Kind != "restart-armed" {
		t.Fatalf("armed: %+v", events)
	}
	c.mu.Lock()
	c.menu.Reattach = ReattachPass{Phase: ReattachRunning, InFlight: map[uint64]couchcore.ThreadAddress{1: menuAddress("worker")}}
	c.mu.Unlock()
	c.observeLifecyclePass()
	if len(events) != 1 {
		t.Fatalf("flushed running pass: %+v", events)
	}
	c.finishPreparedOperationForTest(operationCompletion{origin: MenuOperationOrigin{Operation: "resume", Background: true, Attempt: 1, Address: menuAddress("worker")}, err: errors.New("reattach unavailable")})
	c.observeLifecyclePass()
	if len(events) != 2 || events[1].Kind != "restart-drained" {
		t.Fatalf("drain once: %+v", events)
	}
}

func TestLifecycleInventoryCannotOverwriteHostedReplacement(t *testing.T) {
	f := newFixture(t, 24, 80)
	var events []LifecycleObservation
	f.con.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
	f.con.mu.Lock()
	address := f.con.panes["c1"].thread
	f.con.mu.Unlock()
	f.con.observeLifecycleInventory([]couchcore.ActionableThreadSummary{{Address: address, State: couchcore.ThreadUnusable, Reason: couchcore.ReasonNeedsRelaunch}})
	if len(events) != 0 {
		t.Fatalf("stale inventory overwrote adopted pane: %+v", events)
	}
}

func TestLifecycleParkOnlyReportsFinalizedResult(t *testing.T) {
	for _, tc := range []struct {
		name              string
		finalized, failed bool
		want              int
	}{{"requested", false, false, 0}, {"failed", true, true, 0}, {"finalized", true, false, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(c.Stop)
			var events []LifecycleObservation
			c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
			r := couchcore.ThreadRecord{Address: menuAddress("park"), Revision: 7}
			if tc.finalized {
				r.VerifiedPark = &couchcore.VerifiedPark{}
			}
			var err error
			if tc.failed {
				err = errors.New("finalization failed")
			}
			c.finishPreparedOperationForTest(operationCompletion{origin: MenuOperationOrigin{Operation: "park", Attempt: 1}, value: couchcore.ParkResult{Thread: r, CleanupError: errors.New("cleanup warning")}, err: err})
			if len(events) != tc.want {
				t.Fatalf("park observations: %+v", events)
			}
		})
	}
}

func TestLifecycleInventoryDeduplicatesAndBoundsEvidence(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	var events []LifecycleObservation
	c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
	row := couchcore.ActionableThreadSummary{Address: menuAddress("needs"), State: couchcore.ThreadUnusable, Reason: couchcore.ReasonNeedsRelaunch}
	c.observeLifecycleInventory([]couchcore.ActionableThreadSummary{row, row})
	if len(events) != 1 {
		t.Fatalf("duplicate observation: %+v", events)
	}
	for i := 0; i < 200; i++ {
		row.Address.Tag = couchcore.ThreadTag(strconv.Itoa(i))
		c.observeLifecycleInventory([]couchcore.ActionableThreadSummary{row})
	}
	if len(c.lifecycleNeeds) > 128 {
		t.Fatalf("unbounded evidence: %d", len(c.lifecycleNeeds))
	}
}

func TestLifecycleObserverInstalledBeforeArm(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	var events []LifecycleObservation
	c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
	c.ArmReattachPass(menuAddress("root"))
	if len(events) != 1 || events[0].Kind != "restart-armed" {
		t.Fatalf("arm not observed: %+v", events)
	}
}

func TestLifecycleMetadataPreservesTypedSlotAndCachedBranch(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	address := menuAddress("slot")
	target := menuSlotRow(1, "slot").Target
	c.mu.Lock()
	c.menu = NewMenuState([]couchcore.ActionableThreadSummary{{Address: address, Target: target, Agent: "codex", WorkingPath: "/repo3"}}, address)
	c.menu.SlotGit = map[string]couchcore.SlotGitStatus{target.Slot.WorktreeRoot: {Branch: "000574-work"}}
	c.mu.Unlock()
	e := c.lifecycleMetadata(address, "")
	if e.Target != target || e.Branch != "000574-work" || e.Issue != "574" || e.Agent != "codex" {
		t.Fatalf("metadata: %+v", e)
	}
}

func TestLifecycleMetadataAddresslessSlotsMatchOnlyNamedPath(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	a, b := menuSlotRow(1, ""), menuSlotRow(2, "")
	c.mu.Lock()
	c.menu = NewMenuState([]couchcore.ActionableThreadSummary{a, b}, couchcore.ThreadAddress{})
	c.mu.Unlock()
	got := c.lifecycleMetadata(couchcore.ThreadAddress{}, b.Target.Slot.WorktreeRoot)
	if got.Target != b.Target {
		t.Fatalf("wrong addressless slot: %+v", got)
	}
	unknown := c.lifecycleMetadata(couchcore.ThreadAddress{}, "")
	if unknown.Target != (couchcore.ThreadTarget{}) {
		t.Fatalf("unknown address selected a slot: %+v", unknown)
	}
}

func TestLifecycleNewAdoptionResetsPreviousNeedsRelaunchEvidence(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	row := menuSlotRow(1, "slot")
	row.Reason = couchcore.ReasonNeedsRelaunch
	c.mu.Lock()
	c.menu = NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
	c.mu.Unlock()
	var events []LifecycleObservation
	c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
	c.observeLifecycleInventory([]couchcore.ActionableThreadSummary{row})
	c.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) { return nil, nil })
	c.finishPreparedOperationForTest(operationCompletion{origin: MenuOperationOrigin{Operation: "resume", Attempt: 1}, value: couchcore.StartResult{Record: couchcore.ActorRecord{Thread: row.Address, PID: 123, Identity: "new", Args: couchcore.StartArgs{Worktree: couchcore.Worktree(row.Target.Slot.WorktreeRoot)}}, Handle: stubHandle{id: "new"}}})
	// The replacement becomes unusable before a live inventory is accepted.
	c.observeLifecycleInventory([]couchcore.ActionableThreadSummary{row})
	if len(events) != 3 || events[0].Kind != "needs-relaunch" || events[1].Kind != "resumed" || events[2].Kind != "needs-relaunch" {
		t.Fatalf("old incarnation suppressed replacement state: %+v", events)
	}
}

func TestLifecycleCompletionPrefersAdoptedAgent(t *testing.T) {
	for _, previousAgent := range []string{"", "claude"} {
		for _, operation := range []string{"start", "resume", "switch-agent", "relaunch", "reboot", "continue-thread"} {
			t.Run(operation+"/previous-"+previousAgent, func(t *testing.T) {
				c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
				t.Cleanup(c.Stop)
				row := menuSlotRow(1, "slot")
				row.Agent = previousAgent
				c.menu = NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
				var events []LifecycleObservation
				c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
				c.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) { return nil, nil })
				started := couchcore.StartResult{Record: couchcore.ActorRecord{Thread: row.Address, PID: 123, Identity: "adopted", Args: couchcore.StartArgs{Worktree: couchcore.Worktree(row.Target.Slot.WorktreeRoot), Stack: "codex"}}, Handle: stubHandle{id: "adopted"}}
				var result any = started
				switch operation {
				case "switch-agent":
					result = couchcore.SwitchAgentResult{Outcome: couchcore.SwitchStarted, Record: started.Record, Handle: started.Handle}
				case "relaunch":
					result = couchcore.RelaunchResult{Outcome: couchcore.Relaunched, Record: started.Record, Handle: started.Handle}
				case "reboot":
					result = couchcore.RebootResult{Start: started}
				case "continue-thread":
					result = couchcore.ContinuationResult{Record: started.Record, Handle: started.Handle}
				}
				c.finishPreparedOperationForTest(operationCompletion{origin: MenuOperationOrigin{Operation: operation, Attempt: 1}, value: result})
				if len(events) != 1 || events[0].Agent != "codex" || events[0].Target != row.Target {
					t.Fatalf("completion used stale/empty row agent: %+v", events)
				}
			})
		}
	}
}

func TestLifecycleRecoveryCompletionPreservesConfirmedMode(t *testing.T) {
	for _, operation := range []string{"recover", "recovery-confirmation"} {
		for _, tc := range []struct {
			mode couchcore.ContinuityAction
			want string
		}{{couchcore.Continue, "Continue"}, {couchcore.ResumeDisk, "Resume"}, {couchcore.Fresh, "Fresh"}, {"", ""}} {
			t.Run(operation+"/"+string(tc.mode), func(t *testing.T) {
				c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
				t.Cleanup(c.Stop)
				var events []LifecycleObservation
				c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
				c.SetOperationDispatcher(func(couchcore.OperationCall) (any, error) { return nil, nil })
				c.finishPreparedOperationForTest(operationCompletion{origin: MenuOperationOrigin{Operation: operation, Attempt: 1}, value: couchcore.RecoveryResumeResult{
					Mode: tc.mode, Notice: "deliberately unrelated presentation text",
					Start: couchcore.StartResult{Record: couchcore.ActorRecord{Thread: menuAddress("slot"), PID: 123, Identity: "recovered", Args: couchcore.StartArgs{Worktree: "/repo", Stack: "codex"}}, Handle: stubHandle{id: "recovered"}},
				}})
				if len(events) != 1 || events[0].Kind != "recovered" || events[0].Mode != tc.want || events[0].Agent != "codex" {
					t.Fatalf("confirmed recovery mode lost: %+v", events)
				}
			})
		}
	}
}

// A verified native exit is emitted as one exited observation, in the same
// per-console sequence as every other lifecycle observation (pair#657).
func TestLifecycleNativeExitObservation(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	var events []LifecycleObservation
	c.SetLifecycleObserver(func(e LifecycleObservation) { events = append(events, e) })
	target := couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "pair", Number: 2, WorktreeRoot: "/repo2"}}
	c.ObserveNativeExit(menuAddress("worker"), target, "3:child", 3)
	c.ObserveNativeExit(menuAddress("worker"), target, "4:child", 0)
	if len(events) != 2 || events[0].Kind != "exited" || events[0].Mode != "code 3" || events[0].Incarnation != "3:child" || events[0].Address != menuAddress("worker") || events[0].Target != target || events[0].Path != "/repo2" {
		t.Fatalf("exit observations: %+v", events)
	}
	if events[1].Observation <= events[0].Observation {
		t.Fatal("exit observations are not ordered")
	}
}
