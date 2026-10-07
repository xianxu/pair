package couchcore

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

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

// Strategy (DeriveRecoverPreview, pure): one case per report decision recover
// can meet -- a step list, a hold, steps it does not run, nothing at all -- and
// one per ActorActions answer for a thread the report has no row for.
func TestDeriveRecoverPreview(t *testing.T) {
	parked := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "t"}, WorkingPath: "/w/p", State: ThreadParked}
	orphan := ActionableThreadSummary{Address: ThreadAddress{RepoScope: "s", Tag: "t"}, WorkingPath: "/w/p", State: ThreadUnusable,
		Reason: ReasonOrphanedServer, Orphan: &launcher.SessionServerIdentity{PID: 812, Session: "📁p-1"}}
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
		name    string
		row     ActionableThreadSummary
		report  *RecoverRow
		steps   []string
		confirm bool
		text    []string // substrings of Text, in order
		hold    []string // substrings of Hold
	}{
		{name: "parked resumes", row: parked, report: report([]string{"resume"}), steps: []string{"resume"}, text: []string{"resume", "p"}},
		{name: "orphan reaps then resumes", row: orphan, report: report([]string{"reap", "resume"}), steps: []string{"reap", "resume"}, confirm: true, text: []string{"reap", "PID 812", "then resume"}},
		{name: "unresumable reboots", row: rebootOnly, report: report([]string{"reboot"}), steps: []string{"reboot"}, confirm: true, text: []string{"archive this conversation and start a fresh agent"}},
		{name: "held", row: parked, report: report(nil, "agent-unknown", "conflict:claim-elsewhere"), hold: []string{"agent-unknown", "conflict:claim-elsewhere", "the report's reason"}},
		{name: "non-actor steps are not run", row: parked, report: report([]string{"resume", "ask-agent-restore"}), steps: []string{"resume"}, text: []string{"resume"}},
		{name: "a report row with no step falls back", row: parked, report: report(nil), steps: []string{"resume"}, text: []string{"resume"}},
		{name: "no report row: parked", row: parked, steps: []string{"resume"}, text: []string{"resume"}},
		{name: "no report row: orphan", row: orphan, steps: []string{"reap", "resume"}, confirm: true, text: []string{"PID 812", "then resume"}},
		{name: "no report row: reboot only", row: rebootOnly, steps: []string{"reboot"}, confirm: true, text: []string{"archive this conversation"}},
		{name: "no report row: nothing offered", row: unknown, hold: []string{string(HoldNoActorAction)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveRecoverPreview(tc.row, tc.report)
			if !slices.Equal(got.Steps, tc.steps) || got.Confirm != tc.confirm {
				t.Fatalf("preview = %+v, want steps %v confirm %v", got, tc.steps, tc.confirm)
			}
			if (got.Hold != "") != (tc.hold != nil) {
				t.Fatalf("hold = %q, want %v", got.Hold, tc.hold)
			}
			for _, want := range tc.hold {
				if !strings.Contains(got.Hold, want) {
					t.Fatalf("hold %q lacks %q", got.Hold, want)
				}
			}
			rest := got.Text
			for _, want := range tc.text {
				i := strings.Index(rest, want)
				if i < 0 {
					t.Fatalf("text %q lacks %q (in order)", got.Text, want)
				}
				rest = rest[i+len(want):]
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
// repository, so its preview is the ActorActions fallback [reap, resume]; the
// reaper's hook records how many children had started when it ran, proving
// reap precedes resume, and the result is resume's own started child.
func TestRecoverReapsThenResumes(t *testing.T) {
	env, address, server, reaper := resumableOrphanEnv(t)
	ctx := context.Background()
	preview, err := env.Couch.PrepareRecover(ctx, RecoverTarget{Address: address})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(preview.Steps, []string{"reap", "resume"}) || !preview.Confirm || !strings.Contains(preview.Text, "PID 9090") {
		t.Fatalf("preview = %+v", preview)
	}
	startsAtReap := -1
	reaper.hook = func() { startsAtReap = countStarts(env.Runner) }
	before := countStarts(env.Runner)
	value, err := env.Couch.Recover(ctx, RecoverTarget{Address: address}, preview.Steps, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(reaper.reaped) != 1 || reaper.reaped[0] != server || startsAtReap != before {
		t.Fatalf("reaped %+v with %d starts (before %d)", reaper.reaped, startsAtReap, before)
	}
	child, ok := value.(StartedChild)
	if !ok {
		t.Fatalf("result %T is not resume's started child", value)
	}
	if started, ok := child.Started(); !ok || started.Handle == nil || countStarts(env.Runner) != before+1 {
		t.Fatalf("resume started %+v; starts %d → %d", started, before, countStarts(env.Runner))
	}
}

// A preview the row no longer matches, an unconfirmed destructive plan and a
// held row each refuse before any step runs.
func TestRecoverRefusesWithNoEffect(t *testing.T) {
	for _, tc := range []struct {
		name      string
		steps     []string
		confirmed bool
		setup     func(*testEnv, ThreadAddress)
		code      string
	}{
		{name: "stale preview", steps: []string{"resume"}, confirmed: true, code: RecoverStale},
		{name: "unconfirmed", steps: []string{"reap", "resume"}, confirmed: false, code: RecoverUnconfirmed},
		{name: "held", steps: []string{"reap", "resume"}, confirmed: true, code: RecoverHeld, setup: func(env *testEnv, address ThreadAddress) {
			env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionUnresolved})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, address, _, reaper := resumableOrphanEnv(t)
			if tc.setup != nil {
				tc.setup(env, address)
			}
			before := countStarts(env.Runner)
			_, err := env.Couch.Recover(context.Background(), RecoverTarget{Address: address}, tc.steps, tc.confirmed)
			var refusal *RecoverRefusal
			if !errors.As(err, &refusal) || refusal.Code != tc.code {
				t.Fatalf("err = %v, want refusal %s", err, tc.code)
			}
			if len(reaper.reaped) != 0 || countStarts(env.Runner) != before {
				t.Fatalf("a refused recover acted: reaped %+v, starts %d → %d", reaper.reaped, before, countStarts(env.Runner))
			}
		})
	}
}

// The CLI sends no preview: its --confirm is plan-blind, so a confirmed
// recover runs whatever the plan is, and an unconfirmed destructive one is
// refused naming what it would do.
func TestRecoverWithoutPreviewStepsAppliesThePlan(t *testing.T) {
	env, address, _, reaper := resumableOrphanEnv(t)
	_, err := env.Couch.Recover(context.Background(), RecoverTarget{Address: address}, nil, false)
	var refusal *RecoverRefusal
	if !errors.As(err, &refusal) || refusal.Code != RecoverUnconfirmed || !strings.Contains(refusal.Detail, "PID 9090") || !strings.Contains(refusal.Detail, "--confirm") {
		t.Fatalf("err = %v", err)
	}
	if _, err := env.Couch.Recover(context.Background(), RecoverTarget{Address: address}, nil, true); err != nil || len(reaper.reaped) != 1 {
		t.Fatalf("confirmed recover: err %v, reaped %+v", err, reaper.reaped)
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
