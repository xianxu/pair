package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/panebirth"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadactivity"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

type couchTitleFacts struct {
	Ready         bool
	PaneID, Count string
	Activity      time.Time
}
type couchTitleSource interface {
	Poll(context.Context) (couchTitleFacts, error)
}

// One stateful seam is shared by the actual updater and external-system fake.
// Open creates one compact source for an exact invocation; it never starts work.
type couchTitleIO interface {
	Open(couchcore.TitleTarget) (couchTitleSource, error)
	Now() time.Time
	WorkspaceOwner(context.Context, couchcore.TitleTarget) (string, error)
	WriteWorkspaceOwner(context.Context, couchcore.TitleTarget, string) error
	RenamePane(context.Context, couchcore.TitleTarget, string, string) error
	RenameWorkspace(context.Context, string) error
}
type couchTitleKey struct {
	Address               couchcore.ThreadAddress
	Agent, Session, Nonce string
	Ordinal               uint64
}

func titleKey(t couchcore.TitleTarget) couchTitleKey {
	return couchTitleKey{t.Address, t.Agent, t.Session, t.StartNonce, t.LaunchOrdinal}
}
func (k couchTitleKey) generation() string {
	return fmt.Sprintf("%q/%q/%q/%q/%q/%d", k.Address.RepoScope, k.Address.Tag, k.Agent, k.Session, k.Nonce, k.Ordinal)
}

type couchTitleEntry struct {
	target   couchcore.TitleTarget
	source   couchTitleSource
	desired  []titlepoller.TitleUpdate
	activity time.Time
}

// CouchTitleUpdater is owned by the console's serialized activity worker. It
// uses that worker's pass deadline and shutdown context; it owns no goroutines.
type CouchTitleUpdater struct {
	io        couchTitleIO
	current   func(context.Context, couchcore.TitleTarget) bool
	workspace string
	entries   map[couchTitleKey]*couchTitleEntry
	state     titlepoller.TitleUpdateState
	cursor    int
}

func newCouchTitleUpdater(dataDir, home, workspaceID string, current func(context.Context, couchcore.TitleTarget) bool) *CouchTitleUpdater {
	return &CouchTitleUpdater{io: &osCouchTitles{dataDir: dataDir, home: home, workspace: workspaceID}, current: current, workspace: workspaceID, entries: map[couchTitleKey]*couchTitleEntry{}}
}
func (u *CouchTitleUpdater) isCurrent(ctx context.Context, t couchcore.TitleTarget) bool {
	return ctx.Err() == nil && u.current != nil && u.current(ctx, t)
}
func (u *CouchTitleUpdater) canClaim(ctx context.Context, t couchcore.TitleTarget) bool {
	if t.Sessions == nil {
		return false
	}
	raw, err := u.io.WorkspaceOwner(ctx, t)
	if err != nil {
		return false
	}
	tag, session := titlepoller.ParseCmuxOwner(raw)
	if session == "" && tag != "" {
		session = "pair-" + tag
	}
	return titlepoller.ShouldClaimWorkspace(tag, string(t.Address.Tag), t.Sessions[session])
}

// Update returns latest activity and failed/unresolved addresses. Budget-skipped
// targets retain both their compact projection and desired-title cache; only
// absent or replaced exact generations are pruned. The rotating start position
// prevents one slow projection from monopolizing every pass.
func (u *CouchTitleUpdater) Update(ctx context.Context, rows []couchcore.ActionableThreadSummary) (map[couchcore.ThreadAddress]time.Time, map[couchcore.ThreadAddress]bool) {
	return u.UpdateWithActivity(ctx, rows, nil)
}

