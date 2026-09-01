package couchcore

import (
	"reflect"
	"testing"
)

func TestResolveStartResolutionUsesLaunchProfileOracleAndOwnsValues(t *testing.T) {
	input := validStartResolutionInput()
	want, err := ResolveLaunchProfile(input.LaunchProfileInputs)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveStartResolution(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolved.Profile, want.Profile) || resolved.AgentSource != want.AgentSource || resolved.ArgvSource != want.ArgvSource {
		t.Fatalf("resolution profile = %+v, oracle = %+v", resolved, want)
	}
	input.LaunchProfileInputs.Path.ArgvByAgent["codex"][0] = "mutated"
	input.LaunchProfileInputs.RepoDefault.Argv[0] = "mutated"
	if !reflect.DeepEqual(resolved.Profile.Argv, []string{"--sandbox", "workspace-write"}) {
		t.Fatalf("resolution aliases inputs: %+v", resolved)
	}
}

func TestResolveStartResolutionFingerprintCoversNormalizedAuthority(t *testing.T) {
	base := validStartResolutionInput()
	baseline, err := ResolveStartResolution(base)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*StartResolutionInput){
		"canonical path": func(in *StartResolutionInput) {
			in.CanonicalPath = "/repo/other"
			in.LaunchProfileInputs.Path.PhysicalPath = "/repo/other"
		},
		"worktree":       func(in *StartResolutionInput) { in.Worktree = "/other" },
		"selected agent": func(in *StartResolutionInput) { in.LaunchProfileInputs.ExplicitAgent = "claude" },
		"selected argv element": func(in *StartResolutionInput) {
			in.LaunchProfileInputs.Path.ArgvByAgent["codex"][1] = "danger-full-access"
		},
		"preference revision": func(in *StartResolutionInput) { in.LaunchProfileInputs.Path.Revision++ },
		"repository default": func(in *StartResolutionInput) {
			in.LaunchProfileInputs.RepoDefault.Argv[0] = "--different-default"
		},
		"policy digest": func(in *StartResolutionInput) {
			in.CandidatePolicy.PolicyDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		},
		"admission key": func(in *StartResolutionInput) { in.CandidatePolicy.AdmissionKey = "/repo/other-key" },
		"capacity":      func(in *StartResolutionInput) { in.CandidatePolicy.Capacity.Limit++ },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := cloneStartResolutionInput(base)
			mutate(&changed)
			resolved, resolveErr := ResolveStartResolution(changed)
			if resolveErr != nil {
				t.Fatal(resolveErr)
			}
			if resolved.Fingerprint == baseline.Fingerprint {
				t.Fatalf("fingerprint unchanged after %s mutation: %s", name, resolved.Fingerprint)
			}
		})
	}

	again, err := ResolveStartResolution(cloneStartResolutionInput(base))
	if err != nil || again.Fingerprint != baseline.Fingerprint {
		t.Fatalf("same input fingerprint = %s, %v; want %s", again.Fingerprint, err, baseline.Fingerprint)
	}
}

func TestResolveStartResolutionRejectsMalformedAuthority(t *testing.T) {
	tests := map[string]func(*StartResolutionInput){
		"relative path":     func(in *StartResolutionInput) { in.CanonicalPath = "relative" },
		"relative worktree": func(in *StartResolutionInput) { in.Worktree = "relative" },
		"invalid policy":    func(in *StartResolutionInput) { in.CandidatePolicy.PolicyDigest = "invalid" },
		"preference repo":   func(in *StartResolutionInput) { in.LaunchProfileInputs.Path.RepoIdentity = "other" },
		"preference path":   func(in *StartResolutionInput) { in.LaunchProfileInputs.Path.PhysicalPath = "/other" },
		"nil default argv":  func(in *StartResolutionInput) { in.LaunchProfileInputs.RepoDefault.Argv = nil },
		"unsupported agent": func(in *StartResolutionInput) { in.LaunchProfileInputs.ExplicitAgent = "unknown" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			input := cloneStartResolutionInput(validStartResolutionInput())
			mutate(&input)
			if _, err := ResolveStartResolution(input); err == nil {
				t.Fatal("malformed start authority accepted")
			}
		})
	}
}

func validStartResolutionInput() StartResolutionInput {
	return StartResolutionInput{
		CanonicalPath: "/repo/task",
		Worktree:      "/repo",
		LaunchProfileInputs: LaunchProfileInputs{
			ExplicitAgent: "codex",
			RootAgent:     "claude",
			Path: &PathLaunchPreference{
				SchemaVersion: PathLaunchPreferenceSchemaVersion,
				RepoIdentity:  "/repo/.git",
				PhysicalPath:  "/repo/task",
				LastAgent:     "claude",
				ArgvByAgent: map[string][]string{
					"claude": {"--model", "opus"},
					"codex":  {"--sandbox", "workspace-write"},
				},
				Revision: 2,
			},
			RepoDefault: &LaunchProfile{Agent: "codex", Argv: []string{"--search"}},
		},
		CandidatePolicy: PolicyResult{
			PolicyVersion: 1,
			PolicyDigest:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			RepoIdentity:  "/repo/.git",
			AdmissionKey:  "/repo",
			Capacity:      PolicyCapacity{Kind: CapacityBounded, Limit: 2},
			OnCapacity:    CapacityReject,
		},
	}
}

func cloneStartResolutionInput(input StartResolutionInput) StartResolutionInput {
	out := input
	if input.LaunchProfileInputs.Path != nil {
		path := clonePathLaunchPreference(*input.LaunchProfileInputs.Path)
		out.LaunchProfileInputs.Path = &path
	}
	if input.LaunchProfileInputs.RepoDefault != nil {
		profile := cloneLaunchProfile(*input.LaunchProfileInputs.RepoDefault)
		out.LaunchProfileInputs.RepoDefault = &profile
	}
	return out
}
