package couchcore

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const LaunchFailureOutputLimit = 16 << 10
const launchFailureErrorLimit = 4 << 10
const launchFailureEncodedLimit = 1 << 20
const launchFailureHistoryLimit = 64

// LifecycleFailure is evidence for an exact attempt, independent of thread lifecycle.
// Nil exit observations mean unobserved, never success. Output preserves raw bytes.
type LifecycleFailure struct {
	Unknown          map[string]json.RawMessage `json:"-"`
	addressUnknown   map[string]json.RawMessage
	Target           string                `json:"target,omitempty"`
	Slot             string                `json:"slot,omitempty"`
	Phase            string                `json:"phase,omitempty"`
	Outcome          string                `json:"outcome,omitempty"`
	Holders          []ProcessIdentityLine `json:"holders,omitempty"`
	HoldersTruncated int                   `json:"holders_truncated,omitempty"`
	LaunchNonce      string                `json:"launch_nonce,omitempty"`
	ParkNonce        string                `json:"park_nonce,omitempty"`
	Address          ThreadAddress         `json:"address"`
	Nonce            string                `json:"nonce"`
	StartingPath     string                `json:"starting_path,omitempty"`
	Agent            string                `json:"agent,omitempty"`
	Operation        string                `json:"operation,omitempty"`
	Stage            string                `json:"stage,omitempty"`
	FailedAt         time.Time             `json:"failed_at"`
	Output           []byte                `json:"output,omitempty"`
	Stream           string                `json:"stream,omitempty"`
	HelperStarted    bool                  `json:"helper_started"`
	BeforeExit       *int                  `json:"before_exit,omitempty"`
	AfterExit        *int                  `json:"after_exit,omitempty"`
	LaunchError      string                `json:"launch_error,omitempty"`
	FinalError       string                `json:"final_error,omitempty"`
	CleanupError     string                `json:"cleanup_error,omitempty"`
}

type LaunchFailure = LifecycleFailure
type LifecycleFailureHistory []LifecycleFailure
type LaunchFailureHistory = LifecycleFailureHistory

func cloneLaunchFailure(f LaunchFailure) LaunchFailure {
	f.Output = append([]byte(nil), f.Output...)
	f.Unknown = strictjson.CloneRaw(f.Unknown)
	f.addressUnknown = strictjson.CloneRaw(f.addressUnknown)
	f.Holders = cloneFailureHolders(f.Holders)
	if f.BeforeExit != nil {
		v := *f.BeforeExit
		f.BeforeExit = &v
	}
	if f.AfterExit != nil {
		v := *f.AfterExit
		f.AfterExit = &v
	}
	return f
}
func boundLaunchFailureError(s string) string {
	s = strings.ToValidUTF8(s, "�")
	if len(s) <= launchFailureErrorLimit {
		return s
	}
	const marker = "[truncated] "
	tail := s[len(s)-(launchFailureErrorLimit-len(marker)):]
	for !utf8.ValidString(tail) {
		tail = tail[1:]
	}
	return marker + tail
}
func normalizeLaunchFailure(f LaunchFailure) (LaunchFailure, error) {
	f = cloneLaunchFailure(f)
	if f.Outcome == "" {
		f.Outcome = "failed"
	}
	if f.HoldersTruncated < 0 {
		return f, fmt.Errorf("invalid holder truncation count")
	}
	identities := []string{f.Nonce}
	if f.Address != (ThreadAddress{}) {
		identities = append(identities, f.Address.RepoScope, string(f.Address.Tag))
	} else if !filepath.IsAbs(f.StartingPath) && f.Target == "" {
		return f, fmt.Errorf("failure requires an exact address, absolute starting path, or target")
	}
	for _, v := range identities {
		if v == "" || len(v) > 1024 || !utf8.ValidString(v) {
			return f, fmt.Errorf("invalid launch failure identity")
		}
	}
	if len(f.Target) > 1024 || !utf8.ValidString(f.Target) {
		return f, fmt.Errorf("invalid failure target")
	}
	if f.Phase != "" && !failureEnum(f.Phase, "observe", "preserve", "stop", "sweep", "prove", "bind", "launch", "commit") {
		return f, fmt.Errorf("invalid failure phase")
	}
	if f.Outcome != "" && !failureEnum(f.Outcome, "failed", "unproven", "refused") {
		return f, fmt.Errorf("invalid failure outcome")
	}
	if len(f.Holders) > 32 {
		f.HoldersTruncated += len(f.Holders) - 32
		f.Holders = f.Holders[len(f.Holders)-32:]
	}
	for i := range f.Holders {
		if f.Holders[i].Process.PID <= 0 || f.Holders[i].Process.Identity == "" || len(f.Holders[i].Process.Identity) > 1024 || !utf8.ValidString(f.Holders[i].Process.Identity) {
			return f, fmt.Errorf("invalid failure holder identity")
		}
		f.Holders[i].Kind = safeFailureProcessKind(f.Holders[i].Kind)
	}
	if f.FailedAt.IsZero() {
		return f, fmt.Errorf("missing launch failure time")
	}
	for _, v := range []string{f.StartingPath, f.Agent, f.Operation, f.Stage, f.Stream, f.Slot, f.LaunchNonce, f.ParkNonce} {
		if len(v) > 4096 || !utf8.ValidString(v) {
			return f, fmt.Errorf("invalid launch failure metadata")
		}
	}
	if len(f.Output) > LaunchFailureOutputLimit {
		f.Output = f.Output[len(f.Output)-LaunchFailureOutputLimit:]
	}
	f.LaunchError = boundLaunchFailureError(f.LaunchError)
	f.FinalError = boundLaunchFailureError(f.FinalError)
	f.CleanupError = boundLaunchFailureError(f.CleanupError)
	return cloneLaunchFailure(f), nil
}

