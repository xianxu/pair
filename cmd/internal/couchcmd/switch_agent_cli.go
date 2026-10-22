package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func parseSwitchAgentCLI(args []string) (cliInvocation, error) {
	bad := func() (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf("invalid %s form; use couch --help", args[0])
	}
	if len(args) < 2 || args[1] == "" || strings.HasPrefix(args[1], "-") {
		return bad()
	}
	inv := cliInvocation{kind: cliMessage, messageOp: strings.TrimPrefix(args[0], "--"), ref: args[1]}
	if inv.messageOp == "operation-status" {
		if len(args) != 2 && !(len(args) == 3 && args[2] == "--json") {
			return bad()
		}
		inv.jsonOutput = len(args) == 3
		// Use the protocol's one ID grammar without inventing another parser.
		if couchmessage.ValidateRequest(couchmessage.Request{Op: "operation-status", Scope: "s", Tag: "t", Session: "s", Nonce: "n", ID: inv.ref}) != nil {
			return bad()
		}
		return inv, nil
	}
	r, ok, err := couchcore.ParseWorkspaceReference(inv.ref)
	if err != nil || !ok || r.Repo == "" {
		return bad()
	}

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
		case "--agent":
			if i+1 == len(args) || args[i+1] == "" {
				return bad()
			}
			i++
			inv.messageAgent = args[i]

		default:
			return bad()
		}
	}
	if !inv.confirmed || !launcher.IsSupportedAgent(inv.messageAgent) {
		return bad()
	}

	return inv, nil
}

// runSwitchAgentCLI admits one remote switch and leaves status on the existing receipt path.
func runSwitchAgentCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	fail := func(err error) int { fmt.Fprintln(stderr, "couch:", err); return 1 }
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		return fail(fmt.Errorf("--%s requires a live Couch slot", inv.messageOp))
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return fail(err)
	}
	r := callerIdentity(rt, inv.messageOp)
	r.ID = inv.ref
	if r.Op == "switch-agent" {
		r.ID, err = newRequestID()
		if err != nil {
			return fail(err)
		}
		r.Target, r.Agent, r.Confirmed = inv.ref, inv.messageAgent, inv.confirmed
	}
	if err = couchmessage.ValidateRequest(r); err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err = call(ctx, socket, r, &result); err != nil {
		result = couchmessage.Response{Code: "uncertain", Error: err.Error()}
	}
	if result.Operation == nil && (result.Code == "uncertain" || result.Code == "accepted" || result.Code == "ok") {
		result.Code = "uncertain"
		result.Operation = &couchmessage.OperationReceipt{ID: r.ID, Op: r.Op, Target: r.Target, Status: couchmessage.ReceiptUnknown, Detail: result.Error}
	}
	success := (result.Code == "accepted" || result.Code == "ok") && result.Operation != nil
	if result.Operation != nil && (result.Operation.Status == couchmessage.ReceiptUnknown || result.Operation.Status == couchmessage.ReceiptFailed || result.Operation.Status == couchmessage.ReceiptRefused) {
		success = false
	}
	if inv.jsonOutput {
		if err = json.NewEncoder(stdout).Encode(result); err != nil {
			return fail(err)
		}
	} else if result.Operation != nil {
		if result.Operation.Drain != nil {
			printDrainOperation(stdout, *result.Operation)
		} else {
			fmt.Fprintf(stdout, "%s %s: %s\n", result.Operation.ID, result.Operation.Op, result.Operation.Status)
		}
		if result.Operation.Detail != "" {
			fmt.Fprintln(stdout, result.Operation.Detail)
		}
		if result.Operation.Drive != nil {
			for _, line := range result.Operation.Drive.Tail {
				fmt.Fprintln(stdout, line)
			}
		}
	} else {
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
	}
	id := r.ID
	if result.Operation != nil && result.Operation.ID != "" {
		id = result.Operation.ID
	}
	if !inv.jsonOutput || !success {
		fmt.Fprintf(stderr, "request %s; check couch --operation-status %s; do not automatically resend\n", id, id)
	}
	if success {
		return 0
	}
	return 1
}
