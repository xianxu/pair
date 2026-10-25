package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// TestDisposableCouchFleet is the shared helper's first native consumer.
// Drain/park/resume policy qualification belongs to its owning issues.
func TestDisposableCouchFleet(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 or make test-couch-fleet")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			started := time.Now()
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 3, AgentKind: agent})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			if err := fleet.Prepare(setup); err != nil {
				cancelSetup()
				t.Fatal(err)
			}
			cancelSetup()
			ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatal(err)
			}
			for n := 1; n <= 3; n++ {
				if err := fleet.StartWorker(ctx, n); err != nil {
					t.Fatal(err)
				}
			}
			actors, err := fleet.Actors(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(actors.Actors) != 4 {
				t.Fatalf("wrong genuine cohort: %+v", actors)
			}
			seen := map[string]bool{}
			for _, actor := range actors.Actors {
				seen[actor.Binding.Slot] = true
			}
			for _, slot := range []string{"fixture:0", "fixture:1", "fixture:2", "fixture:3"} {
				if !seen[slot] {
					t.Fatalf("missing %s in %+v", slot, actors)
				}
			}
			for n := 0; n <= 3; n++ {
				ready, err := fleet.Agent(n)
				if err != nil || ready.Agent != agent {
					t.Fatalf("actor %d personality: %+v %v", n, ready, err)
				}
			}
			t.Logf("native cohort: agent=%s steward=fixture:0 workers=fixture:1,fixture:2,fixture:3 elapsed=%s", agent, time.Since(started))
			// The same genuine actor may own simultaneous requests; each result is
			// correlated separately and traverses the real CLI/broker transport.
			var group sync.WaitGroup
			for i := 0; i < 3; i++ {
				group.Add(1)
				go func() {
					defer group.Done()
					result, err := fleet.ActorCLI(ctx, 0, "--actors", "--json")
					if err != nil || result.ExitCode != 0 {
						t.Errorf("concurrent actor CLI: %+v %v", result, err)
					}
				}()
			}
			group.Wait()
			message := "shared-native-fleet-ordinary-message-" + agent
			result, err := fleet.ActorCLI(ctx, 0, "--send-to", "fixture:1", "--message", message)
			if err != nil || result.ExitCode != 0 {
				t.Fatalf("ordinary send: %+v %v", result, err)
			}
			// Extract the actual receipt ID, then prove submission through public
			// status and independently through the fixture's submitted-input log.
			fields := strings.Fields(result.Stdout)
			if len(fields) == 0 {
				t.Fatalf("missing receipt: %+v", result)
			}
			id := fields[0]
			if err := waitFleet(ctx, func() (bool, error) {
				status, e := fleet.ActorCLI(ctx, 0, "--message-status", id, "--json")
				if e != nil {
					return false, e
				}
				var response couchmessage.Response
				if err := json.Unmarshal([]byte(status.Stdout), &response); err != nil {
					return false, err
				}
				if response.Receipt == nil {
					return false, fmt.Errorf("missing receipt: %s", status.Stdout)
				}
				return response.Receipt.Status == couchmessage.Submitted, nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := waitFleet(ctx, func() (bool, error) {
				accepted, e := fleet.Accepted(1)
				if e != nil {
					if errors.Is(e, os.ErrNotExist) {
						return false, nil
					}
					return false, e
				}
				for _, input := range accepted {
					if strings.Contains(input.Body, message) {
						return true, nil
					}
				}
				return false, nil
			}); err != nil {
				t.Fatal(err)
			}
			t.Logf("native delivery: receipt=%s status=submitted recipient=fixture:1 accepted_body=%q elapsed=%s", id, message, time.Since(started))
			if err := fleet.Resize(36, 132); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Detach(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.RestartSupervisor(ctx); err != nil {
				t.Fatal(err)
			}
			if err := waitFleet(ctx, func() (bool, error) {
				actors, e := fleet.Actors(ctx)
				if errors.Is(e, agentfixture.ErrMailboxFull) {
					return false, e
				}
				return e == nil && len(actors.Actors) == 4, nil
			}); err != nil {
				t.Fatal(err)
			}
			t.Logf("native detach/restart: all four actors returned elapsed=%s", time.Since(started))
			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelCleanup()
			if err := fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err := fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(fleet.Roots.Root); !os.IsNotExist(err) {
				t.Fatalf("owned root retained after successful cleanup: %v", err)
			}
		}) {
			break
		}
	}
}

