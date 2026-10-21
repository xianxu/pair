package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestRecoveryFreshMissingRecordCarriesOrientationAndStrictReadiness(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	env.Git.replies[GitCall{Dir: local.slot.WorktreeRoot, Args: "symbolic-ref --quiet --short HEAD"}] = "000556-recover-missing-record"
	var launched launcher.TrustedLaunchProfile
	registered := map[string]bool{}
	after := env.Runner.AfterAcknowledge
	env.Runner.AfterAcknowledge = func(id string) error {
		if err := after(id); err != nil {
			return err
		}
		child := env.Runner.Child(id)
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &launched); err != nil {
			return err
		}
		address := ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(environmentValue(child.Env, "COUCH_THREAD_TAG"))}
		current, err := local.GetThread(address)
		if err != nil {
			return err
		}
		claim := current.Incarnations[0].Start
		if claim == nil {
			t.Fatal("helper acknowledged without committed claim")
		}
		if !requireLauncherAcceptsRecovery(t, env.Runner, id) {
			return errors.New("recordless Fresh child lacks committed recovery authority")
		}
		if current.RecoveryIntent == nil || claim.RecoveryIntent == nil || !reflect.DeepEqual(current.RecoveryIntent, claim.RecoveryIntent) || !reflect.DeepEqual(current.RecoveryIntent, &launched.Recovery.Intent) {
			return errors.New("recordless Fresh intent differs between committed record, start and child")
		}
		env.Proc.Set(current.Incarnations[0].PID, current.Incarnations[0].Identity)
		var intent launcher.CouchSessionIntent
		if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchSessionIntentEnv)), &intent); err != nil {
			return err
		}
		env.Artifacts.SetPairSession(address, intent.Name, true)
		env.Artifacts.SetNativeBinding(address, launched.Agent, sessioninventory.BindingEstablished, "new-native-missing-record")
		registered[launched.Agent+"/"+launched.FreshNonce] = launched.FreshNonce == claim.Nonce
		return nil
	}
	readyChecks := 0
	env.Couch.FreshRegistration = func(ctx context.Context, address ThreadAddress, agent, nonce string) (bool, error) {
		readyChecks++
		binding, err := env.Artifacts.PairSession(address)
		return registered[agent+"/"+nonce] && binding.Present, err
	}
	result, err := env.Couch.startFreshRecoveryWithoutRecord(t.Context(), RecoverySelection{Target: ResumeTarget{Path: local.slot.WorktreeRoot}, Kind: RecoveryFresh})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != Fresh {
		t.Fatalf("missing-record recovery mode = %q, want Fresh", result.Mode)
	}
	if result.Start.Handle == nil || !launched.FreshRequired || launched.ResumeRequired || launched.FreshNonce == "" || readyChecks == 0 {
		t.Fatalf("Fresh missing strict readiness: profile=%+v checks=%d", launched, readyChecks)
	}
	for _, want := range []string{"000556-recover-missing-record", "#556", "Previous TTY capture unavailable"} {
		if !strings.Contains(launched.Prompt, want) {
			t.Fatalf("missing %q in orientation %q", want, launched.Prompt)
		}
	}
	if strings.Contains(launched.Prompt, "Original capture coordinates") || launched.RequiredSessionID != "" {
		t.Fatalf("invented prior screen/native identity: %+v", launched)
	}
}

