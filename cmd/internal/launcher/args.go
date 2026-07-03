package launcher

import (
	"fmt"
	"strings"
)

// LaunchArgs is the pure parse result for the guarded pair-go launch prototype.
type LaunchArgs struct {
	Command     string // "" = launch; "list" (#99 M5a); "rename"/"continue" (#99 M5b)
	Agent       string
	ForcedTag   string
	SelectedTag string
	AgentArgs   []string

	// rename (#99 M5b): `pair rename [--restart-check] <old> <new>`. Raw tags —
	// normalized + gated in runRename so it owns the operator-facing messages.
	RenameOld       string
	RenameNew       string
	RenameCheckOnly bool

	// continue (#99 M5b): the raw slug (normalized at resolve time). "" with
	// Command=="continue" is the bare list mode. Agent/AgentArgs above carry the
	// optional agent port + `-- <forwarded>` args.
	ContinueSlug string
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
	case "-h", "--help", "help":
		// Native help (#99 M5c — the shell owned this before retirement).
		return LaunchArgs{Command: "help"}, nil
	case "list", "ls":
		// The read-only session listing (#99 M5a). No further args (shell
		// `list|ls)` ignores extras); a bare command marker is enough.
		return LaunchArgs{Command: "list"}, nil
	case "rename":
		return parseRename(argv[1:]) // #99 M5b
	case "continue":
		return parseContinue(argv[1:]) // #99 M5b
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
			// A leading flag that isn't -h/--help (handled above) is not an agent
			// name — agents never start with '-'. Refuse with a usage error;
			// LaunchNative prints it + exits 2 (#99 M5c — no shell to defer to).
			if strings.HasPrefix(arg, "-") {
				return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair: %q is a flag, not an agent (use '--' to forward args, or -h for help)", arg)}
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

// parseRename parses `rename [--restart-check] [--] <old> <new>` (#99 M5b, shell
// 329-354). Structural only — tag normalization/length/old!=new gates live in
// runRename (validateRenameTags) so it owns the operator-facing messages.
func parseRename(args []string) (LaunchArgs, error) {
	out := LaunchArgs{Command: "rename"}
	i := 0
	for i < len(args) {
		if args[i] == "--restart-check" {
			out.RenameCheckOnly = true
			i++
			continue
		}
		if args[i] == "--" {
			i++ // end of flags; positionals follow
		}
		break
	}
	rest := args[i:]
	if len(rest) < 2 {
		return LaunchArgs{}, UsageError{Message: "usage: pair rename [--restart-check] <old> <new>"}
	}
	if len(rest) > 2 {
		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair rename: unexpected arg '%s'", rest[2])}
	}
	out.RenameOld, out.RenameNew = rest[0], rest[1]
	return out, nil
}

// parseContinue parses `continue [slug] [agent] [-- args...]` (#99 M5b, shell
// 612-648). Bare (no slug) is the list mode. After the slug, an optional agent
// port (unless it's `--`) overrides the doc's frontmatter agent; everything from
// `--` onward forwards to the agent. The slug stays raw (normalized at resolve).
func parseContinue(args []string) (LaunchArgs, error) {
	out := LaunchArgs{Command: "continue"}
	if len(args) == 0 {
		return out, nil // bare list
	}
	out.ContinueSlug = args[0]
	rest := args[1:]
	if len(rest) > 0 && rest[0] != "--" {
		out.Agent = rest[0] // explicit port
		rest = rest[1:]
	}
	if len(rest) > 0 {
		if rest[0] != "--" {
			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair continue: unexpected arg '%s' (use '--' to forward args)", rest[0])}
		}
		out.AgentArgs = rest[1:]
	}
	return out, nil
}