// Upsert enriches only the matching attempt; first failure time and pre-cleanup
// exit observation are immutable. Newest-first retention also applies to late updates.
func (h LaunchFailureHistory) Upsert(f LaunchFailure) (LaunchFailureHistory, error) {
	suppliedOutcome := f.Outcome != ""
	f, err := normalizeLaunchFailure(f)
	if err != nil {
		return nil, err
	}
	out := make(LaunchFailureHistory, 0, len(h)+1)
	found := false
	for _, old := range h {
		old = cloneLaunchFailure(old)
		if old.Address == f.Address && old.Nonce == f.Nonce && lifecycleFailureTargetKey(failureTarget(old)) == lifecycleFailureTargetKey(failureTarget(f)) {
			old.Unknown = strictjson.MergeRaw(old.Unknown, f.Unknown)
			old.addressUnknown = strictjson.MergeRaw(old.addressUnknown, f.addressUnknown)
			if f.Phase != "" {
				old.Phase = f.Phase
			}
			if suppliedOutcome {
				old.Outcome = f.Outcome
			}
			if f.Holders != nil {
				old.Holders = mergeFailureHolders(old.Holders, f.Holders)
				old.HoldersTruncated = f.HoldersTruncated
			}
			if f.LaunchNonce != "" {
				old.LaunchNonce = f.LaunchNonce
			}
			if f.ParkNonce != "" {
				old.ParkNonce = f.ParkNonce
			}
			found = true
			if f.Output != nil {
				old.Output = f.Output
			}
			if old.Stream == "" {
				old.Stream = f.Stream
			}
			if f.Operation != "" {
				old.Operation = f.Operation
			}
			if f.AfterExit != nil {
				old.AfterExit = f.AfterExit
			}
			if old.LaunchError == "" {
				old.LaunchError = f.LaunchError
			}
			if f.FinalError != "" {
				old.FinalError = f.FinalError
			}
			if f.CleanupError != "" {
				old.CleanupError = f.CleanupError
			}
		}
		out = append(out, old)
	}
	if !found {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if recency := compareLifecycleFailureRecency(a, b); recency != 0 {
			return recency > 0
		}
		if a.Address.RepoScope != b.Address.RepoScope {
			return a.Address.RepoScope < b.Address.RepoScope
		}
		if a.Address.Tag != b.Address.Tag {
			return a.Address.Tag < b.Address.Tag
		}
		return a.Nonce < b.Nonce
	})
	if len(out) > launchFailureHistoryLimit {
		out = out[:launchFailureHistoryLimit]
	}
	return fitLifecycleFailureHistory(out, launchFailureEncodedLimit, encodeLifecycleFailureRow)
}
func encodeLaunchFailureHistory(h LaunchFailureHistory) ([]byte, error) {
	return encodeLifecycleFailureHistory(h)
}
