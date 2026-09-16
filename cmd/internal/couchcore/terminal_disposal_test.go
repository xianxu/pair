package couchcore

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestFailedUnacceptedPTYCleanupCancelsBlockedPublication(t *testing.T) {
	entered := make(chan struct{}, 1)
	runner := &PtyRunner{Sink: func(ctx context.Context, _ string, _ ptychild.OutputBatch) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return ctx.Err()
	}}
	handle, err := runner.Start(t.TempDir(), []string{"/bin/sh", "-c", "printf final-output; exec sleep 30"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	child := handle.(TerminalHandle).Terminal()
	defer child.Close()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("real PTY did not enter blocked delivery")
	}
	// A failed warm attachment owns the client, never the existing session.
	// No artifact service is supplied: this path must not quiesce that session.
	couch := &Couch{}
	done := make(chan error, 1)
	go func() {
		done <- couch.quiescePostAckStart(ThreadAddress{RepoScope: "test", Tag: "warm"}, handle, StartWarmReattach)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Logf("cleanup retained OS signal diagnostics after proven disposal: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cleanup waited on final UI acknowledgement instead of canceling unaccepted publication")
	}
	if _, err := child.Endpoint().Snapshot(time.Now()); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("rollback endpoint remains readable: %v", err)
	}
	if handle.Alive() {
		t.Fatal("rollback retained owned process group")
	}
}
