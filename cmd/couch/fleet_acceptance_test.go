package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch"
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
			waitFleet(t, ctx, func() bool {
				status, e := fleet.ActorCLI(ctx, 0, "--message-status", id, "--json")
				if e != nil {
					return false
				}
				var response couchmessage.Response
				return json.Unmarshal([]byte(status.Stdout), &response) == nil && response.Receipt != nil && response.Receipt.Status == couchmessage.Submitted
			})
			waitFleet(t, ctx, func() bool {
				accepted, e := fleet.Accepted(1)
				if e != nil {
					return false
				}
				for _, input := range accepted {
					if strings.Contains(input.Body, message) {
						return true
					}
				}
				return false
			})
			if err := fleet.Resize(36, 132); err != nil {
				t.Fatal(err)
			}
			if err := fleet.Detach(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.RestartSupervisor(ctx); err != nil {
				t.Fatal(err)
			}
			waitFleet(t, ctx, func() bool { actors, e := fleet.Actors(ctx); return e == nil && len(actors.Actors) == 4 })
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

func waitFleet(t *testing.T, ctx context.Context, check func() bool) {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if check() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
}
