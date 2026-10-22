package couchcore

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Repeated observations of one stage must not reset the visible elapsed time.
// This in-memory observer cannot admit an effect or alter thread state.
func trackStartupProgress(parent context.Context) (context.Context, func(error) error) {
	var mu sync.Mutex
	stage, started := "preparing startup", time.Now()
	ctx := WithOperationProgress(parent, func(text string) {
		mu.Lock()
		if text != stage {
			stage, started = text, time.Now()
		}
		mu.Unlock()
		reportProgress(parent, "%s", text)
	})
	return ctx, func(cause error) error {
		mu.Lock()
		current, elapsed := stage, time.Since(started)
		mu.Unlock()
		return fmt.Errorf("%s after %s: %w", current, elapsed.Round(time.Second), cause)
	}
}
