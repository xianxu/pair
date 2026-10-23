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

	"github.com/charmbracelet/x/ansi"
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

func openWarningAttention(t *testing.T, input *io.PipeWriter, host *hostty.FakeHost, want string) {
	t.Helper()
	if _, err := input.Write([]byte("\x1b[97;7u")); err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "attention panel renders "+want, func() bool {
		text := strings.Join(strings.Fields(ansi.Strip(host.Written())), " ")
		return strings.Contains(text, "Waiting on you") && strings.Contains(text, want)
	})
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
