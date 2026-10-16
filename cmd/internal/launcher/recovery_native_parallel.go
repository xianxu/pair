package launcher

import (
	"context"
	"sync"
)

// runOrderedNativeProofs overlaps independent observations while keeping all
// ownership decisions and errors in wrapper order. Workers own only their result;
// consume runs serially in the caller and may update the phase's shared state.
func runOrderedNativeProofs(ctx context.Context, wrappers []recoveryNativeWrapper, prove func(context.Context, recoveryNativeWrapper) (recoveryNativeProof, error), consume func(recoveryNativeWrapper, recoveryNativeProof) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(wrappers) == 0 {
		return nil
	}
	workerCtx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
	}()

	type result struct {
		proof recoveryNativeProof
		err   error
	}
	results := make([]chan result, len(wrappers))
	jobs := make(chan int, len(wrappers))
	for i := range wrappers {
		results[i] = make(chan result, 1)
		jobs <- i
	}
	close(jobs)
	for range min(4, len(wrappers)) {
		workers.Go(func() {
			for i := range jobs {
				if workerCtx.Err() != nil {
					return
				}
				proof, err := prove(workerCtx, wrappers[i])
				// Each index has one producer and enough room even if the caller stops
				// consuming. Cancellation therefore never leaves a worker blocked here.
				results[i] <- result{proof: proof, err: err}
			}
		})
	}
	for i, wrapper := range wrappers {
		var next result
		select {
		case <-ctx.Done():
			return ctx.Err()
		case next = <-results[i]:
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if next.err != nil {
			return next.err
		}
		if err := consume(wrapper, next.proof); err != nil {
			return err
		}
	}
	return ctx.Err()
}
