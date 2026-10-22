package couchcmd

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"io"
)

func parseDrainCLI(args []string) (cliInvocation, error) {
	d := &couchmessage.DrainRequest{Action: "start"}
	if args[0] == "--undrain" {
		d.Action = "undrain"
	}
	inv := cliInvocation{kind: cliMessage, messageOp: "drain", drain: d}
	seen := map[string]bool{}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if seen[arg] {
			return cliInvocation{}, fmt.Errorf("duplicate drain option %s", arg)
		}
		seen[arg] = true
		switch arg {
		case "--json":
			inv.jsonOutput = true
		case "--fast":
			if args[0] != "--drain" {
				return cliInvocation{}, fmt.Errorf("--fast applies only to --drain")
			}
			d.Fast = true
		case "--status":
			if args[0] != "--drain" {
				return cliInvocation{}, fmt.Errorf("--status applies only to --drain")
			}
			d.Action = "status"
		case "--message":
			if args[0] != "--undrain" || i+1 >= len(args) {
				return cliInvocation{}, fmt.Errorf("--undrain --message requires text")
			}
			i++
			d.Message = args[i]
		default:
			return cliInvocation{}, fmt.Errorf("unsupported drain option %q; see couch --help", arg)
		}
	}
	if d.Action == "status" && d.Fast {
		return cliInvocation{}, fmt.Errorf("drain status cannot carry --fast")
	}
	if d.Action == "undrain" {
		if _, err := couchcore.UndrainNotice(d.Message); err != nil {
			return cliInvocation{}, err
		}
	}
	return inv, nil
}

// printDrainOperation reports delivery separately from requested posture.
func printDrainOperation(w io.Writer, receipt couchmessage.OperationReceipt) {
	fmt.Fprintf(w, "drain %s %s: %s (notice delivery; not checkpoint confirmation)\n", receipt.ID, receipt.Drain.Action, receipt.Status)
	for _, child := range receipt.Drain.Children {
		fmt.Fprintf(w, "%s: notice %s; child %s", notifyosc.Sanitize([]byte(child.Workspace)), couchmessage.DrainNoticeState(child), child.ID)
		if child.Detail != "" {
			fmt.Fprintf(w, "; %s", notifyosc.Sanitize([]byte(child.Detail)))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "inspect couch --operation-status %s; do not automatically resend\n", receipt.ID)
}
