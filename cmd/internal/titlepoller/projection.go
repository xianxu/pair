package titlepoller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// titleProjection owns compact launch/activity evidence for one poller's
// lifetime. Activity and occupancy share strict owner-ledger decoding; token
// occupancy still uses contextcmd's complete authorized transcript scan.
type titleProjection struct {
	env        contextcmd.Env
	runtime    sessioninventory.OSRuntime
	activities *sessioninventory.ActivityCache
}

func newTitleProjection(env contextcmd.Env) *titleProjection {
	data := env.PairDataDir
	if data == "" {
		base := env.XDGDataHome
		if base == "" {
			base = filepath.Join(env.Home, ".local", "share")
		}
		data = filepath.Join(base, "pair")
	}
	return &titleProjection{
		env:        env,
		runtime:    sessioninventory.NewOSRuntime(env.Home, data).WithOwnerLaunchCache(sessioninventory.NewOwnerLaunchCache()),
		activities: sessioninventory.NewActivityCache(),
	}
}

func (p *titleProjection) activity(ctx context.Context, tag, agent string) (time.Time, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	a, ok, err := p.activities.Query(ctx, titleInventoryRuntime{OSRuntime: p.runtime, ctx: ctx}, p.env.PairScopeKey, tag, sessioninventory.Agent(agent))
	if err != nil || ctx.Err() != nil {
		return time.Time{}, false
	}
	return a.LastActivityAt, ok
}

func (p *titleProjection) count(ctx context.Context, tag, agent string) string {
	if ctx != nil && ctx.Err() != nil {
		return ""
	}
	var b bytes.Buffer
	if ctx == nil {
		ctx = context.Background()
	}
	contextcmd.RunWithRuntimeContext(ctx, []string{tag, agent}, p.env, titleInventoryRuntime{OSRuntime: p.runtime, ctx: ctx}, &b, io.Discard)
	if ctx != nil && ctx.Err() != nil {
		return ""
	}
	return strings.TrimSpace(b.String())
}

// Preserve OSRuntime's optional catalog/cache/observer capabilities while
// checking this title lifetime at every potentially large range-read boundary.
type titleInventoryRuntime struct {
	sessioninventory.OSRuntime
	ctx context.Context
}

func (r titleInventoryRuntime) ReadAt(a sessioninventory.Artifact, off, size int64) ([]byte, bool, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, false, err
	}
	raw, eof, err := r.OSRuntime.ReadAt(a, off, size)
	if e := r.ctx.Err(); e != nil {
		return nil, false, e
	}
	return raw, eof, err
}
func (r titleInventoryRuntime) ReadFile(a sessioninventory.Artifact, size int64) ([]byte, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := r.OSRuntime.ReadFile(a, size)
	if e := r.ctx.Err(); e != nil {
		return nil, e
	}
	return raw, err
}
func (r titleInventoryRuntime) StatFile(a sessioninventory.Artifact) (sessioninventory.FileEntry, error) {
	if err := r.ctx.Err(); err != nil {
		return sessioninventory.FileEntry{}, err
	}
	entry, err := r.OSRuntime.StatFile(a)
	return entry, errors.Join(err, r.ctx.Err())
}
func (r titleInventoryRuntime) ListFiles(root sessioninventory.StorageRoot) ([]sessioninventory.FileEntry, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := r.OSRuntime.ListFiles(root)
	if e := r.ctx.Err(); e != nil {
		return nil, e
	}
	return entries, err
}
