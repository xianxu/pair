package couchcmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type cliKind uint8

const (
	cliInvalid cliKind = iota
	cliLaunch
	cliList
	cliArchived
	cliRecoverPlan
	cliShow
	cliReconcile
	cliPeek
	cliInternal
	cliMessage
	cliSkill
	cliAdopt
	cliHelp
)

type adoptionArgs struct {
	StoreDir, PairDataDir, IdentityDir, Expect string
	Stores, Exclude                            []string
}

type cliInvocation struct {
	adoption    adoptionArgs
	kind        cliKind
	path        string
	ref         string
	operation   string
	args        []string
	messageOp   string
	messageBody string
	// messageAgent narrows --send-to to slots running that agent.
	messageAgent string
	// confirmed is --confirm on a slot operation whose declaration requires
	// a confirmation (reboot).
	confirmed  bool
	jsonOutput bool
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
	// Message bodies are opaque argv values. Parse these closed forms before
	// scanning layout flags, which could otherwise consume a literal body.
	if len(args) > 0 {
		switch args[0] {
		case "--adopt-store":
			return parseAdoptionCLI(args)
		case "--actors", "--send-to", "--message-status", "--skill", "--resume", "--reboot", "--reap", "--recover":
			return parseMessageCLI(args)
		}
	}
	invalid := func(format string, values ...any) (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf(format, values...)
	}
	args, layout, layoutGiven, err := extractLayoutFlag(args)
	if err != nil {
		return cliInvocation{}, err
	}
	if layout == "" {
		layout = couchcore.DefaultLayout
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
	case "--recover-plan-from-sdlc":
		if len(args) != 1 {
			return invalid("--recover-plan-from-sdlc takes no arguments")
		}
		if err := refuseLayout("--recover-plan-from-sdlc"); err != nil {
			return cliInvocation{}, err
		}
		return cliInvocation{kind: cliRecoverPlan}, nil
	case "--show":
		if len(args) != 2 || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return invalid("--show requires exactly one non-empty reference")
		}
		if err := refuseLayout("--show"); err != nil {
			return cliInvocation{}, err
		}
		return cliInvocation{kind: cliShow, ref: args[1]}, nil
	case "--reconcile":
		if len(args) != 2 || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return invalid("--reconcile requires exactly one slot reference (repo:N)")
		}
		if err := refuseLayout("--reconcile"); err != nil {
			return cliInvocation{}, err
		}
		return cliInvocation{kind: cliReconcile, ref: args[1]}, nil
	case "--peek":
		// couch --peek ref [--lines N] [--json]: each option at most once.
		if len(args) < 2 || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return invalid("--peek requires exactly one slot reference (repo:N)")
		}
		if err := refuseLayout("--peek"); err != nil {
			return cliInvocation{}, err
		}
		inv := cliInvocation{kind: cliPeek, ref: args[1]}
		seen := map[string]bool{}
		for i := 2; i < len(args); i++ {
			flag := args[i]
			if seen[flag] {
				return invalid("--peek takes a slot reference, then optional --lines N and --json, each once")
			}
			seen[flag] = true
			switch {
			case flag == "--json":
				inv.args = append(inv.args, "--json")
			case flag == "--lines" && i+1 < len(args):
				i++
				inv.args = append(inv.args, "--lines="+args[i])
			default:
				return invalid("--peek takes a slot reference, then optional --lines N and --json, each once")
			}
		}
		return inv, nil
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

func parseAdoptionCLI(args []string) (cliInvocation, error) {
	var adoption adoptionArgs
	seen := make(map[string]bool)
	for i := 0; i < len(args); i += 2 {
		flag := args[i]
		switch flag {
		case "--adopt-store", "--pair-data", "--identity-dir", "--legacy-store", "--exclude-store", "--apply":
		default:
			return cliInvocation{}, fmt.Errorf("unknown adoption option %q; use couch --help", flag)
		}
		if i+1 == len(args) {
			return cliInvocation{}, fmt.Errorf("%s requires a value", flag)
		}
		if flag != "--legacy-store" && flag != "--exclude-store" && seen[flag] {
			return cliInvocation{}, fmt.Errorf("%s may only be given once", flag)
		}
		seen[flag] = true
		value := args[i+1]
		if flag == "--apply" {
			if len(value) != 64 || strings.IndexFunc(value, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) >= 0 {
				return cliInvocation{}, fmt.Errorf("--apply requires the report's 64 lowercase hexadecimal digest")
			}
		} else if !filepath.IsAbs(value) || !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return cliInvocation{}, fmt.Errorf("%s requires an absolute path without control characters", flag)
		}
		switch flag {
		case "--adopt-store":
			adoption.StoreDir = value
		case "--pair-data":
			adoption.PairDataDir = value
		case "--identity-dir":
			adoption.IdentityDir = value
		case "--legacy-store":
			adoption.Stores = append(adoption.Stores, value)
		case "--exclude-store":
			adoption.Exclude = append(adoption.Exclude, value)
		case "--apply":
			adoption.Expect = value
		}
	}
	return cliInvocation{kind: cliAdopt, adoption: adoption}, nil
}

func parseMessageCLI(args []string) (cliInvocation, error) {
	bad := func() (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf("invalid %s form; use couch --help", args[0])
	}
	switch args[0] {
	case "--skill":
		if len(args) != 1 {
			return bad()
		}
		return cliInvocation{kind: cliSkill}, nil
	case "--actors":
		if len(args) != 1 && !(len(args) == 2 && args[1] == "--json") {
			return bad()
		}
		return cliInvocation{kind: cliMessage, messageOp: "actors", jsonOutput: len(args) == 2}, nil
	case "--message-status":
		if (len(args) != 2 && !(len(args) == 3 && args[2] == "--json")) || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return bad()
		}
		return cliInvocation{kind: cliMessage, messageOp: "status", ref: args[1], jsonOutput: len(args) == 3}, nil
	case "--resume", "--reboot", "--reap", "--recover":
		// One exact slot, then --json and the declared --confirm, each once.
		op := strings.TrimPrefix(args[0], "--")
		if len(args) < 2 {
			return bad()
		}
		if ref, recognized, err := couchcore.ParseWorkspaceReference(args[1]); err != nil || !recognized || ref.Repo == "" {
			return cliInvocation{}, fmt.Errorf("%s requires one exact repo:N slot, not %q", args[0], args[1])
		}
		seen := map[string]bool{}
		for _, flag := range args[2:] {
			if (flag != "--json" && flag != "--confirm") || seen[flag] {
				return bad()
			}
			seen[flag] = true
		}
		if confirms, _ := couchcore.OperationConfirms(op); seen["--confirm"] != confirms {
			if confirms {
				return cliInvocation{}, fmt.Errorf("%s requires --confirm", args[0])
			}
			return cliInvocation{}, fmt.Errorf("%s takes no --confirm", args[0])
		}
		return cliInvocation{kind: cliMessage, messageOp: op, ref: args[1], confirmed: seen["--confirm"], jsonOutput: seen["--json"]}, nil
	case "--send-to":
		agent := ""
		if len(args) == 6 && args[2] == "--agent" {
			agent = args[3]
			if agent == "" || strings.HasPrefix(agent, "-") {
				return bad()
			}
			args = append(args[:2:2], args[4:]...)
		}
		if len(args) != 4 || args[1] == "" || strings.HasPrefix(args[1], "-") || args[2] != "--message" || strings.TrimSpace(args[3]) == "" {
			return bad()
		}
		return cliInvocation{kind: cliMessage, messageOp: "send", ref: args[1], messageAgent: agent, messageBody: args[3]}, nil
	}
	return bad()
}
