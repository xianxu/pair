package launcher

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func (r OSRuntime) DeleteSessionContext(ctx context.Context, session string) error {
	ops := r.sessionQuiescence
	if ops == nil {
		ops = newOSSessionQuiescenceOps()
	}
	wait := r.sessionQuiesceWait
	if wait <= 0 || wait > zjTimeout {
		wait = zjTimeout
	}
	poll := r.sessionQuiescePoll
	if poll <= 0 {
		poll = 25 * time.Millisecond
	}
	return quiesceZellijSession(ctx, session, ops, wait, poll)
}

func (r OSRuntime) ReapNvimContext(ctx context.Context, paths lifecycleEditorPaths) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var result error
	for _, pidPath := range paths.pids {
		if raw, readErr := r.ReadFile(pidPath); readErr == nil {
			if pid := strings.TrimSpace(raw); pid != "" {
				if killErr := exec.CommandContext(ctx, "kill", "-9", pid).Run(); killErr != nil && ctx.Err() != nil {
					result = errors.Join(result, ctx.Err())
				}
			}
			r.Remove(pidPath)
		}
	}
	for _, pattern := range []string{
		"nvim --embed.*" + regexp.QuoteMeta(paths.draft) + `$`,
		"nvim --embed.*" + regexp.QuoteMeta(paths.scrollbackPrefix),
	} {
		_ = exec.CommandContext(ctx, "pkill", "-9", "-f", pattern).Run()
		if ctx.Err() != nil {
			result = errors.Join(result, ctx.Err())
			break
		}
	}
	return result
}

func (r OSRuntime) KillTitlePollerContext(ctx context.Context, pidPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if raw, readErr := r.ReadFile(pidPath); readErr == nil {
		if pid := strings.TrimSpace(raw); pid != "" {
			if killErr := exec.CommandContext(ctx, "kill", pid).Run(); killErr != nil && ctx.Err() != nil {
				return ctx.Err()
			}
		}
		r.Remove(pidPath)
	}
	return nil
}

func (r OSRuntime) CleanupCmuxContext(ctx context.Context, tag, title string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	workspaceID := os.Getenv("CMUX_WORKSPACE_ID")
	if workspaceID == "" {
		return nil
	}
	if _, err := exec.LookPath("cmux"); err == nil {
		if err := r.WriteAtomic(filepath.Join(r.DataDir, "cmux-owner-"+workspaceID), tag+"\n"); err != nil {
			return err
		}
		if err := exec.CommandContext(ctx, "cmux", "rename-workspace", title).Run(); err != nil {
			return err
		}
	}
	r.ClearCmuxOwner()
	return nil
}

var _ contextualCleanupRuntime = OSRuntime{}
