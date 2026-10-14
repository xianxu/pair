package couchcmd

import (
	"context"
	"fmt"
	"io"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func parseNotifyCLI(args []string) (cliInvocation, error) {
	title := "Couch"
	var body string
	switch {
	case len(args) == 2 && args[1] != "--title":
		body = args[1]
	case len(args) == 4 && args[1] == "--title":
		title, body = args[2], args[3]
	default:
		return cliInvocation{}, fmt.Errorf("use couch --notify [--title T] MESSAGE")
	}
	if err := notifyosc.ValidateDesktop(title, body); err != nil {
		return cliInvocation{}, err
	}
	return cliInvocation{kind: cliMessage, messageOp: "notify", notifyTitle: title, messageBody: body}, nil
}

func runNotifyCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		fmt.Fprintln(stderr, "couch: notify requires a live Couch hub; send to the hub instead")
		return 1
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	r := callerIdentity(rt, "notify")
	r.Title, r.Body = inv.notifyTitle, inv.messageBody
	if err := couchmessage.ValidateRequest(r); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, r, &result); err != nil {
		fmt.Fprintln(stderr, "couch: notification outcome unconfirmed; do not retry automatically:", err)
		return 1
	}
	if result.Code == "uncertain" {
		fmt.Fprintln(stderr, "couch: notification outcome unconfirmed; do not retry automatically:", result.Error)
		return 1
	}
	// The pre-notify supervisor rejects Title during strict decoding, before
	// admission. Unlike a lost reply, this proves the new operation is unsupported.
	if result.Code == "invalid-request" && result.Error == `json: unknown field "Title"` {
		fmt.Fprintln(stderr, "couch: running Couch does not support --notify; restart Couch to load the new command")
		return 1
	}
	if result.Code != "ok" {
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
		return 1
	}
	fmt.Fprintln(stdout, "Notification written to the operator terminal.")
	return 0
}
