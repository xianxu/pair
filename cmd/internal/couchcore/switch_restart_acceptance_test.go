package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestSwitchRestartPreviewPinsNativeAnchorAndRechecksAcceptedIntent(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{}})
			env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, "native-A")
			argv := []string{"--model", "chosen"}
			prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, agent, &argv)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.restart == nil || prepared.restart.ExpectedNativeID != "native-A" || prepared.restart.SourceFingerprint != prepared.Fingerprint || prepared.restart.SourceRevision != source.Revision || !reflect.DeepEqual(prepared.restart.Profile, prepared.Profile) {
				t.Fatalf("preview lost accepted anchor: %+v", prepared.restart)
			}
			accepted := prepared.AcceptAgentSwitch(false)
			if accepted == nil || accepted.restart == nil {
				t.Fatal("acceptance lost restart intent")
			}
			env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingUnbound, "")
			checked, err := env.Couch.prepareAgentSwitchAccepted(context.Background(), source.Address, agent, &argv, false, accepted.restart)
			if err != nil || checked.Fingerprint != prepared.Fingerprint || checked.restart.ExpectedNativeID != "native-A" {
				fresh, freshErr := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, agent, &argv)
				t.Fatalf("accepted A disappearing invalidated acceptance: %+v %v; before state=%s/%s, after state=%s/%s freshErr=%v", checked.restart, err, prepared.state, prepared.sourceReason, fresh.state, fresh.sourceReason, freshErr)
			}
			if checked.state != ThreadUnusable || checked.sourceReason != ReasonBindingLost || checked.SourceEvidence().state != ThreadUnusable || checked.restart.SourceState != prepared.state {
				t.Fatal("anchor reconstruction replaced current admission facts")
			}
			env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, "native-B")
			if _, err := env.Couch.prepareAgentSwitchAccepted(context.Background(), source.Address, agent, &argv, false, accepted.restart); !errors.Is(err, ErrSwitchResolutionChanged) {
				t.Fatalf("replacement accepted: %v", err)
			}
		})
	}
}

func TestSwitchRestartPreviewPositiveAbsenceDiffersFromDifferentAgent(t *testing.T) {
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingUnbound, "")
	argv := []string{}
	same, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "claude", &argv)
	if err != nil || same.restart == nil || same.restart.ExpectedNativeID != "" {
		t.Fatalf("positive unbound preview: %+v %v", same.restart, err)
	}
	other, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &argv)
	if err != nil || other.restart != nil {
		t.Fatalf("different-agent switch acquired native authority: %+v %v", other.restart, err)
	}
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingEstablished, "new-native")
	if _, err := env.Couch.prepareAgentSwitchAccepted(context.Background(), source.Address, "claude", &argv, false, same.restart); !errors.Is(err, ErrSwitchResolutionChanged) {
		t.Fatalf("initial absence replaced: %v", err)
	}
}

func TestSwitchRestartPreviewRefusesUnknownAndAcceptedAnchorMutation(t *testing.T) {
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	argv := []string{}
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingProvisional, "")
	if _, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "claude", &argv); err == nil {
		t.Fatal("unknown native identity admitted")
	}
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingEstablished, "native-A")
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "claude", &argv)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"address", "revision", "profile", "nil-profile", "fingerprint", "native", "source-state", "source-reason"} {
		t.Run(fault, func(t *testing.T) {
			accepted := p.AcceptAgentSwitch(false)
			if accepted == nil || accepted.restart == nil {
				t.Fatal("acceptance lost restart")
			}
			switch fault {
			case "address":
				accepted.restart.Address.Tag = "other"
			case "revision":
				accepted.restart.SourceRevision++
			case "profile":
				accepted.restart.Profile.Argv = []string{"--model", "other"}
			case "nil-profile":
				accepted.restart.Profile.Argv = nil
			case "fingerprint":
				accepted.restart.SourceFingerprint = "other"
			case "native":
				accepted.restart.ExpectedNativeID = "native-B"
			case "source-state":
				accepted.restart.SourceState = ThreadLive
			case "source-reason":
				accepted.restart.SourceReason = ReasonBindingLost
			}
			if _, err := env.Couch.prepareAgentSwitchAccepted(context.Background(), source.Address, "claude", &argv, false, accepted.restart); !errors.Is(err, ErrSwitchResolutionChanged) {
				t.Fatalf("mutated %s anchor accepted: %v", fault, err)
			}
		})
	}
}

