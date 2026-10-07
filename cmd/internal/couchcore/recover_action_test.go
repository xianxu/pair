package couchcore

import (
	"context"
	"errors"
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

// Strategy (Couch.Recover, shell): the orphan fixture is outside any enrolled
// repository, so its steps are the ActorActions fallback [reap, resume]; the
// reaper's hook records how many children had started when it ran, proving
// reap precedes resume, and the result is resume's own started child. Recover
// asks nothing: choosing it is the consent (#399, 2026-10-07).
func TestRecoverReapsThenResumes(t *testing.T) {
	env, address, server, reaper := resumableOrphanEnv(t)
	startsAtReap := -1
	reaper.hook = func() { startsAtReap = countStarts(env.Runner) }
	before := countStarts(env.Runner)
	value, err := env.Couch.Recover(context.Background(), RecoverTarget{Address: address})
	if err != nil {
		t.Fatal(err)
	}
	if len(reaper.reaped) != 1 || reaper.reaped[0] != server || startsAtReap != before {
		t.Fatalf("reaped %+v with %d starts (before %d)", reaper.reaped, startsAtReap, before)
	}
	assertResumedChild(t, env, value, before)
}

func assertResumedChild(t *testing.T, env *testEnv, value any, before int) {
	t.Helper()
	child, ok := value.(StartedChild)
	if !ok {
		t.Fatalf("result %T is not resume's started child", value)
	}
	if started, ok := child.Started(); !ok || started.Handle == nil || countStarts(env.Runner) != before+1 {
		t.Fatalf("resume started %+v; starts %d → %d", started, before, countStarts(env.Runner))
	}
}

// A held row refuses, typed and naming the hold, before any step runs.
func TestRecoverRefusesAHeldRowWithNoEffect(t *testing.T) {
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

// Reap admits a live orphan: the row carries Orphan though it stays live.
func TestReapAdmitsALiveOrphan(t *testing.T) {
	env, address, server, reaper, _ := liveOrphanEnv(t)
	if _, err := env.Couch.Reap(context.Background(), ReapTarget{Address: address}); err != nil {
		t.Fatal(err)
	}
	if len(reaper.reaped) != 1 || reaper.reaped[0] != server {
		t.Fatalf("reaped %+v", reaper.reaped)
	}
}

// Recover on a live orphan reaps, waits for the hosted client's exit to land
// (it does a moment after the reap returns), then resumes, returning resume's
// child. Without the wait the row still reads live and resume is refused.
func TestRecoverOnALiveOrphanReapsWaitsThenResumes(t *testing.T) {
	env, address, server, reaper, exit := liveOrphanEnv(t)
	polls := 0
	env.Couch.sleep = func(d time.Duration) {
		if d == recoverSettlePoll {
			if polls++; polls == 2 {
				exit()
			}
		}
	}
	before := countStarts(env.Runner)
	value, err := env.Couch.Recover(context.Background(), RecoverTarget{Address: address})
	if err != nil {
		t.Fatal(err)
	}
	if len(reaper.reaped) != 1 || reaper.reaped[0] != server || polls != 2 {
		t.Fatalf("reaped %+v after %d polls", reaper.reaped, polls)
	}
	assertResumedChild(t, env, value, before)
}

// A hosted client that never exits bounds the wait: recover stops before
// resume, typed, having reaped.
func TestRecoverStopsWhenTheRowNeverAdmitsTheNextStep(t *testing.T) {
	env, address, _, reaper, _ := liveOrphanEnv(t)
	before := countStarts(env.Runner)
	_, err := env.Couch.Recover(context.Background(), RecoverTarget{Address: address})
	var refusal *RecoverRefusal
	if !errors.As(err, &refusal) || refusal.Code != RecoverChanged || !strings.Contains(err.Error(), "after reap") {
		t.Fatalf("err = %v", err)
	}
	if len(reaper.reaped) != 1 || countStarts(env.Runner) != before {
		t.Fatalf("reaped %+v, starts %d → %d", reaper.reaped, before, countStarts(env.Runner))
	}
}
