package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
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
	env, local := slotRecoveryOperationFixture(t)
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
