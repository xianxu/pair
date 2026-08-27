package couchcore

import (
	"fmt"
	"path/filepath"
)

const PathLaunchPreferenceSchemaVersion = 1

type AgentSource string

const (
	AgentSourceExplicit AgentSource = "explicit"
	AgentSourcePath     AgentSource = "path"
	AgentSourceRoot     AgentSource = "root"
)

type ArgvSource string

const (
	ArgvSourcePath        ArgvSource = "path"
	ArgvSourceRepoDefault ArgvSource = "repo-default"
)

// LaunchProfile is the exact agent invocation persisted only after Pair has
// registered the corresponding incarnation.
type LaunchProfile struct {
	Agent string   `json:"agent"`
	Argv  []string `json:"argv"`
}

// PathLaunchPreference remembers the last successful agent at one physical
// path and the last successful argv for each agent independently.
type PathLaunchPreference struct {
	SchemaVersion int                 `json:"schema_version"`
	RepoIdentity  string              `json:"repo_identity"`
	PhysicalPath  string              `json:"physical_path"`
	LastAgent     string              `json:"last_agent"`
	ArgvByAgent   map[string][]string `json:"argv_by_agent"`
	Revision      uint64              `json:"revision"`
}

type LaunchProfileInputs struct {
	ExplicitAgent string
	Path          *PathLaunchPreference
	RootAgent     string
	RepoDefault   *LaunchProfile
}

type LaunchProfileResolution struct {
	Profile     LaunchProfile
	AgentSource AgentSource
	ArgvSource  ArgvSource
}

// ResolveLaunchProfile resolves agent and argv on independent axes. Selecting
// another agent may reuse that agent's path argv, but can never inherit argv
// recorded for a different harness.
func ResolveLaunchProfile(in LaunchProfileInputs) (LaunchProfileResolution, error) {
	var out LaunchProfileResolution
	switch {
	case in.ExplicitAgent != "":
		out.Profile.Agent = in.ExplicitAgent
		out.AgentSource = AgentSourceExplicit
	case in.Path != nil && in.Path.LastAgent != "":
		out.Profile.Agent = in.Path.LastAgent
		out.AgentSource = AgentSourcePath
	case in.RootAgent != "":
		out.Profile.Agent = in.RootAgent
		out.AgentSource = AgentSourceRoot
	default:
		return LaunchProfileResolution{}, fmt.Errorf("launch profile has no agent source")
	}

	if in.Path != nil {
		if argv, ok := in.Path.ArgvByAgent[out.Profile.Agent]; ok {
			out.Profile.Argv = cloneArgv(argv)
			out.ArgvSource = ArgvSourcePath
			return out, nil
		}
	}
	if in.RepoDefault != nil {
		if in.RepoDefault.Agent != out.Profile.Agent {
			return LaunchProfileResolution{}, fmt.Errorf("repository default agent %q does not match selected agent %q", in.RepoDefault.Agent, out.Profile.Agent)
		}
		out.Profile.Argv = cloneArgv(in.RepoDefault.Argv)
	} else {
		out.Profile.Argv = []string{}
	}
	out.ArgvSource = ArgvSourceRepoDefault
	return out, nil
}

func cloneLaunchProfile(profile LaunchProfile) LaunchProfile {
	return LaunchProfile{Agent: profile.Agent, Argv: cloneArgv(profile.Argv)}
}

func validatePathLaunchPreference(preference PathLaunchPreference) error {
	if preference.SchemaVersion != PathLaunchPreferenceSchemaVersion {
		return fmt.Errorf("unsupported path launch preference schema %d", preference.SchemaVersion)
	}
	if preference.RepoIdentity == "" {
		return fmt.Errorf("path launch preference has no repository identity")
	}
	if !filepath.IsAbs(preference.PhysicalPath) {
		return fmt.Errorf("path launch preference path must be absolute")
	}
	if preference.LastAgent == "" {
		return fmt.Errorf("path launch preference has no last agent")
	}
	if preference.ArgvByAgent == nil {
		return fmt.Errorf("path launch preference has no per-agent arguments")
	}
	for agent, argv := range preference.ArgvByAgent {
		if agent == "" {
			return fmt.Errorf("path launch preference has an empty agent key")
		}
		if argv == nil {
			return fmt.Errorf("path launch preference has null arguments for %q", agent)
		}
	}
	if _, ok := preference.ArgvByAgent[preference.LastAgent]; !ok {
		return fmt.Errorf("path launch preference has no arguments for last agent %q", preference.LastAgent)
	}
	if preference.Revision == 0 {
		return fmt.Errorf("path launch preference revision must be positive")
	}
	return nil
}

func clonePathLaunchPreference(preference PathLaunchPreference) PathLaunchPreference {
	out := preference
	out.ArgvByAgent = make(map[string][]string, len(preference.ArgvByAgent))
	for agent, argv := range preference.ArgvByAgent {
		out.ArgvByAgent[agent] = cloneArgv(argv)
	}
	return out
}

func RecordSuccessfulLaunch(current *PathLaunchPreference, repoIdentity, physicalPath string, profile LaunchProfile) (PathLaunchPreference, error) {
	if repoIdentity == "" || !filepath.IsAbs(physicalPath) {
		return PathLaunchPreference{}, fmt.Errorf("successful launch requires repository identity and absolute path")
	}
	if profile.Agent == "" || profile.Argv == nil {
		return PathLaunchPreference{}, fmt.Errorf("successful launch has incomplete profile")
	}
	var next PathLaunchPreference
	if current == nil {
		next = PathLaunchPreference{
			SchemaVersion: PathLaunchPreferenceSchemaVersion,
			RepoIdentity:  repoIdentity, PhysicalPath: physicalPath,
			ArgvByAgent: map[string][]string{}, Revision: 1,
		}
	} else {
		if err := validatePathLaunchPreference(*current); err != nil {
			return PathLaunchPreference{}, err
		}
		if current.RepoIdentity != repoIdentity || current.PhysicalPath != physicalPath {
			return PathLaunchPreference{}, fmt.Errorf("path launch preference address mismatch")
		}
		next = clonePathLaunchPreference(*current)
		next.Revision++
	}
	next.LastAgent = profile.Agent
	next.ArgvByAgent[profile.Agent] = cloneArgv(profile.Argv)
	if err := validatePathLaunchPreference(next); err != nil {
		return PathLaunchPreference{}, err
	}
	return next, nil
}

func cloneArgv(argv []string) []string {
	out := append([]string(nil), argv...)
	if out == nil {
		out = []string{}
	}
	return out
}
