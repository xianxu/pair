package couchcore

import (
	"encoding/json"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type ProcessIdentityLine struct {
	Process        ProcessIdentity            `json:"process"`
	Kind           string                     `json:"kind"`
	Unknown        map[string]json.RawMessage `json:"-"`
	processUnknown map[string]json.RawMessage
}
type LifecycleFailureDetails interface {
	error
	FailurePhase() string
	FailureHolders() []ProcessIdentityLine
	FailureOutcome() string
}
type LifecycleFailureTarget struct {
	Address      ThreadAddress
	StartingPath string
	Slot         string
	ParkNonce    string
	Target       string
}

// LifecycleFailureSlotKey derives identity from a previously verified physical path.
func LifecycleFailureSlotKey(t LifecycleFailureTarget) string {
	address, err := NewSlotAddress(SlotIdentity{}, t.StartingPath)
	if err != nil {
		return ""
	}
	return address.Key()
}

func lifecycleFailureTargetKey(t LifecycleFailureTarget) string {
	if slot := LifecycleFailureSlotKey(t); slot != "" {
		return slot
	}
	if t.Address.RepoScope != "" && t.Address.Tag != "" {
		return "thread:" + t.Address.RepoScope + "/" + string(t.Address.Tag)
	}
	if t.Target != "" {
		return "target:" + t.Target
	}
	return ""
}
func failureTarget(f LifecycleFailure) LifecycleFailureTarget {
	return LifecycleFailureTarget{Address: f.Address, StartingPath: f.StartingPath, Slot: f.Slot, ParkNonce: f.ParkNonce, Target: f.Target}
}

// compareLifecycleFailureRecency uses completion time and a stable nonce tie.
func compareLifecycleFailureRecency(a, b LifecycleFailure) int {
	if recency := a.FailedAt.Compare(b.FailedAt); recency != 0 {
		return recency
	}
	return strings.Compare(a.Nonce, b.Nonce)
}
func failureEnum(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
func safeFailureProcessKind(v string) string {
	if failureEnum(v, "pair", "pair-wrap", "pair-launch-helper", "zellij", "nvim") {
		return v
	}
	for _, agent := range launcher.AgentInventory() {
		if v == agent {
			return v
		}
	}
	return "process"
}
func cloneFailureHolders(h []ProcessIdentityLine) []ProcessIdentityLine {
	if h == nil {
		return nil
	}
	out := append([]ProcessIdentityLine(nil), h...)
	for i := range out {
		out[i].Unknown = strictjson.CloneRaw(out[i].Unknown)
		out[i].processUnknown = strictjson.CloneRaw(out[i].processUnknown)
	}
	return out
}
func mergeFailureHolders(old, next []ProcessIdentityLine) []ProcessIdentityLine {
	out := cloneFailureHolders(next)
	for i := range out {
		for _, o := range old {
			if o.Process == out[i].Process {
				out[i].Unknown = strictjson.MergeRaw(o.Unknown, out[i].Unknown)
				out[i].processUnknown = strictjson.MergeRaw(o.processUnknown, out[i].processUnknown)
			}
		}
	}
	return out
}
