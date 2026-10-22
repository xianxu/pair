package testcouch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

func TestActorReadinessAllowsAlreadyReconnectedWorkers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	observe := func(context.Context) (couchmessage.Response, error) {
		calls++
		return couchmessage.Response{Actors: make([]couchmessage.Candidate, 4)}, nil
	}
	if err := waitActorCount(ctx, 1, observe); err != nil {
		t.Fatalf("steward readiness rejected reconnected cohort: %v", err)
	}
	if calls != 1 {
		t.Fatalf("snapshots=%d; complete cohort should settle once", calls)
	}
}

func TestActorReadinessReportsExhaustedMailboxImmediately(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	calls := 0
	err := waitActorCount(ctx, 4, func(context.Context) (couchmessage.Response, error) {
		calls++
		return couchmessage.Response{}, agentfixture.ErrMailboxFull
	})
	if !errors.Is(err, agentfixture.ErrMailboxFull) || calls != 1 {
		t.Fatalf("permanent refusal was retried: calls=%d err=%v", calls, err)
	}
}
