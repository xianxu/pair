package couchcore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// resumableOrphanEnv is orphanedEnv whose conversation resolves, so the resume
// after the reap has a conversation to come back to.
func resumableOrphanEnv(t *testing.T) (*testEnv, ThreadAddress, launcher.SessionServerIdentity, *fakeOrphanReaper) {
	t.Helper()
	env, address, server, reaper := orphanedEnv(t)
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	// The cold resume's session coming up births its agent pane, as the
	// recover-plan acceptance world models.
	env.Artifacts.SetPairSession(address, "pair-orphan", false)
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	return env, address, server, reaper
}

// Strategy (deriveRecoverSteps, pure): one case per report decision recover
// can meet -- a step list, a hold, steps it does not run, nothing at all -- and
// one per ActorActions answer for a thread the report has no row for.
func TestDeriveRecoverSteps(t *testing.T) {
	t.Parallel()

	parked := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "t"}, WorkingPath: "/w/p", State: ThreadParked}
	orphan := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "t"}, WorkingPath: "/w/p", State: ThreadUnusable,
		Reason: ReasonOrphanedServer, Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁p-1"}}
	liveOrphan := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "t"}, WorkingPath: "/w/p", State: ThreadLive,
		Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁p-1"}}
	rebootOnly := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "t"}, WorkingPath: "/w/p", State: ThreadUnusable, Reason: ReasonBindingLost}
	unknown := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "t"}, WorkingPath: "/w/p", State: ThreadUnusable, Reason: ReasonUnknown}
	report := func(steps []string, hold ...string) *RecoverRow {
		row := &RecoverRow{Address: "p:0", Reason: "the report's reason", Next: RecoverNext{Hold: hold}}
		for _, step := range steps {
			row.Next.Steps = append(row.Next.Steps, RecoverStep{Action: step})
		}
		return row
	}
	for _, tc := range []struct {
		name   string
		row    ActionableThreadSummary
		report *RecoverRow
		steps  []string
		hold   []string // substrings of the hold
	}{
		{name: "parked resumes", row: parked, report: report([]string{"resume"}), steps: []string{"resume"}},
		{name: "orphan reaps then resumes", row: orphan, report: report([]string{"reap", "resume"}), steps: []string{"reap", "resume"}},
		{name: "unresumable reboots", row: rebootOnly, report: report([]string{"reboot"}), steps: []string{"reboot"}},
		{name: "held", row: parked, report: report(nil, "agent-unknown", "conflict:claim-elsewhere"), hold: []string{"agent-unknown", "conflict:claim-elsewhere", "the report's reason"}},
		{name: "non-actor steps are not run", row: parked, report: report([]string{"resume", "ask-agent-restore"}), steps: []string{"resume"}},
		{name: "a report row with no step falls back", row: parked, report: report(nil), steps: []string{"resume"}},
		{name: "no report row: parked", row: parked, steps: []string{"resume"}},
		{name: "no report row: orphan", row: orphan, steps: []string{"reap", "resume"}},
		{name: "no report row: live orphan", row: liveOrphan, steps: []string{"reap", "resume"}},
		{name: "no report row: reboot only", row: rebootOnly, steps: []string{"reboot"}},
		{name: "no report row: nothing offered", row: unknown, hold: []string{string(HoldNoActorAction)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			steps, hold := deriveRecoverSteps(tc.row, tc.report)
			if !slices.Equal(steps, tc.steps) || (hold != "") != (tc.hold != nil) {
				t.Fatalf("steps %v hold %q, want steps %v hold %v", steps, hold, tc.steps, tc.hold)
			}
			for _, want := range tc.hold {
				if !strings.Contains(hold, want) {
					t.Fatalf("hold %q lacks %q", hold, want)
				}
			}
		})
	}
}

// Strategy (recoverReportRowFor, pure): a slot row by its host checkout, a
// :0 row by its primary path and scope, and a thread no report row stands for.
func TestRecoverReportRowFor(t *testing.T) {
	t.Parallel()

	scope, err := launcher.ResolveRepoScope("/src/pair")
	if err != nil {
		t.Fatal(err)
	}
	slot := ActionableThreadSummary{Target: ThreadTarget{Kind: ThreadTargetSlot, Slot: SlotIdentity{Repo: "pair", PrimaryRoot: "/src/pair", WorktreeRoot: "/src/worktree/pair-slot2/pair", Number: 2}}}
	primary := ActionableThreadSummary{Address: ThreadAddress{RepoScope: scope.Key, Tag: "p"}, StartingPath: "/src/pair"}
	sub := ActionableThreadSummary{Address: ThreadAddress{RepoScope: scope.Key, Tag: "q"}, StartingPath: "/src/pair/cmd"}
	plan := RecoverPlan{Rows: []RecoverRow{{Address: "pair:0", Path: "/src/pair"}, {Address: "pair:2", Path: "/src/worktree/pair-slot2/pair"}}}
	if got := recoverReportRowFor(plan, slot); got == nil || got.Address != "pair:2" {
		t.Fatalf("slot row → %+v", got)
	}
	if got := recoverReportRowFor(plan, primary); got == nil || got.Address != "pair:0" {
		t.Fatalf("primary row → %+v", got)
	}
	if got := recoverReportRowFor(plan, sub); got != nil {
		t.Fatalf("a subdirectory thread is no slot's row, got %+v", got)
	}
}

