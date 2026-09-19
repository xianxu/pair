package launcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/panebirth"
)

// The create path's birth watch (#288). A zellij server can die before its
// first client initializes the session -- zellij 0.45.1 panics when a
// connection it accepted in that window closes, and any machine-wide
// `list-sessions` is one. Most such clients notice and exit 1, but some hang
// forever on a cooked tty, and LaunchSession never returns: the operator sees a
// blank pane, and Couch keeps the thread live because its helper is alive.

// birthBound is how long a create waits for its agent pane before asking
// zellij whether the session is alive. Measured with probes/zellijbirthrace,
// launch to pane, end to end from `pair resume`: 0.6-1.3 s idle, 1.5-2.9 s
// with every core saturated, 0.4-0.5 s with a cold zellij cache. 10 s is 3.4x
// the loaded worst case and still under Couch's 15 s registration deadline.
const birthBound = 10 * time.Second

// maxProbeWait caps the backoff between probes zellij does not answer. Each
// probe is a machine-wide list-sessions, itself the birth-window risk for other
// threads' launches, so a launch whose probe keeps failing asks ever less often
// -- but it keeps asking, because standing down would leave a dead birth hung.
const maxProbeWait = 5 * time.Minute

// birthPoll is the watch's stat cadence: one stat per tick. A client that
// exits first ends the wait at once (the sleep watches ctx), not a tick late.
const birthPoll = 100 * time.Millisecond

type birthVerdict int

const (
	birthStopped birthVerdict = iota // the client exited before a verdict
	birthBorn                        // this launch's pane wrote its sidecar
	birthAlive                       // no pane, but zellij lists the session live: not ours to end
	birthUnknown                     // no pane, and zellij could not be asked: no proof of death
	birthDead                        // no pane, and zellij lists no live session
)

func (v birthVerdict) String() string {
	return [...]string{"stopped", "born", "alive", "unknown", "dead"}[v]
}

// judgeUnborn is the verdict on a create whose pane did not appear within the
// bound, from one liveness snapshot. zellij's own empty inventory is already
// (nil, nil) by the time it gets here (ZellijSource.runContext).
func judgeUnborn(sessions []Session, err error, session string) birthVerdict {
	if err != nil {
		return birthUnknown // a failed observation is not an absence
	}
	for _, s := range sessions {
		if s.Name == session {
			if s.State == SessionExited {
				return birthDead
			}
			return birthAlive
		}
	}
	return birthDead
}

// failedAtBirth is whether a create's handoff failed at birth: the watch
// proved death AND the client did not end cleanly. The watch's kill reports -1
// and a client that noticed the dead server exits 1, while a client that was
// quitting cleanly as the verdict landed exits 0 -- that is a normal end, and
// it keeps the normal path and its quit cleanup.
func failedAtBirth(v birthVerdict, code int) bool {
	return v == birthDead && code != 0
}

// nextProbeWait is the wait before asking zellij again after it did not answer:
// double, capped at maxProbeWait.
func nextProbeWait(wait time.Duration) time.Duration {
	return min(2*wait, maxProbeWait)
}

// watchBirth waits for the create's birth evidence while LaunchSession blocks.
// Only proof of death -- no pane AND no live session -- calls abort. It never
// acts on a session zellij has listed live: that session may be healthy with a
// missing sidecar, and ending it would skip the quit cleanup. An unanswered
// probe earns a longer wait and another question, never a teardown. ctx ends
// when LaunchSession returns; a verdict reached after that is not acted on.
// clock paces the waits; production passes panebirth.WallClock.
func watchBirth(ctx context.Context, rt Runtime, clock panebirth.Clock, evidence, session string, bound time.Duration, abort func()) birthVerdict {
	for wait := bound; ; wait = nextProbeWait(wait) {
		err := panebirth.Await(ctx, clock, birthPoll, wait, func() (bool, error) {
			_, ok := rt.FileSize(evidence)
			return ok, nil
		})
		switch {
		case err == nil:
			return birthBorn
		case ctx.Err() != nil:
			return birthStopped
		}
		sessions, probeErr := rt.SessionLiveness()
		if ctx.Err() != nil {
			return birthStopped // the client ended while we asked: not ours to judge
		}
		switch v := judgeUnborn(sessions, probeErr, session); v {
		case birthDead:
			abort()
			return v
		case birthAlive:
			return v
		}
	}
}

// launchWatched runs the create handoff under the birth watch. The watch starts
// just before the blocking LaunchSession and is stopped and joined as soon as
// it returns, so nothing it spawned outlives the call.
func launchWatched(rt Runtime, evidence, session, configDir, layout string, bound time.Duration) (int, birthVerdict, error) {
	launchCtx, abort := context.WithCancel(context.Background())
	defer abort()
	watchCtx, stopWatch := context.WithCancel(context.Background())
	verdict := make(chan birthVerdict, 1)
	go func() { verdict <- watchBirth(watchCtx, rt, panebirth.WallClock{}, evidence, session, bound, abort) }()
	code, err := rt.LaunchSession(launchCtx, session, configDir, layout)
	stopWatch()
	return code, <-verdict, err
}

// zellijLogPath is where zellij writes its log by default: a pointer for the
// operator in the dead-birth message, not something Pair reads. The same
// layout is encoded by probes/zellijbirthrace's zellijTmp, which cannot import
// cmd/internal; a zellij layout change must update both.
func zellijLogPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("zellij-%d", os.Getuid()), "zellij-log", "zellij.log")
}
