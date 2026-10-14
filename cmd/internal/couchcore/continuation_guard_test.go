package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestContinuationBlocksCompetingTransitions(t *testing.T) {
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	request := testContinuationRequest(t, source)
	source, err := env.Couch.Threads.PublishContinuation(source.Address, source.Revision, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.Relaunch(context.Background(), source.Address); err == nil {
		t.Fatal("relaunch bypassed continuation")
	}
	if _, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", nil); err == nil {
		t.Fatal("agent switch bypassed continuation")
	}
	empty, err := env.Couch.Threads.updateExistingThread(source.Address, source.Revision, func(next *ThreadRecord) error { next.Incarnations = nil; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := archivableRecord(empty); err != nil {
		t.Fatalf("explicit archive must retain an unoccupied pending continuation: %v", err)
	}
	profile := LaunchProfile{Agent: "codex", Argv: []string{}}
	event := StartEvent{Kind: StartClaimed, Nonce: "start-1111111111111111", Owner: SupervisorOwner{PID: 100, Identity: "owner"}, Profile: &profile}
	if _, err := env.Couch.Threads.CommitStartClaim(empty.Address, empty.Revision, "repo", env.Now, event); err == nil {
		t.Fatal("cold start bypassed continuation")
	}
	running, err := env.Couch.Threads.AdvanceContinuation(empty.Address, empty.Revision, checkpoint.Event{Kind: checkpoint.Begin, RequestID: request.ID, Attempt: event.Nonce})
	if err != nil {
		t.Fatal(err)
	}
	event.Shape = StartFreshExisting
	if _, err := env.Couch.Threads.CommitStartClaim(running.Address, running.Revision, "repo", env.Now, event); err != nil {
		t.Fatalf("authorized attempt refused: %v", err)
	}
}

// The reported symptom (#280, again on 2026-10-12 for #522): a compaction
// that didn't finish blocked the next relaunch, switch and compaction until
// someone found dismiss-continuation. A request that didn't finish now leaves
// one line and blocks nothing -- driven to the real outcome, for a stored
// pre-#522 failed request too.
func TestAMissedContinuationBlocksNothing(t *testing.T) {
	t.Parallel()

	env, live := envWithLiveThread(t)
	read, _ := legacyFailedContinuation(t, env.Couch.Threads, live)
	if read.ContinuationMissed == nil {
		t.Fatal("no line")
	}
	result, err := env.Couch.Relaunch(context.Background(), read.Address)
	if err != nil || result.Outcome != Relaunched {
		t.Fatalf("Relaunch with the line = %+v, %v", result.Outcome, err)
	}
	after, err := env.Couch.Threads.GetThread(read.Address)
	if err != nil || after.ContinuationMissed != nil {
		t.Fatalf("a registered start keeps the line: %+v %v", after.ContinuationMissed, err)
	}
}

// continuationRefuses is the single statement of continuationGuard's reach --
// relaunch (relaunch.go), switch-agent (switchagent.go) and every non-warm
// start claim (threadstore.go); park and detach never read the request. Every row
// action is driven through the PRODUCTION dispatcher on a live thread holding a
// request in flight: a listed operation must be refused BY THE GUARD (its phrase),
// and an unlisted one must SUCCEED -- not merely fail some other way, which
// would read as "not guarded" when it never got that far. Every declared row
// action is either driven here or exempt with a reason, so a new one cannot
// arrive unclassified (#280).
func TestContinuationRefusesMatchesTheGuardForEveryRowAction(t *testing.T) {
	t.Parallel()

	type drive struct {
		args map[string]string
	}
	driven := map[string]drive{
		"relaunch":             {args: map[string]string{}},
		"prepare-switch-agent": {args: map[string]string{"agent": "codex"}}, // SwitchAgent re-runs this preview (switchagent.go)
		"park":                 {args: map[string]string{}},
		"detach":               {args: map[string]string{}},
	}
	rowActionDrivenAs := map[string]string{"switch-agent": "prepare-switch-agent"}
	exempt := map[string]string{
		"reboot":  "never offered on a live row; its retirement is prepareRetirement, whose admission is archive's archiveContinuationVacant",
		"alias":   "repository metadata keyed by primary root; it addresses no thread, so no thread's continuation gates it",
		"reap":    "offered on a live row only when its server is orphaned, where every lifecycle action refuses; it ends a process tree and touches no record, so no continuation gates it (#399)",
		"recover": "offered on a live row only when its server is orphaned; it composes resume, reap and reboot, each gated as itself through DispatchOperation (#399)",
		"resume":  "#363: routes a retained request to Continue/RecoverThread instead of the guard; pinned by TestChooseResumeRoute",
	}
	for _, op := range Operations() {
		if !op.RowAction {
			continue
		}
		name := op.Name
		if as, ok := rowActionDrivenAs[name]; ok {
			name = as
		}
		if _, ok := driven[name]; !ok && exempt[op.Name] == "" {
			t.Errorf("row action %q is neither driven against the guard nor exempt with a reason", op.Name)
		}
	}
	for name, d := range driven {
		t.Run(name, func(t *testing.T) {
			env, live := switchEnvWithLiveThread(t)
			if name == "detach" {
				// Detach SIGTERMs the helper and waits for it, as newDetachFixture
				// models; without this the fake never exits and detach fails for a
				// reason that has nothing to do with the guard.
				env.Proc.DiesOn = map[int]os.Signal{42: syscall.SIGTERM}
			}
			failed, request := runningContinuation(t, env.Couch.Threads, live)
			guardPrefix := "continuation " + request.ID + " is running; " // the guard's own words, not the wrapper's
			args := map[string]string{"repo-scope": failed.Address.RepoScope, "tag": string(failed.Address.Tag)}
			for k, v := range d.args {
				args[k] = v
			}
			_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch), DirectStore: DirectStoreExecutor(env.Couch)},
				OperationCall{Name: name, Args: args, Implicit: true, Context: context.Background()})
			if continuationRefuses(name) {
				if err == nil || !strings.Contains(err.Error(), guardPrefix) {
					t.Fatalf("%s is listed in continuationRefuses but the guard did not refuse it: %v", name, err)
				}
				// Refused BY THE GUARD means refused before any effect. The same
				// words from a later stage -- relaunch parking first, then failing
				// its cold resume -- would pass the message check and be exactly
				// the destructive order the front guard exists to prevent.
				after, err := env.Couch.Threads.GetThread(failed.Address)
				if err != nil || after.Revision != failed.Revision {
					t.Fatalf("%s wrote before its refusal: revision %d -> %d (%v)", name, failed.Revision, after.Revision, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s is not listed in continuationRefuses but did not succeed: %v", name, err)
			}
		})
	}
	// start is not a row action: the guard's reach there is the store's cold
	// start claim, driven directly.
	t.Run("start", func(t *testing.T) {
		env, live := switchEnvWithLiveThread(t)
		failed, request := runningContinuation(t, env.Couch.Threads, live)
		cold, err := env.Couch.Threads.updateExistingThread(failed.Address, failed.Revision, func(r *ThreadRecord) error { r.Incarnations = nil; return nil })
		if err != nil {
			t.Fatal(err)
		}
		profile := LaunchProfile{Agent: "claude", Argv: []string{}}
		_, err = env.Couch.Threads.CommitStartClaim(cold.Address, cold.Revision, "repo", env.Now, StartEvent{Kind: StartClaimed, Nonce: "start-2222222222222222", Owner: SupervisorOwner{PID: 100, Identity: "owner"}, Profile: &profile})
		if !continuationRefuses("start") || err == nil || !strings.Contains(err.Error(), "continuation "+request.ID+" is running; ") {
			t.Fatalf("cold start claim must be refused by the guard, and listed: %v", err)
		}
	})
}

// Every refusal a request in flight CAUSES says the same thing, not only the
// guard's. The non-guard checks each wrap their refusals ONCE, at their
// boundary, in withContinuationExits. This table has one row per call site,
// each driven for real, and the scan below fails when a site is added without a
// row -- a claim of reach over a set of sites is checked against that set.
func TestEveryRefusalARequestInFlightCausesSaysSo(t *testing.T) {
	t.Parallel()

	const both = "; the continuation is running: " + checkpoint.InFlight
	rows := []struct {
		site  string // the function holding the withContinuationExits call
		drive func(t *testing.T) error
		cause string
	}{
		{"RecoverThread", func(t *testing.T) error {
			env, live := switchEnvWithLiveThread(t)
			failed, _ := runningContinuation(t, env.Couch.Threads, live)
			other := filepath.Join(t.TempDir(), "other.md")
			if err := os.WriteFile(other, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nanother handoff\n"), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := env.Couch.RecoverThread(context.Background(), failed.Address, other)
			return err
		}, "another continuation is unresolved"},
		{"prepareAbsentContinuation", func(t *testing.T) error {
			f := newContinuationFixture(t)
			c := f.env.Couch
			request := f.status.RequestID
			if _, err := c.Threads.AdvanceContinuation(f.source.Address, mustRevision(t, c, f.source.Address), checkpoint.Event{Kind: checkpoint.Begin, RequestID: request, Attempt: "start-3333333333333333"}); err != nil {
				t.Fatal(err)
			}
			f.env.Proc.Kill(f.source.Incarnations[0].PID)
			f.env.Artifacts.SetPairSession(f.source.Address, f.sessionName(t), false)
			// The source generation advanced with no ownership proof: the retained
			// request's admission refuses inside the retained branch.
			c.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
				return ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 5}, nil
			}
			_, err := c.RecoverThread(context.Background(), f.source.Address, "")
			return err
		}, "generation"},
		{"archiveContinuationVacant", func(t *testing.T) error {
			env, live := switchEnvWithLiveThread(t)
			failed, _ := runningContinuation(t, env.Couch.Threads, live)
			return env.Couch.archiveContinuationVacant(failed, RecoveryEvidence{Presence: PresencePresent})
		}, "continuation source or target session is still occupied"},
		{"validateContinuationWarm", func(t *testing.T) error {
			env, live := switchEnvWithLiveThread(t)
			failed, _ := runningContinuation(t, env.Couch.Threads, live)
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return false, nil }
			env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
				return ContinuationSource{Agent: "claude", Session: "some-other-session", LaunchOrdinal: 9}, nil
			}
			return env.Couch.validateContinuationWarm(context.Background(), failed)
		}, "warm session is neither"},
	}
	for _, row := range rows {
		t.Run(row.site, func(t *testing.T) {
			err := row.drive(t)
			if err == nil || !strings.Contains(err.Error(), row.cause) || !strings.Contains(err.Error(), both) {
				t.Fatalf("%s refusal must carry its cause %q and the in-flight wording: %v", row.site, row.cause, err)
			}
		})
	}

	sites := map[string]bool{}
	for _, row := range rows {
		sites[row.site] = true
	}
	calls := productionCallsTo(t, "withContinuationExits(")
	if len(calls) != len(rows) {
		t.Fatalf("withContinuationExits has %d production call sites %v but %d driven rows; add a row per site", len(calls), calls, len(rows))
	}
	for _, fn := range calls {
		if !sites[fn] {
			t.Errorf("withContinuationExits is called in %s, which no row drives", fn)
		}
	}
}

