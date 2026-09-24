package couchcore

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"os"
	"path/filepath"
	"testing"
)

func TestFreshSlotTransmitsRegistrationNonce(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	data := t.TempDir()
	reader := OSOrientationStatusReader{DataDir: data, Proc: OSProcOps{}, Session: func(ThreadAddress) (PairSessionBinding, error) {
		return PairSessionBinding{Name: "fresh-session", Present: true}, nil
	}}
	env.Runner.AfterAcknowledge = func(id string) error {
		child := env.Runner.Child(id)
		var profile map[string]json.RawMessage
		raw := childEnvValue(child.Env, launcher.CouchLaunchProfileEnv)
		if err := json.Unmarshal([]byte(raw), &profile); err != nil {
			return err
		}
		var nonce string
		if err := json.Unmarshal(profile["launch_nonce"], &nonce); err != nil {
			t.Fatalf("fresh profile missing transaction nonce: %s", raw)
		}
		args, _, err := launcher.ApplyCouchLaunchProfile(launcher.LaunchArgs{ForcedTag: childEnvValue(child.Env, "COUCH_THREAD_TAG")}, raw)
		if err != nil {
			return err
		}
		if args.Orientation != nil {
			t.Fatal("plain fresh invented orientation")
		}
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: childEnvValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: args.ForcedTag})
		if err != nil {
			return err
		}
		path := paths.AgentReady(args.Agent)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		ready, err := readiness.Encode(readiness.ReadyRecord{Tag: args.ForcedTag, Agent: args.Agent, Session: "fresh-session", Nonce: nonce, PID: os.Getpid()})
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte(ready), 0600)
	}
	env.Couch.FreshRegistration = reader.Registered
	ctx, cancel := context.WithTimeout(context.Background(), 3e9)
	defer cancel()
	result, err := env.Couch.StartFreshSlot(ctx, local.slot.WorktreeRoot, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if accepted, err := reader.Registered(ctx, result.Record.Thread, "claude", "stale-attempt"); err != nil || accepted {
		t.Fatalf("stale nonce: %v %v", accepted, err)
	}
}
