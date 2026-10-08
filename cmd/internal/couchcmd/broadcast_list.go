package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// runBroadcastListCLI answers `couch --broadcast-list [--json]` (pair#413):
// the running couch's broadcast and its viewer count, from any shell. The
// broker socket is found through the store, not a slot's identity, and the
// answer never carries the link or its token.
func runBroadcastListCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	socket, err := couchmessage.SocketPath(rt.StoreDir(), "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, couchmessage.Request{Op: "broadcast-status"}, &result); err != nil {
		fmt.Fprintf(stderr, "couch: no running couch answered (%v)\n", err)
		return 1
	}
	switch {
	case result.Code == "invalid-request" && strings.Contains(result.Error, "unknown message operation"),
		result.Code == "unsupported":
		fmt.Fprintln(stderr, "couch: the running Couch predates `couch --broadcast-list`; restart Couch to use it")
		return 1
	case result.Code != "ok":
		fmt.Fprintf(stderr, "couch: %s: %s\n", result.Code, result.Error)
		return 1
	}
	if inv.jsonOutput {
		if err := json.NewEncoder(stdout).Encode(struct {
			Broadcast *couchmessage.BroadcastStatus `json:"broadcast"`
		}{result.Broadcast}); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintln(stdout, formatBroadcastStatus(result.Broadcast))
	return 0
}

// formatBroadcastStatus is the one-line listing; nil means no broadcast.
func formatBroadcastStatus(status *couchmessage.BroadcastStatus) string {
	if status == nil {
		return "no broadcast"
	}
	viewers := fmt.Sprintf("%d viewers", status.Viewers)
	if status.Viewers == 1 {
		viewers = "1 viewer"
	}
	if status.StartedAt.IsZero() {
		return fmt.Sprintf("%s — %s", status.State, viewers)
	}
	return fmt.Sprintf("%s since %s (%s) — %s", status.State, status.StartedAt.Local().Format("15:04:05"), status.Mode, viewers)
}