func TestSwitchAcceptanceClonesAndNeverSerializesAuthority(t *testing.T) {
	p := PreparedAgentSwitch{restart: &EditedRestartIntent{ExpectedNativeID: "native-A", Profile: LaunchProfile{Agent: "claude", Argv: []string{"--model", "accepted"}}}, Force: &AgentSwitchForceOffer{}}
	accepted := p.AcceptAgentSwitch(true)
	if accepted == nil || accepted.restart == nil || accepted.force == nil || accepted.force.offer != p.Force {
		t.Fatal("accepted local authority missing")
	}
	accepted.restart.Profile.Argv[1] = "changed"
	if p.restart.Profile.Argv[1] != "accepted" {
		t.Fatal("accepted argv aliases preview")
	}
	raw, err := json.Marshal(accepted)
	if err != nil || string(raw) != "{}" {
		t.Fatalf("serialized authority: %s %v", raw, err)
	}
	var decoded AgentSwitchAcceptance
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.restart != nil || decoded.force != nil {
		t.Fatal("serialized acceptance recreated permission")
	}
	raw, err = json.Marshal(SwitchAgentRequest{Restart: accepted.restart, Force: accepted.force})
	if err != nil {
		t.Fatal(err)
	}
	var request SwitchAgentRequest
	if err := json.Unmarshal(raw, &request); err != nil || request.Restart != nil || request.Force != nil {
		t.Fatal("serialized request recreated permission")
	}
}

func TestSwitchAcceptanceDispatchExtractsOnlyLocalAuthority(t *testing.T) {
	p := PreparedAgentSwitch{restart: &EditedRestartIntent{ExpectedNativeID: "native-A", Profile: LaunchProfile{Agent: "claude", Argv: []string{"--model", "accepted"}}}, Force: &AgentSwitchForceOffer{}}
	accepted := p.AcceptAgentSwitch(true)
	restart, force := acceptedAgentSwitchPayload(accepted)
	if restart == nil || restart.ExpectedNativeID != "native-A" || force != accepted.force {
		t.Fatal("dispatcher lost local acceptance")
	}
	restart.Profile.Argv[1] = "other"
	if accepted.restart.Profile.Argv[1] != "accepted" {
		t.Fatal("dispatcher request aliases accepted anchor")
	}
	legacy := p.Force.Confirm()
	if restart, force = acceptedAgentSwitchPayload(legacy); restart != nil || force != legacy {
		t.Fatal("legacy local consent lost")
	}
	for _, payload := range []any{nil, (*AgentSwitchAcceptance)(nil), &AgentSwitchAcceptance{}, map[string]any{"restart": accepted.restart, "force": accepted.force}} {
		restart, force = acceptedAgentSwitchPayload(payload)
		if restart != nil || force != nil {
			t.Fatal("serialized/untyped acceptance recreated authority")
		}
	}
}

func TestSwitchRestartAcceptedAnchorRefusesFreshSourceOrNativeUncertainty(t *testing.T) {
	for _, fault := range []string{"source-revision", "unknown-native"} {
		t.Run(fault, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingEstablished, "native-A")
			argv := []string{}
			prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "claude", &argv)
			if err != nil {
				t.Fatal(err)
			}
			accepted := prepared.AcceptAgentSwitch(false)
			if fault == "source-revision" {
				name := "changed since preview"
				if _, err := env.Couch.Threads.ApplyThreadMetadata(source.Address, source.Revision, ThreadMetadataPatch{Name: &name}); err != nil {
					t.Fatal(err)
				}
			} else {
				env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingProvisional, "")
			}
			if _, err := env.Couch.prepareAgentSwitchAccepted(context.Background(), source.Address, "claude", &argv, false, accepted.restart); err == nil {
				t.Fatal("changed or uncertain source accepted")
			}
		})
	}
}
