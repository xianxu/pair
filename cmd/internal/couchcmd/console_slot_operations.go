package couchcmd

import (
	"context"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// consoleSlotOperations runs slot operations on the console's queue;
// PrepareSlotOperation runs on it, so admission is judged against the
// inventory at execution time, and the queue key resolves repository names
// from the thread store.
func consoleSlotOperations(console *couchtty.Console, c *couchcore.Couch, probe *liveRestartProbe) *slotOperations {
	run := func(key, op, target string, opts couchcore.LiveRestartOptions, guard func(context.Context) error, started func(), finished func(any, error)) error {
		// The admission note and readiness wait are written on the queue
		// goroutine by prepare and read by finished after the job; the queue
		// orders the two.
		var note string
		var await func(context.Context) error
		return console.EnqueueRemoteOperation(key, op, func(ctx context.Context) (couchcore.OperationCall, error) {
			if guard != nil {
				if err := guard(ctx); err != nil {
					return couchcore.OperationCall{}, err
				}
			}
			call, n, err := c.PrepareSlotOperation(ctx, op, target, opts)
			note = n
			if err == nil {
				await = probe.readinessAfter(op, call.Args)
				if guard != nil {
					call.TypedPayload = hubMutationPayload{original: call.TypedPayload, guard: guard}
				}
			}
			return call, err
		}, started, awaitReadiness(&await, withAdmissionNote(&note, finished)))
	}
	ops := newSlotOperations(func(key, op, target string, opts couchcore.LiveRestartOptions, started func(), finished func(any, error)) error {
		return run(key, op, target, opts, nil, started, finished)
	}, func(ctx context.Context) ([]couchcore.RepositoryName, error) {
		if c.Threads == nil {
			return nil, errors.New("no thread store")
		}
		return c.Threads.RepositoryNamesContext(ctx)
	}, time.Now)
	ops.reference = c.RecoveryConfirmationReference
	ops.recovery = func(key, actor string, request couchmessage.Request, guard func(context.Context) error, started func(), finished func(any, error)) error {
		return console.EnqueueRemoteOperation(key, request.Op, func(ctx context.Context) (couchcore.OperationCall, error) {
			ctx = couchcore.WithRecoveryConfirmer(ctx, actor)
			if guard != nil {
				if err := guard(ctx); err != nil {
					return couchcore.OperationCall{}, err
				}
			}
			var call couchcore.OperationCall
			if request.Op == "recovery-confirmation" {
				ref, err := c.RecoveryConfirmationReference(ctx, request.RecoveryConfirmation)
				if err != nil {
					return call, err
				}
				if ref != request.Target {
					return call, errors.New("confirmation target changed")
				}
				call = couchcore.OperationCall{Name: request.Op, Args: map[string]string{"token": request.RecoveryConfirmation}, Context: ctx, Implicit: true}
			} else if request.CandidateKey != "" {
				resolved, err := recoveryTargetForReference(ctx, c, request.Target)
				if err != nil {
					return call, err
				}
				args := map[string]string{"ref": request.Target, "candidate": request.CandidateKey, "repo-scope": resolved.Address.RepoScope}
				if resolved.Path != "" {
					args["path"] = resolved.Path
				} else {
					args["tag"] = string(resolved.Address.Tag)
				}
				call = couchcore.OperationCall{Name: "resume", Args: args, Context: ctx, Implicit: true}
			} else {
				var err error
				call, _, err = c.PrepareSlotOperation(ctx, request.Op, request.Target, couchcore.LiveRestartOptions{SameBinary: request.SameBinary, ForceUnknown: request.ForceUnknown})
				if err != nil {
					return call, err
				}
				if call.Args["warm-only"] != "true" {
					call.Args["ref"] = request.Target
				}
			}
			if guard != nil {
				call.TypedPayload = hubMutationPayload{original: call.TypedPayload, guard: guard}
			}
			return call, nil
		}, started, finished)
	}
	ops.guardedRun = run
	ops.preview = func(ctx context.Context, target string) (couchcore.RecoveryPreview, error) {
		resolved, err := recoveryTargetForReference(ctx, c, target)
		if err != nil {
			return couchcore.RecoveryPreview{}, err
		}
		return c.ResumeCandidates(ctx, resolved)
	}
	ops.candidate = func(key, target, candidate string, started func(), finished func(any, error), guards ...func(context.Context) error) error {
		return console.EnqueueRemoteOperation(key, "resume", func(ctx context.Context) (couchcore.OperationCall, error) {
			var guard func(context.Context) error
			if len(guards) > 0 {
				guard = guards[0]
				if err := guard(ctx); err != nil {
					return couchcore.OperationCall{}, err
				}
			}
			resolved, err := recoveryTargetForReference(ctx, c, target)
			if err != nil {
				return couchcore.OperationCall{}, err
			}
			args := map[string]string{"candidate": candidate, "repo-scope": resolved.Address.RepoScope}
			if resolved.Path != "" {
				args["path"] = resolved.Path
			} else {
				args["tag"] = string(resolved.Address.Tag)
			}
			call := couchcore.OperationCall{Name: "resume", Args: args, Implicit: true, Context: ctx}
			if guard != nil {
				call.TypedPayload = hubMutationPayload{guard: guard}
			}
			return call, nil
		}, started, finished)
	}
	return ops
}
