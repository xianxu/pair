package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/reviewcmd"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// The pane layout behind a terminal note's anchor (#488): which pane of the
// slot in view holds a box, and for a draft or review nvim the file it is
// on. Couch asks the slot's own zellij session, as the shortcut focus probe
// does; couchtty never calls zellij (its seam is couchtty.PaneRects).

// maxNoteFile bounds how much of a draft or review file is read to find the
// lines under a note. A larger file names no lines.
const maxNoteFile = 1 << 20

// queryPaneGeometry is `zellij action list-panes` with geometry, for one
// session. It costs about 590 ms on zellij 0.44.3 (#220); the caller bounds
// it.
func queryPaneGeometry(ctx context.Context, session string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "zellij", "--session", session, "action", "list-panes", "--json", "--command", "--state", "--geometry")
	// A short WaitDelay fails complete output whose copy lags under load
	// (pair#613); the caller's context still bounds the query.
	cmd.WaitDelay = time.Second
	return cmd.Output()
}

// readNoteFile reads at most maxNoteFile+1 bytes of path.
func readNoteFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxNoteFile+1))
}

// paneRectsOf is Couch's couchtty.PaneRects: the slot's panes from its zellij
// session (query), each with its Pair role; floating panes first, as they
// draw over the tiled ones. A draft pane's file is the slot's draft; a
// review pane's is the file its nvim was opened on. read reads a file
// (readNoteFile); an unreadable or oversized one names no lines.
func paneRectsOf(artifacts shortcutFocusArtifacts, query func(context.Context, string) ([]byte, error), read func(string) ([]byte, error)) couchtty.PaneRects {
	return func(ctx context.Context, address couchcore.ThreadAddress) ([]couchtty.PaneRect, error) {
		if artifacts == nil {
			return nil, errors.New("Pair session observer unavailable")
		}
		binding, err := artifacts.PairSessionContext(ctx, address)
		if err != nil {
			return nil, err
		}
		if !binding.Present || binding.Name == "" {
			return nil, errors.New("Pair session binding unavailable")
		}
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: artifacts.PairLifecycleDataDir(), RepoScope: address.RepoScope, Tag: string(address.Tag)})
		if err != nil {
			return nil, err
		}
		var panes []zellijpane.Pane
		var ids []string
		readPanes := func(ctx context.Context) ([]byte, error) {
			raw, err := query(ctx, binding.Name)
			if err != nil {
				return nil, err
			}
			ids, err = (workbenchshortcut.TerminalPaneRegistry{DataDir: paths.ScopeDir(), Tag: string(address.Tag)}).LiveIDs(func(pid int) bool { return procutil.Alive(strconv.Itoa(pid)) })
			return raw, err
		}
		_, err = zellijreply.Confirm(ctx, readPanes, func(raw []byte) (string, error) {
			var err error
			panes, err = zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, State: true, Geometry: true, Tab: true})
			if err != nil {
				return "", err
			}
			if len(panes) == 0 {
				return "", errors.New("zellij reported no panes")
			}
			type fact struct {
				ID                   string
				Plugin               bool
				Role                 workbenchshortcut.PaneRole
				Command, File        string
				TabID, TabPosition   *int
				X, Y, W, H           int
				Floating, Fullscreen bool
			}
			facts := make([]fact, 0, len(panes))
			for _, p := range panes {
				role := workbenchshortcut.RoleForPaneWith(p, ids)
				file := ""
				if role == workbenchshortcut.PaneRoleLeftDraft {
					file = paths.Draft()
				} else if role == workbenchshortcut.PaneRoleReview {
					file = reviewFile(p.TerminalCommand)
				}
				facts = append(facts, fact{p.ID, p.IsPlugin, role, p.TerminalCommand, file, p.TabID, p.TabPosition, p.X, p.Y, p.Columns, p.Rows, p.IsFloating, *p.IsFullscreen})
			}
			encoded, err := json.Marshal(facts)
			return string(encoded), err
		})
		if err != nil {
			return nil, fmt.Errorf("cannot confirm note panes: %w", err)
		}
		if err := artifacts.RevalidatePairSession(ctx, binding); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return notePanes(panes, ids, paths.Draft(), read), ctx.Err()
	}
}

// notePanes maps zellij's panes to couchtty's: pure but for read.
func notePanes(panes []zellijpane.Pane, terminalIDs []string, draft string, read func(string) ([]byte, error)) []couchtty.PaneRect {
	var floating, tiled []couchtty.PaneRect
	for _, p := range panes {
		if p.IsPlugin || p.Columns <= 0 || p.Rows <= 0 {
			continue
		}
		r := couchtty.PaneRect{X: p.X, Y: p.Y, W: p.Columns, H: p.Rows}
		switch workbenchshortcut.RoleForPaneWith(p, terminalIDs) {
		case workbenchshortcut.PaneRoleLeftAgent:
			r.Role = couchtty.PaneAgent
		case workbenchshortcut.PaneRoleLeftDraft:
			r.Role, r.File = couchtty.PaneDraft, draft
		case workbenchshortcut.PaneRoleReview:
			r.Role, r.File = couchtty.PaneReview, reviewFile(p.TerminalCommand)
		case workbenchshortcut.PaneRoleRightTerminal:
			r.Role = couchtty.PaneTerminal
		}
		if r.File != "" {
			if data, err := read(r.File); err == nil && len(data) <= maxNoteFile {
				r.Lines = strings.Split(string(data), "\n")
			}
		}
		if p.IsFloating {
			floating = append(floating, r)
		} else {
			tiled = append(tiled, r)
		}
	}
	return append(floating, tiled...)
}

// reviewFile is the file a review pane's nvim was opened on: what follows
// the review script in `nvim -u …/nvim/review.lua FILE`
// (reviewcmd.SpawnReviewPane, reviewcmd.ReviewScript).
func reviewFile(command string) string {
	marker := "/" + reviewcmd.ReviewScript + " "
	i := strings.Index(command, marker)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(command[i+len(marker):])
}
