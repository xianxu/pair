package vmcmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLaunchClaimPreservesUnknownAndRefusesMissingAuthority(t *testing.T) {
	raw := []byte(`{"version":1,"lane":"train","vm":"pair-conformance-run-train","nonce":"` + strings.Repeat("a", 32) + `","stage":"launching","pid":0,"start_identity":"","future": { "x" : 2 }}`)
	c, err := DecodeLaunchClaim(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeLaunchClaim(c)
	if err != nil || !bytes.Contains(encoded, []byte(`{ "x" : 2 }`)) {
		t.Fatalf("lost additive bytes: %s %v", encoded, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for key := range claimFields {
		copyFields := make(map[string]json.RawMessage, len(fields))
		for name, value := range fields {
			copyFields[name] = value
		}
		delete(copyFields, key)
		incomplete, err := json.Marshal(copyFields)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeLaunchClaim(incomplete); err == nil {
			t.Fatalf("missing %q accepted", key)
		}
	}
}

func TestRecoverLaunchClaimRequiresPositiveDeathAndStoppedVM(t *testing.T) {
	c := LaunchClaim{Version: 1, Lane: TrainLane, VM: ManagedVMName(TrainLane), Nonce: strings.Repeat("a", 32), Stage: ClaimLaunched, PID: 12, StartIdentity: "darwin:123.456"}
	for _, proof := range []RecoveryEvidence{
		{PID: 12, Identity: c.StartIdentity, Process: ProcessUnknown, VM: VMStopped},
		{PID: 12, Identity: c.StartIdentity, Process: ProcessAlive, VM: VMStopped},
		{PID: 13, Identity: c.StartIdentity, Process: ProcessGone, VM: VMStopped},
		{PID: 12, Identity: "darwin:other", Process: ProcessGone, VM: VMStopped},
		{PID: 12, Identity: c.StartIdentity, Process: ProcessGone, VM: VMUnknown},
		{PID: 12, Identity: c.StartIdentity, Process: ProcessGone, VM: VMRunning},
	} {
		if err := RecoverLaunchClaim(c, proof); err == nil {
			t.Fatalf("uncertain/foreign proof admitted: %+v", proof)
		}
	}
	proof := RecoveryEvidence{PID: c.PID, Identity: c.StartIdentity, Process: ProcessGone, VM: VMStopped}
	if err := RecoverLaunchClaim(c, proof); err != nil {
		t.Fatal(err)
	}
	c.Stage, c.PID, c.StartIdentity = ClaimLaunching, 0, ""
	proof.PID, proof.Identity = 0, ""
	if err := RecoverLaunchClaim(c, proof); err == nil {
		t.Fatal("stopped VM sample cleared pre-publication startup uncertainty")
	}
}

func FuzzLaunchClaim(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"version":1,"version":1}`))
	f.Add([]byte(`{"version":1,"lane":"adhoc","vm":"pair-conformance-run-adhoc","nonce":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","stage":"launching","pid":0,"start_identity":""}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		c, err := DecodeLaunchClaim(raw)
		if err != nil {
			return
		}
		out, err := EncodeLaunchClaim(c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeLaunchClaim(out); err != nil {
			t.Fatal(err)
		}
	})
}
