package launcher

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func (r OSRuntime) DeleteSessionContext(ctx context.Context, session string) error {
	ops, wait, poll := r.sessionQuiescenceSettings()
	return zellijquery.RetireReadyAfter(ctx, nil, session, func(ctx context.Context) error {
		return quiesceZellijSession(ctx, session, ops, wait, poll)
	})
}

// DeleteOwnedSessionContext retains the positively observed server generation
// even after stopping its children removes their live pane command evidence.
// Later observations may confirm absence, but never authorize another server.
func (r OSRuntime) DeleteOwnedSessionContext(ctx context.Context, owner SessionOwnerObservation) error {
	if owner.State != SessionOwnerOwned || owner.Name == "" || filepath.IsAbs(owner.Name) || filepath.Base(owner.Name) != owner.Name || !validQuiescenceServer(owner.Server, owner.Name) {
		return errors.New("delete owned session requires positive exact server ownership")
	}
	ops, wait, poll := r.sessionQuiescenceSettings()
	return zellijquery.RetireReadyAfter(ctx, nil, owner.Name, func(ctx context.Context) error {
		return quiesceAuthorizedSession(ctx, owner.Name, []SessionServerIdentity{owner.Server}, ops, wait, poll)
	})
}

func (r OSRuntime) sessionQuiescenceSettings() (sessionQuiescenceOps, time.Duration, time.Duration) {
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
	return ops, wait, poll
}

func (r OSRuntime) ReapNvimContext(ctx context.Context, paths lifecycleEditorPaths) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := reapNvimSidecars(ctx, r.processTable(), r, paths.sidecars)
	return err
}

func (r OSRuntime) KillTitlePollerContext(ctx context.Context, pidPath string) error {
	// Cancellation remains part of the existing lifecycle interface. Teardown
	// belongs to the wrapper lease, never a launcher reading a numeric pidfile.
	return ctx.Err()
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

func (r OSRuntime) TakeQuitIntent(session string) (QuitIntent, bool, error) {
	path, ok := quitMarkerPath(session)
	if !ok {
		return QuitIntent{}, false, errors.New("invalid quit marker session")
	}
	raw, err := osfs.ReadRegularMetadata(context.Background(), path, 64*1024)
	if os.IsNotExist(err) {
		return QuitIntent{}, false, nil
	}
	if err != nil {
		return QuitIntent{}, false, err
	}
	r.Remove(path)
	intent, err := ReadQuitIntent(raw)
	return intent, true, err
}

func (r OSRuntime) WriteQuitIntent(session string, intent QuitIntent) error {
	path, ok := quitMarkerPath(session)
	if !ok {
		return errors.New("invalid quit marker session")
	}
	raw, err := WriteQuitIntent(intent)
	if err != nil {
		return err
	}
	return r.WriteAtomic(path, string(raw))
}

var _ contextualCleanupRuntime = OSRuntime{}