// Unsafe terminals require explicit relaunch. Recover must not turn the
// inventory's orphan identity into permission to kill and resume automatically.
func TestRecoverOrphansRequireManualRelaunch(t *testing.T) {
	t.Parallel()
	for _, hosted := range []bool{false, true} {
		t.Run(fmt.Sprintf("hosted=%v", hosted), func(t *testing.T) {
			var env *testEnv
			var address ThreadAddress
			var reaper *fakeOrphanReaper
			if hosted {
				env, address, _, reaper, _ = liveOrphanEnv(t)
			} else {
				env, address, _, reaper = resumableOrphanEnv(t)
			}
			before := countStarts(env.Runner)
			polls := 0
			env.Couch.sleep = func(time.Duration) { polls++ }
			var progress []string
			ctx := WithOperationProgress(context.Background(), func(text string) { progress = append(progress, text) })
			_, err := env.Couch.Recover(ctx, RecoverTarget{Address: address})
			var refusal *RecoverRefusal
			if !errors.As(err, &refusal) || refusal.Code != RecoverHeld || !strings.Contains(refusal.Detail, string(HoldNoActorAction)) {
				t.Fatalf("err=%v want held", err)
			}
			if len(reaper.reaped) != 0 || countStarts(env.Runner) != before || polls != 0 || len(progress) != 0 {
				t.Fatalf("held recovery acted: reaped=%v starts=%d polls=%d progress=%v", reaper.reaped, countStarts(env.Runner)-before, polls, progress)
			}
		})
	}
}

func TestRecoverRefusesAHeldRowWithNoEffect(t *testing.T) {
	t.Parallel()

	env, address, _, reaper := resumableOrphanEnv(t)
	env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionUnresolved})
	before := countStarts(env.Runner)
	_, err := env.Couch.Recover(context.Background(), RecoverTarget{Address: address})
	var refusal *RecoverRefusal
	if !errors.As(err, &refusal) || refusal.Code != RecoverHeld || !strings.Contains(refusal.Detail, string(HoldNoActorAction)) {
		t.Fatalf("err = %v, want a held refusal", err)
	}
	if len(reaper.reaped) != 0 || countStarts(env.Runner) != before {
		t.Fatalf("a refused recover acted: reaped %+v, starts %d → %d", reaper.reaped, before, countStarts(env.Runner))
	}
}

func countStarts(r *FakeRunner) int {
	n := 0
	for _, op := range r.Ops {
		if strings.HasPrefix(op, "start ") {
			n++
		}
	}
	return n
}

// liveOrphanEnv is the 2026-10-07 acceptance gap: Couch still hosts the
// thread (its recorded process is alive) and its server lost its socket. exit
// ends the hosted process the way its client exits once the server is gone.
func liveOrphanEnv(t *testing.T) (env *testEnv, address ThreadAddress, server launcher.SessionServerIdentity, reaper *fakeOrphanReaper, exit func()) {
	t.Helper()
	env = newTestEnv(t, "/repo")
	first, h := env.spawn(t, StartArgs{Worktree: "/repo"})
	address = first.Thread
	server = launcher.SessionServerIdentity{PID: 9191, Identity: "t9191", Session: "📁repo-1"}
	env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionOrphaned, Orphan: &server})
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	reaper = &fakeOrphanReaper{artifacts: env.Artifacts, address: address}
	env.Couch.Reaper = reaper
	env.Couch.sleep = func(time.Duration) {}
	exit = func() {
		env.Runner.SetExited(h.ID(), 0)
		env.Proc.Kill(first.PID)
	}
	return env, address, server, reaper, exit
}

// Explicit Reap admits an orphan with a hosted helper; the row retains its exact identity.
func TestReapAdmitsALiveOrphan(t *testing.T) {
	t.Parallel()

	env, address, server, reaper, _ := liveOrphanEnv(t)
	if _, err := env.Couch.Reap(context.Background(), ReapTarget{Address: address}); err != nil {
		t.Fatal(err)
	}
	if len(reaper.reaped) != 1 || reaper.reaped[0] != server {
		t.Fatalf("reaped %+v", reaper.reaped)
	}
}

// Without a reporter, progress is a no-op.
func TestProgressWithoutAReporterIsSilent(t *testing.T) {
	t.Parallel()

	reportProgress(context.Background(), "nothing listens")
	reportProgress(nil, "nothing listens")
}
