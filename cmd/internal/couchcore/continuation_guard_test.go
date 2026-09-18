package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestContinuationBlocksCompetingTransitions(t *testing.T) {
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

// The reported symptom (#280): a live thread whose handoff the operator took
// over could never be relaunched, because the retained FAILED request refused
// it and named only retry. The refusal now names both exits, and once the
// request is dismissed the relaunch completes -- the real outcome, not merely
// the absence of the guard's error.
func TestFailedContinuationRelaunchesOnceDismissed(t *testing.T) {
	env, live := envWithLiveThread(t)
	failed, request := failedContinuation(t, env.Couch.Threads, live)
	_, err := env.Couch.Relaunch(context.Background(), failed.Address)
	if err == nil || !strings.Contains(err.Error(), "Retry continuation re-delivers it and Dismiss continuation drops it") ||
		!strings.Contains(err.Error(), "couch --internal dismiss-continuation") {
		t.Fatalf("relaunch refusal must name both exits: %v", err)
	}
	if _, err := env.Couch.DismissContinuation(context.Background(), failed.Address, request.ID); err != nil {
		t.Fatal(err)
	}
	result, err := env.Couch.Relaunch(context.Background(), failed.Address)
	if err != nil || result.Outcome != Relaunched {
		t.Fatalf("Relaunch after dismissal = %+v, %v", result.Outcome, err)
	}
}

// ContinuationRefuses is the switcher's copy of the guard's reach. Every row
// action is driven through the PRODUCTION dispatcher on a live thread holding a
// failed request: a listed operation must be refused BY THE GUARD (its phrase),
// and an unlisted one must SUCCEED -- not merely fail some other way, which
// would read as "not guarded" when it never got that far. Every declared row
// action is either driven here or exempt with a reason, so a new one cannot
// arrive unclassified (#280).
func TestContinuationRefusesMatchesTheGuardForEveryRowAction(t *testing.T) {
	type drive struct {
		args map[string]string
		cold bool // retire the helper first: the guard reaches only a COLD resume
	}
	driven := map[string]drive{
		"relaunch":             {args: map[string]string{}},
		"prepare-switch-agent": {args: map[string]string{"agent": "codex"}}, // SwitchAgent re-runs this preview (switchagent.go)
		"park":                 {args: map[string]string{}},
		"detach":               {args: map[string]string{}},
		"name":                 {args: map[string]string{"name": "renamed"}},
		"describe":             {args: map[string]string{}},
		"resume":               {args: map[string]string{}, cold: true},
	}
	rowActionDrivenAs := map[string]string{"switch-agent": "prepare-switch-agent"}
	exempt := map[string]string{
		"retry-continuation":   "an exit from the failed request, not an operation it gates",
		"dismiss-continuation": "an exit from the failed request, not an operation it gates",
		"archive":              "never offered on a live row; its own admission is archiveContinuationVacant",
		"recover-thread":       "offered only on recovery rows, never composed",
		"recover-checkpoint":   "offered only on recovery rows, never composed",
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
			if d.cold {
				var err error
				live, err = env.Couch.Threads.updateExistingThread(live.Address, live.Revision, func(r *ThreadRecord) error { r.Incarnations = nil; return nil })
				if err != nil {
					t.Fatal(err)
				}
			}
			if name == "detach" {
				// Detach SIGTERMs the helper and waits for it, as newDetachFixture
				// models; without this the fake never exits and detach fails for a
				// reason that has nothing to do with the guard.
				env.Proc.DiesOn = map[int]os.Signal{42: syscall.SIGTERM}
			}
			failed, request := failedContinuation(t, env.Couch.Threads, live)
			guardPrefix := "continuation " + request.ID + " is failed; " // the guard's own words, not the wrapper's
			args := map[string]string{"repo-scope": failed.Address.RepoScope, "tag": string(failed.Address.Tag)}
			for k, v := range d.args {
				args[k] = v
			}
			_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch), DirectStore: DirectStoreExecutor(env.Couch)},
				OperationCall{Name: name, Args: args, Implicit: true, Context: context.Background()})
			if ContinuationRefuses(name) {
				if err == nil || !strings.Contains(err.Error(), guardPrefix) {
					t.Fatalf("%s is listed in ContinuationRefuses but the guard did not refuse it: %v", name, err)
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
				t.Fatalf("%s is not listed in ContinuationRefuses but did not succeed: %v", name, err)
			}
		})
	}
	// start is not a row action: the guard's reach there is the store's cold
	// start claim, driven directly.
	t.Run("start", func(t *testing.T) {
		env, live := switchEnvWithLiveThread(t)
		failed, request := failedContinuation(t, env.Couch.Threads, live)
		cold, err := env.Couch.Threads.updateExistingThread(failed.Address, failed.Revision, func(r *ThreadRecord) error { r.Incarnations = nil; return nil })
		if err != nil {
			t.Fatal(err)
		}
		profile := LaunchProfile{Agent: "claude", Argv: []string{}}
		_, err = env.Couch.Threads.CommitStartClaim(cold.Address, cold.Revision, "repo", env.Now, StartEvent{Kind: StartClaimed, Nonce: "start-2222222222222222", Owner: SupervisorOwner{PID: 100, Identity: "owner"}, Profile: &profile})
		if !ContinuationRefuses("start") || err == nil || !strings.Contains(err.Error(), "continuation "+request.ID+" is failed; ") {
			t.Fatalf("cold start claim must be refused by the guard, and listed: %v", err)
		}
	})
}