// productionCallsTo names the enclosing function of every non-test call of
// needle in this package (the definition itself excluded).
func productionCallsTo(t *testing.T, needle string) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		fn := ""
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "func ") {
				name := strings.TrimPrefix(line, "func ")
				if strings.HasPrefix(name, "(") {
					name = name[strings.Index(name, ")")+2:]
				}
				fn = name[:strings.IndexAny(name, "([")]
			}
			if strings.Contains(line, needle) && !strings.HasPrefix(line, "func ") {
				out = append(out, fn)
			}
		}
	}
	return out
}

func mustRevision(t *testing.T, c *Couch, address ThreadAddress) uint64 {
	t.Helper()
	r, err := c.Threads.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	return r.Revision
}

// Once a compaction didn't finish, its request is gone, so
// PublishContinuation's generation check has nothing to compare against: the
// source proof is what stops a dead writer's stale handoff from coming back,
// while the CURRENT session can compact again at once (#522).
func TestPublishAfterAMissAcceptsOnlyTheCurrentSource(t *testing.T) {
	t.Parallel()

	env, live := switchEnvWithLiveThread(t)
	failed, request := legacyFailedContinuation(t, env.Couch.Threads, live)
	current := ContinuationSource{Agent: "claude", Session: "pair-source", LaunchOrdinal: 3}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return current, nil }
	path := filepath.Join(t.TempDir(), "handoff.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nfresh handoff\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stale := ContinuationSource{Agent: "claude", Session: "pair-source", LaunchOrdinal: request.Source.LaunchOrdinal}
	if _, err := env.Couch.RequestContinuation(context.Background(), failed.Address, stale, path); err == nil || err.Error() != "continuation source generation is obsolete" {
		t.Fatalf("the missed request's own generation must stay refused: %v", err)
	}
	status, err := env.Couch.RequestContinuation(context.Background(), failed.Address, current, path)
	if err != nil || status.Phase != checkpoint.Pending {
		t.Fatalf("the current session could not compact again: %+v, %v", status, err)
	}
}

