package couchcmd

import (
	"context"
	"errors"
	"fmt"
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
	lifecycleQueue := newLifecycleRemoteQueue(console, c)
	run := func(key, op, target string, opts couchcore.LiveRestartOptions, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		// The admission note and readiness wait are written on the queue
		// goroutine by prepare and read by finished after the job; the queue
		// orders the two.
		var note string
		var await func(context.Context) error
		return lifecycleQueue.EnqueueForReference(key, op, target, func(ctx context.Context) (couchcore.OperationCall, error) {
			ctx = couchcore.WithOperationProgress(ctx, progress)
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
		}, started, withAdmissionNote(&note, finished), &await)
	}
	ops := newSlotOperations(func(key, op, target string, opts couchcore.LiveRestartOptions, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		return run(key, op, target, opts, nil, started, progress, finished)
	}, func(ctx context.Context) ([]couchcore.RepositoryName, error) {
		if c.Threads == nil {
			return nil, errors.New("no thread store")
		}
		return c.Threads.RepositoryNamesContext(ctx)
	}, time.Now)
	ops.preparePark = func(ctx context.Context, target string, repository couchcore.RepositoryName, guard func(context.Context) error) (slotOperationRunner, error) {
		snapshot := console.ResourceInventory()
		if !snapshot.InventoryKnown {
			return nil, errors.New("recovery inventory is unavailable; refresh the switcher inventory, inspect the row and retry")
		}
		ref, _, err := couchcore.ParseWorkspaceReference(target)
		if err != nil {
			return nil, err
		}
		row, err := couchcore.SelectCachedSlotRow(snapshot.Rows, repository.Key, ref.Number)
		if err != nil {
			return nil, fmt.Errorf("%w; refresh the switcher inventory, inspect the row and retry", err)
		}
		bound := couchcore.OperationCall{Name: couchcore.OpRecoverPark, Args: couchcore.ActorOperationArgs(row, couchcore.OpRecoverPark)}
		check := func(ctx context.Context) error {
			if guard != nil {
				if err := guard(ctx); err != nil {
					return err
				}
			}
			current, _, err := c.PrepareSlotOperation(ctx, couchcore.OpRecoverPark, target, couchcore.LiveRestartOptions{})
			if err != nil {
				return err
			}
			return validateParkRecoveryTarget(bound, current)
		}
		return func(key, op, _ string, _ couchcore.LiveRestartOptions, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
			return lifecycleQueue.EnqueueForReference(key, op, target, func(ctx context.Context) (couchcore.OperationCall, error) {
				ctx = couchcore.WithOperationProgress(ctx, progress)
				if err := check(ctx); err != nil {
					return couchcore.OperationCall{}, err
				}
				call := bound
				call.Context = ctx
				call.TypedPayload = hubMutationPayload{guard: check}
				return call, nil
			}, started, finished)
		}, nil
	}
	ops.reference = c.RecoveryConfirmationReference
	ops.recovery = func(key, actor string, request couchmessage.Request, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		return lifecycleQueue.EnqueueForReference(key, request.Op, request.Target, func(ctx context.Context) (couchcore.OperationCall, error) {
			ctx = couchcore.WithOperationProgress(ctx, progress)
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
	ops.candidate = func(key, target, candidate string, started func(), progress couchcore.OperationProgress, finished func(any, error), guards ...func(context.Context) error) error {
		return lifecycleQueue.EnqueueForReference(key, "resume", target, func(ctx context.Context) (couchcore.OperationCall, error) {
			ctx = couchcore.WithOperationProgress(ctx, progress)
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

// A settled transaction is an idempotent repeat; a different conversation or
// newly opened park is not the transaction the operator confirmed.
func validateParkRecoveryTarget(bound, current couchcore.OperationCall) error {
	if current.Args["repo-scope"] != bound.Args["repo-scope"] || current.Args["tag"] != bound.Args["tag"] ||
		(current.Args["park-nonce"] != "" && current.Args["park-nonce"] != bound.Args["park-nonce"]) {
		return errors.New("park recovery target changed; inspect the current thread before recovering")
	}
	return nil
}
