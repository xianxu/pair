package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

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
