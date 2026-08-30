package pairlifecycle

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRunCleanupStageMatrix(t *testing.T) {
	t.Parallel()
	stages := []CleanupStage{
		StageSessionQuiescence,
		StageEditorReap,
		StageScrollbackPreserve,
		StageSidecarCleanup,
		StagePollerCleanup,
		StageCmuxCleanup,
	}
	for _, failed := range stages {
		failed := failed
		t.Run(string(failed), func(t *testing.T) {
			t.Parallel()
			ops := newCleanupOps()
			ops.failures[failed] = errors.New("injected " + string(failed))
			result := RunCleanup(context.Background(), CleanupDirect, ops)
			if result.Outcome != CompletionFailure || len(result.Failures) != 1 {
				t.Fatalf("result=%#v", result)
			}
			if result.Failures[0].Stage != failed || result.Failures[0].Code != FailureCleanupFailed {
				t.Fatalf("failure=%#v", result.Failures[0])
			}
			wantTrace := stages
			if failed == StageSessionQuiescence {
				wantTrace = stages[:1]
			}
			if !reflect.DeepEqual(ops.trace, wantTrace) {
				t.Fatalf("trace=%v, want %v", ops.trace, wantTrace)
			}
		})
	}

	ops := newCleanupOps()
	ops.failures[StageEditorReap] = errors.New("editor")
	ops.failures[StageSidecarCleanup] = errors.New("sidecar")
	result := RunCleanup(context.Background(), CleanupDirect, ops)
	if got := []CleanupStage{result.Failures[0].Stage, result.Failures[1].Stage}; !reflect.DeepEqual(got, []CleanupStage{StageEditorReap, StageSidecarCleanup}) {
		t.Fatalf("failure order=%v", got)
	}

	success := RunCleanup(context.Background(), CleanupDirect, newCleanupOps())
	if success.Outcome != CompletionSuccess || len(success.Failures) != 0 || success.CompletedAt.IsZero() {
		t.Fatalf("success=%#v", success)
	}
}

func TestRunCleanupIntentPolicy(t *testing.T) {
	t.Parallel()
	for _, intent := range []CleanupIntent{CleanupDirect, CleanupCouch} {
		ops := newCleanupOps()
		result := RunCleanup(context.Background(), intent, ops)
		if result.Outcome != CompletionSuccess {
			t.Fatalf("intent %q result=%#v", intent, result)
		}
		if !reflect.DeepEqual(ops.intents, []CleanupIntent{intent}) {
			t.Fatalf("intent %q preserve calls=%v", intent, ops.intents)
		}
	}
}

func TestRunCleanupClassifiesContextExpiry(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ops := newCleanupOps()
	result := RunCleanup(ctx, CleanupDirect, ops)
	if result.Outcome != CompletionFailure || len(result.Failures) != 1 || result.Failures[0].Stage != StageSessionQuiescence || result.Failures[0].Code != FailureTimeout {
		t.Fatalf("result=%#v", result)
	}
	if len(ops.trace) != 0 {
		t.Fatalf("cancelled cleanup performed effects: %v", ops.trace)
	}
}

type cleanupOps struct {
	now      time.Time
	trace    []CleanupStage
	intents  []CleanupIntent
	failures map[CleanupStage]error
}

func newCleanupOps() *cleanupOps {
	return &cleanupOps{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC), failures: map[CleanupStage]error{}}
}

func (o *cleanupOps) run(stage CleanupStage) error {
	o.trace = append(o.trace, stage)
	return o.failures[stage]
}
func (o *cleanupOps) QuiesceSession(context.Context) error { return o.run(StageSessionQuiescence) }
func (o *cleanupOps) ReapEditors(context.Context) error    { return o.run(StageEditorReap) }
func (o *cleanupOps) PreserveScrollback(_ context.Context, intent CleanupIntent) error {
	o.intents = append(o.intents, intent)
	return o.run(StageScrollbackPreserve)
}
func (o *cleanupOps) CleanupSidecars(context.Context) error { return o.run(StageSidecarCleanup) }
func (o *cleanupOps) CleanupPoller(context.Context) error   { return o.run(StagePollerCleanup) }
func (o *cleanupOps) CleanupCmux(context.Context) error     { return o.run(StageCmuxCleanup) }
func (o *cleanupOps) Now() time.Time                        { return o.now }
