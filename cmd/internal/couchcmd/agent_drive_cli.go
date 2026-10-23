package couchcmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func parseAgentDriveCLI(args []string) (cliInvocation, error) {
	bad := func() (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf("invalid %s form; use couch --help", args[0])
	}
	if args[0] == "--keys" && len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		return cliInvocation{kind: cliKeysHelp}, nil
	}
	if len(args) < 3 && (args[0] != "--interrupt" || len(args) < 2) {
		return bad()
	}
	ref, ok, err := couchcore.ParseWorkspaceReference(args[1])
	if err != nil || !ok || ref.Repo == "" {
		return bad()
	}
	inv := cliInvocation{kind: cliMessage, messageOp: strings.TrimPrefix(args[0], "--"), ref: args[1], drive: &agentdrive.Request{}}
	if inv.messageOp == "keys" {
		specs := args[2:]
		if specs[len(specs)-1] == "--json" {
			inv.jsonOutput = true
			specs = specs[:len(specs)-1]
		}
		inv.drive.Keys, err = agentdrive.ParseKeySequence(specs)
		if err != nil {
			return cliInvocation{}, err
		}
		return inv, nil
	}
	if inv.messageOp == "interrupt" {
		for _, flag := range args[2:] {
			switch {
			case flag == "--confirm" && !inv.confirmed:
				inv.confirmed = true
			case flag == "--json" && !inv.jsonOutput:
				inv.jsonOutput = true
			default:
				return bad()
			}
		}
		if !inv.confirmed {
			return cliInvocation{}, fmt.Errorf("--interrupt requires --confirm: it sends Ctrl+C into the slot's working turn")
		}
		inv.drive.Interrupt = true
		return inv, nil
	}
	model := &agentdrive.ModelRequest{Scope: agentdrive.ScopeSession}
	seen := map[string]bool{}
	for i := 2; i < len(args); i++ {
		flag := args[i]
		if seen[flag] {
			return bad()
		}
		seen[flag] = true
		switch flag {
		case "--confirm":
			inv.confirmed = true
		case "--json":
			inv.jsonOutput = true
		case "--model", "--effort", "--scope":
			if i+1 == len(args) {
				return bad()
			}
			i++
			switch flag {
			case "--model":
				model.Model = args[i]
			case "--effort":
				model.Effort = args[i]
			case "--scope":
				model.Scope = agentdrive.ModelScope(args[i])
			}
		default:
			return bad()
		}
	}
	if !inv.confirmed {
		return cliInvocation{}, fmt.Errorf("--set-model requires --confirm")
	}
	inv.drive.Model = model
	if err = inv.drive.Validate(); err != nil {
		return cliInvocation{}, err
	}
	return inv, nil
}

// keysUsage renders `couch --keys --help` from agentdrive.Steps and its limit
// constants, so the help cannot list a step the parser refuses (#716).
func keysUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: couch --keys repo:N STEP... [--json]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Hub only: types named keys into another slot's agent, which the hub must own.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Steps:")
	for _, step := range agentdrive.StepGroups() {
		fmt.Fprintf(w, "  %-14s %s\n", step.Syntax, step.Meaning)
	}
	fmt.Fprintln(w, "Not steps (refused before anything is sent):")
	refused := make([]string, 0, len(agentdrive.RefusedSteps))
	for name := range agentdrive.RefusedSteps {
		refused = append(refused, string(name))
	}
	sort.Strings(refused)
	for _, name := range refused {
		fmt.Fprintf(w, "  %-14s %s\n", name, agentdrive.RefusedSteps[agentdrive.KeyKind(name)])
	}
	fmt.Fprintf(w, "Limits: %d steps, %d bytes of text, waits of at most %d ms each and %d ms in total.\n",
		agentdrive.MaxKeySteps, agentdrive.MaxKeyTextBytes, agentdrive.MaxKeyWaitMS, agentdrive.MaxKeyTotalWaitMS)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Admission:")
	fmt.Fprintln(w, "  Any steps need a settled agent with an empty composer.")
	fmt.Fprintln(w, "  Only esc and clear-line steps may start on a recognized occupied composer.")
	fmt.Fprintln(w, "  A lone esc may close a recognized slash-command panel; every other request")
	fmt.Fprintln(w, "  refuses while one shows. The panel's own footer (Esc to cancel) is the proof.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Slash-command panel round trip (/usage, /status):")
	fmt.Fprintln(w, "  couch --keys pair:7 clear-line text:/usage enter   open it")
	fmt.Fprintln(w, "  couch --peek pair:7 --lines 40                      read it")
	fmt.Fprintln(w, "  couch --keys pair:7 esc                             close it; done when the empty composer returns")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "An unconfirmed result is never retried for you: peek at the slot before sending more input.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Interrupting a working turn is its own verb, never a step:")
	fmt.Fprintln(w, "  couch --interrupt pair:7 --confirm   one Ctrl+C, only while a turn is working; done when it ends")
}
