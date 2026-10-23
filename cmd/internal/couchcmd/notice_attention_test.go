package couchcmd

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func warningConsole(t *testing.T) (*couchtty.Console, *hostty.FakeHost, *io.PipeWriter) {
	t.Helper()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 120})
	reader, input := io.Pipe()
	console := couchtty.New(host, reader)
	child := ptychild.NewFakeChild(nil)
	console.Attach("unrelated-focused-slot", "unrelated focused slot", child)
	done := make(chan int, 1)
	go func() { done <- console.Run() }()
	t.Cleanup(func() {
		console.Stop()
		child.Exit(0)
		input.Close()
		reader.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("warning console failed to stop")
		}
	})
	return console, host, input
}

func openWarningAttention(t *testing.T, input io.Writer, host interface {
	Written() string
	Size() (ptychild.Size, error)
}, want string) {
	t.Helper()
	size, err := host.Size()
	if err != nil {
		t.Fatal(err)
	}
	screen := startupWarmScreen(t, int(size.Cols), int(size.Rows), host.Written)
	// An empty toggle stays closed, including when a warning arrives later.
	// Read the CURRENT terminal cells, so a dismissed historical badge in the
	// accumulated output cannot acknowledge a later asynchronous publication.
	testwait.Until(t, "warning publication before attention input", func() bool {
		return strings.Contains(screen(), couchtty.AttentionGlyph+"1")
	})
	if _, err := input.Write([]byte("\x1b[97;7u")); err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "attention panel renders "+want, func() bool {
		text := strings.Join(strings.Fields(screen()), " ")
		return strings.Contains(text, "Waiting on you") && strings.Contains(text, want)
	})
}

// warningAttentionScreen and warningAttentionInput keep the integration's real
// console and input path, while controlling which event happens first.
type warningAttentionScreen struct {
	*hostty.FakeHost
	read func() string
}

func (screen warningAttentionScreen) Written() string { return screen.read() }

type warningAttentionInput func([]byte) (int, error)

func (write warningAttentionInput) Write(p []byte) (int, error) { return write(p) }

func TestOpenWarningAttentionWaitsForPublication(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		publicationFirst, historical bool
	}{
		{name: "publication-before-helper", publicationFirst: true},
		{name: "helper-before-publication"},
		{name: "dismissed-historical-badge-before-publication", historical: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			console, host, input := warningConsole(t)
			if tc.historical {
				console.Warn(couchtty.Notice{Body: "old dismissed warning"})
				openWarningAttention(t, input, host, "old dismissed warning")
				if _, err := input.Write([]byte("c")); err != nil {
					t.Fatal(err)
				}
				current := startupWarmScreen(t, 120, 30, host.Written)
				testwait.Until(t, "historical warning dismissed", func() bool {
					return !strings.Contains(current(), couchtty.AttentionGlyph)
				})
			}
			const body = "controlled publication warning"
			published := false
			publish := func() {
				console.Warn(couchtty.Notice{Body: body})
				published = true
			}
			if tc.publicationFirst {
				publish()
			}
			// In the helper-first ordering, publication is released only when
			// the helper first observes the screen. An eager toggle therefore
			// fails deterministically, without a sleep or scheduler assumption.
			reads := 0
			screen := warningAttentionScreen{FakeHost: host, read: func() string {
				reads++
				if tc.historical && reads == 1 {
					// Historical bytes contain a badge, current cells do not.
					return host.Written()
				}
				if !published {
					publish()
				}
				return host.Written()
			}}
			toggles := 0
			guardedInput := warningAttentionInput(func(p []byte) (int, error) {
				if !published {
					return 0, errors.New("attention input preceded warning publication acknowledgment")
				}
				toggles++
				return input.Write(p)
			})
			openWarningAttention(t, guardedInput, screen, body)
			if toggles != 1 {
				t.Fatalf("attention toggled %d times, want one after publication", toggles)
			}
		})
	}
}

