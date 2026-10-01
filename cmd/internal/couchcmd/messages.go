package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type messageCall func(context.Context, string, any, any) error

func runMessageCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer) int {
	return runMessageCLIWithCall(inv, rt, stdout, stderr, couchmessage.Call)
}

func runMessageCLIWithCall(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
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
	r := couchmessage.Request{Op: inv.messageOp, Scope: rt.Getenv("COUCH_THREAD_SCOPE"), Tag: rt.Getenv("COUCH_THREAD_TAG"), Session: rt.Getenv("PAIR_SESSION_NAME"), Nonce: rt.Getenv("PAIR_LAUNCH_NONCE")}
	switch inv.messageOp {
	case "send":
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			fmt.Fprintln(stderr, "couch: create message ID:", err)
			return 1
		}
		bytes[6] = (bytes[6] & 0x0f) | 0x40
		bytes[8] = (bytes[8] & 0x3f) | 0x80
		r.ID = fmt.Sprintf("%x-%x-%x-%x-%x", bytes[:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:])
		r.Target = inv.ref
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
			fmt.Fprintf(stdout, "%s\t%s\tallowance=%d\n", a.Binding.Slot, messageActorAvailability(a, now), a.Remaining)
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
