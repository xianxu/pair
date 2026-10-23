package couchcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func operationIntentDigest(r couchmessage.Request) string {
	if r.Op == "switch-agent" {
		raw, _ := json.Marshal(struct {
			Arguments      couchmessage.SwitchArguments `json:",omitzero"`
			InterruptFirst bool
		}{r.AgentArgs, r.InterruptFirst})
		digest := sha256.Sum256(raw)
		return hex.EncodeToString(digest[:])
	}
	if r.Drive == nil {
		return ""
	}
	return r.Drive.IntentDigest()
}

// Admission is sent exactly once. Every subsequent exchange only reads its receipt.
func runAgentDriveCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
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
	r.Target, r.Drive, r.Confirmed = inv.ref, inv.drive, inv.confirmed
	r.ID, err = newRequestID()
	if err != nil {
		return fail(err)
	}
	if err = couchmessage.ValidateRequest(r); err != nil {
		return fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 36*time.Second)
	defer cancel()
	var response couchmessage.Response
	exchange := func(request couchmessage.Request) error {
		rpc, stop := context.WithTimeout(ctx, couchmessage.AdmissionTimeout)
		defer stop()
		response = couchmessage.Response{}
		return call(rpc, socket, request, &response)
	}
	uncertain := func(err error) int {
		fmt.Fprintf(stderr, "request %s: outcome unconfirmed (%v); check couch --operation-status %s and couch --peek %s before sending anything else; do not resend automatically\n", r.ID, err, r.ID, r.Target)
		return 1
	}
	if err = exchange(r); err != nil {
		return uncertain(err)
	}
	if response.Operation == nil {
		if inv.jsonOutput {
			_ = json.NewEncoder(stdout).Encode(response)
		}
		return fail(fmt.Errorf("%s: %s; relaunch Couch and the slot on the new build if unsupported", response.Code, response.Error))
	}
	for !response.Operation.Status.Terminal() {
		if response.Operation.Status == couchmessage.ReceiptUnknown {
			return uncertain(fmt.Errorf("receipt unavailable"))
		}
		select {
		case <-ctx.Done():
			return uncertain(ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
		status := callerIdentity(rt, "operation-status")
		status.ID = r.ID
		if err = exchange(status); err != nil {
			return uncertain(err)
		}
		if response.Operation == nil {
			return uncertain(fmt.Errorf("missing receipt"))
		}
	}
	receipt := response.Operation
	if inv.jsonOutput {
		if err = json.NewEncoder(stdout).Encode(response); err != nil {
			return fail(err)
		}
	} else {
		fmt.Fprintf(stdout, "%s %s: %s\n", receipt.ID, receipt.Op, receipt.Status)
		if receipt.Detail != "" {
			fmt.Fprintln(stdout, receipt.Detail)
		}
		if receipt.Drive != nil {
			for _, line := range receipt.Drive.Tail {
				fmt.Fprintln(stdout, line)
			}
		}
	}
	if receipt.Code == "unconfirmed" {
		return uncertain(fmt.Errorf("%s", receipt.Detail))
	}
	if receipt.Status != couchmessage.ReceiptSucceeded {
		return 1
	}
	return 0
}