func waitFleet(ctx context.Context, check func() (bool, error)) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if done, err := check(); done || err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func TestFleetWaitStopsOnPermanentRefusal(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	calls := 0
	err := waitFleet(ctx, func() (bool, error) {
		calls++
		return false, agentfixture.ErrMailboxFull
	})
	if !errors.Is(err, agentfixture.ErrMailboxFull) || calls != 1 {
		t.Fatalf("permanent refusal retried: calls=%d err=%v", calls, err)
	}
}

// TestDisposableCouchFleetFullQuit is #827's physical full quit: one Alt+x and
// one confirmation park every genuine actor in a 4- and an 8-slot cohort, for
// each agent kind, and Couch exits with every slot parked.
func TestDisposableCouchFleetFullQuit(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 or make test-couch-fleet")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		for _, workers := range []int{3, 7} {
			if !t.Run(fmt.Sprintf("%s/%d", agent, workers+1), func(t *testing.T) {
				started := time.Now()
				fleet := testcouch.New(t, binaries, testcouch.Options{Workers: workers, AgentKind: agent})
				setup, cancelSetup := context.WithTimeout(t.Context(), 240*time.Second)
				if err := fleet.Prepare(setup); err != nil {
					cancelSetup()
					t.Fatal(err)
				}
				cancelSetup()
				ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
				defer cancel()
				if err := fleet.Start(ctx); err != nil {
					t.Fatal(err)
				}
				for n := 1; n <= workers; n++ {
					if err := fleet.StartWorker(ctx, n); err != nil {
						t.Fatal(err)
					}
				}
				pids := map[int]int{}
				for n := 0; n <= workers; n++ {
					ready, err := fleet.Agent(n)
					if err != nil || ready.Agent != agent {
						t.Fatalf("actor %d personality: %+v %v", n, ready, err)
					}
					pids[n] = ready.PID
				}
				if err := fleet.Resize(36, 132); err != nil {
					t.Fatal(err)
				}
				quit := time.Now()
				if err := fleet.ParkAndQuit(ctx); err != nil {
					t.Fatal(err)
				}
				t.Logf("native full quit: agent=%s slots=%d quit=%s elapsed=%s", agent, workers+1, time.Since(quit), time.Since(started))
				for n, pid := range pids {
					if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
						t.Errorf("fixture:%d agent pid %d still running after full quit: %v", n, pid, err)
					}
				}
				// #827's contract is verified inactive history for every targeted
				// incarnation, read from the private store the console wrote. The
				// native resume binding is a separate concern: a fixture quit
				// before Couch discovered its transcript parks verified yet lists
				// "binding lost", so --list is evidence here, not the assertion.
				namespace, err := couchcore.ResolveCouchNamespace(fleet.Roots.Mutable["store"], fleet.Workspaces.Primary)
				if err != nil {
					t.Fatal(err)
				}
				snapshot, err := couchcore.NewThreadStore(namespace).Snapshot()
				if err != nil {
					t.Fatal(err)
				}
				if len(snapshot.Records) != workers+1 || len(snapshot.Unreadable) != 0 {
					t.Fatalf("store holds %d threads (%d unreadable), want %d", len(snapshot.Records), len(snapshot.Unreadable), workers+1)
				}
				for _, record := range snapshot.Records {
					if record.VerifiedPark == nil || record.Park != nil || len(record.Incarnations) != 0 {
						t.Errorf("%s not verified parked: park=%+v verified=%v incarnations=%+v", record.Address.Tag, record.Park, record.VerifiedPark != nil, record.Incarnations)
					}
				}
				list, err := fleet.ListThreads(ctx)
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("post-quit --list:\n%s", list)
				cleanup, cancelCleanup := context.WithTimeout(context.Background(), 30*time.Second)
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
}

// pair#830 T6: a full quit puts every slot to sleep. Couch starts again with
// its hubs and brings none back (#739). The operator resumes the main hub
// from the switcher; its `couch --wake` brings back exactly the other asleep
// hub, and the plain worker stays asleep for that hub to wake.
func TestDisposableCouchFleetWake(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 or make test-couch-fleet")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			// fixture:1 is a second hub; fixture:2 is a plain worker.
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 2, ExplicitHub: true, Hubs: []int{1}, AgentKind: agent})
			setup, cancelSetup := context.WithTimeout(t.Context(), 240*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 300*time.Second)
			defer cancel()
			// A fresh namespace starts both configured hubs by itself (#863);
			// the main hub then launches the plain worker.
			if err := fleet.Start(ctx); err != nil {
				t.Fatalf("fresh startup: %v\n%s", err, fleet.Screen())
			}
			if err := fleet.WaitAgent(ctx, 1); err != nil {
				t.Fatalf("second hub start: %v\n%s", err, fleet.Screen())
			}
			if err := fleet.WaitActors(ctx, 2); err != nil {
				t.Fatalf("hub registration: %v\n%s", err, fleet.Screen())
			}
			if err := fleet.StartWorker(ctx, 2); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Resize(36, 132); err != nil {
				t.Fatal(err)
			}
			// Only a conversation is resumable from disk: each hub the test
			// later brings back accepts one turn first.
			wakeFleetConverse(t, ctx, fleet, 1, 0, "wake-main-hub-turn-"+agent)
			wakeFleetConverse(t, ctx, fleet, 0, 1, "wake-second-hub-turn-"+agent)
			if err := fleet.ParkAndQuit(ctx); err != nil {
				t.Fatal(err)
			}
			asleep := func() map[string]bool {
				t.Helper()
				namespace, err := couchcore.ResolveCouchNamespace(fleet.Roots.Mutable["store"], fleet.Workspaces.Primary)
				if err != nil {
					t.Fatal(err)
				}
				snapshot, err := couchcore.NewThreadStore(namespace).Snapshot()
				if err != nil {
					t.Fatal(err)
				}
				// Keyed by fixture slot: tags are generated, workspaces are not.
				slots := map[string]string{fleet.Workspaces.Primary: "0", fleet.Workspaces.Workers[1]: "1", fleet.Workspaces.Workers[2]: "2"}
				out := map[string]bool{}
				for _, record := range snapshot.Records {
					slot, known := slots[record.StartingPath]
					if !known {
						t.Fatalf("unexpected thread %s at %s", record.Address.Tag, record.StartingPath)
					}
					_, ok := couchcore.ThreadAsleep(record)
					out[slot] = ok
				}
				return out
			}
			slept := asleep()
			if len(slept) != 3 {
				t.Fatalf("store threads after full quit: %v", slept)
			}
			for tag, ok := range slept {
				if !ok {
					t.Fatalf("%s not asleep after full quit: %v", tag, slept)
				}
			}
			if err := fleet.QuitConsole(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.StartCold(ctx); err != nil {
				t.Fatal(err)
			}
			// Startup brings no hub back after a full quit (#739).
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(3 * time.Second):
			}
			for n := 0; n <= 2; n++ {
				if ready, err := fleet.Agent(n); err == nil && syscall.Kill(ready.PID, 0) == nil {
					t.Fatalf("fixture:%d came back at startup: %+v", n, ready)
				}
			}
			// The operator resumes the main hub from the switcher.
			// Its row reads as the bare repo name (slot 0) and is selected
			// when the cold start opens the switcher; Tab opens its actions.
			wakeFleetOpenSwitcher(t, ctx, fleet)
			forceFleetWaitScreen(t, ctx, fleet, "▸ fixture  "+fleet.Workspaces.Primary+" ")
			forceFleetPress(t, fleet, "\t", 9, 1)
			forceFleetWaitScreen(t, ctx, fleet, "Resume from disk")
			if err := fleet.Input([]byte("Resume from disk")); err != nil {
				t.Fatal(err)
			}
			forceFleetPress(t, fleet, "\r", 13, 1)
			if err := fleet.WaitAgent(ctx, 0); err != nil {
				t.Fatalf("main hub resume: %v\n%s", err, fleet.Screen())
			}
			if err := fleet.WaitActors(ctx, 1); err != nil {
				t.Fatalf("main hub registration: %v\n%s", err, fleet.Screen())
			}
			result, err := fleet.ActorCLI(ctx, 0, "--wake")
			if err != nil || result.ExitCode != 0 {
				t.Fatalf("couch --wake: %+v %v", result, err)
			}
			t.Logf("couch --wake: %s", strings.TrimSpace(result.Stdout))
			if err := fleet.WaitAgent(ctx, 1); err != nil {
				t.Fatalf("asleep hub not woken: %v\n%s", err, fleet.Screen())
			}
			if ready, err := fleet.Agent(2); err == nil && syscall.Kill(ready.PID, 0) == nil {
				t.Fatalf("plain worker woken by couch --wake: %+v", ready)
			}
			if woke := asleep(); woke["0"] || woke["1"] || !woke["2"] {
				t.Fatalf("asleep after wake: %v", woke)
			}
			// A repeat finds nothing left for it to wake.
			again, err := fleet.ActorCLI(ctx, 0, "--wake")
			if err != nil || again.ExitCode != 0 {
				t.Fatalf("repeat couch --wake: %+v %v", again, err)
			}
			t.Logf("repeat couch --wake: %s", strings.TrimSpace(again.Stdout))
			if !strings.Contains(again.Stdout, "no asleep hubs to wake") {
				t.Fatalf("repeat couch --wake woke something: %+v", again)
			}
			if ready, err := fleet.Agent(2); err == nil && syscall.Kill(ready.PID, 0) == nil {
				t.Fatalf("plain worker started by a repeat couch --wake: %+v", ready)
			}
			if woke := asleep(); woke["0"] || woke["1"] || !woke["2"] {
				t.Fatalf("asleep after repeat wake: %v", woke)
			}
			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 30*time.Second)
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

