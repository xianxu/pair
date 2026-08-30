package pairlifecycletest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

func TestFakePairLifecycleStateTransitions(t *testing.T) {
	t.Parallel()
	fake := New(time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC))
	request := testRequest()
	fake.SetSession(request.Session, true)
	if err := fake.PrepareRequest(request); err != nil {
		t.Fatal(err)
	}
	if err := fake.DeliverTrigger(request); err == nil {
		t.Fatal("prepared request was trigger authority")
	}
	if err := fake.CommitRequest(request); err != nil {
		t.Fatal(err)
	}
	wrong := request
	wrong.Session = "pair-other"
	if err := fake.DeliverTrigger(wrong); err == nil {
		t.Fatal("wrong exact session accepted")
	}
	if err := fake.DeliverTrigger(request); err != nil {
		t.Fatal(err)
	}
	if err := fake.DeliverTrigger(request); err != nil {
		t.Fatal(err)
	}
	if got := fake.DeliveredTriggers(request); got != 2 {
		t.Fatalf("delivered triggers=%d", got)
	}
	if !fake.InProgress(request) {
		t.Fatal("attempt has no process-local owner")
	}

	result := pairlifecycle.RunCleanup(context.Background(), pairlifecycle.CleanupCouch, fake)
	if result.Outcome != pairlifecycle.CompletionSuccess {
		t.Fatalf("cleanup result=%#v", result)
	}
	wantTrace := []pairlifecycle.CleanupStage{
		pairlifecycle.StageSessionQuiescence,
		pairlifecycle.StageEditorReap,
		pairlifecycle.StageScrollbackPreserve,
		pairlifecycle.StageSidecarCleanup,
		pairlifecycle.StagePollerCleanup,
		pairlifecycle.StageCmuxCleanup,
	}
	if got := fake.EffectiveTrace(); !reflect.DeepEqual(got, wantTrace) {
		t.Fatalf("effective trace=%v, want %v", got, wantTrace)
	}
	if fake.SessionPresent(request.Session) {
		t.Fatal("quiescence left exact session present")
	}
	if second := pairlifecycle.RunCleanup(context.Background(), pairlifecycle.CleanupCouch, fake); second.Outcome != pairlifecycle.CompletionSuccess {
		t.Fatalf("idempotent cleanup=%#v", second)
	}
	if got := fake.EffectiveTrace(); !reflect.DeepEqual(got, wantTrace) {
		t.Fatalf("idempotent cleanup repeated effects: %v", got)
	}

	if err := fake.PrepareCompletion(request, result); err != nil {
		t.Fatal(err)
	}
	fake.Restart()
	if fake.InProgress(request) {
		t.Fatal("restart retained process-local ownership")
	}
	if state := fake.CompletionState(request); state != Prepared {
		t.Fatalf("restart lost prepared completion: %q", state)
	}
	if err := fake.CommitCompletion(request, result); err != nil {
		t.Fatal(err)
	}
	different := result
	different.Outcome = pairlifecycle.CompletionFailure
	if err := fake.CommitCompletion(request, different); err == nil {
		t.Fatal("immutable completion was replaced")
	}
	if state := fake.CompletionState(request); state != Committed {
		t.Fatalf("completion state=%q", state)
	}
}

func TestFakePairLifecycleStageFailureIsRetryable(t *testing.T) {
	t.Parallel()
	fake := New(time.Now())
	request := testRequest()
	fake.SetSession(request.Session, true)
	if err := fake.CommitRequest(request); err != nil {
		t.Fatal(err)
	}
	if err := fake.DeliverTrigger(request); err != nil {
		t.Fatal(err)
	}
	fake.FailStage(pairlifecycle.StageEditorReap, errors.New("editor busy"))
	first := pairlifecycle.RunCleanup(context.Background(), pairlifecycle.CleanupDirect, fake)
	if first.Outcome != pairlifecycle.CompletionFailure {
		t.Fatalf("first=%#v", first)
	}
	fake.FailStage(pairlifecycle.StageEditorReap, nil)
	second := pairlifecycle.RunCleanup(context.Background(), pairlifecycle.CleanupDirect, fake)
	if second.Outcome != pairlifecycle.CompletionSuccess {
		t.Fatalf("second=%#v", second)
	}
	count := 0
	for _, stage := range fake.AttemptTrace() {
		if stage == pairlifecycle.StageEditorReap {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("failed stage attempts=%d, want retry", count)
	}
}

func TestQuitLifecycleConformanceScenarioFake(t *testing.T) {
	fake := New(time.Unix(100, 0).UTC())
	request := testRequest()
	fake.SetSession(request.Session, true)
	trace, err := RunConformanceScenario(context.Background(), fake, request)
	if err != nil {
		t.Fatal(err)
	}
	want := EffectTrace{
		"request:prepared", "restart:prepared-request", "request:committed", "restart:committed-request",
		"trigger:delivered", "trigger:delivered",
		"cleanup:session-quiescence", "cleanup:editor-reap", "cleanup:preserve-scrollback",
		"cleanup:sidecar-cleanup", "cleanup:poller-cleanup", "cleanup:cmux-cleanup",
		"completion:prepared", "restart:prepared-completion", "completion:committed",
	}
	if !reflect.DeepEqual(trace, want) {
		t.Fatalf("trace = %q, want %q", trace, want)
	}
}

func testRequest() pairlifecycle.QuitRequest {
	return pairlifecycle.QuitRequest{
		SchemaVersion: pairlifecycle.SchemaVersion,
		Identity: pairlifecycle.Identity{
			Nonce: "nonce-1", RepoScope: "scope", Tag: "work", PID: 42, ProcessIdentity: "start:1",
		},
		Attempt: 1, Session: "pair-work", Mode: pairlifecycle.CleanupPreserveScrollback,
		CompletionKey: "quit-completion-1",
	}
}
