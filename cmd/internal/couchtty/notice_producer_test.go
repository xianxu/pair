package couchtty

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Observe the production routing handoff without reentering the Console lock.
func observeProducerNotices(c *Console) <-chan Notice {
	out := make(chan Notice, 32)
	c.SetNoticeRouter(func(n Notice) { out <- n })
	return out
}

func requireProducerNotice(t *testing.T, out <-chan Notice, class couchcore.NoticeClass, origin couchcore.NoticeOrigin, thread couchcore.ThreadAddress) Notice {
	t.Helper()
	select {
	case n := <-out:
		if n.Class != class || n.Origin != origin || n.Thread != thread || n.HubHandled {
			t.Fatalf("producer notice = %+v, want class=%s origin=%d thread=%+v without prior delivery", n, class, origin, thread)
		}
		return n
	case <-testwait.After(t):
		t.Fatal("producer did not hand off its notice")
	}
	return Notice{}
}

func requireNoProducerNotice(t *testing.T, out <-chan Notice) {
	t.Helper()
	select {
	case n := <-out:
		t.Fatalf("unexpected actionable notice: %+v", n)
	default:
	}
}

func TestNoticeTTYProducerSynchronousPaths(t *testing.T) {
	tests := []struct {
		name     string
		class    couchcore.NoticeClass
		operator bool
		emit     func(*testing.T, *Console, couchcore.ContinuationStatus)
	}{
		{"input trace", couchcore.NoticeInputTraceFailed, false, func(t *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			if c.SetInputTrace(filepath.Join(t.TempDir(), "missing", "trace")) == nil {
				t.Fatal("trace unexpectedly opened")
			}
		}},
		{"mouse trace", couchcore.NoticeMouseTraceFailed, false, func(t *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			if c.SetMouseTrace(filepath.Join(t.TempDir(), "missing", "trace")) == nil {
				t.Fatal("trace unexpectedly opened")
			}
		}},
		{"broadcast start", couchcore.NoticeCastStartFailed, true, func(_ *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			c.bcast = broadcastState{attempt: 1, phase: broadcastStarting}
			c.broadcastStarted(1, nil, errors.New("start refused"), nil)
		}},
		{"chrome start", couchcore.NoticeChromeStartFailed, true, func(_ *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			c.wcast = webcastState{attempt: 1, phase: webcastStarting}
			c.webcastStarted(1, nil, errors.New("start refused"), nil)
		}},
		{"chrome recovery", couchcore.NoticeChromeRecovery, true, func(_ *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			cast := &webcast.Cast{}
			c.wcast = webcastState{cast: cast, phase: webcastRunning}
			defer func() { c.wcast = webcastState{} }()
			c.webcastUpdated(cast, webcast.Update{State: webcast.State{Recovery: "active tab disappeared"}})
		}},
		{"drive failure", couchcore.NoticeOperatorDriveFailed, true, func(_ *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			c.driveResult(errors.New("writer refused"), "")
		}},
		{"event trace", couchcore.NoticeEventTraceFailed, false, func(t *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			if c.SetEventTrace(filepath.Join(t.TempDir(), "missing", "trace"), time.Now()) == nil {
				t.Fatal("trace unexpectedly opened")
			}
		}},
		{"continuation scan", couchcore.NoticeContinuationScannerFailed, false, func(_ *testing.T, c *Console, _ couchcore.ContinuationStatus) {
			c.acceptContinuationRequests(continuationScanResult{err: errors.New("scanner failed")})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, status := continuationConsole(t)
			out := observeProducerNotices(c)
			test.emit(t, c, status)
			origin := couchcore.NoticeSystemOrigin
			if test.operator {
				origin = couchcore.NoticeOperatorOrigin
			}
			n := requireProducerNotice(t, out, test.class, origin, couchcore.ThreadAddress{})
			if n.Body == "" || c.menuSnapshot().LatestFeed != n.Body {
				t.Fatalf("feed changed or omitted: notice=%+v feed=%q", n, c.menuSnapshot().LatestFeed)
			}
			requireNoProducerNotice(t, out)
		})
	}
}

