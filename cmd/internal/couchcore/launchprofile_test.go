package couchcore

import (
	"reflect"
	"testing"
)

func TestLaunchProfileResolutionKeepsAgentAndArgvProvenanceIndependent(t *testing.T) {
	path := &PathLaunchPreference{
		SchemaVersion: 1,
		RepoIdentity:  "repo-identity",
		PhysicalPath:  "/repo/task",
		LastAgent:     "claude",
		ArgvByAgent: map[string][]string{
			"claude": {"--model", "opus"},
			"codex":  {"--sandbox", "workspace-write"},
		},
		Revision: 2,
	}
	tests := []struct {
		name         string
		in           LaunchProfileInputs
		wantAgent    string
		wantArgv     []string
		wantAgentSrc AgentSource
		wantArgvSrc  ArgvSource
	}{
		{
			name: "explicit agent reuses that agent's path argv",
			in: LaunchProfileInputs{
				ExplicitAgent: "codex", Path: path, RootAgent: "muse",
				RepoDefault: &LaunchProfile{Agent: "codex", Argv: []string{"--search"}},
			},
			wantAgent: "codex", wantArgv: []string{"--sandbox", "workspace-write"},
			wantAgentSrc: AgentSourceExplicit, wantArgvSrc: ArgvSourcePath,
		},
		{
			name: "path agent reuses its own path argv",
			in: LaunchProfileInputs{
				Path: path, RootAgent: "muse",
				RepoDefault: &LaunchProfile{Agent: "claude", Argv: []string{"--permission-mode", "plan"}},
			},
			wantAgent: "claude", wantArgv: []string{"--model", "opus"},
			wantAgentSrc: AgentSourcePath, wantArgvSrc: ArgvSourcePath,
		},
		{
			name: "root agent uses matching repo default when path has no history",
			in: LaunchProfileInputs{
				RootAgent:   "muse",
				RepoDefault: &LaunchProfile{Agent: "muse", Argv: []string{"--model", "spark"}},
			},
			wantAgent: "muse", wantArgv: []string{"--model", "spark"},
			wantAgentSrc: AgentSourceRoot, wantArgvSrc: ArgvSourceRepoDefault,
		},
		{
			name: "switching agents never crosses path argv",
			in: LaunchProfileInputs{
				ExplicitAgent: "muse", Path: path, RootAgent: "codex",
				RepoDefault: &LaunchProfile{Agent: "muse", Argv: []string{"--provider", "meta"}},
			},
			wantAgent: "muse", wantArgv: []string{"--provider", "meta"},
			wantAgentSrc: AgentSourceExplicit, wantArgvSrc: ArgvSourceRepoDefault,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveLaunchProfile(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.Profile.Agent != tc.wantAgent || !reflect.DeepEqual(got.Profile.Argv, tc.wantArgv) ||
				got.AgentSource != tc.wantAgentSrc || got.ArgvSource != tc.wantArgvSrc {
				t.Fatalf("ResolveLaunchProfile = %+v, want agent=%q argv=%q sources=%s/%s", got, tc.wantAgent, tc.wantArgv, tc.wantAgentSrc, tc.wantArgvSrc)
			}
		})
	}
}

func TestLaunchProfileResolutionReturnsDefensiveValuesAndRejectsCrossAgentDefault(t *testing.T) {
	pathArgv := []string{"--model", "opus"}
	path := &PathLaunchPreference{LastAgent: "claude", ArgvByAgent: map[string][]string{"claude": pathArgv}}
	got, err := ResolveLaunchProfile(LaunchProfileInputs{Path: path, RootAgent: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	pathArgv[0] = "mutated"
	path.ArgvByAgent["claude"][1] = "mutated"
	if !reflect.DeepEqual(got.Profile.Argv, []string{"--model", "opus"}) {
		t.Fatalf("resolution aliases preference argv: %q", got.Profile.Argv)
	}

	_, err = ResolveLaunchProfile(LaunchProfileInputs{
		RootAgent:   "codex",
		RepoDefault: &LaunchProfile{Agent: "claude", Argv: []string{"--dangerously-skip-permissions"}},
	})
	if err == nil {
		t.Fatal("cross-agent repository default accepted")
	}
}
