package couchcore

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// A record from before the field binds once through the ledger, and the
// resume's own start claim stores the binding, so the next resume reads it
// from the record (pair#695 P2).
func TestColdResumeStoresTheBindingOnTheRecord(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	if parked.Transcript != nil {
		t.Fatalf("fixture already has a binding: %+v", parked.Transcript)
	}
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Artifacts.SetPaneSidecar(address, "claude")
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}

	if _, _, err := env.Couch.Resume(address); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	after, err := env.Couch.Threads.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	got := after.Transcript
	if got == nil || got.NativeID != "native-root-1" || got.Agent != "claude" || got.Source != threadrecord.TranscriptSourceLegacyLedger {
		t.Fatalf("stored binding = %+v", got)
	}
}
