package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestStartupFailureNamesConsumingStageAndElapsed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var projected []string
		ctx := WithOperationProgress(context.Background(), func(s string) { projected = append(projected, s) })
		ctx, describe := trackStartupProgress(ctx)
		reportProgress(ctx, "waiting for agent readiness")
		// Observe this timer in testing/synctest virtual time; no wall-clock guess.
		<-time.After(20 * time.Second)
		reportProgress(ctx, "waiting for agent readiness")
		// Observe this timer in testing/synctest virtual time; no wall-clock guess.
		<-time.After(5 * time.Second)
		err := describe(context.DeadlineExceeded)
		if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "waiting for agent readiness after 25s") {
			t.Fatalf("%v", err)
		}
		if len(projected) != 2 {
			t.Fatalf("stage projection=%v", projected)
		}
		reportProgress(ctx, "verifying terminal registration")
		// Observe this timer in testing/synctest virtual time; no wall-clock guess.
		<-time.After(2 * time.Second)
		if got := describe(context.DeadlineExceeded).Error(); !strings.Contains(got, "verifying terminal registration after 2s") {
			t.Fatal(got)
		}
	})
}