func TestRecoveryFreshMissingRecordRefusesFutureVersion(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	if err := os.MkdirAll(local.root, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(local.root, "thread.json")
	original := []byte(`{"schema_version":999}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := env.Couch.startFreshRecoveryWithoutRecord(t.Context(), RecoverySelection{Target: ResumeTarget{Path: local.slot.WorktreeRoot}, Kind: RecoveryFresh})
	if err == nil {
		t.Fatal("future record overwritten")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(after, original) || len(env.Runner.children) != 0 {
		t.Fatalf("future record changed: %q %v", after, readErr)
	}
}

func TestRecoveryFreshOrdinarySlotKeepsOrdinaryLaunch(t *testing.T) {
	t.Cleanup(func() {
		if t.Failed() {
			t.Log(testwait.HostState())
		}
	})
	env, local := slotRecoverySemanticFixture(t)
	readyChecks := 0
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { readyChecks++; return false, nil }
	result, err := env.Couch.StartFreshSlot(t.Context(), local.slot.WorktreeRoot, "claude")
	if err != nil {
		t.Fatal(err)
	}
	var profile launcher.TrustedLaunchProfile
	if err := json.Unmarshal([]byte(environmentValue(env.Runner.Child(result.Handle.ID()).Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Prompt != "" || profile.FreshRequired || profile.FreshNonce != "" || readyChecks != 0 {
		t.Fatalf("ordinary Fresh changed: %+v checks=%d", profile, readyChecks)
	}
}

type recordlessRecoveryArrival struct {
	*FakeThreadArtifactCollisionChecker
	arrive func()
}

func (a recordlessRecoveryArrival) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	claim, err := a.FakeThreadArtifactCollisionChecker.Claim(address)
	if err == nil && a.arrive != nil {
		a.arrive()
	}
	return claim, err
}

func TestRecoveryFreshMissingRecordLeavesForeignOwnersUntouched(t *testing.T) {
	for _, mode := range []string{"listed-foreign", "unknown-foreign", "arrives-during-prepare"} {
		t.Run(mode, func(t *testing.T) {
			t.Cleanup(func() {
				if t.Failed() {
					t.Log(testwait.HostState())
				}
			})
			env, local := slotRecoverySemanticFixture(t)
			scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
			foreign := ThreadAddress{RepoScope: scope.Key, Tag: "couch-5555555555555555"}
			arrive := func() {
				env.Artifacts.SetSessionPresence(foreign, SessionObservation{State: SessionPresent})
				env.Artifacts.SetPairSession(foreign, "foreign-terminal", true)
				env.Proc.Set(777, "foreign-generation")
			}
			if mode == "arrives-during-prepare" {
				env.Couch.Artifacts = recordlessRecoveryArrival{FakeThreadArtifactCollisionChecker: env.Artifacts, arrive: arrive}
			} else {
				arrive()
			}
			if mode == "unknown-foreign" {
				env.Artifacts.BeforePairSession = func(address ThreadAddress) error {
					if address == foreign {
						return errors.New("foreign identity unavailable")
					}
					return nil
				}
			}
			preparations := 0
			env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
				preparations++
				return ProvisionResult{}, errors.New("must not reconcile beside unproved owners")
			})
			nativeArtifact := filepath.Join(local.slot.WorktreeRoot, "foreign-session-context")
			original := []byte("foreign session state must stay byte-identical")
			if err := os.WriteFile(nativeArtifact, original, 0600); err != nil {
				t.Fatal(err)
			}
			var profile launcher.TrustedLaunchProfile
			readyAddress := ThreadAddress{}
			readyNonce := ""
			env.Runner.AfterAcknowledge = func(id string) error {
				child := env.Runner.Child(id)
				if err := json.Unmarshal([]byte(environmentValue(child.Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
					return err
				}
				address := ThreadAddress{RepoScope: environmentValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: ThreadTag(profile.Tag)}
				current, err := local.GetThread(address)
				if err != nil {
					return err
				}
				if current.Incarnations[0].Start == nil || current.Incarnations[0].Start.Nonce != profile.FreshNonce {
					return errors.New("Fresh nonce differs from claim")
				}
				readyAddress, readyNonce = address, profile.FreshNonce
				env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
				env.Artifacts.SetSessionPresence(address, SessionObservation{State: SessionPresent})
				return nil
			}
			env.Couch.FreshRegistration = func(_ context.Context, address ThreadAddress, _ string, nonce string) (bool, error) {
				return nonce != "" && nonce == readyNonce && address == readyAddress, nil
			}
			result, err := env.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Path: local.slot.WorktreeRoot}, Kind: RecoveryFresh})
			if err != nil {
				t.Fatal(err)
			}
			if result.Start.Handle == nil || result.Start.Record.Thread == foreign || profile.Prompt == "" || !profile.FreshRequired || profile.ResumeRequired || profile.FreshNonce == "" || (!strings.Contains(result.Notice, "untouched") || !strings.Contains(result.Notice, "workspace setup was not refreshed")) {
				t.Fatalf("recordless Fresh lost isolation: %+v profile=%+v", result, profile)
			}
			raw, readErr := os.ReadFile(nativeArtifact)
			identity, identityErr := env.Proc.Identity(777)
			if preparations != 0 || readErr != nil || !bytes.Equal(raw, original) || identityErr != nil || identity != "foreign-generation" || len(env.Proc.Signals[777]) != 0 || len(env.Proc.GroupSignals[777]) != 0 || len(env.Artifacts.Quiesces()) != 0 {
				t.Fatalf("foreign owner changed: prepare=%d bytes=%q identity=%q signals=%v groups=%v quiesces=%v err=%v/%v", preparations, raw, identity, env.Proc.Signals, env.Proc.GroupSignals, env.Artifacts.Quiesces(), readErr, identityErr)
			}
		})
	}
}

func TestRecoveryFreshMissingRecordNeverRepairsBesideForeignOwner(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	foreign := ThreadAddress{RepoScope: scope.Key, Tag: "couch-5555555555555555"}
	env.Artifacts.SetSessionPresence(foreign, SessionObservation{State: SessionPresent})
	env.Artifacts.SetPairSession(foreign, "foreign-terminal", true)
	fixture := env.Couch.Slots.(*OSSlotCatalog).IO.(*ProvisionFixture)
	admin := fixture.git(local.slot.WorktreeRoot, "rev-parse", "--absolute-git-dir")
	if err := os.RemoveAll(admin); err != nil {
		t.Fatal(err)
	}
	preparations := 0
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		preparations++
		return ProvisionResult{}, errors.New("repair forbidden")
	})
	_, err := env.Couch.RecoverSelection(t.Context(), RecoverySelection{Target: ResumeTarget{Path: local.slot.WorktreeRoot}, Kind: RecoveryFresh})
	if err == nil || preparations != 0 || len(env.Runner.Ops) != 0 || len(env.Artifacts.Quiesces()) != 0 {
		t.Fatalf("unverified host acted: err=%v preparations=%d starts=%v quiesces=%v", err, preparations, env.Runner.Ops, env.Artifacts.Quiesces())
	}
	if _, err := os.Stat(admin); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unproved owner's host repaired: %v", err)
	}
}

func TestRecoveryFreshMissingRecordConfirmedFailureRetainsIntent(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Runner.SetExited(id, 23)
		return nil
	}
	_, err := env.Couch.startFreshRecoveryWithoutRecord(t.Context(), RecoverySelection{Target: ResumeTarget{Path: local.slot.WorktreeRoot}, Kind: RecoveryFresh})
	if err == nil {
		t.Fatal("exited helper became successful Fresh startup")
	}
	var unconfirmed *RecoveryLaunchUnconfirmed
	if errors.As(err, &unconfirmed) {
		t.Fatalf("confirmed dead helper remained unconfirmed: %v", err)
	}
	current, readErr := local.observeSlotCurrent()
	if readErr != nil || current.Record == nil || len(current.Record.Incarnations) != 0 || current.Record.RecoveryIntent == nil || current.Record.RecoveryIntent.Kind() != RecoveryFresh {
		t.Fatalf("confirmed failure lost Fresh selection or retained creating helper: %+v %v", current.Record, readErr)
	}
}
