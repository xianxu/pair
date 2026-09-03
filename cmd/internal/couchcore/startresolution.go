package couchcore

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type StartResolutionFingerprint string

var ErrStartResolutionChanged = errors.New("start resolution changed")

type StartResolutionInput struct {
	CanonicalPath       string
	Worktree            Worktree
	Issue               string
	LaunchProfileInputs LaunchProfileInputs
	// RepoIdentity is the Git common directory. It replaces the fleet-policy
	// record this input used to carry (pair#170 M4): the only value anything
	// downstream still wanted from that record was this string, which keys the
	// path's saved launch preference.
	RepoIdentity string
}

// StartResolution is the immutable authority shared by preview and launch.
type StartResolution struct {
	CanonicalPath      string                     `json:"canonical_path"`
	Worktree           Worktree                   `json:"worktree"`
	Issue              string                     `json:"issue,omitempty"`
	RequestedAgent     string                     `json:"requested_agent,omitempty"`
	Profile            LaunchProfile              `json:"profile"`
	AgentSource        AgentSource                `json:"agent_source"`
	ArgvSource         ArgvSource                 `json:"argv_source"`
	RepoIdentity       string                     `json:"repo_identity"`
	PreferenceRevision uint64                     `json:"preference_revision,omitempty"`
	DefaultDigest      string                     `json:"default_digest,omitempty"`
	Fingerprint        StartResolutionFingerprint `json:"fingerprint"`
}

func ResolveStartResolution(input StartResolutionInput) (StartResolution, error) {
	if !filepath.IsAbs(input.CanonicalPath) || !filepath.IsAbs(string(input.Worktree)) {
		return StartResolution{}, errors.New("start resolution paths must be absolute")
	}
	if input.RepoIdentity == "" {
		return StartResolution{}, errors.New("start resolution has no repository identity")
	}
	var preferenceRevision uint64
	if preference := input.LaunchProfileInputs.Path; preference != nil {
		if err := validatePathLaunchPreference(*preference); err != nil {
			return StartResolution{}, err
		}
		if preference.RepoIdentity != input.RepoIdentity || preference.PhysicalPath != input.CanonicalPath {
			return StartResolution{}, errors.New("start resolution preference does not match candidate")
		}
		preferenceRevision = preference.Revision
	}
	defaultDigest := ""
	if profile := input.LaunchProfileInputs.RepoDefault; profile != nil {
		if profile.Agent == "" || profile.Argv == nil || !launcher.IsSupportedAgent(profile.Agent) {
			return StartResolution{}, errors.New("start resolution has invalid repository default")
		}
		defaultDigest = launchProfileDigest(*profile)
	}
	selected, err := ResolveLaunchProfile(input.LaunchProfileInputs)
	if err != nil {
		return StartResolution{}, err
	}
	if !launcher.IsSupportedAgent(selected.Profile.Agent) {
		return StartResolution{}, errors.New("start resolution selected an unsupported agent")
	}
	resolution := StartResolution{
		CanonicalPath:      input.CanonicalPath,
		Worktree:           input.Worktree,
		Issue:              input.Issue,
		RequestedAgent:     input.LaunchProfileInputs.ExplicitAgent,
		Profile:            cloneLaunchProfile(selected.Profile),
		AgentSource:        selected.AgentSource,
		ArgvSource:         selected.ArgvSource,
		RepoIdentity:       input.RepoIdentity,
		PreferenceRevision: preferenceRevision,
		DefaultDigest:      defaultDigest,
	}
	resolution.Fingerprint = fingerprintStartResolution(resolution)
	return resolution, nil
}

func cloneStartResolution(resolution StartResolution) StartResolution {
	resolution.Profile = cloneLaunchProfile(resolution.Profile)
	return resolution
}

func launchProfileDigest(profile LaunchProfile) string {
	digest := sha256.New()
	writeFingerprintField(digest, "pair-launch-profile-v1")
	writeFingerprintField(digest, profile.Agent)
	writeFingerprintUint(digest, uint64(len(profile.Argv)))
	for _, arg := range profile.Argv {
		writeFingerprintField(digest, arg)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func fingerprintStartResolution(resolution StartResolution) StartResolutionFingerprint {
	digest := sha256.New()
	writeFingerprintField(digest, "pair-start-resolution-v1")
	writeFingerprintField(digest, resolution.CanonicalPath)
	writeFingerprintField(digest, string(resolution.Worktree))
	writeFingerprintField(digest, resolution.Issue)
	writeFingerprintField(digest, resolution.RequestedAgent)
	writeFingerprintField(digest, resolution.Profile.Agent)
	writeFingerprintUint(digest, uint64(len(resolution.Profile.Argv)))
	for _, arg := range resolution.Profile.Argv {
		writeFingerprintField(digest, arg)
	}
	writeFingerprintField(digest, string(resolution.AgentSource))
	writeFingerprintField(digest, string(resolution.ArgvSource))
	writeFingerprintField(digest, resolution.RepoIdentity)
	writeFingerprintUint(digest, resolution.PreferenceRevision)
	writeFingerprintField(digest, resolution.DefaultDigest)
	return StartResolutionFingerprint(hex.EncodeToString(digest.Sum(nil)))
}

func writeFingerprintField(digest hash.Hash, value string) {
	writeFingerprintUint(digest, uint64(len(value)))
	_, _ = digest.Write([]byte(value))
}

func writeFingerprintUint(digest hash.Hash, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	_, _ = digest.Write(raw[:])
}

// CommitArgs renders the operation arguments that commit this preview: the
// SAME inputs it resolved from, plus the fingerprint the operator accepted.
//
// It exists because "the args that reproduce this resolution" was being
// rebuilt by hand at every call site (CLI, console menu, tests), and a call
// site that guessed wrong -- passing the RESOLVED agent where the operator
// gave none, say -- changes AgentSource and so re-resolves to a different
// fingerprint, failing the start with a drift error nobody drifted into.
// One owner, one contract (ARCH-DRY).
func (r StartResolution) CommitArgs() map[string]string {
	return map[string]string{
		"path":        r.CanonicalPath,
		"worktree":    string(r.Worktree),
		"agent":       r.RequestedAgent,
		"issue":       r.Issue,
		"fingerprint": string(r.Fingerprint),
	}
}
