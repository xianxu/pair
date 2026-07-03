package launcher

import (
	"fmt"
	"strings"
)

// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
type LaunchArgs struct {
	Command     string // "" = launch; "list" = the read-only `list`/`ls` subcommand (#99 M5a)
	Agent       string
	ForcedTag   string
	SelectedTag string
	AgentArgs   []string
}

// UsageError is an operator-facing parse error.
type UsageError struct {
	Message string
}

func (e UsageError) Error() string {
	return e.Message
}

// ParseArgs parses pair-go launch args. It intentionally supports only the
// decision-phase subset for #75; unsupported shell-owned launcher verbs fail
// explicitly.
func ParseArgs(argv []string) (LaunchArgs, error) {
	var out LaunchArgs
	if len(argv) == 0 {
		out.Agent = "claude"
		return out, nil
	}

	switch argv[0] {
	case "list", "ls":
		// The read-only session listing (#99 M5a). No further args (shell
		// `list|ls)` ignores extras); a bare command marker is enough.
		return LaunchArgs{Command: "list"}, nil
	case "continue", "rename":
		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %s is not implemented by pair-go launch; use pair", argv[0])}
	case "resume":
		if len(argv) < 2 {
			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
		}
		tag, err := NormalizeTag(argv[1])
		if err != nil {
			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: invalid tag: %v", err)}
		}
		if len(argv) > 2 {
			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", argv[2])}
		}
		out.ForcedTag = tag
		return out, nil
	}

	seenSeparator := false
	for _, arg := range argv {
		if seenSeparator {
			out.AgentArgs = append(out.AgentArgs, arg)
			continue
		}
		if arg == "--" {
			seenSeparator = true
			continue
		}
		if out.Agent == "" {
			// A leading flag (e.g. --help, -h) is not an agent name — agents
			// never start with '-'. The shell owns help/flag handling, so refuse
			// here; LaunchNative maps this to ErrFallbackToShell → bin/pair-shell
			// (#99 M4, once native is the default entry).
			if strings.HasPrefix(arg, "-") {
				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: %q is a flag, not an agent (shell-owned)", arg)}
			}
			out.Agent = arg
			continue
		}
		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair-go launch: unexpected positional arg %q (use '--' to forward args to the agent)", arg)}
	}

	if out.Agent == "" {
		out.Agent = "claude"
	}
	return out, nil
}