func TestCrashCaptureFailureRaisesAttention(t *testing.T) {
	console, host, input := warningConsole(t)
	store := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(store, nil, 0600); err != nil {
		t.Fatal(err)
	}
	installCrashReport(console, store)
	openWarningAttention(t, input, host, "crash capture unavailable")
}

func TestHubStartupFailureRaisesAttention(t *testing.T) {
	r := newHubStartupRig(t, false)
	r.rt.runner.FailNextStartIn(r.paths[1], errors.New("second hub unavailable"))
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	if call := r.await(t, "start-hub"); call.err == nil {
		t.Fatal("hub launch unexpectedly succeeded")
	}
	// Completion sends the report after the dispatcher returns; wait for the
	// badge rather than racing the asynchronous warning publication.
	testwait.Until(t, "hub warning attention", func() bool {
		return strings.Contains(r.host.Written(), couchtty.AttentionGlyph+"1")
	})
	openWarningAttention(t, r.input, r.host, "hub failed")
}

func TestSupervisionWarningCompositionRaisesAttention(t *testing.T) {
	raw, _, _ := scratchWiringRuntime(t)
	rt, lease, err := raw.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	c, err := rt.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	console, host, input := warningConsole(t)
	service, err := startMessageService(console, c)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	service.supervision.Load().notice("startup resend prompt unavailable; request fresh status")
	openWarningAttention(t, input, host, "startup resend prompt unavailable")
}

func TestLifecycleQueueCleanupRaisesAttention(t *testing.T) {
	console, host, input := warningConsole(t)
	c := &couchcore.Couch{LaunchFailures: couchcore.NewFakeLaunchFailureStore()}
	q := newLifecycleRemoteQueue(console, c)
	var finish func(any, error)
	q.enqueue = func(_ string, _ string, _ func(context.Context) (couchcore.OperationCall, error), _ func(), done func(any, error)) error {
		finish = done
		return nil
	}
	if err := q.Queue("hub-start:/test-hub", "park", func(context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{}, nil
	}, func() {}, func(any, error) {}); err != nil {
		t.Fatal(err)
	}
	finish(couchcore.ParkResult{CleanupError: errors.New("sidecar cleanup unavailable")}, nil)
	openWarningAttention(t, input, host, "unproven cleanup")
}

func TestHubStartupSuccessDoesNotRaiseAttention(t *testing.T) {
	r := newHubStartupRig(t, false)
	drained := make(chan struct{})
	var once sync.Once
	r.console.SetLifecycleObserver(func(observation couchtty.LifecycleObservation) {
		if observation.Kind == "restart-drained" {
			once.Do(func() { close(drained) })
		}
	})
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	if call := r.await(t, "start-hub"); call.err != nil {
		t.Fatal(call.err)
	}
	r.await(t, "attach")
	// Join the warm-only background attachment as well as the explicit hub.
	// A zero-attention assertion before that terminal completion can miss a
	// legitimate borrowed-client warning from the background fixture.
	select {
	case <-drained:
	case <-testwait.After(t):
		t.Fatal("background reattachment completion never drained")
	}
	r.assertFirstUsable(t)
	if strings.Contains(r.host.Written(), couchtty.AttentionGlyph) {
		t.Fatal("ordinary successful startup raised attention")
	}
}

