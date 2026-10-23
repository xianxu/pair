package couchcore

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"testing"
)

func TestEditedRestartRollbackReceiptNamesExactCASResult(t *testing.T) {
	for _, scenario := range []string{"exact", "wrong-nonce", "stale-revision"} {
		t.Run(scenario, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			parked := createParkedThreadInCouch(t, env, profile)
			claim, err := env.Couch.Threads.CommitStartClaim(parked.Address, parked.Revision, "/repo/.git", env.Now, StartEvent{Kind: StartClaimed, Nonce: "edited-attempt", Owner: SupervisorOwner{PID: 777, Identity: "owner"}, Profile: &profile, Shape: StartColdResume})
			if err != nil {
				t.Fatal(err)
			}
			expected := claim
			nonce := "edited-attempt"
			if scenario == "wrong-nonce" {
				nonce = "foreign-attempt"
			}
			if scenario == "stale-revision" {
				name := "metadata"
				if _, err := env.Couch.Threads.ApplyThreadMetadata(claim.Address, claim.Revision, ThreadMetadataPatch{Name: &name}); err != nil {
					t.Fatal(err)
				}
			}
			receipt := &startRollbackReceipt{}
			err = env.Couch.rollbackTrackedStartObserved(expected, nonce, receipt)
			if scenario != "exact" {
				if err == nil || receipt.Nonce != "" || receipt.Record.Revision != 0 {
					t.Fatal("unproved rollback produced receipt", receipt, err)
				}
				return
			}
			if err != nil || receipt.Nonce != nonce || receipt.Record.Address != claim.Address || receipt.Record.Revision <= claim.Revision || hasOccupiedIncarnation(receipt.Record) {
				t.Fatal("missing exact rollback receipt", receipt, err)
			}
			before := receipt.Record.Revision
			name := "later metadata"
			changed, err := env.Couch.Threads.ApplyThreadMetadata(claim.Address, before, ThreadMetadataPatch{Name: &name})
			if err != nil {
				t.Fatal(err)
			}
			if changed.Revision <= before || receipt.Record.Revision != before {
				t.Fatal("later read replaced exact CAS receipt")
			}
		})
	}
}

func TestTrackedLaunchReportsItsOwnExactRollback(t *testing.T) {
	for _, phase := range []string{"spawn", "pre-ack", "post-ack"} {
		t.Run(phase, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			parked := createParkedThreadInCouch(t, env, profile)
			claim, err := env.Couch.Threads.CommitStartClaim(parked.Address, parked.Revision, "/repo/.git", env.Now, StartEvent{Kind: StartClaimed, Nonce: "edited-attempt", Owner: SupervisorOwner{PID: 777, Identity: "owner"}, Profile: &profile, Shape: StartColdResume})
			if err != nil {
				t.Fatal(err)
			}
			raw, err := launcher.BuildCouchResumeLaunchProfile(string(claim.Address.Tag), profile.Agent, profile.Argv, bindID(3))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch phase {
			case "spawn":
				env.Runner.FailNextStart(errors.New("spawn failure"))
			case "pre-ack":
				env.Runner.AfterBlockedStart = func(string) { cancel() }
			case "post-ack":
				env.Runner.BeforeAcknowledge = func(string) error { return errors.New("acknowledgement failure") }
			}
			receipt := &startRollbackReceipt{}
			_, h, err := env.Couch.launchTrackedThread(trackedThreadLaunch{Context: ctx, Thread: claim, Nonce: "edited-attempt", Args: StartArgs{Worktree: "/repo", Cwd: "/repo", Stack: profile.Agent, ExtraArgs: []string{}}, StartedAt: env.Now, ProfileRaw: raw, Resume: true, Rollback: receipt})
			if err == nil {
				t.Fatal("failure was swallowed")
			}
			if receipt.Nonce != "edited-attempt" || receipt.Record.Revision <= claim.Revision || hasOccupiedIncarnation(receipt.Record) {
				t.Fatal("existing cleanup lost exact rollback result", receipt, err)
			}
			if h != nil && h.Alive() {
				t.Fatal("receipt preceded exact helper death")
			}
		})
	}
}
