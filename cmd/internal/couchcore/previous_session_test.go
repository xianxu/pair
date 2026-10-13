package couchcore

import (
	"bytes"
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
		t.Fatal("tty-only switch resolved native context")
		return orientation.OrientationContext{}, nil
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
	if saved.Previous == nil || saved.Previous.Agent != "claude" || saved.Previous.NativeSession != "" || len(saved.Previous.Transcripts) != 0 {
		t.Fatalf("outgoing evidence lost after registration: %s", raw)
	}
}

func TestPreviousSessionEvidenceContainsOnlyTTYCoordinates(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	previous := previousSessionEvidence(source, source)
	if previous == nil || previous.Binding != nil || previous.NativeSession != "" || len(previous.Transcripts) != 0 || len(previous.Unavailable) != 0 {
		t.Fatalf("context retained: %+v", previous)
	}
	raw, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"binding", "native_session", "transcripts", "unavailable"} {
		if bytes.Contains(raw, []byte(`"`+key+`"`)) {
			t.Fatalf("persisted context key %s: %s", key, raw)
		}
	}
	if previous.Capture != nil {
		t.Fatal("fixture unexpectedly includes capture")
	}
}
