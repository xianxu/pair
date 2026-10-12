package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/rowtext"
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
	// A couch older than this command refuses the identity-free request with
	// its generic identity check (or, older still, the unknown-op error).
	case result.Code == "invalid-request" && (strings.Contains(result.Error, "unknown message operation") ||
		strings.Contains(result.Error, "requires the calling conversation identity")),
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
			Shares    []couchmessage.ShareStatus    `json:"shares"`
		}{result.Broadcast, result.Shares}); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		return 0
	}
	fmt.Fprintln(stdout, formatShares(result.Broadcast, result.Shares))
	return 0
}

// formatShares is one line per link out, by kind (pair#467): the screen
// broadcast as before, then the web cast's view and drive links. A Couch
// that reports no shares lists its broadcast alone.
func formatShares(screen *couchmessage.BroadcastStatus, shares []couchmessage.ShareStatus) string {
	if len(shares) == 0 {
		return formatBroadcastStatus(screen)
	}
	var lines []string
	for _, s := range shares {
		if s.Kind == couchmessage.ShareScreen && screen != nil {
			lines = append(lines, s.Kind+": "+formatBroadcastStatus(screen))
			continue
		}
		line := fmt.Sprintf("%s: %s — %s", s.Kind, s.State, viewerCount(s.Viewers))
		if s.Kind == couchmessage.ShareWebDrive {
			if s.Driver != "" {
				line += "; " + rowtext.Sanitize(s.Driver) + " driving"
			}
			if s.Asking > 0 {
				line += fmt.Sprintf("; %d asking", s.Asking)
			}
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func viewerCount(n int) string {
	if n == 1 {
		return "1 viewer"
	}
	return fmt.Sprintf("%d viewers", n)
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
