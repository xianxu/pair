package couchcore

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLaunchOutputSetupReaderCloseFailureRetainsRealChild(t *testing.T) {
	starter := func(dir string, _ []string, env []string, extra []*os.File) (Handle, error) {
		// Simulate a starter which consumed/closed the parent's reader. The second
		// Close in startBlockedChild must fail after the real process exists.
		if err := extra[0].Close(); err != nil {
			return nil, err
		}
		h, err := startExecChild(dir, []string{"sh", "-c", "printf setup-failure >&2; exit 23"}, env, nil)
		if err == nil {
			h.Wait()
		}
		return h, err
	}
	h, err := startBlockedChild(context.Background(), starter, "ignored", t.TempDir(), []string{"pair"}, nil, time.Second)
	if h != nil || err == nil {
		t.Fatalf("handle=%v err=%v", h, err)
	}
	var failure *blockedStartError
	if !errors.As(err, &failure) {
		t.Fatalf("missing retained setup evidence: %v", err)
	}
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("lost original close error: %v", err)
	}
	if !failure.HelperStarted || failure.BeforeExit == nil || *failure.BeforeExit != 23 || failure.AfterExit == nil || *failure.AfterExit != 23 {
		t.Fatalf("exit evidence=%+v", failure)
	}
	if failure.Stream != "stderr" || string(failure.Output) != "setup-failure" {
		t.Fatalf("output=%q %q", failure.Stream, failure.Output)
	}
}

func TestLaunchOutputSetupSharedFailureSeparatesCleanup(t *testing.T) {
	r := NewFakeRunner()
	h, err := r.Start(t.TempDir(), []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Terminal(h.ID()).Feed([]byte("before"))
	cause := errors.New("priority setup refused")
	cleanupErr := errors.New(strings.Repeat("cleanup", 1000))
	got := failedBlockedStart(h, cause, func() error {
		r.Terminal(h.ID()).Feed([]byte(" after"))
		r.SetExited(h.ID(), 23)
		return cleanupErr
	})
	var failure *blockedStartError
	if !errors.As(got, &failure) {
		t.Fatal("missing setup error")
	}
	if !errors.Is(got, cause) || !errors.Is(got, cleanupErr) {
		t.Fatal("lost error chain")
	}
	if failure.BeforeExit != nil || failure.AfterExit == nil || *failure.AfterExit != 23 {
		t.Fatalf("exit evidence=%+v", failure)
	}
	if !failure.HelperStarted || string(failure.Output) != "before after" || len(failure.CleanupError) > 4096 {
		t.Fatalf("evidence=%+v", failure)
	}
	r.Terminal(h.ID()).Feed([]byte(" later"))
	if string(failure.Output) != "before after" {
		t.Fatal("evidence aliases later handle output")
	}
}

func TestLaunchOutputSetupStarterReturnsHandleAndError(t *testing.T) {
	r := NewFakeRunner()
	cause := errors.New("starter setup failed after launch")
	starter := func(dir string, _ []string, _ []string, _ []*os.File) (Handle, error) {
		h, err := r.Start(dir, []string{"pair"}, nil)
		if err != nil {
			return nil, err
		}
		r.Terminal(h.ID()).Feed([]byte(strings.Repeat("x", 20000)))
		r.SetExited(h.ID(), 23)
		return h, cause
	}
	h, err := startBlockedChild(context.Background(), starter, "ignored", t.TempDir(), []string{"pair"}, nil, time.Second)
	var failure *blockedStartError
	if h != nil || !errors.As(err, &failure) || !errors.Is(err, cause) {
		t.Fatalf("lost starter failure: h=%v err=%v", h, err)
	}
	if !failure.HelperStarted || failure.BeforeExit == nil || *failure.BeforeExit != 23 || len(failure.Output) != 16*1024 {
		t.Fatalf("bad started evidence: %+v", failure)
	}
}
