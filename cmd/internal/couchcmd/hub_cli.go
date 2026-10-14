package couchcmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func parseHubLaunchCLI(paths []string, layout couchcore.Layout, quoted bool) (cliInvocation, error) {
	if len(paths) == 0 || len(paths) > couchcore.MaxHubSlots {
		return cliInvocation{}, fmt.Errorf("expected 1..%d hub directories", couchcore.MaxHubSlots)
	}
	for _, p := range paths {
		if p == "" || !quoted && strings.HasPrefix(p, "-") {
			return cliInvocation{}, fmt.Errorf("invalid hub directory %q", p)
		}
	}
	return cliInvocation{kind: cliLaunch, path: paths[0], hubPaths: slices.Clone(paths), layout: layout}, nil
}
func parseHubMoveCLI(args []string) (cliInvocation, error) {
	bad := func() (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf("use --move-slot repo:N[,repo:N...] --to-hub HUB [--persist]")
	}
	if len(args) != 4 && len(args) != 5 || args[2] != "--to-hub" || strings.TrimSpace(args[3]) == "" || strings.HasPrefix(args[3], "-") {
		return bad()
	}
	if len(args) == 5 && args[4] != "--persist" {
		return bad()
	}
	slots := strings.Split(args[1], ",")
	if len(slots) > couchcore.MaxHubSlots {
		return bad()
	}
	for _, s := range slots {
		ref, ok, err := couchcore.ParseWorkspaceReference(s)
		if err != nil || !ok || ref.Repo == "" {
			return bad()
		}
	}
	return cliInvocation{kind: cliMessage, messageOp: "move-slot", ref: args[1], toHub: args[3], persistHub: len(args) == 5}, nil
}
