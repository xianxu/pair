package launcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const CouchLaunchProfileEnv = "PAIR_COUCH_LAUNCH_PROFILE"

type LaunchDiagnosticCode string

const NativeBindingChanged LaunchDiagnosticCode = "native-binding-changed"

type LaunchRefusal struct {
	Code       LaunchDiagnosticCode
	Diagnostic string
}

func (e *LaunchRefusal) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Diagnostic)
}

func LaunchDiagnosticOf(err error) LaunchDiagnosticCode {
	var refusal *LaunchRefusal
	if errors.As(err, &refusal) {
		return refusal.Code
	}
	return ""
}

func RequireNativeResumeBinding(required, actual string, status sessioninventory.BindingStatus) error {
	if required == "" || status != sessioninventory.BindingEstablished || actual == "" || actual != required {
		return &LaunchRefusal{Code: NativeBindingChanged, Diagnostic: "required native session binding changed before launch"}
	}
	return nil
}

type TrustedLaunchProfile struct {
	SchemaVersion     int      `json:"schema_version"`
	Tag               string   `json:"tag"`
	Agent             string   `json:"agent"`
	Argv              []string `json:"argv"`
	AgentSource       string   `json:"agent_source"`
	ArgvSource        string   `json:"argv_source"`
	ResumeRequired    bool     `json:"resume_required,omitempty"`
	RequiredSessionID string   `json:"required_session_id,omitempty"`
}

func BuildCouchLaunchProfile(tag, agent string, argv []string, agentSource, argvSource string) (string, error) {
	profile := TrustedLaunchProfile{
		SchemaVersion: 1, Tag: tag, Agent: agent, Argv: append([]string(nil), argv...),
		AgentSource: agentSource, ArgvSource: argvSource,
	}
	if profile.Argv == nil {
		profile.Argv = []string{}
	}
	if err := ValidateTrustedLaunchProfile(profile); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(profile); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func BuildCouchResumeLaunchProfile(tag, agent string, argv []string, requiredSessionID string) (string, error) {
	profile := TrustedLaunchProfile{
		SchemaVersion: 1, Tag: tag, Agent: agent, Argv: append([]string(nil), argv...),
		AgentSource: "saved", ArgvSource: "saved", ResumeRequired: true, RequiredSessionID: requiredSessionID,
	}
	if profile.Argv == nil {
		profile.Argv = []string{}
	}
	if err := ValidateTrustedLaunchProfile(profile); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(profile); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ValidateTrustedLaunchProfile(profile TrustedLaunchProfile) error {
	if profile.SchemaVersion != 1 {
		return fmt.Errorf("unsupported couch launch profile schema %d", profile.SchemaVersion)
	}
	if profile.Tag == "" {
		return fmt.Errorf("couch launch profile has no tag")
	}
	if !IsSupportedAgent(profile.Agent) {
		return fmt.Errorf("unsupported couch launch agent %q", profile.Agent)
	}
	if profile.Argv == nil {
		return fmt.Errorf("couch launch profile has null argv")
	}
	if profile.ResumeRequired {
		if profile.RequiredSessionID == "" {
			return fmt.Errorf("resume-required couch launch profile has no required session ID")
		}
		if profile.AgentSource != "saved" || profile.ArgvSource != "saved" {
			return fmt.Errorf("resume-required couch launch profile requires saved agent and argv sources")
		}
		return nil
	}
	if profile.RequiredSessionID != "" {
		return fmt.Errorf("ordinary couch launch profile carries a required session ID")
	}
	switch profile.AgentSource {
	case "explicit", "path", "root":
	default:
		return fmt.Errorf("unsupported couch agent source %q", profile.AgentSource)
	}
	switch profile.ArgvSource {
	case "path", "repo-default":
	default:
		return fmt.Errorf("unsupported couch argv source %q", profile.ArgvSource)
	}
	return nil
}

// ApplyCouchLaunchProfile binds one trusted, tag-addressed Couch resolution to
// Pair's ordinary launch policy without exposing a second public CLI grammar.
func ApplyCouchLaunchProfile(args LaunchArgs, raw string) (LaunchArgs, string, error) {
	var profile TrustedLaunchProfile
	if err := strictjson.Decode([]byte(raw), &profile); err != nil {
		return LaunchArgs{}, "", fmt.Errorf("decode couch launch profile: %w", err)
	}
	if args.ForcedTag == "" || profile.Tag != args.ForcedTag {
		return LaunchArgs{}, "", fmt.Errorf("couch launch profile does not match forced tag %q", args.ForcedTag)
	}
	if err := ValidateTrustedLaunchProfile(profile); err != nil {
		return LaunchArgs{}, "", err
	}
	args.Agent = profile.Agent
	args.AgentExplicit = true
	args.AgentArgs = append([]string(nil), profile.Argv...)
	if args.AgentArgs == nil {
		args.AgentArgs = []string{}
	}
	args.AgentArgsExplicit = true
	args.AgentArgsFromCouch = true
	args.ResumeRequired = profile.ResumeRequired
	args.RequiredSessionID = profile.RequiredSessionID
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
