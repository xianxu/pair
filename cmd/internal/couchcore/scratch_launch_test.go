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
				_, h, err = env.Couch.Spawn(StartArgs{Worktree: "/repo", Stack: "codex", ExtraArgs: []string{"--sandbox", "workspace-write"}})
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
			wantAgent := "codex"
			if warm {
				wantAgent = "claude"
			}
			if owner.LatestLaunchProfile == nil || owner.LatestLaunchProfile.Agent != wantAgent {
				t.Fatalf("scratch admission did not receive selected %s profile: %+v", wantAgent, owner.LatestLaunchProfile)
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

func TestShowProjectsScratchWithoutPreparingEnvironment(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude"})
	called := 0
	env.Couch.SlotEnvironment = func(context.Context, ThreadRecord) []string { t.Fatal("show prepared scratch"); return nil }
	env.Couch.ScratchDiagnostic = func(_ context.Context, got ThreadRecord) string {
		called++
		if got.Address != record.Address || got.LatestLaunchProfile.Agent != "claude" {
			t.Fatalf("wrong diagnostic authority: %+v", got)
		}
		return "scratch: normal temp (sandbox doesn't allow /Volumes/TestScratch)"
	}
	result, err := dispatchTestOperation(env.Couch, "show", map[string]string{"ref": string(record.Address.Tag), "repo-scope": record.Address.RepoScope})
	if err != nil {
		t.Fatal(err)
	}
	show := result.(ShowResult)
	if called != 1 || len(show.Scratch) != 1 || !strings.Contains(show.Scratch[0], "sandbox doesn't allow") {
		t.Fatalf("calls=%d show=%+v", called, show)
	}
}
