package couchcore

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const LaunchFailureOutputLimit = 16 << 10
const launchFailureErrorLimit = 4 << 10
const launchFailureEncodedLimit = 1 << 20
const launchFailureHistoryLimit = 16

// LaunchFailure is evidence for an exact attempt, independent of thread lifecycle.
// Nil exit observations mean unobserved, never success. Output preserves raw bytes.
type LaunchFailure struct {
	Address       ThreadAddress `json:"address"`
	Nonce         string        `json:"nonce"`
	StartingPath  string        `json:"starting_path,omitempty"`
	Agent         string        `json:"agent,omitempty"`
	Operation     string        `json:"operation,omitempty"`
	Stage         string        `json:"stage,omitempty"`
	FailedAt      time.Time     `json:"failed_at"`
	Output        []byte        `json:"output,omitempty"`
	Stream        string        `json:"stream,omitempty"`
	HelperStarted bool          `json:"helper_started"`
	BeforeExit    *int          `json:"before_exit,omitempty"`
	AfterExit     *int          `json:"after_exit,omitempty"`
	LaunchError   string        `json:"launch_error,omitempty"`
	FinalError    string        `json:"final_error,omitempty"`
	CleanupError  string        `json:"cleanup_error,omitempty"`
}

type LaunchFailureHistory []LaunchFailure

func cloneLaunchFailure(f LaunchFailure) LaunchFailure {
	f.Output = append([]byte(nil), f.Output...)
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
	for _, v := range []string{f.Address.RepoScope, string(f.Address.Tag), f.Nonce} {
		if v == "" || len(v) > 1024 || !utf8.ValidString(v) {
			return f, fmt.Errorf("invalid launch failure identity")
		}
	}
	if f.FailedAt.IsZero() {
		return f, fmt.Errorf("missing launch failure time")
	}
	for _, v := range []string{f.StartingPath, f.Agent, f.Operation, f.Stage, f.Stream} {
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
	f, err := normalizeLaunchFailure(f)
	if err != nil {
		return nil, err
	}
	out := make(LaunchFailureHistory, 0, len(h)+1)
	found := false
	for _, old := range h {
		old = cloneLaunchFailure(old)
		if old.Address == f.Address && old.Nonce == f.Nonce {
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
		if !a.FailedAt.Equal(b.FailedAt) {
			return a.FailedAt.After(b.FailedAt)
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
	if _, err := encodeLaunchFailureHistory(out); err != nil {
		return nil, err
	}
	return out, nil
}
func encodeLaunchFailureHistory(h LaunchFailureHistory) ([]byte, error) {
	raw, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}
	if len(raw) > launchFailureEncodedLimit {
		return nil, fmt.Errorf("launch failure history exceeds %d bytes", launchFailureEncodedLimit)
	}
	return raw, nil
}