// UpdateWithActivity merges the admitted wrapper snapshot before either title
// or fading decisions, so both consumers use the same latest work evidence.
func (u *CouchTitleUpdater) UpdateWithActivity(ctx context.Context, rows []couchcore.ActionableThreadSummary, work map[couchcore.ThreadAddress]time.Time) (map[couchcore.ThreadAddress]time.Time, map[couchcore.ThreadAddress]bool) {
	activity := map[couchcore.ThreadAddress]time.Time{}
	failed := map[couchcore.ThreadAddress]bool{}
	next := make(map[couchTitleKey]*couchTitleEntry)
	var keys []couchTitleKey
	for _, row := range rows {
		if !row.Live() {
			continue
		}
		failed[row.Address] = true
		t := row.TitleTarget
		if t == nil || t.Address != row.Address || t.Agent != row.Agent || t.Session == "" || t.StartNonce == "" || t.LaunchOrdinal == 0 {
			continue
		}
		key := titleKey(*t)
		if _, seen := next[key]; seen {
			continue
		}
		e := u.entries[key]
		if e == nil {
			e = &couchTitleEntry{}
		}
		e.target = *t
		next[key] = e
		keys = append(keys, key)
		if !e.activity.IsZero() {
			activity[row.Address] = e.activity
		}
	}
	u.entries = next
	var ordered []couchTitleKey
	if len(keys) > 0 {
		start := u.cursor % len(keys)
		for n := 0; n < len(keys); n++ {
			ordered = append(ordered, keys[(start+n)%len(keys)])
		}
		for n, key := range ordered {
			if ctx.Err() != nil {
				break
			}
			// Move before IO so cancellation of the first target advances fairness.
			u.cursor = (start + n + 1) % len(keys)
			e := next[key]
			if !u.isCurrent(ctx, e.target) {
				e.desired = nil
				continue
			}
			if e.source == nil {
				var err error
				e.source, err = u.io.Open(e.target)
				if err != nil {
					e.desired = nil
					continue
				}
			}
			facts, err := e.source.Poll(ctx)
			if err != nil || !facts.Ready {
				e.desired = nil
				continue
			}
			if work[key.Address].After(facts.Activity) {
				facts.Activity = work[key.Address]
			}
			e.activity = facts.Activity
			if !facts.Activity.IsZero() {
				activity[key.Address] = facts.Activity
				delete(failed, key.Address)
			}
			e.desired = nil
			generation := key.generation()
			if facts.PaneID != "" && !facts.Activity.IsZero() && u.io.Now().Sub(facts.Activity) < 2*time.Minute {
				e.desired = append(e.desired, titlepoller.TitleUpdate{Target: titlepoller.TitleUpdateTarget{Generation: generation, PaneID: facts.PaneID}, Title: titlepoller.FrameTitle(e.target.Agent, facts.Count)})
			}
			if u.workspace != "" && !facts.Activity.IsZero() && u.canClaim(ctx, e.target) {
				e.desired = append(e.desired, titlepoller.TitleUpdate{Target: titlepoller.TitleUpdateTarget{Generation: generation + "/workspace"}, Title: titlepoller.CmuxWorkspaceTitle(titlepoller.PrefixForAge(u.io.Now().Sub(facts.Activity)), e.target.Session)})
			}
			u.advance(ctx, ordered, &key)
		}
		if ctx.Err() == nil {
			u.cursor = (start + 1) % len(keys)
		}
	} else {
		u.cursor = 0
	}
	// Refresh once more to prune absent, replaced and unresolved targets even
	// when the pass expired before it could process any sources.
	u.advance(ctx, ordered, nil)
	return activity, failed
}

// Each target's effects run before the next projection can consume the budget.
// Every reducer refresh still includes skipped targets' retained desires.
func (u *CouchTitleUpdater) advance(ctx context.Context, ordered []couchTitleKey, selected *couchTitleKey) {

	var desired []titlepoller.TitleUpdate
	owners := map[titlepoller.TitleUpdateTarget]couchTitleKey{}
	for _, key := range ordered {
		for _, d := range u.entries[key].desired {
			desired = append(desired, d)
			owners[d.Target] = key
		}
	}
	effects := titlepoller.AdvanceTitleUpdate(&u.state, titlepoller.TitleUpdateEvent{Kind: titlepoller.RefreshTitleUpdates, Titles: desired})
	for _, effect := range effects {
		key, ok := owners[effect.Target]
		if !ok || selected == nil || key != *selected {
			continue
		}
		e := u.entries[key]
		if !u.isCurrent(ctx, e.target) {
			continue
		}
		var err error
		if effect.Target.PaneID != "" {
			err = u.io.RenamePane(ctx, e.target, effect.Target.PaneID, effect.Title)
		} else {
			if !u.canClaim(ctx, e.target) || !u.isCurrent(ctx, e.target) {
				continue
			}
			err = u.io.WriteWorkspaceOwner(ctx, e.target, titlepoller.FormatCmuxOwner(string(e.target.Address.Tag), e.target.Session))
			if err == nil {
				if !u.isCurrent(ctx, e.target) {
					continue
				}
				err = u.io.RenameWorkspace(ctx, effect.Title)
			}
		}
		// A completion after generation replacement cannot confirm its cache.
		success := err == nil && u.isCurrent(ctx, e.target)
		titlepoller.AdvanceTitleUpdate(&u.state, titlepoller.TitleUpdateEvent{Kind: titlepoller.CompleteTitleUpdate, Result: titlepoller.TitleUpdateResult{Effect: effect, Success: success}})
	}
}

type osCouchTitles struct{ dataDir, home, workspace string }

