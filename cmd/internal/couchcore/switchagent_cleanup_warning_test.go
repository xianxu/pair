package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSwitchAgentRetainsTypedCleanupWarningOnSuccessfulReplacement(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	env.Lifecycle.cleanupErr = errors.New("source sidecar cleanup refused")
	args := []string{}
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	result, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: prepared.Fingerprint})
	if err != nil || result.Outcome != SwitchStarted {
		t.Fatalf("cleanup warning incorrectly refused replacement: %+v, %v", result, err)
	}
	if result.CleanupWarning != "source sidecar cleanup refused" || !strings.Contains(result.Warning(), result.CleanupWarning) {
		t.Fatalf("successful replacement lost its typed cleanup fact or existing warning: %+v", result)
	}
}
