package couchcore

import (
	"context"
	"strings"
	"testing"
)

func TestScratchEnvironmentReachesFreshAndWarmLaunches(t *testing.T) {
	for _, warm := range []bool{false, true} {
		name := "fresh"
		if warm {
			name = "warm"
		}
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			var owner ThreadRecord
			env.Couch.SlotEnvironment = func(_ context.Context, record ThreadRecord) []string {
				owner = record
				return []string{"TMPDIR=/private/ram/slot", "GOTMPDIR=/private/ram/slot", "CLAUDE_CODE_TMPDIR=/private/ram/slot"}
			}
			var h Handle
			var err error
			if !warm {
				_, h, err = env.Couch.Spawn(StartArgs{Worktree: "/repo"})
			} else {
				record := validThreadRecord(t)
				record.StartingPath, record.WorkingPath = "/repo", "/repo"
				record.Reservation = false
				record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
				record, err = env.Couch.Threads.CreateThread(record)
				if err != nil {
					t.Fatal(err)
				}
				env.Artifacts.SetDetachedSession(record.Address, "pair-"+string(record.Address.Tag))
				env.Runner.AfterAcknowledge = func(string) error {
					env.Artifacts.SetPairSession(record.Address, "pair-"+string(record.Address.Tag), true)
					return nil
				}
				_, h, err = env.Couch.Resume(record.Address)
			}
			if err != nil {
				t.Fatal(err)
			}
			if owner.StartingPath != "/repo" || owner.Address.Tag == "" {
				t.Fatalf("wrong retained owner: %+v", owner)
			}
			got := map[string]string{}
			for _, entry := range env.Runner.Child(h.ID()).Env {
				k, v, _ := strings.Cut(entry, "=")
				got[k] = v
			}
			for _, key := range []string{"TMPDIR", "GOTMPDIR", "CLAUDE_CODE_TMPDIR"} {
				if got[key] != "/private/ram/slot" {
					t.Fatalf("%s=%q", key, got[key])
				}
			}
		})
	}
}
