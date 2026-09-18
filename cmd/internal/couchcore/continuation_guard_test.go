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
	const guardPhrase = "Dismiss continuation drops it"
	driven := map[string]map[string]string{
		"relaunch":             {},
		"prepare-switch-agent": {"agent": "codex"}, // the switcher's switch-agent begins with this preview
		"park":                 {},
		"detach":               {},
		"name":                 {"name": "renamed"},
		"describe":             {},
	}
	rowActionDrivenAs := map[string]string{"switch-agent": "prepare-switch-agent"}
	exempt := map[string]string{
		"retry-continuation":   "an exit from the failed request, not an operation it gates",
		"dismiss-continuation": "an exit from the failed request, not an operation it gates",
		"resume":               "never offered on a live row; a COLD resume is gated (see TestContinuationBlocksCompetingTransitions)",
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
	for name, extra := range driven {
		t.Run(name, func(t *testing.T) {
			env, live := switchEnvWithLiveThread(t)
			if name == "detach" {
				// Detach SIGTERMs the helper and waits for it, as newDetachFixture
				// models; without this the fake never exits and detach fails for a
				// reason that has nothing to do with the guard.
				env.Proc.DiesOn = map[int]os.Signal{42: syscall.SIGTERM}
			}
			failed, _ := failedContinuation(t, env.Couch.Threads, live)
			args := map[string]string{"repo-scope": failed.Address.RepoScope, "tag": string(failed.Address.Tag)}
			for k, v := range extra {
				args[k] = v
			}
			_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch), DirectStore: DirectStoreExecutor(env.Couch)},
				OperationCall{Name: name, Args: args, Implicit: true, Context: context.Background()})
			if ContinuationRefuses(name) {
				if err == nil || !strings.Contains(err.Error(), guardPhrase) {
					t.Fatalf("%s is listed in ContinuationRefuses but the guard did not refuse it: %v", name, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s is not listed in ContinuationRefuses but did not succeed: %v", name, err)
			}
		})
	}
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
	if _, err := env.Couch.RequestContinuation(context.Background(), failed.Address, current, path); err == nil || !strings.Contains(err.Error(), "retry it (re-deliver) or dismiss it (drop it)") {
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
