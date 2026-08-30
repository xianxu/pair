package pairlifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type CleanupStage string

const (
	StageSessionQuiescence  CleanupStage = "session-quiescence"
	StageEditorReap         CleanupStage = "editor-reap"
	StageScrollbackPreserve CleanupStage = "preserve-scrollback"
	StageSidecarCleanup     CleanupStage = "sidecar-cleanup"
	StagePollerCleanup      CleanupStage = "poller-cleanup"
	StageCmuxCleanup        CleanupStage = "cmux-cleanup"
)

type CleanupIntent string

const (
	CleanupDirect CleanupIntent = "direct"
	CleanupCouch  CleanupIntent = "couch"
)

type StageFailure struct {
	Stage CleanupStage
	Code  FailureCode
	Err   error
}

type CleanupResult struct {
	Outcome     CompletionOutcome
	Failures    []StageFailure
	CompletedAt time.Time
}

// QuitLifecycleOps is the narrow ordered effect seam for Pair's full quit.
// PreserveScrollback receives the entry intent so direct policy can retain its
// prompt while Couch can later force preservation without a second cleanup.
type QuitLifecycleOps interface {
	QuiesceSession(context.Context) error
	ReapEditors(context.Context) error
	PreserveScrollback(context.Context, CleanupIntent) error
	CleanupSidecars(context.Context) error
	CleanupPoller(context.Context) error
	CleanupCmux(context.Context) error
	Now() time.Time
}

// RunCleanup executes Pair's single full-quit sequence. Quiescence gates every
// destructive later stage; once it succeeds, independent cleanup stages are
// all attempted and their failures accumulate in stable order.
func RunCleanup(ctx context.Context, intent CleanupIntent, ops QuitLifecycleOps) CleanupResult {
	result := CleanupResult{Outcome: CompletionFailure}
	if ops == nil {
		result.Failures = []StageFailure{{Stage: StageSessionQuiescence, Code: FailureCleanupFailed, Err: errors.New("nil quit lifecycle operations")}}
		return result
	}
	result.CompletedAt = ops.Now()
	if intent != CleanupDirect && intent != CleanupCouch {
		result.Failures = []StageFailure{{Stage: StageSessionQuiescence, Code: FailureCleanupFailed, Err: fmt.Errorf("unknown cleanup intent %q", intent)}}
		return result
	}
	if err := ctx.Err(); err != nil {
		result.Failures = []StageFailure{{Stage: StageSessionQuiescence, Code: FailureTimeout, Err: err}}
		return result
	}
	if err := ops.QuiesceSession(ctx); err != nil {
		result.Failures = []StageFailure{stageFailure(StageSessionQuiescence, err)}
		result.CompletedAt = ops.Now()
		return result
	}

	stages := []struct {
		stage CleanupStage
		run   func(context.Context) error
	}{
		{StageEditorReap, ops.ReapEditors},
		{StageScrollbackPreserve, func(ctx context.Context) error { return ops.PreserveScrollback(ctx, intent) }},
		{StageSidecarCleanup, ops.CleanupSidecars},
		{StagePollerCleanup, ops.CleanupPoller},
		{StageCmuxCleanup, ops.CleanupCmux},
	}
	for _, stage := range stages {
		if err := stage.run(ctx); err != nil {
			result.Failures = append(result.Failures, stageFailure(stage.stage, err))
		}
	}
	result.CompletedAt = ops.Now()
	if len(result.Failures) == 0 {
		result.Outcome = CompletionSuccess
	}
	return result
}

func stageFailure(stage CleanupStage, err error) StageFailure {
	code := FailureCleanupFailed
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = FailureTimeout
	}
	return StageFailure{Stage: stage, Code: code, Err: err}
}
