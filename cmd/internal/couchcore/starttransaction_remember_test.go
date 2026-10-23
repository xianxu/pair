package couchcore

import (
	"encoding/json"
	"testing"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestStartClaimRememberConfigurationIsExplicitAndSurvivesPersistence(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		record := admittedStartRecord(t)
		profile := LaunchProfile{Agent: "codex", Argv: []string{}}
		claimed, err := AdvanceStartTransaction(record, StartEvent{Kind: StartClaimed, Nonce: "explicit-intent", Owner: SupervisorOwner{PID: 7, Identity: "owner"}, Profile: &profile, RememberConfiguration: accepted})
		if err != nil {
			t.Fatal(err)
		}
		if claimed.Incarnations[0].Start.RememberConfiguration != accepted {
			t.Fatalf("claim acceptance = %v, want %v", claimed.Incarnations[0].Start.RememberConfiguration, accepted)
		}
		raw, err := encodeThreadRecord(claimed)
		if err != nil {
			t.Fatal(err)
		}
		persisted, err := threadrecord.DecodePersisted(raw, toPersistedThreadAddress(claimed.Address), threadRecordValidators)
		if err != nil {
			t.Fatal(err)
		}
		restored := cloneThreadRecord(fromPersistedThreadRecord(persisted))
		if restored.Incarnations[0].Start.RememberConfiguration != accepted {
			t.Fatalf("persisted acceptance lost: %s", raw)
		}
		var fields map[string]json.RawMessage
		claimRaw, err := json.Marshal(restored.Incarnations[0].Start)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(claimRaw, &fields); err != nil {
			t.Fatal(err)
		}
		if _, exists := fields["remember_configuration"]; exists != accepted {
			t.Fatalf("default intent should remain omitted: %s", claimRaw)
		}
	}
}
