package threadrecord

import (
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const NativeRecoverySelectionLifetime = 5 * time.Minute

// NativeRecoverySource pins the native inventory observation window and record.
type NativeRecoverySource struct {
	Kind       sessioninventory.RecoverySourceKind `json:"kind"`
	Cutoff     time.Time                           `json:"cutoff"`
	Revision   uint64                              `json:"revision"`
	LaunchHint *time.Time                          `json:"launch_hint,omitempty"`
	Unknown    map[string]json.RawMessage          `json:"-"`
}

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
	CWD                  string                     `json:"cwd,omitempty"`
	Unknown              map[string]json.RawMessage `json:"-"`
	Key                  string                     `json:"key"`
	Agent                string                     `json:"agent"`
	NativeID             string                     `json:"native_id"`
	ArtifactRoot         string                     `json:"artifact_root"`
	ArtifactRelativePath string                     `json:"artifact_relative_path"`
	StableFileID         string                     `json:"stable_file_id"`
	GenerationToken      string                     `json:"generation_token"`
	MutationToken        string                     `json:"mutation_token"`
	Size                 int64                      `json:"size"`
	ModTime              time.Time                  `json:"mod_time"`
	Score                float64                    `json:"score"`
	Matched              int                        `json:"matched"`
	Total                int                        `json:"total"`
	LastActivity         time.Time                  `json:"last_activity"`
	Excerpt              string                     `json:"excerpt"`
	Eligible             bool                       `json:"eligible"`
}

// InventoryEntry restores the exact persisted fingerprint for bounded native
// revalidation. It does not resolve or authorize the artifact's storage root.
func (c RecoveryCandidate) InventoryEntry() sessioninventory.FileEntry {
	return sessioninventory.FileEntry{
		Artifact:        sessioninventory.Artifact{StorageRoot: c.ArtifactRoot, RelativePath: c.ArtifactRelativePath},
		StableFileID:    sessioninventory.StableFileID(c.StableFileID),
		GenerationToken: sessioninventory.GenerationToken(c.GenerationToken),
		MutationToken:   sessioninventory.MutationToken(c.MutationToken),
		Size:            c.Size,
		ModTime:         &c.ModTime,
	}
}

type RecoveryContext struct {
	Source           *NativeRecoverySource      `json:"source,omitempty"`
	Unknown          map[string]json.RawMessage `json:"-"`
	Version          int                        `json:"version"`
	Address          Address                    `json:"address"`
	Capture          RecoveryCapture            `json:"capture"`
	Generation       string                     `json:"generation"`
	Digest           string                     `json:"digest"`
	CreatedAt        time.Time                  `json:"created_at"`
	ExpiresAt        time.Time                  `json:"expires_at"`
	Branch           string                     `json:"branch,omitempty"`
	Issue            string                     `json:"issue,omitempty"`
	Candidates       []RecoveryCandidate        `json:"candidates,omitempty"`
	SelectedKey      string                     `json:"selected_key"`
	SelectedNativeID string                     `json:"selected_native_id,omitempty"`
}

type RecoveryMode string

const (
	RecoveryInferred RecoveryMode = "inferred"
	RecoveryProven   RecoveryMode = "proven"
	RecoveryFresh    RecoveryMode = "fresh"
)

type RecoveryOrientation struct {
	Branch      string           `json:"branch,omitempty"`
	Issue       string           `json:"issue,omitempty"`
	Capture     *RecoveryCapture `json:"capture,omitempty"`
	Unavailable string           `json:"unavailable,omitempty"`
}

type RecoveryIntent struct {
	Mode           RecoveryMode         `json:"mode,omitempty"`
	Address        Address              `json:"address,omitzero"`
	SourceRevision uint64               `json:"source_revision,omitempty"`
	CreatedAt      time.Time            `json:"created_at,omitzero"`
	Agent          string               `json:"agent,omitempty"`
	Orientation    *RecoveryOrientation `json:"orientation,omitempty"`
	Context        RecoveryContext      `json:"context,omitzero"`
	SelectedKey    string               `json:"selected_key"`
	NativeID       string               `json:"native_id,omitempty"`
}

