package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type messageCall func(context.Context, string, any, any) error

func runMessageCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer) int {
	return runMessageCLIWithCall(inv, rt, stdout, stderr, couchmessage.Call)
}

func runMessageCLIWithCall(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	// The one verb list (pair#424): a hand-kept list here let relaunch and
	// reload-context fall through to the message path, which sends no ID.
	if couchcore.IsSlotOperation(inv.messageOp) {
		return runSlotOperationCLI(inv, rt, stdout, stderr, call, slotPollClock{now: time.Now, sleep: time.Sleep})
	}
	if inv.messageOp == "broadcast-status" {
		return runBroadcastListCLI(inv, rt, stdout, stderr, call)
	}
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		fmt.Fprintln(stderr, "couch: messaging requires a live Couch slot")
		return 1
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	r := callerIdentity(rt, inv.messageOp)
	switch inv.messageOp {
	case "send":
		if r.ID, err = newRequestID(); err != nil {
			fmt.Fprintln(stderr, "couch: create message ID:", err)
			return 1
		}
		r.Target = inv.ref
		r.Agent = inv.messageAgent
		r.Body = inv.messageBody
	case "status":
		r.ID = inv.ref
	}
	if err := couchmessage.ValidateRequest(r); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, r, &result); err != nil {
		if r.Op == "send" {
			printUncertainMessage(stderr, r.ID, err.Error())
		} else {
			fmt.Fprintln(stderr, "couch:", err)
		}
		return 1
	}
	if result.Code == "uncertain" && r.Op == "send" {
		printUncertainMessage(stderr, r.ID, result.Error)
		return 1
	}
	success := result.Code == "ok" || result.Code == "accepted" || result.Code == "not-dispatched"
	if inv.jsonOutput {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
	} else if !success {
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
	} else if result.Code == "not-dispatched" {
		fmt.Fprintf(stdout, "not-dispatched: %s\n", result.Error)
	} else if r.Op == "actors" {
		if len(result.Actors) == 0 {
			fmt.Fprintln(stdout, "No live message-capable slots.")
		}
		now := time.Now()
		for _, a := range result.Actors {
			slot := a.Binding.Slot
			if a.Alias != "" {
				slot += " (" + a.Alias + ")"
			}
			fmt.Fprintf(stdout, "%s\t%s\t%s\tallowance=%d\n", slot, a.Binding.Agent, messageActorAvailability(a, now), a.Remaining)
		}
	} else if result.Receipt != nil {
		receipt := result.Receipt
		fmt.Fprintf(stdout, "%s %s -> %s %s\n", receipt.Message.ID, receipt.Message.From.Slot, receipt.Message.To.Slot, receipt.Status)
		if receipt.Detail != "" {
			fmt.Fprintln(stdout, receipt.Detail)
		}
		if r.Op == "status" {
			fmt.Fprintln(stdout, receipt.Message.Body)
		}
	} else {
		fmt.Fprintln(stderr, "couch: broker returned no message receipt")
		return 1
	}
	if !success {
		return 1
	}
	return 0
}

// callerIdentity is a request from this slot's conversation: the identity
// the broker resolves to a live, registered binding.
func callerIdentity(rt Runtime, op string) couchmessage.Request {
	return couchmessage.Request{Op: op, Scope: rt.Getenv("COUCH_THREAD_SCOPE"), Tag: rt.Getenv("COUCH_THREAD_TAG"), Session: rt.Getenv("PAIR_SESSION_NAME"), Nonce: rt.Getenv("PAIR_LAUNCH_NONCE")}
}

// newRequestID is a random (version 4) UUID naming one request.
func newRequestID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:]), nil
}

// Slot operations admit, then poll: a resume can outlast the socket's 2 s
// exchange, so the CLI polls the receipt within this budget (pair#367 M2).
const (
	slotOperationPollInterval = 500 * time.Millisecond
	slotOperationPollBudget   = 3 * time.Minute
)

// slotPollClock is the poll loop's time: the budget is measured on now and
// waits use sleep (tests advance both). Each exchange still runs on a real
// per-call context.
type slotPollClock struct {
	now   func() time.Time
	sleep func(time.Duration)
}

