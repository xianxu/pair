package couchcore

import "testing"

func TestExplicitEmptyArgvWins(t *testing.T) {
	argv := []string{}
	got, err := ResolveLaunchProfile(LaunchProfileInputs{ExplicitAgent: "codex", ExplicitArgv: &argv, Path: &PathLaunchPreference{ArgvByAgent: map[string][]string{"codex": {"--old"}}}, RepoDefault: &LaunchProfile{Agent: "claude", Argv: []string{"wrong"}}})
	if err != nil || got.ArgvSource != ArgvSourceExplicit || got.Profile.Argv == nil || len(got.Profile.Argv) != 0 {
		t.Fatalf("explicit empty: %#v %v", got, err)
	}
}
