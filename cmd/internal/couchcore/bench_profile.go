package couchcore

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// BenchProfile is a bench slot's exact launch profile (pair#847): a supported
// agent and explicit argv, never the saved configuration. Fresh recovery
// launches it once; it is not remembered for the repository.
type BenchProfile struct {
	Agent string
	Argv  []string
}

// ParseBenchProfile validates an explicit profile. Nil argv is absent and
// refused; an empty slice explicitly selects no startup parameters.
func ParseBenchProfile(agent string, argv []string) (BenchProfile, error) {
	if agent == "" || !launcher.IsSupportedAgent(agent) {
		return BenchProfile{}, fmt.Errorf("bench profile needs a supported agent, got %q", agent)
	}
	if argv == nil {
		return BenchProfile{}, errors.New("bench profile needs explicit argv (empty for none)")
	}
	for _, arg := range argv {
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return BenchProfile{}, errors.New("bench profile argv contains invalid UTF-8 or NUL")
		}
	}
	if len(launcher.FormatLaunchParameters(argv)) > MaxRepoArgumentBytes {
		return BenchProfile{}, fmt.Errorf("bench profile argv exceeds %d formatted bytes", MaxRepoArgumentBytes)
	}
	return BenchProfile{Agent: agent, Argv: append([]string{}, argv...)}, nil
}

func (p BenchProfile) resolution() LaunchProfileResolution {
	return LaunchProfileResolution{Profile: LaunchProfile{Agent: p.Agent, Argv: append([]string{}, p.Argv...)}, AgentSource: AgentSourceExplicit, ArgvSource: ArgvSourceExplicit}
}

// selectionBenchProfile is the explicit profile a Fresh selection carries, if any.
func selectionBenchProfile(selection RecoverySelection) (*BenchProfile, error) {
	if selection.Argv == nil {
		return nil, nil
	}
	if selection.Kind != RecoveryFresh {
		return nil, errors.New("an explicit launch profile starts fresh; native resume keeps its saved agent")
	}
	profile, err := ParseBenchProfile(selection.Agent, *selection.Argv)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
