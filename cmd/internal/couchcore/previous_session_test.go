package couchcore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xianxu/pair/cmd/internal/orientation"
)

func TestPreviousSessionSurvivesSuccessfulSwitch(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t, "/repo")
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Couch.SwitchContext = switchContextFunc(func(context.Context, ThreadRecord) (orientation.OrientationContext, error) {
		return orientation.OrientationContext{Tag: string(source.Address.Tag), WorkingPath: source.WorkingPath,
			SourceAgent: "claude", SourceSession: "source-native", NativeTranscripts: []string{"/fixture/source.jsonl"}}, nil
	})
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	args := []string{}
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address,
		Agent: "codex", Argv: args, AcceptedFingerprint: prepared.Fingerprint}); err != nil {
		t.Fatal(err)
	}
	after, err := env.Couch.Threads.GetThread(source.Address)
	if err != nil {
		t.Fatal(err)
	}
	if after.VerifiedPark != nil {
		t.Fatal("fixture did not complete target registration")
	}
	// Check the persisted shape independently of a new Go field: successful
	// registration must not erase the evidence the incoming agent will need.
	raw, err := json.Marshal(toPersistedThreadRecord(after))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Previous *struct {
			Agent         string   `json:"agent"`
			NativeSession string   `json:"native_session"`
			Transcripts   []string `json:"transcripts"`
		} `json:"previous_session"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Previous == nil || saved.Previous.Agent != "claude" || saved.Previous.NativeSession != "source-native" ||
		len(saved.Previous.Transcripts) != 1 || saved.Previous.Transcripts[0] != "/fixture/source.jsonl" {
		t.Fatalf("outgoing evidence lost after registration: %s", raw)
	}
}