func TestNoticeTTYProducerContinuationOwnership(t *testing.T) {
	for _, kind := range []string{"routed miss", "watched miss", "failure"} {
		t.Run(kind, func(t *testing.T) {
			c, status := continuationConsole(t)
			out := observeProducerNotices(c)
			completed := operationCompletion{name: "resume", origin: MenuOperationOrigin{Address: status.Address}}
			want := couchcore.NoticeContinuationMissed
			var err error
			if kind != "routed miss" {
				completed.origin.ContinuationID = status.RequestID
				c.continuations[status.Address] = continuationWatch{status: status}
			}
			if kind == "failure" {
				want = couchcore.NoticeContinuationFailed
				err = errors.New("continuation failed")
			} else {
				completed.value = couchcore.ContinuationResult{Missed: &checkpoint.Miss{RequestID: status.RequestID, Path: "continuation.md", Reason: "source unavailable"}}
			}
			c.finishContinuationOperation(completed, err)
			n := requireProducerNotice(t, out, want, couchcore.NoticeSystemOrigin, status.Address)
			if n.Label == "" {
				t.Fatal("continuation lost source label")
			}
			requireNoProducerNotice(t, out)
		})
	}
}

func TestNoticeTTYProducerContinuationScanEpisode(t *testing.T) {
	c, _ := continuationConsole(t)
	out := observeProducerNotices(c)
	scan := continuationScanResult{err: errors.New("scanner unavailable")}
	c.acceptContinuationRequests(scan)
	c.acceptContinuationRequests(scan)
	requireProducerNotice(t, out, couchcore.NoticeContinuationScannerFailed, couchcore.NoticeSystemOrigin, couchcore.ThreadAddress{})
	requireNoProducerNotice(t, out)
	c.acceptContinuationRequests(continuationScanResult{})
	c.acceptContinuationRequests(scan)
	requireProducerNotice(t, out, couchcore.NoticeContinuationScannerFailed, couchcore.NoticeSystemOrigin, couchcore.ThreadAddress{})
}

func TestNoticeTTYProducerCastDiagnosticAndRoutineLog(t *testing.T) {
	c, _ := continuationConsole(t)
	out := observeProducerNotices(c)
	c.SetWebcast(webcast.CastConfig{})
	c.webcastCfg.Logf("HTTP %d", 503)
	requireNoProducerNotice(t, out)
	c.webcastCfg.Warnf("recorder: %s", "unavailable")
	n := requireProducerNotice(t, out, couchcore.NoticeCastDiagnostic, couchcore.NoticeOperatorOrigin, couchcore.ThreadAddress{})
	if n.Body != "Chrome cast: recorder: unavailable" {
		t.Fatalf("diagnostic text changed: %q", n.Body)
	}
}

func TestNoticeTTYProducerDriveRacesStayFeedOnly(t *testing.T) {
	c, _ := continuationConsole(t)
	out := observeProducerNotices(c)
	for _, err := range []error{drive.ErrNoSeat, drive.ErrControlOp, drive.ErrDriveUnavailable, drive.ErrClosed, drive.ErrNotLive, context.Canceled, context.DeadlineExceeded} {
		c.driveResult(err, "")
		requireNoProducerNotice(t, out)
		if got := c.menuSnapshot().LatestFeed; got != driveFailure(err) {
			t.Fatalf("race feed=%q, want %q", got, driveFailure(err))
		}
	}
}

func TestNoticeTTYProducerCastCodeGeneration(t *testing.T) {
	f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
	f.startLive(t)
	out := observeProducerNotices(f.con)
	attempt := f.screenAttempt()
	rotation := castlock.Rotation{Code: "519351", Reason: castlock.ReasonGuessing, Gen: 10, Capped: true}
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, rotation)
	requireProducerNotice(t, out, couchcore.NoticeCastCodeLockout, couchcore.NoticeOperatorOrigin, couchcore.ThreadAddress{})
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, rotation)
	rotation.Gen--
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, rotation)
	rotation.Gen = 11
	f.con.onCodeRotated(castScreen, attempt+1, broadcast.LinkView, rotation)
	rotation.Reason = castlock.ReasonOperator
	f.con.onCodeRotated(castScreen, attempt, broadcast.LinkView, rotation)
	requireNoProducerNotice(t, out)
}

