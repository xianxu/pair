package threadrecord

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

func validSwitchRecovery() SwitchRecovery {
	now := time.Unix(1000, 0).UTC()
	return SwitchRecovery{RequestID: "request", Owner: Address{RepoScope: "repo", Tag: "owner"}, SourceToken: "authenticated-source", Source: SwitchSourceSnapshot{Process: SwitchProcess{PID: 42, Identity: "source-process"}, Binding: &couchidentity.SessionBinding{Name: "pair-source", ScopeKey: "repo", Tag: "2", StartNonce: "source-nonce", Legacy: true}, NativeSession: "native-source", Profile: LaunchProfile{Agent: "codex", Argv: []string{"--source"}}}, TargetProfile: LaunchProfile{Agent: "claude", Argv: []string{}}, Content: SwitchContent{Kind: SwitchContentAuthored, Body: " exact brief\n"}, Phase: SwitchWaitingSettled, AdmittedAt: now, BriefDeadline: now.Add(time.Minute), OverallDeadline: now.Add(2 * time.Minute), UpdatedAt: now}
}
func recoveryWitness(profile LaunchProfile, nonce string) *SwitchLaunchWitness {
	return &SwitchLaunchWitness{Claim: StartClaim{Nonce: nonce, OwnerPID: 100, OwnerIdentity: "supervisor", LaunchProfile: &profile, SessionBinding: &couchidentity.SessionBinding{Name: "pair-" + nonce, ScopeKey: "repo", Tag: "2", StartNonce: nonce, Legacy: true}}, Helper: &Incarnation{PID: 52, Identity: nonce + "-helper", State: "live", LaunchProfile: &profile}}
}
func TestSwitchRecoveryValidation(t *testing.T) {
	valid := validSwitchRecovery()
	address := Address{RepoScope: "repo", Tag: "2"}
	if err := valid.Validate(address); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*SwitchRecovery)
	}{
		{"missing request", func(r *SwitchRecovery) { r.RequestID = "" }},
		{"invalid owner", func(r *SwitchRecovery) { r.Owner.Tag = "../escape" }},
		{"half process", func(r *SwitchRecovery) { r.Source.Process.PID = 0 }},
		{"foreign binding", func(r *SwitchRecovery) { r.Source.Binding.Tag = "other" }},
		{"invalid native", func(r *SwitchRecovery) { r.Source.NativeSession = "\xff" }},
		{"missing profile", func(r *SwitchRecovery) { r.Source.Profile.Argv = nil }},
		{"unknown phase", func(r *SwitchRecovery) { r.Phase = "invented" }},
		{"hidden body", func(r *SwitchRecovery) { r.Content = SwitchContent{Body: "hidden"} }},
		{"oversized body", func(r *SwitchRecovery) { r.Content.Body = strings.Repeat("x", orientation.MaxBodyBytes+1) }},
		{"bad deadlines", func(r *SwitchRecovery) { r.OverallDeadline = r.BriefDeadline }},
		{"time reversal", func(r *SwitchRecovery) { r.UpdatedAt = r.AdmittedAt.Add(-time.Second) }},
		{"missing selected content", func(r *SwitchRecovery) { r.Content = SwitchContent{} }},
		{"fake success", func(r *SwitchRecovery) { r.Phase = SwitchSucceeded }},
		{"unknown survivor", func(r *SwitchRecovery) { r.Survivor = "invented" }},
		{"missing target survivor", func(r *SwitchRecovery) { r.Survivor = SwitchSurvivorTarget }},
		{"target wrong profile", func(r *SwitchRecovery) { r.Target = recoveryWitness(r.Source.Profile, "target") }},
		{"binding nonce mismatch", func(r *SwitchRecovery) {
			r.Target = recoveryWitness(r.TargetProfile, "target")
			r.Target.Claim.SessionBinding.StartNonce = "other"
		}},
		{"duplicate attempt", func(r *SwitchRecovery) {
			r.Target = recoveryWitness(r.TargetProfile, "attempt")
			r.Restoration = recoveryWitness(r.Source.Profile, "attempt")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := CloneSwitchRecovery(&valid)
			tc.mutate(r)
			if r.Validate(address) == nil {
				t.Fatal("invalid recovery accepted")
			}
		})
	}
	valid.Phase = SwitchWaitingBrief
	valid.Content = SwitchContent{}
	if err := valid.Validate(address); err != nil {
		t.Fatal(err)
	}
	valid.Phase = SwitchRefused
	if err := valid.Validate(address); err != nil {
		t.Fatal(err)
	}
}
func TestSwitchRecoveryCloneAndRoundTrip(t *testing.T) {
	original := validSwitchRecovery()
	original.Target = recoveryWitness(original.TargetProfile, "target")
	original.Restoration = recoveryWitness(original.Source.Profile, "restore")
	before, _ := json.Marshal(original)
	copy := CloneSwitchRecovery(&original)
	copy.Source.Binding.Tag = "changed"
	copy.Source.Profile.Argv[0] = "changed"
	copy.TargetProfile.Argv = append(copy.TargetProfile.Argv, "changed")
	copy.Target.Claim.SessionBinding.Tag = "changed"
	copy.Target.Claim.LaunchProfile.Agent = "changed"
	copy.Target.Helper.Identity = "changed"
	copy.Target.Helper.LaunchProfile.Agent = "changed"
	copy.Restoration.Claim.LaunchProfile.Argv[0] = "changed"
	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatal("clone aliased original")
	}
	var decoded SwitchRecovery
	if err := json.Unmarshal(before, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Fatal("roundtrip changed record")
	}
	if err := decoded.Validate(Address{RepoScope: "repo", Tag: "2"}); err != nil {
		t.Fatal(err)
	}
	if CloneSwitchRecovery(nil) != nil {
		t.Fatal("nil clone")
	}
}

func TestSwitchRecoveryClonePreservesInvalidNilArguments(t *testing.T) {
	r := validSwitchRecovery()
	r.Source.Profile.Argv = nil
	if CloneSwitchRecovery(&r).Source.Profile.Argv != nil {
		t.Fatal("clone repaired malformed evidence")
	}
}
