package couchcore

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"path/filepath"
	"strings"
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
	if filepath.IsAbs(t.StartingPath) {
		return "slot:" + t.StartingPath
	}
	return ""
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
func cloneFailureUnknown(m map[string]json.RawMessage) map[string]json.RawMessage {
	if m == nil {
		return nil
	}
	n := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		n[k] = append(json.RawMessage(nil), v...)
	}
	return n
}
func mergeFailureUnknown(a, b map[string]json.RawMessage) map[string]json.RawMessage {
	n := cloneFailureUnknown(a)
	if n == nil && b != nil {
		n = map[string]json.RawMessage{}
	}
	for k, v := range b {
		n[k] = append(json.RawMessage(nil), v...)
	}
	return n
}
func cloneFailureHolders(h []ProcessIdentityLine) []ProcessIdentityLine {
	if h == nil {
		return nil
	}
	out := append([]ProcessIdentityLine(nil), h...)
	for i := range out {
		out[i].Unknown = cloneFailureUnknown(out[i].Unknown)
		out[i].processUnknown = cloneFailureUnknown(out[i].processUnknown)
	}
	return out
}
func mergeFailureHolders(old, next []ProcessIdentityLine) []ProcessIdentityLine {
	out := cloneFailureHolders(next)
	for i := range out {
		for _, o := range old {
			if o.Process == out[i].Process {
				out[i].Unknown = mergeFailureUnknown(o.Unknown, out[i].Unknown)
				out[i].processUnknown = mergeFailureUnknown(o.processUnknown, out[i].processUnknown)
			}
		}
	}
	return out
}
