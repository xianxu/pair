package launcher

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const CouchLaunchProfileEnv = "PAIR_COUCH_LAUNCH_PROFILE"

type couchLaunchProfileWire struct {
	SchemaVersion int      `json:"schema_version"`
	Tag           string   `json:"tag"`
	Agent         string   `json:"agent"`
	Argv          []string `json:"argv"`
	AgentSource   string   `json:"agent_source"`
	ArgvSource    string   `json:"argv_source"`
}

func BuildCouchLaunchProfile(tag, agent string, argv []string, agentSource, argvSource string) (string, error) {
	profile := couchLaunchProfileWire{
		SchemaVersion: 1, Tag: tag, Agent: agent, Argv: append([]string(nil), argv...),
		AgentSource: agentSource, ArgvSource: argvSource,
	}
	if profile.Argv == nil {
		profile.Argv = []string{}
	}
	if !IsSupportedAgent(agent) {
		return "", fmt.Errorf("unsupported couch launch agent %q", agent)
	}
	if tag == "" {
		return "", fmt.Errorf("couch launch profile has no tag")
	}
	switch agentSource {
	case "explicit", "path", "root":
	default:
		return "", fmt.Errorf("unsupported couch agent source %q", agentSource)
	}
	switch argvSource {
	case "path", "repo-default":
	default:
		return "", fmt.Errorf("unsupported couch argv source %q", argvSource)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(profile); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ApplyCouchLaunchProfile binds one trusted, tag-addressed Couch resolution to
// Pair's ordinary launch policy without exposing a second public CLI grammar.
func ApplyCouchLaunchProfile(args LaunchArgs, raw string) (LaunchArgs, string, error) {
	var profile couchLaunchProfileWire
	if err := strictjson.Decode([]byte(raw), &profile); err != nil {
		return LaunchArgs{}, "", fmt.Errorf("decode couch launch profile: %w", err)
	}
	if profile.SchemaVersion != 1 || args.ForcedTag == "" || profile.Tag != args.ForcedTag {
		return LaunchArgs{}, "", fmt.Errorf("couch launch profile does not match forced tag %q", args.ForcedTag)
	}
	if !IsSupportedAgent(profile.Agent) {
		return LaunchArgs{}, "", fmt.Errorf("unsupported couch launch agent %q", profile.Agent)
	}
	if profile.Argv == nil {
		return LaunchArgs{}, "", fmt.Errorf("couch launch profile has null argv")
	}
	switch profile.AgentSource {
	case "explicit", "path", "root":
	default:
		return LaunchArgs{}, "", fmt.Errorf("unsupported couch agent source %q", profile.AgentSource)
	}
	switch profile.ArgvSource {
	case "path", "repo-default":
	default:
		return LaunchArgs{}, "", fmt.Errorf("unsupported couch argv source %q", profile.ArgvSource)
	}
	args.Agent = profile.Agent
	args.AgentExplicit = true
	args.AgentArgs = append([]string(nil), profile.Argv...)
	if args.AgentArgs == nil {
		args.AgentArgs = []string{}
	}
	args.AgentArgsExplicit = true
	args.AgentArgsFromCouch = true
	return args, profile.ArgvSource, nil
}

type LaunchArgInputs struct {
	Agent        string
	Args         LaunchArgs
	Saved        savedConfig
	SavedResumes bool
	Default      AgentDefault
	DefaultFound bool
}

type LaunchArgDecision struct {
	Args           []string
	ResumeID       string
	Warnings       []string
	PersistDefault bool
	ClearDefault   bool
}

func DecideLaunchArgs(in LaunchArgInputs) LaunchArgDecision {
	var out LaunchArgDecision
	var args []string
	argsSelected := false

	if in.Args.AgentArgsExplicit || len(in.Args.AgentArgs) > 0 {
		args = append([]string(nil), in.Args.AgentArgs...)
		argsSelected = true
		if in.Args.AgentArgsExplicit && !in.Args.AgentArgsFromCouch {
			out.PersistDefault = true
			out.ClearDefault = len(args) == 0
		}
		if sid := extractExplicitResume(in.Agent, args); sid != "" {
			out.ResumeID = sid
			out.Args = args
			return out
		}
	} else if savedConfigUsable(in.Agent, in.Saved) {
		args = persistedConfigArgs(in.Saved.Args)
		argsSelected = true
	} else if in.Saved.Agent != "" && in.Saved.Agent != in.Agent {
		out.Warnings = append(out.Warnings, fmt.Sprintf("saved config agent %q does not match requested agent %q; ignoring it", in.Saved.Agent, in.Agent))
	}

	if !argsSelected && in.DefaultFound && in.Default.Agent == in.Agent {
		args = append([]string(nil), in.Default.Args...)
		argsSelected = true
	}
	if !argsSelected {
		args = []string{}
	}

	if savedConfigUsable(in.Agent, in.Saved) && in.Saved.SessionID != "" && len(resumeToken(in.Agent, in.Saved.SessionID)) > 0 {
		if in.SavedResumes {
			out.ResumeID = in.Saved.SessionID
			args = composeResumeArgs(in.Agent, args, in.Saved.SessionID)
		} else {
			out.Warnings = append(out.Warnings, fmt.Sprintf("saved session %q for %s is not available; starting fresh", in.Saved.SessionID, in.Agent))
		}
	}

	out.Args = append([]string(nil), args...)
	return out
}

func savedConfigUsable(agent string, saved savedConfig) bool {
	return agent != "" && saved.Agent == agent
}