// wakeFleetConverse sends one message from hub from to hub to and waits until
// the receiving fixture records it as an accepted turn.
func wakeFleetConverse(t *testing.T, ctx context.Context, fleet *testcouch.Fleet, from, to int, message string) {
	t.Helper()
	result, err := fleet.ActorCLI(ctx, from, "--send-to", fmt.Sprintf("fixture:%d", to), "--message", message)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("send to fixture:%d: %+v %v", to, result, err)
	}
	if err := waitFleet(ctx, func() (bool, error) {
		accepted, err := fleet.Accepted(to)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		for _, input := range accepted {
			if strings.Contains(input.Body, message) {
				return true, nil
			}
		}
		return false, err
	}); err != nil {
		t.Fatalf("fixture:%d never accepted its turn: %v", to, err)
	}
}

// wakeFleetOpenSwitcher leaves the switcher open at its root. A cold start
// opens it by itself, possibly after a delay; pressing Ctrl+Space then would
// open the start form instead.
func wakeFleetOpenSwitcher(t *testing.T, ctx context.Context, fleet *testcouch.Fleet) {
	t.Helper()
	const root = "Ctrl+Y copy last notice / selected error"
	settle, stop := context.WithTimeout(ctx, 5*time.Second)
	open := waitFleet(settle, func() (bool, error) { return strings.Contains(fleet.Screen(), root), nil }) == nil
	stop()
	if !open {
		forceFleetPress(t, fleet, "\x00", 32, 5)
		forceFleetWaitScreen(t, ctx, fleet, root)
	}
}
