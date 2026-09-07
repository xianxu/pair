package couchcmd

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type cliKind uint8

const (
	cliInvalid cliKind = iota
	cliLaunch
	cliList
	cliArchived
	cliShow
	cliInternal
	cliHelp
)

type cliInvocation struct {
	kind      cliKind
	path      string
	ref       string
	operation string
	args      []string
	// layout is the couch-wide pair layout to launch threads in. Set only on
	// cliLaunch -- it is a property of the session being started, so the
	// read-only forms reject the flag rather than carrying a meaningless value.
	layout couchcore.Layout
}

// extractLayoutFlag strips layout flags before any shape check, mirroring
// launcher.ParseArgs, which runs extractLayoutRequest first (args.go:51) so its
// positional guard never sees them. Doing it here is what lets `--layout3`
// compose with a path on either side without "path cannot be combined with
// other arguments" firing.
func extractLayoutFlag(args []string) (rest []string, layout couchcore.Layout, given bool, err error) {
	rest = make([]string, 0, len(args))
	for i, arg := range args {
		// Everything after `--` is a path the operator quoted deliberately.
		if arg == "--" {
			rest = append(rest, args[i:]...)
			break
		}
		parsed, parseErr := couchcore.ParseLayout(strings.TrimPrefix(arg, "--"))
		if !strings.HasPrefix(arg, "--layout") || parseErr != nil {
			rest = append(rest, arg)
			continue
		}
		if given && parsed != layout {
			return nil, "", false, fmt.Errorf("couch takes one layout, not both %s and %s", layout, parsed)
		}
		layout, given = parsed, true
	}
	return rest, layout, given, nil
}

// ParseCLI classifies the complete public argv vector without performing IO.
// Registry presentation is the only authority for the hidden process boundary.
func ParseCLI(args []string, operations []couchcore.Operation) (cliInvocation, error) {
	invalid := func(format string, values ...any) (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf(format, values...)
	}
	args, layout, layoutGiven, err := extractLayoutFlag(args)
	if err != nil {
		return cliInvocation{}, err
	}
	if layout == "" {
		layout = couchcore.Layout2
	}
	// The read-only forms below reject the flag; only a launch carries it.
	refuseLayout := func(form string) error {
		if layoutGiven {
			return fmt.Errorf("%s does not take a layout: layout applies to the couch session being started", form)
		}
		return nil
	}
	if len(args) == 0 {
		return cliInvocation{kind: cliLaunch, path: ".", layout: layout}, nil
	}
	switch args[0] {
	case "-h", "--help":
		if len(args) != 1 {
			return invalid("%s cannot be combined with other arguments", args[0])
		}
		if err := refuseLayout(args[0]); err != nil {
			return cliInvocation{}, err
		}
		return cliInvocation{kind: cliHelp}, nil
	case "--list":
		if len(args) != 1 {
			return invalid("--list takes no arguments")
		}
		if err := refuseLayout("--list"); err != nil {
			return cliInvocation{}, err
		}
		return cliInvocation{kind: cliList}, nil
	case "--archived":
		if len(args) != 1 {
			return invalid("--archived takes no arguments")
		}
		if err := refuseLayout("--archived"); err != nil {
			return cliInvocation{}, err
		}
		return cliInvocation{kind: cliArchived}, nil
	case "--show":
		if len(args) != 2 || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return invalid("--show requires exactly one non-empty reference")
		}
		if err := refuseLayout("--show"); err != nil {
			return cliInvocation{}, err
		}
		return cliInvocation{kind: cliShow, ref: args[1]}, nil
	case "--":
		if len(args) != 2 || args[1] == "" {
			return invalid("-- requires exactly one non-empty path")
		}
		return cliInvocation{kind: cliLaunch, path: args[1], layout: layout}, nil
	case "--internal":
		if len(args) < 2 || args[1] == "" {
			return invalid("--internal requires an operation")
		}
		for _, arg := range args[2:] {
			if arg == "--" {
				return invalid("-- is not valid within internal arguments")
			}
		}
		for _, operation := range operations {
			if operation.Name == args[1] && operation.Presentation == couchcore.PresentationInternal {
				if err := refuseLayout("--internal"); err != nil {
					return cliInvocation{}, err
				}
				return cliInvocation{kind: cliInternal, operation: operation.Name, args: append([]string(nil), args[2:]...)}, nil
			}
		}
		return invalid("unknown internal operation %q", args[1])
	default:
		if args[0] == "" {
			return invalid("path must not be empty")
		}
		if strings.HasPrefix(args[0], "-") {
			return invalid("unknown option %q", args[0])
		}
		if len(args) != 1 {
			return invalid("path cannot be combined with other arguments")
		}
		return cliInvocation{kind: cliLaunch, path: args[0], layout: layout}, nil
	}
}
