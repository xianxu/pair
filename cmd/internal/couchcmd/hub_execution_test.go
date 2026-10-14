package couchcmd

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"testing"
)

func TestHubExecutionGateWaitsForRunningMutationAndCancelsQueuedMove(t *testing.T) {
	h, _, _ := moveFixture(t)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		_, _ = dispatchHubOperation(h, couchcore.OperationCall{Name: "start-hub", Context: context.Background()}, func(couchcore.OperationCall) (any, error) { close(entered); <-release; return nil, nil })
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	_, err := dispatchHubOperation(h, couchcore.OperationCall{Name: "move-slot", Context: ctx}, func(couchcore.OperationCall) (any, error) { called = true; return nil, nil })
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("blocked move: %v called %v", err, called)
	}
	close(release)
	<-done
	if _, err = dispatchHubOperation(h, couchcore.OperationCall{Name: "move-slot", Context: context.Background()}, func(couchcore.OperationCall) (any, error) { called = true; return nil, nil }); err != nil || !called {
		t.Fatalf("released gate %v", err)
	}
}
