package couchcore

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"path/filepath"
	"strings"
	"time"
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
}

// LifecycleFailureSlotKey derives identity from a previously verified physical path.
func LifecycleFailureSlotKey(t LifecycleFailureTarget) string {
	if filepath.IsAbs(t.StartingPath) {
		return "slot:" + t.StartingPath
	}
	return ""
}
func failureTarget(f LifecycleFailure) LifecycleFailureTarget {
	return LifecycleFailureTarget{Address: f.Address, StartingPath: f.StartingPath, Slot: f.Slot, ParkNonce: f.ParkNonce}
}

// compareLifecycleFailureRecency retains time ordering across owners and uses
// admission order to break identical-clock ties within the same owner.
func compareLifecycleFailureRecency(a, b LifecycleFailure) int {
	if recency := a.FailedAt.Compare(b.FailedAt); recency != 0 {
		return recency
	}
	if epoch := strings.Compare(a.AttemptEpoch, b.AttemptEpoch); epoch != 0 {
		return epoch
	}
	if a.AttemptOrder > b.AttemptOrder {
		return 1
	}
	if a.AttemptOrder < b.AttemptOrder {
		return -1
	}
	return 0
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

// Supersede keeps raw evidence while hiding only failures admitted no later than success.
func (h LifecycleFailureHistory) Supersede(t LifecycleFailureTarget, epoch string, order uint64, at time.Time) (LifecycleFailureHistory, bool) {
	out := make(LifecycleFailureHistory, len(h))
	changed := false
	key := LifecycleFailureSlotKey(t)
	for i, f := range h {
		out[i] = cloneLaunchFailure(f)
		if key != "" && LifecycleFailureSlotKey(failureTarget(f)) == key && f.SupersededAt == nil && (f.AttemptEpoch != epoch || f.AttemptOrder <= order) {
			v := at
			out[i].SupersededAt = &v
			changed = true
		}
	}
	return out, changed
}

// LifecycleFailureStorage is the same durable store used by legacy launch capture.
type LifecycleFailureStorage interface {
	LaunchFailureStorage
	Supersede(context.Context, LifecycleFailureTarget, string, uint64, time.Time) error
}
type LifecycleFailureStore = LaunchFailureStore
type FakeLifecycleFailureStore = FakeLaunchFailureStore

func NewLifecycleFailureStore(n CouchNamespace) *LifecycleFailureStore {
	return NewLaunchFailureStore(n)
}
func NewFakeLifecycleFailureStore() *FakeLifecycleFailureStore { return NewFakeLaunchFailureStore() }
