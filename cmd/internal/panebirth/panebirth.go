// Package panebirth is Pair's evidence that a zellij session has been born
// (#287), and the one wait on it (#288).
//
// The layout's pane command writes the agent pane sidecar as its first act, and
// a pane exists only once the first client has initialized the session. Until
// then zellij 0.45.1 panics when a connection it accepted closes, and
// `list-sessions` connects to every socket. A waiter that asks zellij nothing
// until this evidence appears cannot open that window. Three waiters share the
// loop: the title poller, Couch's cold resume, and the launcher's birth watch.
package panebirth

import (
	"context"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// Evidence is the file a create's birth is judged by: this agent's pane
// sidecar. It is the ONE declaration of that path. The launcher's create path
// clears exactly this file before it spawns anything; if a waiter named a
// different file, the clear would miss and a stale sidecar would pass at once.
//
// Birth is the file's existence, whatever its content or mtime. Each waiter
// stats it through its own runtime seam (the poller's ModTime, the launcher's
// FileSize), so its fake can model the pane command writing it.
func Evidence(dataDir, tag, agent string) (string, error) {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return "", err
	}
	return paths.PaneChecked(agent)
}

// ErrUnborn is a wait that ended without birth.
var ErrUnborn = errors.New("agent pane not born")

// Clock is a wait's time: Now bounds it, Sleep paces it. Sleep returns early
// when ctx ends; a clock that cannot (a test's) may ignore ctx.
type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration)
}

// WallClock is the real clock, for waiters without a clock seam of their own.
type WallClock struct{}

func (WallClock) Now() time.Time { return time.Now() }

func (WallClock) Sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// Await polls born every poll until it reports birth (nil), grace has passed
// on clock, or ctx is done.
//
// A failed observation is "not yet", never birth, and does not end the wait: a
// transient stat error after a good create must not read as a dead one. The
// last such error rides the returned one, for the diagnosis. A grace that
// passes is reported as context.DeadlineExceeded -- it is a deadline -- so a
// caller whose context and grace race sees the same cause whichever fires
// first.
func Await(ctx context.Context, clock Clock, poll, grace time.Duration, born func() (bool, error)) error {
	deadline := clock.Now().Add(grace)
	var lastErr error
	for {
		ok, err := born()
		if err == nil && ok {
			return nil
		}
		if err != nil {
			lastErr = err
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrUnborn, err, lastErr)
		}
		if !clock.Now().Before(deadline) {
			return errors.Join(ErrUnborn, context.DeadlineExceeded, lastErr)
		}
		clock.Sleep(ctx, poll)
	}
}
