package couchcmd

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type shortcutFocusArtifacts interface {
	PairSessionContext(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error)
	PairLifecycleDataDir() string
}

// Query only the exact thread's client focus, not list-panes' per-client union
// of is_focused flags. More than one client is ambiguous until Zellij exposes
// the presenting client's identity; report that rather than choose arbitrarily.
func focusedClientPane(raw []byte) (string, error) {
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 || strings.Join(strings.Fields(lines[0]), " ") != "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND" {
		return "", fmt.Errorf("expected exactly one attached Zellij client; use Couch's switcher to relaunch")
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return "", fmt.Errorf("invalid Zellij client row")
	}
	if _, err := strconv.ParseUint(fields[0], 10, 64); err != nil {
		return "", fmt.Errorf("invalid Zellij client id")
	}
	kind, id, ok := strings.Cut(fields[1], "_")
	if _, err := strconv.ParseUint(id, 10, 64); !ok || err != nil || (kind != "terminal" && kind != "plugin") {
		return "", fmt.Errorf("invalid focused Zellij pane")
	}
	if kind == "plugin" {
		return "", nil
	}
	return id, nil
}

func queryShortcutClients(ctx context.Context, session string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "zellij", "--session", session, "action", "list-clients")
	cmd.WaitDelay = 100 * time.Millisecond
	return cmd.Output()
}

func rightTerminalFocusProbe(artifacts shortcutFocusArtifacts, query func(context.Context, string) ([]byte, error)) func(context.Context, couchcore.ThreadAddress) (bool, error) {
	return func(ctx context.Context, address couchcore.ThreadAddress) (bool, error) {
		if artifacts == nil {
			return false, fmt.Errorf("Pair session focus observer unavailable")
		}
		binding, err := artifacts.PairSessionContext(ctx, address)
		if err != nil {
			return false, err
		}
		if !binding.Present || binding.Name == "" {
			return false, fmt.Errorf("Pair session binding unavailable")
		}
		raw, err := query(ctx, binding.Name)
		if err != nil {
			return false, err
		}
		id, err := focusedClientPane(raw)
		if err != nil {
			return false, err
		}
		if id == "" { // A positively identified plugin cannot be a right terminal.
			return false, ctx.Err()
		}
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: artifacts.PairLifecycleDataDir(), RepoScope: address.RepoScope, Tag: string(address.Tag)})
		if err != nil {
			return false, err
		}
		ids, err := (workbenchshortcut.TerminalPaneRegistry{DataDir: paths.ScopeDir(), Tag: string(address.Tag)}).LiveIDs(func(pid int) bool { return procutil.Alive(strconv.Itoa(pid)) })
		if err != nil {
			return false, err
		}
		if workbenchshortcut.Registered(ids, id) {
			return true, ctx.Err()
		}
		// Absence from a best-effort registry is not proof of another role:
		// registration can fail, and LiveIDs skips malformed/dead entries.
		line := strings.Split(strings.TrimSpace(string(raw)), "\n")[1]
		fields := strings.Fields(line)
		role := workbenchshortcut.RoleForPane(zellijpane.Pane{TerminalCommand: strings.Join(fields[2:], " ")})
		if role == workbenchshortcut.PaneRoleLeftDraft || role == workbenchshortcut.PaneRoleLeftAgent {
			return false, ctx.Err()
		}
		return false, fmt.Errorf("focused pane has no confirmed Pair role; use Couch's switcher to relaunch")
	}
}