func (i RecoveryIntent) Kind() RecoveryMode {
	if i.Mode == "" {
		return RecoveryInferred
	}
	return i.Mode
}

func recoveryText(s string, max int, required bool) bool {
	return (!required || s != "") && len(s) <= max && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
}
func recoveryTime(t time.Time) bool { return !t.IsZero() && t.Year() >= 1 && t.Year() <= 9999 }
func recoveryPath(p string) bool {
	return recoveryText(p, 4096, true) && filepath.IsAbs(p) && filepath.Clean(p) == p && p != "/"
}
func recoveryWorkingDirectory(p string) bool {
	return recoveryText(p, 4096, true) && filepath.IsAbs(p) && filepath.Clean(p) == p
}
func recoveryFileIdentity(i artifactpath.CaptureFileIdentity) bool {
	return i.Size >= 0 && i.Device != 0 && i.Inode != 0 && recoveryTime(i.ModifiedAt)
}

// SourceKind keeps legacy capture contexts implicit.
func (c RecoveryContext) SourceKind() sessioninventory.RecoverySourceKind {
	if c.Version == 1 {
		return sessioninventory.RecoverySourceCapture
	}
	if c.Source != nil {
		return c.Source.Kind
	}
	return ""
}

func validateNativeRecoverySource(source *NativeRecoverySource) error {
	if source == nil || source.Kind != sessioninventory.RecoverySourceNative || source.Revision == 0 || (!recoveryTime(source.Cutoff) || !recoveryTime(source.Cutoff.Add(NativeRecoverySelectionLifetime))) {
		return errors.New("invalid native recovery source")
	}
	_, offset := source.Cutoff.Zone()
	if offset != 0 {
		return errors.New("native recovery cutoff must be UTC")
	}
	if source.LaunchHint != nil && (!recoveryTime(*source.LaunchHint) || source.LaunchHint.After(source.Cutoff)) {
		return errors.New("invalid native recovery launch hint")
	}
	return nil
}

// ValidateNativeRecoverySourceAt is action admission; structural reads never use now.
func ValidateNativeRecoverySourceAt(source *NativeRecoverySource, now time.Time, expectedRevision uint64) error {
	if err := validateNativeRecoverySource(source); err != nil {
		return err
	}
	if source.Revision != expectedRevision || !recoveryTime(now) || now.Before(source.Cutoff) || !now.Before(source.Cutoff.Add(NativeRecoverySelectionLifetime)) {
		return errors.New("native recovery evidence changed or expired; refresh resume candidates")
	}
	return nil
}

