package launcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/panebirth"
)

// #288: once a create's pane has not appeared within the bound, one liveness
// snapshot decides. Only an answer that shows no running session is death; a
// failed question is not an answer.
func TestJudgeUnborn(t *testing.T) {
	const ours = "📁work-bugfix"
	other := Session{Name: "📁work-other", State: SessionLive}
	for _, tc := range []struct {
		name     string
		sessions []Session
		err      error
		want     birthVerdict
	}{
		{"the probe failed", nil, errors.New("zellij list-sessions: timeout"), birthUnknown},
		{"zellij's empty inventory", nil, nil, birthDead},
		{"only other sessions", []Session{other}, nil, birthDead},
		{"ours exited", []Session{other, {Name: ours, State: SessionExited}}, nil, birthDead},
		{"ours live", []Session{{Name: ours, State: SessionLive}}, nil, birthAlive},
		{"ours attached", []Session{{Name: ours, State: SessionAttached}}, nil, birthAlive},
		{"ours detached", []Session{{Name: ours, State: SessionDetached}}, nil, birthAlive},
	} {
		if got := judgeUnborn(tc.sessions, tc.err, ours); got != tc.want {
			t.Errorf("%s: judgeUnborn = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Death is proven by the watch, but a handoff failed at birth only when the
// client did not end cleanly: -1 from the watch's kill, 1 from a client that
// noticed the dead server. A 0 is a normal end whatever the watch concluded.
func TestFailedAtBirth(t *testing.T) {
	for _, tc := range []struct {
		v    birthVerdict
		code int
		want bool
	}{
		{birthDead, -1, true},
		{birthDead, 1, true},
		{birthDead, 0, false},
		{birthStopped, -1, false},
		{birthBorn, -1, false},
		{birthAlive, 1, false},
		{birthUnknown, -1, false},
	} {
		if got := failedAtBirth(tc.v, tc.code); got != tc.want {
			t.Errorf("failedAtBirth(%v, %d) = %v, want %v", tc.v, tc.code, got, tc.want)
		}
	}
}

// Unanswered probes back off, because each is a machine-wide list-sessions,
// but never stop: standing down would leave a dead birth hung.
func TestNextProbeWait(t *testing.T) {
	for _, tc := range []struct{ in, want time.Duration }{
		{10 * time.Second, 20 * time.Second},
		{3 * time.Minute, maxProbeWait},
		{maxProbeWait, maxProbeWait},
	} {
		if got := nextProbeWait(tc.in); got != tc.want {
			t.Errorf("nextProbeWait(%s) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// A launch that ended while the watch was asking zellij is not the watch's to
// judge: by then an empty listing describes a session that was quit, not one
// that never came up, and the abort would land on a finished launch.
func TestWatchBirthDoesNotJudgeALaunchThatEndedDuringItsProbe(t *testing.T) {
	rt := newFakeRuntime()
	rt.launchStarted = true // the probe below is the watch's
	ctx, launchReturned := context.WithCancel(context.Background())
	defer launchReturned()
	rt.livenessHook = func(int) { launchReturned() } // the client exits while zellij is asked
	rt.livenessScript = []livenessAnswer{{}}         // and by then nothing is listed
	aborted := false
	v := watchBirth(ctx, rt, panebirth.WallClock{}, "/data/pane-work-claude.json", watchedSession, 10*time.Millisecond, func() { aborted = true })
	if v != birthStopped || aborted {
		t.Fatalf("verdict = %v, aborted = %v; want stopped and no abort", v, aborted)
	}
}

// watchClock stands still until slept on, so the watch's waits are exact.
type watchClock struct{ now time.Time }

func (c *watchClock) Now() time.Time                           { return c.now }
func (c *watchClock) Sleep(_ context.Context, d time.Duration) { c.now = c.now.Add(d) }

// The backoff is wired, not only defined: with zellij unanswering, the watch
// asks at bound, then after 2x, then after 4x -- 1 s, 3 s and 7 s on a stood-
// still clock -- and ends the client only on the third, answered probe.
func TestWatchBirthBacksOffBetweenUnansweredProbes(t *testing.T) {
	rt := newFakeRuntime()
	rt.launchStarted = true
	start := time.Unix(1_700_000_000, 0)
	clock := &watchClock{now: start}
	unanswered := livenessAnswer{err: errors.New("zellij list-sessions: context deadline exceeded")}
	rt.livenessScript = []livenessAnswer{unanswered, unanswered, {}}
	var askedAt []time.Duration
	rt.livenessHook = func(int) { askedAt = append(askedAt, clock.now.Sub(start)) }
	aborted := false
	v := watchBirth(context.Background(), rt, clock, "/data/pane-work-claude.json", watchedSession, time.Second, func() { aborted = true })
	if v != birthDead || !aborted {
		t.Fatalf("verdict = %v, aborted = %v; want dead and the client ended", v, aborted)
	}
	want := []time.Duration{time.Second, 3 * time.Second, 7 * time.Second}
	if len(askedAt) != len(want) {
		t.Fatalf("probes at %v, want %v", askedAt, want)
	}
	for i := range want {
		if askedAt[i] != want[i] {
			t.Fatalf("probes at %v, want %v", askedAt, want)
		}
	}
}
