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
}

// ParseCLI classifies the complete public argv vector without performing IO.
// Registry presentation is the only authority for the hidden process boundary.
func ParseCLI(args []string, operations []couchcore.Operation) (cliInvocation, error) {
	invalid := func(format string, values ...any) (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf(format, values...)
	}
	if len(args) == 0 {
		return cliInvocation{kind: cliLaunch, path: "."}, nil
	}
	switch args[0] {
	case "-h", "--help":
		if len(args) != 1 {
			return invalid("%s cannot be combined with other arguments", args[0])
		}
		return cliInvocation{kind: cliHelp}, nil
	case "--list":
		if len(args) != 1 {
			return invalid("--list takes no arguments")
		}
		return cliInvocation{kind: cliList}, nil
	case "--show":
		if len(args) != 2 || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return invalid("--show requires exactly one non-empty reference")
		}
		return cliInvocation{kind: cliShow, ref: args[1]}, nil
	case "--":
		if len(args) != 2 || args[1] == "" {
			return invalid("-- requires exactly one non-empty path")
		}
		return cliInvocation{kind: cliLaunch, path: args[1]}, nil
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
		return cliInvocation{kind: cliLaunch, path: args[0]}, nil
	}
}
