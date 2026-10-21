package couchcmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
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
	// cliKeysHelp is `couch --keys --help`: the step table and recipes (#716).
	cliKeysHelp
	cliSessionSync
	// cliWebQuery is couch broadcast-web's agent queries (#462); args holds
	// the argv after the flag, which query.Run parses.
	cliWebQuery
	// cliWebReplay serves broadcast-web recordings for replay (#471); args
	// holds the argv after the flag.
	cliWebReplay
	// cliWebFidelity checks a broadcast-web recording against its frames
	// (#478); args holds the argv after the flag.
	cliWebFidelity
	cliSubmission
	cliSchedule
)

type adoptionArgs struct {
	StoreDir, PairDataDir, IdentityDir, Expect string
	Stores, Exclude                            []string
}

type cliInvocation struct {
	schedule   *scheduleArgs
	drive      *agentdrive.Request
	hubPaths   []string
	toHub      string
	persistHub bool
	// relaunch overrides (pair#421)
	sameBinary, forceUnknown bool
	adoption                 adoptionArgs
	kind                     cliKind
	path                     string
	ref                      string
	operation                string
	args                     []string
	messageOp                string
	messageBody              string
	notifyTitle              string
	// messageFile is --message-file's path (pair#486); the send reads the
	// body from it, so the parser stays free of IO.
	messageFile string
	// messageAgent narrows --send-to to slots running that agent.
	messageAgent string
	// confirmed is --confirm on a slot operation whose declaration requires
	// a confirmation (reboot).
	candidateKey         string
	recoveryConfirmation string
	confirmed            bool
	jsonOutput           bool
	// sessionSync is set on cliSessionSync (pair#437).
	sessionSync *sessionSyncArgs
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
		case "--schedule":
			return parseScheduleCLI(args)
		case "--keys", "--set-model", "--interrupt":
			return parseAgentDriveCLI(args)
		case "--move-slot":
			return parseHubMoveCLI(args)
		case "--notify":
			return parseNotifyCLI(args)
		case submissionFlag:
			return parseSubmissionCLI(args)
		case "--adopt-store":
			return parseAdoptionCLI(args)
		case query.Flag:
			return cliInvocation{kind: cliWebQuery, args: args[1:]}, nil
		case replayFlag:
			if _, err := parseReplayArgs(args[1:]); err != nil {
				return cliInvocation{}, fmt.Errorf("%s: %w", replayFlag, err)
			}
			return cliInvocation{kind: cliWebReplay, args: args[1:]}, nil
		case fidelityFlag:
			if _, err := parseFidelityArgs(args[1:]); err != nil {
				return cliInvocation{}, fmt.Errorf("%s: %w", fidelityFlag, err)
			}
			return cliInvocation{kind: cliWebFidelity, args: args[1:]}, nil
		case "--session-sync":
			return parseSessionSyncCLI(args)
		case "--switch-agent", "--operation-status":
			return parseSwitchAgentCLI(args)
		case "--recovery-confirmation", "--resume-candidates", "--actors", "--send-to", "--message-status", "--skill", "--broadcast-list", "--resources", "--reap-orphans":
			return parseMessageCLI(args)
		}
		// Slot operations come from the one verb list (couchcore.IsSlotOperation).
		if strings.HasPrefix(args[0], "--") && couchcore.IsSlotOperation(strings.TrimPrefix(args[0], "--")) {
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
		// couch --peek ref [--lines N] [--json] [--transcripts]: each option at most once.
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
				return invalid("--peek takes a slot reference, then optional --lines N, --json, --transcripts, --previous and --all, each once")
			}
			seen[flag] = true
			switch {
			case flag == "--json" || flag == "--transcripts" || flag == "--previous" || flag == "--all":
				inv.args = append(inv.args, flag)
			case flag == "--lines" && i+1 < len(args):
				i++
				inv.args = append(inv.args, "--lines="+args[i])
			default:
				return invalid("--peek takes a slot reference, then optional --lines N, --json, --transcripts, --previous and --all, each once")
			}
		}
		return inv, nil
	case "--":
		return parseHubLaunchCLI(args[1:], layout, true)
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
		return parseHubLaunchCLI(args, layout, false)
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
	case "--resources":
		if len(args) != 1 && !(len(args) == 2 && args[1] == "--json") {
			return cliInvocation{}, fmt.Errorf("--resources accepts only --json")
		}
		return cliInvocation{kind: cliMessage, messageOp: "resources", jsonOutput: len(args) == 2}, nil
	case "--reap-orphans":
		// No slot: the sweep this confirms is fleet-wide, so the two switches are
		// the whole surface, each at most once and in either order (#583).
		seen := map[string]bool{}
		for _, flag := range args[1:] {
			if (flag != "--confirm" && flag != "--json") || seen[flag] {
				return cliInvocation{}, fmt.Errorf("--reap-orphans accepts only --confirm and --json, once each")
			}
			seen[flag] = true
		}
		return cliInvocation{kind: cliMessage, messageOp: orphanReapOp, confirmed: seen["--confirm"], jsonOutput: seen["--json"]}, nil
	case "--broadcast-list":
		if len(args) != 1 && !(len(args) == 2 && args[1] == "--json") {
			return bad()
		}
		return cliInvocation{kind: cliMessage, messageOp: "broadcast-status", jsonOutput: len(args) == 2}, nil
	case "--message-status":
		if (len(args) != 2 && !(len(args) == 3 && args[2] == "--json")) || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return bad()
		}
		return cliInvocation{kind: cliMessage, messageOp: "status", ref: args[1], jsonOutput: len(args) == 3}, nil
	case "--recovery-confirmation":
		if (len(args) != 2 && !(len(args) == 3 && args[2] == "--json")) || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return bad()
		}
		return cliInvocation{kind: cliMessage, messageOp: "recovery-confirmation", recoveryConfirmation: args[1], jsonOutput: len(args) == 3}, nil
	case "--resume-candidates":
		if len(args) != 2 && !(len(args) == 3 && args[2] == "--json") {
			return bad()
		}
		if ref, ok, err := couchcore.ParseWorkspaceReference(args[1]); err != nil || !ok || ref.Repo == "" {
			return bad()
		}
		return cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: args[1], jsonOutput: len(args) == 3}, nil
	case slotOperationFlag(args[0]):
		// One exact slot, then --json and the declared --confirm, each once.
		// relaunch (pair#421) also takes its two overrides.
		op := strings.TrimPrefix(args[0], "--")
		if len(args) < 2 {
			return bad()
		}
		if ref, recognized, err := couchcore.ParseWorkspaceReference(args[1]); err != nil || !recognized || ref.Repo == "" {
			return cliInvocation{}, fmt.Errorf("%s requires one exact repo:N slot, not %q", args[0], args[1])
		}
		seen := map[string]bool{}
		allowed := map[string]bool{"--json": true, "--confirm": true}
		allowed["--same-binary"] = couchcore.SlotOperationTakesSameBinary(op)
		allowed["--force-unknown"] = couchcore.SlotOperationTakesForceUnknown(op)
		candidateKey := ""
		for i := 2; i < len(args); i++ {
			flag := args[i]
			if flag == "--candidate" && op == "resume" {
				if candidateKey != "" || i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "-") {
					return bad()
				}
				i++
				candidateKey = args[i]
				continue
			}
			if !allowed[flag] || seen[flag] {
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
		return cliInvocation{kind: cliMessage, messageOp: op, ref: args[1], confirmed: seen["--confirm"], jsonOutput: seen["--json"],
			sameBinary: seen["--same-binary"], forceUnknown: seen["--force-unknown"], candidateKey: candidateKey}, nil
	case "--send-to":
		agent := ""
		if len(args) == 6 && args[2] == "--agent" {
			agent = args[3]
			if agent == "" || strings.HasPrefix(agent, "-") {
				return bad()
			}
			args = append(args[:2:2], args[4:]...)
		}
		if len(args) != 4 || args[1] == "" || strings.HasPrefix(args[1], "-") {
			return bad()
		}
		inv := cliInvocation{kind: cliMessage, messageOp: "send", ref: args[1], messageAgent: agent}
		switch {
		case args[2] == "--message" && strings.TrimSpace(args[3]) != "":
			inv.messageBody = args[3]
		case args[2] == "--message-file" && args[3] != "":
			inv.messageFile = args[3]
		default:
			return bad()
		}
		return inv, nil
	}
	return bad()
}

// slotOperationFlag returns flag when it names a slot operation ("--relaunch"),
// so a switch on the flag can match the one verb list; "" never matches a flag.
func slotOperationFlag(flag string) string {
	if strings.HasPrefix(flag, "--") && couchcore.IsSlotOperation(strings.TrimPrefix(flag, "--")) {
		return flag
	}
	return "\x00not-a-slot-operation"
}