// PQ-1 (#522): a compaction that fails after the source was parked, with an
// incarnation still recorded, may have left the new conversation running.
// The line says so, and the guard against a second agent is the thread's
// incarnations, not the continuation: a cold start claim is refused, and
// relaunch parks that exact incarnation before it starts anything, so there
// is never a second one.
func TestAMissWhoseTargetMayExistNeverYieldsASecondAgent(t *testing.T) {
	t.Parallel()

	for _, parked := range []bool{false, true} {
		env, live := envWithLiveThread(t)
		running, request := runningContinuation(t, env.Couch.Threads, live)
		if parked {
			var err error
			// The source's retirement is recorded (here as its proved absence),
			// so an incarnation now can only be the new conversation's.
			absence := checkpoint.SourceAbsence{Session: request.Source.Session, LaunchOrdinal: request.Source.LaunchOrdinal, ObservedAt: env.Now, RecordRevision: running.Revision}
			running, err = env.Couch.Threads.AdvanceContinuation(running.Address, running.Revision, checkpoint.Event{Kind: checkpoint.SourceAbsent, RequestID: request.ID, Attempt: request.Attempt, SourceAbsence: &absence})
			if err != nil {
				t.Fatal(err)
			}
		}
		result, err := env.Couch.failContinuation(running, errors.New("abort started: the helper did not stop"))
		if err == nil || result.Missed == nil || result.Missed.TargetMayExist != parked {
			t.Fatalf("parked=%v: miss %+v, %v", parked, result.Missed, err)
		}
		if !parked {
			continue // before park the incarnation is the old conversation, still running
		}
		missed, err := env.Couch.Threads.GetThread(running.Address)
		if err != nil || len(missed.Incarnations) != 1 || continuationGuard(missed) != nil {
			t.Fatalf("after the miss: %+v, %v", missed, err)
		}
		profile := LaunchProfile{Agent: "claude", Argv: []string{}}
		if _, err := env.Couch.Threads.CommitStartClaim(missed.Address, missed.Revision, "repo", env.Now, StartEvent{Kind: StartClaimed, Nonce: "start-4444444444444444", Owner: SupervisorOwner{PID: 100, Identity: "owner"}, Profile: &profile}); err == nil || !strings.Contains(err.Error(), "incarnation") {
			t.Fatalf("a cold start beside a possibly live target: %v", err)
		}
		relaunched, err := env.Couch.Relaunch(context.Background(), missed.Address)
		if err != nil || relaunched.Outcome != Relaunched {
			t.Fatalf("relaunch = %+v, %v", relaunched.Outcome, err)
		}
		after, err := env.Couch.Threads.GetThread(missed.Address)
		if err != nil || len(after.Incarnations) != 1 || after.Incarnations[0].PID == 42 {
			t.Fatalf("relaunch left %+v, %v; want exactly the one new incarnation", after.Incarnations, err)
		}
	}
}

func continuationRefuses(operation string) bool {
	switch operation {
	case "relaunch", "switch-agent", "prepare-switch-agent", "start":
		return true
	}
	return false
}
