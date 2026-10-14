package couchcmd

import (
	"context"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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
	ops.guardedRun = run
	return ops
}
