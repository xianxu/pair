package threadrecord

import (
	"errors"
	"math"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// RecoveryCapture is the original immutable screen, not necessarily the slot's
// current PreviousSession. Producer authority is optional, and never a park proof.
type RecoveryCapture struct {
	Token          string                           `json:"token,omitempty"`
	LastActivity   time.Time                        `json:"last_activity,omitempty"`
	Raw            string                           `json:"raw"`
	Events         string                           `json:"events"`
	RawIdentity    artifactpath.CaptureFileIdentity `json:"raw_identity"`
	EventsIdentity artifactpath.CaptureFileIdentity `json:"events_identity"`
	CapturedAt     time.Time                        `json:"captured_at"`
	LaunchOrdinal  uint64                           `json:"launch_ordinal,omitempty"`
	Nonce          string                           `json:"nonce,omitempty"`
}

// RecoveryCandidate stores bounded evidence, not permission to launch. Native
// artifact roots and relative names must be reauthorized by the runtime adapter.
type RecoveryCandidate struct {
	Key                  string    `json:"key"`
	Agent                string    `json:"agent"`
	NativeID             string    `json:"native_id"`
	ArtifactRoot         string    `json:"artifact_root"`
	ArtifactRelativePath string    `json:"artifact_relative_path"`
	StableFileID         string    `json:"stable_file_id"`
	GenerationToken      string    `json:"generation_token"`
	MutationToken        string    `json:"mutation_token"`
	Size                 int64     `json:"size"`
	ModTime              time.Time `json:"mod_time"`
	Score                float64   `json:"score"`
	Matched              int       `json:"matched"`
	Total                int       `json:"total"`
	LastActivity         time.Time `json:"last_activity"`
	Excerpt              string    `json:"excerpt"`
	Eligible             bool      `json:"eligible"`
}

type RecoveryContext struct {
	Version          int                 `json:"version"`
	Address          Address             `json:"address"`
	Capture          RecoveryCapture     `json:"capture"`
	Generation       string              `json:"generation"`
	Digest           string              `json:"digest"`
	CreatedAt        time.Time           `json:"created_at"`
	ExpiresAt        time.Time           `json:"expires_at"`
	Branch           string              `json:"branch,omitempty"`
	Issue            string              `json:"issue,omitempty"`
	Candidates       []RecoveryCandidate `json:"candidates,omitempty"`
	SelectedKey      string              `json:"selected_key"`
	SelectedNativeID string              `json:"selected_native_id,omitempty"`
}

type RecoveryIntent struct {
	Context     RecoveryContext `json:"context"`
	SelectedKey string          `json:"selected_key"`
	NativeID    string          `json:"native_id,omitempty"`
}

func recoveryText(s string, max int, required bool) bool {
	return (!required || s != "") && len(s) <= max && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func recoveryTime(t time.Time) bool { return !t.IsZero() && t.Year() >= 1 && t.Year() <= 9999 }
func recoveryPath(p string) bool {
	return recoveryText(p, 4096, true) && filepath.IsAbs(p) && filepath.Clean(p) == p && p != "/"
}
func recoveryFileIdentity(i artifactpath.CaptureFileIdentity) bool {
	return i.Size >= 0 && i.Device != 0 && i.Inode != 0 && recoveryTime(i.ModifiedAt)
}

// ValidateRecoveryContext is structural and clock-independent so expired
// evidence remains readable after restart. Admission rechecks expiry and binds
// it to the existing capture retention policy; it also verifies physical files.
func ValidateRecoveryContext(c RecoveryContext, address Address) error {
	if c.Version != 1 || c.Address != address {
		return errors.New("recovery context version or address mismatch")
	}
	if !recoveryText(c.Generation, 1024, true) || len(c.Digest) != 64 || strings.Trim(c.Digest, "0123456789abcdef") != "" {
		return errors.New("invalid recovery generation or digest")
	}
	if !recoveryPath(c.Capture.Raw) || !recoveryPath(c.Capture.Events) || c.Capture.Raw == c.Capture.Events || !recoveryFileIdentity(c.Capture.RawIdentity) || !recoveryFileIdentity(c.Capture.EventsIdentity) {
		return errors.New("invalid recovery capture")
	}
	if !recoveryTime(c.Capture.CapturedAt) || !recoveryTime(c.CreatedAt) || !recoveryTime(c.ExpiresAt) || c.CreatedAt.Before(c.Capture.CapturedAt) || !c.ExpiresAt.After(c.CreatedAt) {
		return errors.New("invalid recovery lifetime")
	}
	if c.Capture.Token != "" && (len(c.Capture.Token) > 128 || !componentPattern.MatchString(c.Capture.Token)) {
		return errors.New("invalid recovery capture token")
	}
	if !c.Capture.LastActivity.IsZero() && (!recoveryTime(c.Capture.LastActivity) || c.Capture.LastActivity.After(c.Capture.CapturedAt)) {
		return errors.New("invalid capture activity cutoff")
	}
	if (c.Capture.LaunchOrdinal == 0) != (c.Capture.Nonce == "") || !recoveryText(c.Capture.Nonce, 1024, false) {
		return errors.New("incomplete recovery producer")
	}
	if !recoveryText(c.Branch, 1024, false) || !recoveryText(c.Issue, 1024, false) || len(c.Candidates) > 5 {
		return errors.New("oversized recovery context")
	}
	keys := map[string]bool{}
	for _, candidate := range c.Candidates {
		p := candidate.ArtifactRelativePath
		if !recoveryText(candidate.Key, 4096, true) || keys[candidate.Key] || !recoveryText(candidate.Agent, 128, true) || !componentPattern.MatchString(candidate.Agent) || !recoveryText(candidate.NativeID, 1024, true) || !recoveryText(candidate.ArtifactRoot, 4096, true) || !recoveryText(p, 4096, true) || filepath.IsAbs(p) || filepath.Clean(p) != p || p == "." || p == ".." || strings.HasPrefix(p, "../") {
			return errors.New("invalid recovery candidate identity")
		}
		keys[candidate.Key] = true
		if !recoveryText(candidate.StableFileID, 1024, true) || !recoveryText(candidate.GenerationToken, 1024, false) || !recoveryText(candidate.MutationToken, 1024, true) || candidate.Size < 0 || !recoveryTime(candidate.ModTime) || !recoveryTime(candidate.LastActivity) {
			return errors.New("invalid recovery candidate artifact")
		}
		if math.IsNaN(candidate.Score) || math.IsInf(candidate.Score, 0) || candidate.Score < 0 || candidate.Score > 100 || candidate.Matched < 0 || candidate.Total < 0 || candidate.Matched > candidate.Total || !recoveryText(candidate.Excerpt, 2048, false) {
			return errors.New("invalid recovery candidate evidence")
		}
	}
	if c.SelectedKey == "" && c.SelectedNativeID == "" {
		return nil
	}
	return validateRecoverySelection(c, c.SelectedKey, c.SelectedNativeID)
}
func validateRecoverySelection(c RecoveryContext, key, native string) error {
	if !recoveryText(key, 4096, true) || !recoveryText(native, 1024, false) {
		return errors.New("invalid recovery selection")
	}
	for _, candidate := range c.Candidates {
		if candidate.Key == key {
			if native == "" || native != candidate.NativeID || !candidate.Eligible {
				return errors.New("recovery selection is ineligible or mismatched")
			}
			return nil
		}
	}
	if native != "" {
		return errors.New("recovery selection is absent from snapshot")
	}
	// Fresh keys are source-bound by the caller's codec, not inferred here.
	return nil
}
func ValidateRecoveryIntent(i RecoveryIntent, address Address) error {
	if err := ValidateRecoveryContext(i.Context, address); err != nil {
		return err
	}
	return validateRecoverySelection(i.Context, i.SelectedKey, i.NativeID)
}
func CloneRecoveryContext(c *RecoveryContext) *RecoveryContext {
	if c == nil {
		return nil
	}
	out := *c
	if c.Candidates != nil {
		out.Candidates = append([]RecoveryCandidate{}, c.Candidates...)
	}
	return &out
}
func CloneRecoveryIntent(i *RecoveryIntent) *RecoveryIntent {
	if i == nil {
		return nil
	}
	out := *i
	out.Context = *CloneRecoveryContext(&i.Context)
	return &out
}