func (o *osCouchTitles) Now() time.Time { return time.Now() }
func (o *osCouchTitles) Open(t couchcore.TitleTarget) (couchTitleSource, error) {
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: o.dataDir, RepoScope: t.Address.RepoScope, Tag: string(t.Address.Tag)})
	if err != nil {
		return nil, err
	}
	rt := sessioninventory.NewOSRuntime(o.home, paths.ScopeDir()).WithOwnerLaunchCache(sessioninventory.NewOwnerLaunchCache())
	return &osCouchTitleSource{target: t, paths: paths, runtime: rt, projection: sessioninventory.NewTitleProjection(rt, t.Address.RepoScope, string(t.Address.Tag), sessioninventory.Agent(t.Agent), t.LaunchOrdinal)}, nil
}
func (o *osCouchTitles) ownerPath(t couchcore.TitleTarget) (string, error) {
	name := "cmux-owner-" + o.workspace
	if o.workspace == "" || filepath.Base(name) != name {
		return "", errors.New("invalid workspace publication name")
	}
	owner, err := artifactpath.NewStorageOwner(o.dataDir, t.Address.RepoScope, string(t.Address.Tag))
	if err != nil {
		return "", err
	}
	return filepath.Join(owner.Directory(), name), nil
}
func (o *osCouchTitles) WorkspaceOwner(ctx context.Context, t couchcore.TitleTarget) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := o.ownerPath(t)
	if err != nil {
		return "", err
	}
	raw, err := readCouchTitleFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(raw), err
}
func (o *osCouchTitles) WriteWorkspaceOwner(ctx context.Context, t couchcore.TitleTarget, raw string) error {
	path, err := o.ownerPath(t)
	if err != nil {
		return err
	}
	owner, err := artifactpath.NewStorageOwner(o.dataDir, t.Address.RepoScope, string(t.Address.Tag))
	if err != nil {
		return err
	}
	coordinator, err := storagegc.NewCoordinator(o.dataDir)
	if err != nil {
		return err
	}
	return coordinator.PublishMetadata(ctx, owner, filepath.Base(path), []byte(raw))
}
func (*osCouchTitles) RenamePane(ctx context.Context, t couchcore.TitleTarget, pane, title string) error {
	env := zellijquery.ChildEnvironment(nil)
	return zellijquery.WithReadyProbe(ctx, env, func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "zellij", "--session", t.Session, "action", "rename-pane", "--pane-id", pane, title)
		cmd.Env = env
		_, err := runStaleCommand(cmd)
		return err
	})
}
func (*osCouchTitles) RenameWorkspace(ctx context.Context, title string) error {
	_, err := runStaleCommand(exec.CommandContext(ctx, "cmux", "rename-workspace", title))
	return err
}

type osCouchTitleSource struct {
	target     couchcore.TitleTarget
	paths      artifactpath.Paths
	runtime    sessioninventory.OSRuntime
	projection *sessioninventory.TitleProjection
}

func (s *osCouchTitleSource) Poll(ctx context.Context) (couchTitleFacts, error) {
	t := s.target
	launch, ok, err := sessioninventory.CurrentTitleLaunch(ctx, s.runtime, t.Address.RepoScope, string(t.Address.Tag), sessioninventory.Agent(t.Agent))
	if err != nil {
		return couchTitleFacts{}, err
	}
	if !ok || launch.Ordinal != t.LaunchOrdinal {
		return couchTitleFacts{}, nil
	}
	pane, err := panebirth.Evidence(s.paths.ScopeDir(), string(t.Address.Tag), t.Agent)
	if err != nil {
		return couchTitleFacts{}, err
	}
	raw, err := readCouchTitleFile(pane)
	if errors.Is(err, os.ErrNotExist) {
		return couchTitleFacts{}, nil
	}
	if err != nil {
		return couchTitleFacts{}, err
	}
	var info struct {
		PaneID string `json:"pane_id"`
	}
	if err = json.Unmarshal(raw, &info); err != nil {
		return couchTitleFacts{}, err
	}
	if info.PaneID == "" {
		return couchTitleFacts{}, nil
	}
	result, err := s.projection.PollContext(ctx)
	if err != nil {
		return couchTitleFacts{}, err
	}
	activity := threadactivity.Latest(ctx, couchTitleActivity{native: result.Activity}, threadactivity.Thread{ScopeDir: s.paths.ScopeDir(), Tag: string(t.Address.Tag), Agent: t.Agent})
	count := ""
	if !result.Awaiting && result.HasUsage {
		count = result.Usage.Figure()
	}
	return couchTitleFacts{Ready: true, PaneID: info.PaneID, Count: count, Activity: activity}, nil
}

type couchTitleActivity struct{ native time.Time }

func (a couchTitleActivity) ModTime(path string) (time.Time, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return st.ModTime(), true
}
func (a couchTitleActivity) SessionActivity(context.Context, threadactivity.Thread) (time.Time, bool) {
	return a.native, !a.native.IsZero()
}
func readCouchTitleFile(path string) ([]byte, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > 64<<10 {
		return nil, errors.New("invalid title metadata")
	}
	raw, err := io.ReadAll(io.LimitReader(f, 64<<10+1))
	if len(raw) > 64<<10 {
		return nil, errors.New("title metadata grew beyond limit")
	}
	return raw, err
}
