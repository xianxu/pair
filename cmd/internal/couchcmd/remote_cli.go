package couchcmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func parseRemoteCLI(args []string) (cliInvocation, error) {
	if len(args) != 2 || !couchmessage.IsRemoteOp("remote-"+args[1]) {
		return cliInvocation{}, fmt.Errorf("use couch --remote on|off|status; reset requires a keypress in Couch's own console")
	}
	return cliInvocation{kind: cliMessage, messageOp: "remote-" + args[1]}, nil
}
func runRemoteCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	namespace := rt.Getenv("COUCH_STORE_DIR")
	if namespace == "" {
		namespace = rt.StoreDir()
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, couchmessage.Request{Op: inv.messageOp}, &result); err != nil {
		fmt.Fprintln(stderr, "couch: remote action unconfirmed; read couch --remote status before another action:", err)
		return 1
	}
	if predatesOp(result) {
		fmt.Fprintln(stderr, "couch: running Couch predates --remote; restart Couch to use it")
		return 1
	}
	if result.Code != "ok" || result.Remote == nil {
		fmt.Fprintf(stderr, "couch: remote action refused: %s\n", result.Error)
		return 1
	}
	// Only the secret-free status projection is printed: no raw response/link.
	fmt.Fprintf(stdout, "Remote %s: %s\n", strings.TrimPrefix(inv.messageOp, "remote-"), result.Remote.Status)
	if result.Remote.Detail != "" {
		fmt.Fprintln(stdout, result.Remote.Detail)
	}
	return 0
}