// runSlotOperationCLI admits resume/reboot <slot> through the running Couch
// and polls its receipt. Each call gets a fresh AdmissionTimeout context; the
// 3-minute budget is the outer deadline. After admission any lost or
// unreadable outcome (dial failure, timeout, an unavailable caller after a
// Couch restart, a receipt no longer held) is uncertain: the report, not the
// receipt, is the evidence.
func runSlotOperationCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall, clock slotPollClock) int {
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		fmt.Fprintf(stderr, "couch: --%s requires a live Couch slot\n", inv.messageOp)
		return 1
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	admit := callerIdentity(rt, inv.messageOp)
	admit.Target, admit.Confirmed = inv.ref, inv.confirmed
	admit.SameBinary, admit.ForceUnknown = inv.sameBinary, inv.forceUnknown
	if admit.ID, err = newRequestID(); err != nil {
		fmt.Fprintln(stderr, "couch: create request ID:", err)
		return 1
	}
	if err := couchmessage.ValidateRequest(admit); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	exchange := func(r couchmessage.Request) (couchmessage.Response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
		defer cancel()
		var result couchmessage.Response
		return result, call(ctx, socket, r, &result)
	}
	uncertain := func(detail string) int {
		printUncertainSlotOperation(stderr, admit, detail)
		return 1
	}
	result, err := exchange(admit)
	switch {
	case err != nil:
		return uncertain(err.Error())
	case result.Code == "uncertain":
		return uncertain(result.Error)
	case result.Code == "invalid-request" && (strings.Contains(result.Error, "unknown message operation") || strings.Contains(result.Error, "unknown field")):
		// The running Couch is older than this CLI: it has no slot operations.
		fmt.Fprintf(stderr, "couch: the running Couch predates `couch --%s`; restart Couch (switcher Alt+d, then `couch`) on the new binary to use it\n", inv.messageOp)
		return 1
	case result.Code != "accepted" || result.Operation == nil:
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
		return 1
	}
	receipt := *result.Operation
	status := callerIdentity(rt, "operation-status")
	status.ID = admit.ID
	for start := clock.now(); !receipt.Status.Terminal(); {
		if clock.now().Sub(start) >= slotOperationPollBudget {
			return uncertain(fmt.Sprintf("still %s after %s", receipt.Status, slotOperationPollBudget))
		}
		clock.sleep(slotOperationPollInterval)
		result, err := exchange(status)
		switch {
		case err != nil:
			return uncertain(err.Error())
		case result.Code != "ok" || result.Operation == nil:
			return uncertain(result.Code + ": " + result.Error)
		case result.Operation.Status == couchmessage.ReceiptUnknown:
			return uncertain("Couch no longer holds this request (it may have restarted)")
		}
		receipt = *result.Operation
	}
	return reportSlotOperation(stdout, stderr, receipt, inv.jsonOutput)
}

// reportSlotOperation prints a terminal receipt: JSON, or one line naming the
// outcome. Only success exits 0.
func reportSlotOperation(stdout, stderr io.Writer, receipt couchmessage.OperationReceipt, asJSON bool) int {
	code := 1
	if receipt.Status == couchmessage.ReceiptSucceeded {
		code = 0
	}
	if asJSON {
		if err := json.NewEncoder(stdout).Encode(receipt); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		return code
	}
	switch receipt.Status {
	case couchmessage.ReceiptSucceeded:
		line := receipt.Target + " " + receipt.Op + " succeeded"
		if receipt.Tag != "" {
			line += " (tag " + receipt.Tag + ")"
		}
		if receipt.Archived != "" {
			line += "; archived " + receipt.Archived
		}
		fmt.Fprintln(stdout, line)
		if receipt.Warning != "" {
			fmt.Fprintln(stdout, receipt.Warning)
		}
	case couchmessage.ReceiptRefused:
		fmt.Fprintf(stderr, "couch: %s %s refused (%s): %s\n", receipt.Target, receipt.Op, receipt.Code, receipt.Detail)
	default:
		code := ""
		if receipt.Diagnostic != "" {
			code = " (" + receipt.Diagnostic + ")"
		}
		fmt.Fprintf(stderr, "couch: %s %s failed%s: %s\n", receipt.Target, receipt.Op, code, receipt.Detail)
	}
	return code
}

// printUncertainSlotOperation is the one text for a slot operation whose
// outcome is unknown: verify with the report before any resend.
func printUncertainSlotOperation(w io.Writer, r couchmessage.Request, detail string) {
	fmt.Fprintf(w, "couch: %s %s (request %s): %s\noutcome uncertain; verify with couch --recover-plan-from-sdlc before resending (a resend is refused harmlessly once the slot is live)\n", r.Op, r.Target, r.ID, detail)
}

func printUncertainMessage(w io.Writer, id, detail string) {
	fmt.Fprintf(w, "couch: message %s outcome uncertain: %s\nQuery couch --message-status %s before any further send; do not automatically retry.\n", id, detail, id)
}

func messageActorAvailability(a couchmessage.Candidate, now time.Time) string {
	switch {
	case !a.Supported:
		return "unsupported"
	case a.Pending:
		return "pending"
	case a.Remaining <= 0:
		return "breaker-exhausted"
	case !a.Known:
		return "unknown"
	case !a.Resting:
		return "occupied"
	case a.LastActivity.IsZero() || now.Before(a.LastActivity.Add(couchmessage.QuietInterval)):
		return "active"
	default:
		return "available"
	}
}