func TestArchivePassFailureWarnsOnceWithoutDuplicatingFeedProjection(t *testing.T) {
	console, host, input := warningConsole(t)
	s := newArchiveScheduler(t.TempDir(), func(text string) {
		console.Notify(couchtty.Notice{Kind: "archive", Body: text})
	}, func(text string) {
		console.Warn(couchtty.Notice{Kind: "archive", Body: text})
	})
	if err := os.MkdirAll(s.store, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.store, historyarchive.ContinuousConfigName), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	s.launch = func(context.Context) (historyarchive.ContinuousCommandReport, error) {
		return historyarchive.ContinuousCommandReport{}, errors.New("archive subprocess failed")
	}
	s.tick(t.Context())
	s.tick(t.Context())
	// Feed-only projection is shared with notifyArchiveWarning, whose attention
	// event already has its own owner. It must not raise a second event here.
	s.notify("already represented backup warning")
	openWarningAttention(t, input, host, "history backup pass failed")
	if !strings.Contains(host.Written(), couchtty.AttentionGlyph+"1") || strings.Contains(host.Written(), couchtty.AttentionGlyph+"2") {
		t.Fatal("archive failure or feed projection raised duplicate attention")
	}
}

// Replays the ordinary restart class observed in the operator screenshot
// 2026-10-23.09-44-47.240.jpg: multiple live sessions lack unique retained
// helper proof. They still attach; their diagnostics must not fill the bell.
func TestNormalMultiSlotBorrowedRestartHasNoAttention(t *testing.T) {
	r := newHubStartupOwnershipRig(t, true, false)
	if !strings.Contains(r.first.Warning(), "BORROWED ATTACH: no unique retained client helper") {
		t.Fatalf("initial reattachment did not reproduce captured diagnostic: %q", r.first.Warning())
	}
	drained := make(chan struct{})
	var once sync.Once
	r.console.SetLifecycleObserver(func(o couchtty.LifecycleObservation) {
		if o.Kind == "restart-drained" {
			once.Do(func() { close(drained) })
		}
	})
	var mu sync.Mutex
	var warnings []string
	actual := r.console.Ops()
	r.console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		result, err := actual(call)
		if err == nil {
			if w, ok := result.(interface{ Warning() string }); ok && strings.Contains(w.Warning(), "BORROWED ATTACH") {
				mu.Lock()
				warnings = append(warnings, w.Warning())
				mu.Unlock()
			}
		}
		return result, err
	})
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	select {
	case <-drained:
	case <-testwait.After(t):
		t.Fatal("multi-slot reattachment did not drain")
	}
	r.await(t, "start-hub")
	r.assertFirstUsable(t)
	// A terminal command after restart-drained joins publication and repaint.
	if _, err := r.input.Write([]byte("\x00")); err != nil {
		t.Fatal(err)
	}
	screen := startupWarmScreen(t, 100, 30, r.host.Written)
	testwait.Until(t, "startup switcher renders", func() bool { return strings.Contains(screen(), "threads") })
	mu.Lock()
	count := len(warnings)
	mu.Unlock()
	if count < 2 {
		t.Fatalf("fixture reattached only %d borrowed results, want >=2 plus initial", count)
	}
	if strings.Contains(screen(), couchtty.AttentionGlyph) {
		t.Fatalf("normal multi-slot borrowed restart raised attention: %s", screen())
	}
}

func TestSessionFreshnessSkipDiagnosticsStayFeedOnly(t *testing.T) {
	rig := newHubStartupRig(t, false)
	rig.run(t)
	rig.assertFirstUsable(t)
	console, host, input := rig.console, rig.host, rig.input
	r := newServiceRig(t)
	r.restart(nil)
	watcher := newConsoleSessionStaleWatcher(r.s, t.TempDir(), console)
	for _, cause := range []string{
		"open /fixture/ops/session.side/entry.md: no such file or directory",
		"open /fixture/echo/workshop/projects/status.md: no such file or directory",
	} {
		watcher.diagnostic(errors.New(cause))
	}
	if _, err := input.Write([]byte("\x00")); err != nil {
		t.Fatal(err)
	}
	screen := startupWarmScreen(t, 100, 30, host.Written)
	testwait.Until(t, "freshness skip retained in switcher", func() bool { return strings.Contains(screen(), "freshness reminder skipped") })
	if strings.Contains(screen(), couchtty.AttentionGlyph) {
		t.Fatalf("routine freshness skip raised attention: %s", screen())
	}
}