// ValidateRecoveryContext is structural and clock-independent so expired
// evidence remains readable after restart. Admission separately rechecks expiry
// and the physical capture or native artifact.
func ValidateRecoveryContext(c RecoveryContext, address Address) error {
	if (c.Version != 1 && c.Version != 2) || c.Address != address {
		return errors.New("recovery context version or address mismatch")
	}
	if len(c.Digest) != 64 || strings.Trim(c.Digest, "0123456789abcdef") != "" {
		return errors.New("invalid recovery generation or digest")
	}
	if c.Version == 1 {
		if c.Source != nil || !recoveryText(c.Generation, 1024, true) {
			return errors.New("invalid capture recovery source")
		}
		if err := validateRecoveryCapture(c.Capture); err != nil {
			return err
		}
		if !recoveryTime(c.CreatedAt) || !recoveryTime(c.ExpiresAt) || c.CreatedAt.Before(c.Capture.CapturedAt) || !c.ExpiresAt.After(c.CreatedAt) {
			return errors.New("invalid recovery lifetime")
		}
	} else {
		if err := validateNativeRecoverySource(c.Source); err != nil {
			return err
		}
		if c.Capture != (RecoveryCapture{}) || c.Generation != "" || len(c.Candidates) == 0 {
			return errors.New("mixed or empty native recovery evidence")
		}
		if !c.CreatedAt.Equal(c.Source.Cutoff) || !c.ExpiresAt.Equal(c.Source.Cutoff.Add(NativeRecoverySelectionLifetime)) {
			return errors.New("invalid native recovery lifetime")
		}
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
		if c.Version == 2 && (!recoveryWorkingDirectory(candidate.CWD) || candidate.Score != 0 || candidate.Matched != 0 || candidate.Total != 0 || candidate.LastActivity.After(c.Source.Cutoff)) {
			return errors.New("invalid native-only recovery evidence")
		}
		if c.Version == 1 && candidate.CWD != "" && !recoveryWorkingDirectory(candidate.CWD) {
			return errors.New("invalid recovery cwd")
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
	if native != "" || c.Version == 2 {
		return errors.New("recovery selection is absent from snapshot")
	}
	// Fresh keys are source-bound by the caller's codec, not inferred here.
	return nil
}
func validateRecoveryCapture(c RecoveryCapture) error {
	if !recoveryPath(c.Raw) || !recoveryPath(c.Events) || c.Raw == c.Events || !recoveryFileIdentity(c.RawIdentity) || !recoveryFileIdentity(c.EventsIdentity) || !recoveryTime(c.CapturedAt) {
		return errors.New("invalid recovery capture")
	}
	if c.Token != "" && (len(c.Token) > 128 || !componentPattern.MatchString(c.Token)) {
		return errors.New("invalid recovery capture token")
	}
	if !c.LastActivity.IsZero() && (!recoveryTime(c.LastActivity) || c.LastActivity.After(c.CapturedAt)) {
		return errors.New("invalid capture activity cutoff")
	}
	if (c.LaunchOrdinal == 0) != (c.Nonce == "") || !recoveryText(c.Nonce, 1024, false) {
		return errors.New("incomplete recovery producer")
	}
	return nil
}

func ValidateRecoveryIntent(i RecoveryIntent, address Address) error {
	switch i.Kind() {
	case RecoveryInferred:
		if err := ValidateRecoveryContext(i.Context, address); err != nil {
			return err
		}
		return validateRecoverySelection(i.Context, i.SelectedKey, i.NativeID)
	case RecoveryProven, RecoveryFresh:
		if i.Address != address || i.SourceRevision == 0 || !recoveryTime(i.CreatedAt) || !recoveryText(i.Agent, 128, true) || !componentPattern.MatchString(i.Agent) || !recoveryText(i.SelectedKey, 4096, true) || !recoveryText(i.NativeID, 1024, i.Kind() == RecoveryProven) {
			return errors.New("invalid explicit recovery intent")
		}
		if i.Kind() == RecoveryFresh && i.NativeID != "" {
			return errors.New("fresh recovery cannot select a native identity")
		}
		if o := i.Orientation; o != nil {
			if !recoveryText(o.Branch, 1024, false) || !recoveryText(o.Issue, 1024, false) || !recoveryText(o.Unavailable, 2048, false) {
				return errors.New("invalid recovery orientation")
			}
			if o.Capture != nil {
				if err := validateRecoveryCapture(*o.Capture); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return errors.New("unknown recovery mode")
	}
}
func CloneRecoveryContext(c *RecoveryContext) *RecoveryContext {
	if c == nil {
		return nil
	}
	out := *c
	out.Unknown = strictjson.CloneRaw(c.Unknown)
	out.Source = CloneNativeRecoverySource(c.Source)
	if c.Candidates != nil {
		out.Candidates = append([]RecoveryCandidate{}, c.Candidates...)
		for i := range out.Candidates {
			out.Candidates[i].Unknown = strictjson.CloneRaw(c.Candidates[i].Unknown)
		}
	}
	return &out
}
func CloneRecoveryIntent(i *RecoveryIntent) *RecoveryIntent {
	if i == nil {
		return nil
	}
	out := *i
	out.Context = *CloneRecoveryContext(&i.Context)
	if i.Orientation != nil {
		orientation := *i.Orientation
		if orientation.Capture != nil {
			capture := *orientation.Capture
			orientation.Capture = &capture
		}
		out.Orientation = &orientation
	}
	return &out
}

func CloneNativeRecoverySource(source *NativeRecoverySource) *NativeRecoverySource {
	if source == nil {
		return nil
	}
	out := *source
	out.Unknown = strictjson.CloneRaw(source.Unknown)
	if source.LaunchHint != nil {
		hint := *source.LaunchHint
		out.LaunchHint = &hint
	}
	return &out
}
