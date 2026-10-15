package couchcore

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecoveryExecutionOrder(t *testing.T) {
	for current := recoveryPrepared; current <= recoveryLaunched; current++ {
		for next := recoveryPrepared; next <= recoveryLaunched; next++ {
			got, err := AdvanceRecoveryExecution(current, next)
			allowed := next == current+1
			if allowed != (err == nil) || (allowed && got != next) || (!allowed && got != current) {
				t.Fatalf("%d -> %d = %d, %v", current, next, got, err)
			}
		}
	}
}

func TestRecoveryInvalidExecutionNeverClaimsOrLaunches(t *testing.T) {
	for _, kind := range []RecoverySelectionKind{RecoveryFresh, RecoveryProven} {
		t.Run(string(kind), func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			before, err := f.Couch.Threads.PreviewThreadContext(t.Context(), f.record.Address)
			if err != nil {
				t.Fatal(err)
			}
			selection := RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: kind}
			if kind == RecoveryFresh {
				_, err = f.Couch.launchRecoveryFreshAddress(context.Background(), before, selection, *before.LatestLaunchProfile, threadrecord.RecoveryIntent{}, recoveryPrepared, "repo", nil)
			} else {
				_, err = f.Couch.launchPreparedSelection(context.Background(), before, selection, *before.LatestLaunchProfile, threadrecord.RecoveryIntent{}, RecoveryChoice{}, recoveryPrepared, "repo", nil)
			}
			if err == nil || len(f.Runner.Ops) != 0 {
				t.Fatalf("err=%v ops=%v", err, f.Runner.Ops)
			}
			after, e := f.Couch.Threads.PreviewThreadContext(t.Context(), f.record.Address)
			if e != nil || after.Revision != before.Revision {
				t.Fatalf("invalid ordering wrote: %+v %v", after, e)
			}
		})
	}
}

func TestPreparedFreshUnavailableEvidenceStillLaunches(t *testing.T) {
	for _, state := range []string{"expired", "future", "unsupported"} {
		t.Run(state, func(t *testing.T) {
			f := newRecoveryExecutionFixture(t)
			switch state {
			case "expired":
				f.observation.Capture.CapturedAt = f.Now.Add(-365 * 24 * time.Hour)
			case "future":
				f.observation.Capture.CapturedAt = f.Now.Add(time.Hour)
			case "unsupported":
				f.Couch.RecoveryEvidence = func(context.Context, ThreadRecord) (RecoveryObservation, error) {
					return RecoveryObservation{Incomplete: true, Reasons: []string{"unsupported transcript format"}}, nil
				}
			}
			result, err := f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh})
			if err != nil || result.Start.Handle == nil || f.preserved != 0 {
				t.Fatalf("result=%+v err=%v preserved=%d", result, err, f.preserved)
			}
			child := f.Runner.Child(result.Start.Handle.ID())
			var profile launcher.TrustedLaunchProfile
			if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
				t.Fatal(err)
			}
			if !profile.FreshRequired || profile.Prompt == "" || !strings.Contains(result.Notice, "unavailable") {
				t.Fatalf("profile=%+v notice=%q", profile, result.Notice)
			}
		})
	}
}

func TestPreparedFreshAgentChoiceUsesNormalConfiguration(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	original := cloneLaunchProfile(*f.record.LatestLaunchProfile)
	got, err := f.Couch.recoveryLaunchProfileForSelection(t.Context(), f.record, RecoverySelection{Kind: RecoveryFresh, Agent: "grok"})
	if err != nil || got.Agent != "grok" || len(got.Argv) != 0 {
		t.Fatalf("profile=%+v err=%v", got, err)
	}
	if !reflect.DeepEqual(original, *f.record.LatestLaunchProfile) {
		t.Fatal("agent choice edited source config")
	}
	if _, err = f.Couch.recoveryLaunchProfileForSelection(t.Context(), f.record, RecoverySelection{Kind: RecoveryProven, Agent: "grok"}); err == nil {
		t.Fatal("native resume changed agent")
	}
}
