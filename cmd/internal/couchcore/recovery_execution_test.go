package couchcore

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"os"
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
				f.Couch.RecoveryEvidence = func(context.Context, ThreadRecord, RecoveryObservationRequest) (RecoveryObservation, error) {
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

// pair#600: a Fresh recovery onto a new address launches an authority the
// launcher accepts. Run the real launcher validation on the authority this
// path hands the child, against the record it names, once the old record is
// past revision 1 (an earlier launch, park or rename always moves it there).
func TestFreshRecoveryOntoNewAddressPassesLauncherValidation(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	local := f.Couch.Threads
	for i := 0; i < 4; i++ {
		updated, err := local.updateExistingThread(f.record.Address, f.record.Revision, func(next *ThreadRecord) error { next.Description = fmt.Sprintf("edit %d", i); return nil })
		if err != nil {
			t.Fatal(err)
		}
		f.record = updated
	}
	validated := false
	acknowledge := f.Runner.AfterAcknowledge
	f.Runner.AfterAcknowledge = func(id string) error {
		validated = requireLauncherAcceptsRecovery(t, f.Runner, id)
		return acknowledge(id)
	}
	result, err := f.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Address: f.record.Address}, Kind: RecoveryFresh})
	if err != nil || result.Start.Handle == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !validated {
		t.Fatal("no recovery child started")
	}
}

// requireLauncherAcceptsRecovery runs the launcher's real authority check on a
// recovery child's profile against the record it names. Call it once Couch has
// acknowledged the child: the helper validates then, with its PID on the
// creating incarnation. It reports whether the child carried recovery authority.
func requireLauncherAcceptsRecovery(t *testing.T, runner *FakeRunner, id string) bool {
	t.Helper()
	var profile launcher.TrustedLaunchProfile
	if err := json.Unmarshal([]byte(environmentValue(runner.Child(id).Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
		t.Errorf("launch profile: %v", err)
		return false
	}
	if profile.Recovery == nil {
		return false
	}
	a := *profile.Recovery
	raw, err := os.ReadFile(a.RecordPath)
	if err != nil {
		t.Error(err)
		return true
	}
	record, err := threadrecord.DecodePersisted(raw, a.Address, threadrecord.Validators{RepoScope: launcher.ValidateRepoScopeKey, Tag: launcher.ValidatePairTag})
	if err != nil || len(record.Incarnations) != 1 {
		t.Errorf("record: %+v %v", record, err)
		return true
	}
	inc := record.Incarnations[0]
	if err := launcher.ValidateRecoveryRecord(a, record, launcher.RepoScope{Key: a.Address.RepoScope}, a.Address.Tag, profile.Agent, record.WorkingPath, inc.PID, inc.Identity, a.Intent.CreatedAt.Add(time.Second)); err != nil {
		t.Errorf("launcher refuses the recovery authority: %v (source revision %d, record revision %d)", err, a.Intent.SourceRevision, a.RecordRevision)
	}
	return true
}
