package couchcmd

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func parseAgentDriveCLI(args []string) (cliInvocation, error) {
	bad := func() (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf("invalid %s form; use couch --help", args[0])
	}
	if len(args) < 3 {
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
