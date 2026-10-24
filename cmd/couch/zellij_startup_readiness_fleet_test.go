package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/panebirth"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// This bounded sample qualifies cooperating background commands against real
// creator startup. It makes no crash/interactive-attach coverage or failure-rate
// claim. Native lifetime and failure evidence remain owned by shared testcouch.
func TestDisposableZellijStartupReadiness(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 to run disposable native acceptance")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 1, ExplicitHub: true, AgentKind: agent, RuntimeBudget: 4 * time.Minute, RetainStartupServerLog: true})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			startupArgumentsBaseline(t, fleet)
			ctx, cancel := context.WithTimeout(t.Context(), 240*time.Second)
			defer cancel()
			stop := startupReadinessTraffic(t, ctx, fleet, cancel)
			defer stop()
			started := time.Now()
			if err := startupReadinessStartHub(ctx, fleet, agent); err != nil {
				t.Fatalf("primary startup: %v", err)
			}
			if err := fleet.StartWorker(ctx, 1); err != nil {
				t.Fatalf("worker startup: %v", err)
			}
			startupReadinessExactMarkers(t, ctx, fleet)
			// Require a successful native observation after both exact startup proofs,
			// independently of empty inventory observations before creator admission.
			source := zellijquery.Source{Path: fleet.Binaries.Zellij, Env: fleet.Environment}
			result, err := source.ReadContext(ctx, "list-sessions", "--no-formatting")
			t.Logf("post-startup native result stdout=%q stderr=%q err=%v", result.Stdout, result.Stderr, err)
			if err != nil {
				t.Fatal(err)
			}
			stop()
			startupReadinessServerLog(t, ctx, fleet)
			if t.Failed() {
				return
			}
			t.Logf("startup readiness conformance run=%s agent=%s duration=%s; bounded sample only", fleet.RunID, agent, time.Since(started))
			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelCleanup()
			if err := fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err := fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
		}) {
			break
		}
	}
}

// The shared Start on this branch waits for Agent0, while current attachment-only
// startup intentionally opens the empty switcher. Drive the existing public
// operator form rather than import #863's unlanded harness change. Down requests
// preview; exactly one Enter starts. See TL a4eabbf3 and measured peer 8c904b9c.
func startupReadinessStartHub(parent context.Context, fleet *testcouch.Fleet, agent string) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		err := fleet.Start(ctx)
		if err != nil {
			cancel()
		}
		done <- err
	}()
	driveErr := startupReadinessHubForm(ctx, fleet, agent)
	if driveErr != nil {
		cancel()
	}
	return errors.Join(driveErr, <-done)
}

func startupReadinessHubForm(ctx context.Context, fleet *testcouch.Fleet, agent string) error {
	waitScreen := func(text string) error {
		return startupArgumentsWait(ctx, func() (bool, error) {
			return strings.Contains(fleet.Screen(), text), nil
		})
	}
	const root = "Ctrl+Y copy last notice / selected error"
	settle, stop := context.WithTimeout(ctx, 5*time.Second)
	open := startupArgumentsWait(settle, func() (bool, error) {
		return strings.Contains(fleet.Screen(), root), nil
	}) == nil
	stop()
	if !open {
		if err := fleet.Press("\x00", 32, 5); err != nil {
			return err
		}
		if err := waitScreen(root); err != nil {
			return err
		}
	}
	if err := fleet.Press("\x00", 32, 5); err != nil {
		return err
	}
	if err := waitScreen("start thread"); err != nil {
		return err
	}
	if err := fleet.Input([]byte(fleet.Workspaces.Primary)); err != nil {
		return err
	}
	if err := fleet.Press("\x1b[B", 57353, 1); err != nil {
		return err
	}
	for i := 0; i < 8 && !strings.Contains(fleet.Screen(), "agent "+agent); i++ {
		if err := fleet.Press("\x1b[C", 57351, 1); err != nil {
			return err
		}
		next, stop := context.WithTimeout(ctx, 2*time.Second)
		_ = startupArgumentsWait(next, func() (bool, error) { return strings.Contains(fleet.Screen(), "agent "+agent), nil })
		stop()
	}
	if err := waitScreen("agent " + agent); err != nil {
		return err
	}
	if err := waitScreen("args  repo-default"); err != nil {
		return err
	}
	if err := fleet.Press("\r", 13, 1); err != nil {
		return err
	}
	return startupArgumentsWait(ctx, func() (bool, error) {
		return !strings.Contains(fleet.Screen(), "start thread"), nil
	})
}

