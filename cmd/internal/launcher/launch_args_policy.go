package launcher

import "fmt"

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
		if in.Args.AgentArgsExplicit {
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
