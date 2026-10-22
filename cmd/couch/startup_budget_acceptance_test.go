package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch"
)

// Keep startup acceptance separate from delivery and detach/restart: those
// operations cannot establish whether cold registration consumed its budget.
func TestDisposableCouchStartupBudget(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_FLEET") != "1" {
		t.Skip("set PAIR_NATIVE_COUCH_FLEET=1 for disposable native startup acceptance")
	}
	binaries := testcouch.Build(t)
	for _, agent := range []string{"claude", "codex"} {
		if !t.Run(agent, func(t *testing.T) {
			fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 3, AgentKind: agent})
			setup, cancelSetup := context.WithTimeout(t.Context(), 180*time.Second)
			err := fleet.Prepare(setup)
			cancelSetup()
			if err != nil {
				t.Fatal(err)
			}
			// Reuse the existing fleet caller's budget; no per-phase renewal.
			started := time.Now()
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
			if err != nil || len(actors.Actors) != 4 {
				t.Fatalf("cold registration cohort: %+v: %v", actors, err)
			}
			for n := 0; n <= 3; n++ {
				ready, err := fleet.Agent(n)
				if err != nil || ready.Agent != agent {
					t.Fatalf("cold actor %d: %+v: %v", n, ready, err)
				}
			}
			t.Logf("cold startup: agent=%s steward=fixture:0 workers=fixture:1,fixture:2,fixture:3 elapsed=%s", agent, time.Since(started))
			cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelCleanup()
			if err := fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err := fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(fleet.Roots.Root); !os.IsNotExist(err) {
				t.Fatalf("owned root retained after cleanup: %v", err)
			}
		}) {
			break
		}
	}
}