// Source uses the fleet's child environment, rather than the test process's
// native namespace. Its normal runner owns and joins each bounded query group;
// stop joins this sole producer before the fleet starts native teardown.
func startupReadinessTraffic(t *testing.T, parent context.Context, fleet *testcouch.Fleet, stopStartup context.CancelFunc) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	entered := make(chan struct{})
	var transcript strings.Builder
	var terminal error
	observations := 0
	go func() {
		defer close(done)
		source := zellijquery.Source{Path: fleet.Binaries.Zellij, Env: append([]string(nil), fleet.Environment...)}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		close(entered)
		// A finite observation budget prevents a broken startup from growing an
		// unbounded test log. There is no native command per passive candidate.
		for sample := 0; sample < 256; sample++ {
			if ctx.Err() != nil {
				return
			}
			started := time.Now()
			result, err := source.ReadContext(ctx, "list-sessions", "--no-formatting")
			observations++
			fmt.Fprintf(&transcript, "sample=%d started=%s duration=%s stdout=%q stderr=%q err=%v\n", sample, started.UTC().Format(time.RFC3339Nano), time.Since(started), result.Stdout, result.Stderr, err)
			if err != nil && !errors.Is(err, zellijquery.ErrReadinessUnknown) {
				if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
					return
				}
				terminal = fmt.Errorf("background native observation failed: %w", err)
				stopStartup()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
		terminal = errors.New("background observation sample budget exhausted before startup finished")
		stopStartup()
	}()
	<-entered
	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			<-done
			raw := transcript.String()
			t.Logf("concurrent startup probes run=%s observations=%d sha256=%x\n%s", fleet.RunID, observations, sha256.Sum256([]byte(raw)), raw)
			if terminal != nil {
				t.Error(terminal)
			}
		})
	}
}

func startupReadinessExactMarkers(t *testing.T, ctx context.Context, fleet *testcouch.Fleet) {
	t.Helper()
	actors, err := fleet.Actors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	namespace, err := zellijquery.ReadinessNamespace(fleet.Environment)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := artifactpath.ResolvePairCache(fleet.Roots.Mutable["home"])
	if err != nil {
		t.Fatal(err)
	}
	for slot := 0; slot <= 1; slot++ {
		fixture, err := fleet.Agent(slot)
		if err != nil {
			t.Fatal(err)
		}
		matches := 0
		for _, actor := range actors.Actors {
			binding := actor.Binding
			if binding.Slot != fmt.Sprintf("fixture:%d", slot) {
				continue
			}
			matches++
			paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: filepath.Join(fleet.Roots.Mutable["data"], "pair"), RepoScope: binding.Scope, Tag: binding.Tag})
			if err != nil {
				t.Fatal(err)
			}
			paneRaw, err := osfs.ReadRegularMetadata(ctx, paths.Pane(binding.Agent), panebirth.MaxRecordBytes)
			if err != nil {
				t.Fatal(err)
			}
			pane, err := panebirth.DecodeRecord(paneRaw)
			if err != nil {
				t.Fatal(err)
			}
			identity := pane.Identity
			if identity.Validate() != nil || identity.Namespace != namespace || identity.Scope != binding.Scope || identity.Tag != binding.Tag || identity.Agent != fixture.Agent || identity.Session != binding.Session || identity.Nonce != binding.Nonce {
				t.Fatalf("pane does not name exact registered launch: binding=%+v pane=%+v fixture=%+v", binding, pane, fixture)
			}
			markerPath, err := cache.ZellijReadyMarker(namespace, identity.Session)
			if err != nil {
				t.Fatal(err)
			}
			markerRaw, err := osfs.ReadRegularMetadata(ctx, markerPath, panebirth.MaxRecordBytes)
			if err != nil {
				t.Fatal(err)
			}
			marker, err := zellijquery.DecodeReadyMarker(markerRaw)
			if err != nil || !marker.Identity.Equal(identity) {
				t.Fatalf("namespace marker differs from first-pane proof: marker=%+v pane=%+v err=%v", marker, pane, err)
			}
			readyRaw, err := osfs.ReadRegularMetadata(ctx, paths.AgentReady(binding.Agent), panebirth.MaxRecordBytes)
			if err != nil {
				t.Fatal(err)
			}
			ready, err := readiness.Decode(string(readyRaw))
			if err != nil || ready.Tag != identity.Tag || ready.Agent != identity.Agent || ready.Session != identity.Session || ready.Nonce != identity.Nonce || ready.PID != fixture.PID || ready.AgentIdentity != fixture.Generation {
				t.Fatalf("AgentReady differs from exact launch: ready=%+v pane=%+v fixture=%+v err=%v", ready, pane, fixture, err)
			}
			t.Logf("exact startup run=%s slot=%s identity=%+v fixture_pid=%d generation=%s pane_sha256=%x marker_sha256=%x agent_ready_sha256=%x", fleet.RunID, binding.Slot, identity, fixture.PID, fixture.Generation, sha256.Sum256(paneRaw), sha256.Sum256(markerRaw), sha256.Sum256(readyRaw))
		}
		if matches != 1 {
			t.Fatalf("expected one actor for fixture:%d; matches=%d", slot, matches)
		}
	}
}

// Inspect the same documented private log source already retained by testcouch.
// Refuse incomplete evidence rather than claiming a panic-free qualification.
func startupReadinessServerLog(t *testing.T, ctx context.Context, fleet *testcouch.Fleet) {
	t.Helper()
	path := filepath.Join(fleet.Roots.Mutable["tmp"], fmt.Sprintf("zellij-%d", os.Getuid()), "zellij-log", "zellij.log")
	raw, err := osfs.ReadRegularMetadata(ctx, path, 1<<20)
	if err != nil {
		t.Fatalf("native startup server log unavailable for qualification: %v", err)
	}
	t.Logf("native startup complete server log run=%s bytes=%d sha256=%x output=%q", fleet.RunID, len(raw), sha256.Sum256(raw), raw)
	if strings.Contains(strings.ToLower(string(raw)), "panicked") {
		t.Error("native server panic recorded during startup qualification")
	}
}