func TestNoticeTTYProducerLifecycleDetachedSource(t *testing.T) {
	c, _ := continuationConsole(t)
	out := observeProducerNotices(c)
	store := couchcore.NewFakeLaunchFailureStore()
	store.SaveError = errors.New("diagnostic persistence unavailable")
	recorder := couchcore.NewLifecycleFailureRecorder(store, time.Now, func() (string, error) { return "producer-diagnostic", nil })
	attempt, err := recorder.Begin("park", couchcore.LifecycleFailureTarget{StartingPath: "/detached/source"})
	if err != nil {
		t.Fatal(err)
	}
	c.finishLifecycleAttempt(operationCompletion{attempt: attempt, origin: MenuOperationOrigin{RowKey: couchcore.ThreadRowKey{SlotPath: "/detached/source"}}, err: errors.New("park refused")})
	n := requireProducerNotice(t, out, couchcore.NoticeLifecycleAttemptDiagnostic, couchcore.NoticeSystemOrigin, couchcore.ThreadAddress{})
	if n.SourceRoot != "/detached/source" || n.OwnerRoot != "" {
		t.Fatalf("detached source became inferred hub ownership: %+v", n)
	}
}

func TestNoticeTTYProducerUnexpectedCastEnds(t *testing.T) {
	t.Run("screen", func(t *testing.T) {
		tunnel := &broadcast.FakeTunnel{}
		f := broadcastFixture(t, tunnel, nil)
		session := f.startLive(t)
		out := observeProducerNotices(f.con)
		tunnel.Exit()
		requireProducerNotice(t, out, couchcore.NoticeCastEndedUnexpectedly, couchcore.NoticeOperatorOrigin, couchcore.ThreadAddress{})
		// Joining the command loop makes the duplicate-completion negative exact.
		if err := f.con.runTerminalCommand(context.Background(), func() error { return nil }); err != nil {
			t.Fatal(err)
		}
		f.con.broadcastEnded(session)
		requireNoProducerNotice(t, out)
	})
	t.Run("chrome", func(t *testing.T) {
		r := newWebcastRig(t, func(r *webcastRig) {
			r.wrap = func(b *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser { return warningFailedBrowser{b} }
		})
		browser := r.open(t)
		out := observeProducerNotices(r.con)
		browser.QuitByHand()
		requireProducerNotice(t, out, couchcore.NoticeChromeEndedUnexpectedly, couchcore.NoticeOperatorOrigin, couchcore.ThreadAddress{})
	})
}

func TestNoticeTTYProducerCapabilityFailure(t *testing.T) {
	r := newWebcastRig(t, func(r *webcastRig) {
		r.wrap = func(b *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
			d, err := webcast.NewFakeDriven(b, drive.DrivenPage(100), &recording.MemSink{})
			if err != nil {
				t.Fatal(err)
			}
			return &capabilityWarningBrowser{FakeDriven: d, err: errors.New("clear annotations refused")}
		}
	})
	r.open(t)
	r.con.mu.Lock()
	cast := r.con.wcast.cast
	r.con.mu.Unlock()
	if err := cast.SetCapability(context.Background(), webcast.CapabilityAnnotation, true); err != nil {
		t.Fatal(err)
	}
	out := observeProducerNotices(r.con)
	awaitDrive(t, "typed capability failure", r.con.onWebCapabilityClick(webcast.CapabilityAnnotation, false))
	requireProducerNotice(t, out, couchcore.NoticeOperatorCapabilityFailed, couchcore.NoticeOperatorOrigin, couchcore.ThreadAddress{})
}
