package couchcmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestSwitchRestartAcceptanceSurvivesRemoteHubGuard(t *testing.T) {
	r := newSwitchAdmissionObservationRig(t)
	response := switchCall(t, r.service, r.request(t, "claude"))
	if response.Code != "accepted" || r.queue.count() != 1 {
		t.Fatalf("admission: %+v", response)
	}
	call, err := r.queue.job(0).prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wrapped, ok := call.TypedPayload.(hubMutationPayload)
	if !ok {
		t.Fatal("hub guard missing")
	}
	acceptance, ok := wrapped.original.(*couchcore.AgentSwitchAcceptance)
	if !ok || acceptance == nil {
		t.Fatal("queued remote request lost accepted restart anchor")
	}
	unwrapped, err := unwrapHubMutation(context.Background(), call.TypedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if unwrapped != acceptance {
		t.Fatal("hub unwrap lost owner-local acceptance")
	}
	raw, err := json.Marshal(acceptance)
	if err != nil || string(raw) != "{}" {
		t.Fatalf("anchor escaped wire transport: %s %v", raw, err)
	}
}
