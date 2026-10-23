package webcast

import (
	"bytes"
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/cdp"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Characterize launch cleanup before moving the existing fidelity lifecycle to
// a shared file. A failed browser launch cannot leave its private profile.
func TestReplayImageLaunchFailureRemovesProfile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TMPDIR", root)
	_, err := launchHeadless(context.Background(), filepath.Join(root, "absent-browser"))
	if err == nil {
		t.Fatal("launched absent browser")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("launch failure retained profile: %v", entries)
	}
}

type cleanupCommandPipe struct{ bytes.Buffer }

func (*cleanupCommandPipe) Close() error { return nil }

func TestReplayImageCleanupUsesOneExpiredBudget(t *testing.T) {
	reader, writer := io.Pipe()
	conn := cdp.New(reader, &cleanupCommandPipe{})
	defer reader.Close()
	defer writer.Close()
	eventsDone := make(chan struct{})
	go func() {
		defer close(eventsDone)
		for range conn.Events() {
		}
	}()
	profile := t.TempDir()
	h := &headless{cmd: exec.Command("unused"), conn: conn, profile: profile, eventsDone: eventsDone}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() { done <- h.closeJoinedContext(ctx) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lost cleanup deadline: %v", err)
		}
		if _, err := os.Stat(profile); err != nil {
			t.Fatalf("removed uncertain profile: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cleanup stages renewed an expired budget")
	}
	writer.Close()
	if err := conn.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestReplayImageCleanupBudgetsFitObserverGrace(t *testing.T) {
	if replayBrowserCleanupBudget > 5*time.Second || sketchImageServerCleanupBudget > time.Second || replayBrowserCleanupBudget+sketchImageServerCleanupBudget > 6*time.Second {
		t.Fatal("renderer cleanup exceeds client observer grace")
	}
}
