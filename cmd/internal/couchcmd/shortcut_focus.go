package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijexec"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

type shortcutFocusArtifacts interface {
	PairSessionContext(context.Context, couchcore.ThreadAddress) (couchcore.PairSessionBinding, error)
	PairLifecycleDataDir() string
	RevalidatePairSession(context.Context, couchcore.PairSessionBinding) error
}

func queryShortcutClients(ctx context.Context, session string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "zellij", "--session", session, "action", "list-clients")
	// A short WaitDelay fails complete output whose copy lags under load
	// (pair#613); the caller's context still bounds the query.
	cmd.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := zellijexec.Run(cmd)
	if exit, ok := err.(*exec.ExitError); ok {
		exit.Stderr = append([]byte(nil), stderr.Bytes()...)
	}
	return stdout.Bytes(), err
}

// Query only the exact thread's client focus, not list-panes' per-client union
// of is_focused flags. More than one client is ambiguous until Zellij exposes
// the presenting client's identity; report that rather than choose arbitrarily.
func shortcutFocusProbe(artifacts shortcutFocusArtifacts, query func(context.Context, string) ([]byte, error)) func(context.Context, couchcore.ThreadAddress) (workbenchshortcut.PaneRole, error) {
	return shortcutFocusProbeWithProcesses(artifacts, query, procutil.Source{})
}

func shortcutFocusProbeWithProcesses(artifacts shortcutFocusArtifacts, query func(context.Context, string) ([]byte, error), processes procutil.Source) func(context.Context, couchcore.ThreadAddress) (workbenchshortcut.PaneRole, error) {
	return func(ctx context.Context, address couchcore.ThreadAddress) (workbenchshortcut.PaneRole, error) {
		if artifacts == nil {
			return workbenchshortcut.PaneRoleOther, fmt.Errorf("Pair session focus observer unavailable")
		}
		binding, err := artifacts.PairSessionContext(ctx, address)
		if err != nil {
			return workbenchshortcut.PaneRoleOther, err
		}
		if !binding.Present || binding.Name == "" {
			return workbenchshortcut.PaneRoleOther, fmt.Errorf("Pair session binding unavailable")
		}
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: artifacts.PairLifecycleDataDir(), RepoScope: address.RepoScope, Tag: string(address.Tag)})
		if err != nil {
			return workbenchshortcut.PaneRoleOther, err
		}
		var role workbenchshortcut.PaneRole
		var ids []string
		read := func(ctx context.Context) ([]byte, error) {
			raw, err := query(ctx, binding.Name)
			if err != nil {
				return nil, err
			}
			ids, err = (workbenchshortcut.TerminalPaneRegistry{DataDir: paths.ScopeDir(), Tag: string(address.Tag)}).LiveOwnerIDs(func(pid int) bool { return procutil.Alive(strconv.Itoa(pid)) }, processes)
			return raw, err
		}
		_, err = zellijreply.Confirm(ctx, read, func(raw []byte) (string, error) {
			clients, err := zellijreply.ParseClients(raw)
			if err != nil {
				return "", err
			}
			if len(clients) != 1 {
				return "", fmt.Errorf("expected exactly one attached Zellij client; use Couch's switcher to relaunch")
			}
			client := clients[0]
			kind, id, _ := strings.Cut(client.PaneID, "_")
			role = workbenchshortcut.PaneRoleOther
			registered := false
			command := ""
			if kind != "plugin" {
				registered = workbenchshortcut.Registered(ids, id)
				if registered {
					role = workbenchshortcut.PaneRoleRightTerminal
				} else {
					command = strings.Join(strings.Fields(client.RunningCommand), " ")
					role = workbenchshortcut.RoleForPane(zellijpane.Pane{TerminalCommand: command})
					if role != workbenchshortcut.PaneRoleLeftDraft && role != workbenchshortcut.PaneRoleLeftAgent && role != workbenchshortcut.PaneRoleReview {
						return "", fmt.Errorf("focused pane has no confirmed Pair role; use Couch's switcher to relaunch")
					}
				}
			}
			proof := struct {
				Client     int
				Pane       string
				Role       workbenchshortcut.PaneRole
				Registered bool
				Command    string
			}{client.ID, client.PaneID, role, registered, command}
			encoded, err := json.Marshal(proof)
			return string(encoded), err
		})
		if err != nil {
			return workbenchshortcut.PaneRoleOther, err
		}
		if err := artifacts.RevalidatePairSession(ctx, binding); err != nil {
			return workbenchshortcut.PaneRoleOther, err
		}
		return role, ctx.Err()
	}
}
