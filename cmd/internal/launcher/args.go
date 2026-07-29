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
	Layout      LayoutRequest

	// rename (#99 M5b): `pair rename [--restart-check] <old> <new>`. Raw tags —
	// normalized + gated in runRename so it owns the operator-facing messages.
	RenameOld       string
	RenameNew       string
	RenameCheckOnly bool

	// continue (#99 M5b): the raw slug (normalized at resolve time). "" with
	// Command=="continue" is the bare list mode. Agent/AgentArgs above carry the
	// optional agent port + `-- <forwarded>` args.
	ContinueSlug string

	// restart (#94 M1): `pair restart [--new-session] [--rename-to <tag>]` — the
	// nvim-keybind lifecycle writer ported from bin/pair-restart.sh. Both fields
	// are written into the restart marker; RenameTo is the inside-flow tag rename
	// (distinct from the `rename` command's RenameOld/RenameNew).
	NewSession bool
	RenameTo   string
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
	clean, layout, err := extractLayoutRequest(argv)
	if err != nil {
		return LaunchArgs{}, err
	}
	out, err := parseArgs(clean)
	if err != nil {
		return LaunchArgs{}, err
	}
	if layout.Explicit && !launchArgsAcceptLayout(out) {
		return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair: layout flags do not apply to %q", firstNonEmpty(out.Command, "this command"))}
	}
	out.Layout = layout
	return out, nil
}

func parseArgs(argv []string) (LaunchArgs, error) {
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
	case "restart":
		return parseRestart(argv[1:]) // #94 M1
	case "quit":
		return LaunchArgs{Command: "quit"}, nil // #94 M1
	case "resume":
		if len(argv) < 2 {
			return LaunchArgs{}, UsageError{Message: "pair-go launch: 'resume' requires a tag"}
		}
		// A pasted session name is resolved through the ledger before charset
		// validation (#130). NormalizeTag strips the legacy `pair-` prefix itself,
		// but the 📁 scheme has no string inverse — and 📁 is not in NormalizeTag's
		// charset, so without this a user pasting the tab-title text gets
		// "contains invalid character".
		tag, err := ResumeTagFromArg(argv[1])
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

func extractLayoutRequest(argv []string) ([]string, LayoutRequest, error) {
	clean := make([]string, 0, len(argv))
	var request LayoutRequest
	for i, arg := range argv {
		if arg == "--" {
			clean = append(clean, argv[i:]...)
			break
		}
		var mode LayoutMode
		switch arg {
		case "--layout2":
			mode = Layout2
		case "--layout3":
			mode = Layout3
		default:
			clean = append(clean, arg)
			continue
		}
		if request.Explicit && request.Mode != mode {
			return nil, LayoutRequest{}, UsageError{Message: fmt.Sprintf("pair: conflicting layout flags %q and %q", "--"+string(request.Mode), arg)}
		}
		request = LayoutRequest{Mode: mode, Explicit: true}
	}
	return clean, request, nil
}

func launchArgsAcceptLayout(args LaunchArgs) bool {
	if args.Command == "" {
		return true
	}
	return args.Command == "continue" && args.ContinueSlug != ""
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

// parseRestart parses `restart [--new-session] [--rename-to <tag>]` (#94 M1,
// ported from bin/pair-restart.sh:29-56). Flags only; no positionals. The tag in
// --rename-to stays raw — nvim validates it (`pair rename --restart-check`)
// before ever invoking restart.
func parseRestart(args []string) (LaunchArgs, error) {
	out := LaunchArgs{Command: "restart"}
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--new-session":
			out.NewSession = true
			i++
		case "--rename-to":
			if i+1 >= len(args) || args[i+1] == "" {
				return LaunchArgs{}, UsageError{Message: "pair restart: --rename-to requires a value"}
			}
			out.RenameTo = args[i+1]
			i += 2
		default:
			return LaunchArgs{}, UsageError{Message: fmt.Sprintf("pair restart: unknown arg %q", args[i])}
		}
	}
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

// ResumeTagFromArg accepts what a user may type after `pair resume`: a bare tag,
// a legacy `pair-<tag>` name, or a 📁 session name pasted out of the tab title /
// `zellij list-sessions`.
//
// It stays PURE, so it cannot resolve the 📁 form — that needs the ledger, and
// ParseArgs has no Runtime. A 📁 value is therefore passed through verbatim and
// resolved later by resolveResumeTag, at the first point the index is in hand.
// Without this, NormalizeTag's charset loop rejects 📁 outright and the paste a
// user is most likely to make fails (#130).
func ResumeTagFromArg(raw string) (string, error) {
	if strings.HasPrefix(raw, sessionPrefix) {
		return raw, nil
	}
	return NormalizeTag(raw)
}