// Every refusal a retained failed request CAUSES names both exits, not only
// the guard's. The non-guard checks each wrap their refusals ONCE, at their
// boundary, in withContinuationExits. This table has one row per call site,
// each driven for real, and the scan below fails when a site is added without a
// row -- a claim of reach over a set of sites is checked against that set.
func TestEveryRefusalARetainedRequestCausesNamesBothExits(t *testing.T) {
	const both = "; the retained continuation is failed: Retry continuation re-delivers it and Dismiss continuation drops it"
	rows := []struct {
		site  string // the function holding the withContinuationExits call
		drive func(t *testing.T) error
		cause string
	}{
		{"RecoverThread", func(t *testing.T) error {
			env, live := switchEnvWithLiveThread(t)
			failed, _ := failedContinuation(t, env.Couch.Threads, live)
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
			r, err := c.Threads.AdvanceContinuation(f.source.Address, mustRevision(t, c, f.source.Address), checkpoint.Event{Kind: checkpoint.Begin, RequestID: request, Attempt: "start-3333333333333333"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Threads.AdvanceContinuation(r.Address, r.Revision, checkpoint.Event{Kind: checkpoint.Fail, RequestID: request, Attempt: "start-3333333333333333", Failure: "interrupted"}); err != nil {
				t.Fatal(err)
			}
			f.env.Proc.Kill(f.source.Incarnations[0].PID)
			f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
			// The source generation advanced with no ownership proof: the retained
			// request's admission refuses inside the retained branch.
			c.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) {
				return ContinuationSource{Agent: "claude", Session: "pair-exact", LaunchOrdinal: 5}, nil
			}
			_, err = c.RecoverThread(context.Background(), f.source.Address, "")
			return err
		}, "generation"},
		{"archiveContinuationVacant", func(t *testing.T) error {
			env, live := switchEnvWithLiveThread(t)
			failed, _ := failedContinuation(t, env.Couch.Threads, live)
			return env.Couch.archiveContinuationVacant(failed, RecoveryEvidence{Presence: PresencePresent})
		}, "continuation source or target session is still occupied"},
		{"validateContinuationWarm", func(t *testing.T) error {
			env, live := switchEnvWithLiveThread(t)
			failed, _ := failedContinuation(t, env.Couch.Threads, live)
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
				t.Fatalf("%s refusal must carry its cause %q and both exits: %v", row.site, row.cause, err)
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

// Once a dismissal deletes the request, PublishContinuation's generation check
// has nothing to compare against, so the source proof is what stops the dead
// writer's stale handoff from coming back -- while the CURRENT session, which
// the failed request had blocked from publishing at all, can hand off again.
func TestPublishAfterDismissalAcceptsOnlyTheCurrentSource(t *testing.T) {
	env, live := switchEnvWithLiveThread(t)
	failed, request := failedContinuation(t, env.Couch.Threads, live)
	current := ContinuationSource{Agent: "claude", Session: "pair-source", LaunchOrdinal: 3}
	env.Couch.ContinuationSource = func(context.Context, ThreadAddress) (ContinuationSource, error) { return current, nil }
	path := filepath.Join(t.TempDir(), "handoff.md")
	if err := os.WriteFile(path, []byte("---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nfresh handoff\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.RequestContinuation(context.Background(), failed.Address, current, path); err == nil || !strings.Contains(err.Error(), "Dismiss continuation drops it") {
		t.Fatalf("a failed request must block publishing and name both exits: %v", err)
	}
	if _, err := env.Couch.DismissContinuation(context.Background(), failed.Address, request.ID); err != nil {
		t.Fatal(err)
	}
	stale := ContinuationSource{Agent: "claude", Session: "pair-source", LaunchOrdinal: request.Source.LaunchOrdinal}
	if _, err := env.Couch.RequestContinuation(context.Background(), failed.Address, stale, path); err == nil || err.Error() != "continuation source generation is obsolete" {
		t.Fatalf("the dismissed request's own generation must stay refused: %v", err)
	}
	status, err := env.Couch.RequestContinuation(context.Background(), failed.Address, current, path)
	if err != nil || status.Phase != checkpoint.Pending {
		t.Fatalf("the current session could not hand off after dismissal: %+v, %v", status, err)
	}
}
