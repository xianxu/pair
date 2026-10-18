package couchcore

import (
	"context"
	"os"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// TitleTarget is display authority for an observed terminal and exact launch.
// It is never part of a row's public JSON; StartNonce is an internal fence.
type TitleTarget struct {
	Address                    ThreadAddress
	Agent, Session, StartNonce string
	LaunchOrdinal              uint64
	// Sessions is the successful host-wide listing already obtained by inventory.
	// A nil map means unknown and may never authorize a workspace takeover.
	Sessions map[string]bool
}

// ProjectTitleTarget rejects incomplete, mismatched and non-live observations.
func ProjectTitleTarget(record ThreadRecord, row ActionableThreadSummary, observation SessionObservation, ordinal uint64) (TitleTarget, bool) {
	b := record.SessionBinding
	if !row.Live() || row.Address != record.Address || b == nil || b.Validate() != nil || b.ScopeKey != record.Address.RepoScope || b.Tag != string(record.Address.Tag) || ordinal == 0 || row.Agent != launchProfileAgent(record) || !launcher.IsSupportedAgent(row.Agent) || !observation.Present() || observation.Name != b.Name {
		return TitleTarget{}, false
	}
	return TitleTarget{Address: row.Address, Agent: row.Agent, Session: b.Name, StartNonce: b.StartNonce, LaunchOrdinal: ordinal, Sessions: observation.Listing}, true
}

type titleLaunchResolver interface {
	CurrentTitleOrdinal(context.Context, ThreadAddress, string) (uint64, error)
}

// CurrentTitleOrdinal uses the checker's existing compact owner cache; it does
// not repeat native inventory scans or ask Zellij for session/client state.
func (c ScopedThreadArtifactCollisionChecker) CurrentTitleOrdinal(ctx context.Context, address ThreadAddress, agent string) (uint64, error) {
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: c.GlobalDataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		return 0, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}
	rt := sessioninventory.NewOSRuntime(home, paths.ScopeDir()).WithOwnerLaunchCache(c.OwnerLaunches).WithWorkObserver(c.InventoryWork)
	launch, ok, err := sessioninventory.CurrentTitleLaunch(ctx, rt, address.RepoScope, string(address.Tag), sessioninventory.Agent(agent))
	if err != nil || !ok {
		return 0, err
	}
	return launch.Ordinal, nil
}

func (c *Couch) applyTitleTargets(ctx context.Context, rows []ActionableThreadSummary, snapshot ThreadSnapshot, evidence map[ThreadAddress]ThreadEvidence) []ActionableThreadSummary {
	resolver, ok := c.Artifacts.(titleLaunchResolver)
	if !ok {
		return rows
	}
	records := map[ThreadAddress]ThreadRecord{}
	for _, r := range snapshot.Records {
		records[r.Address] = r
	}
	for i := range rows {
		row := &rows[i]
		if !row.Live() || !evidence[row.Address].Session.Present() {
			continue
		}
		if _, valid := ProjectTitleTarget(records[row.Address], *row, evidence[row.Address].Session, 1); !valid {
			continue
		}
		ordinal, err := resolver.CurrentTitleOrdinal(ctx, row.Address, row.Agent)
		if err != nil {
			continue
		}
		if target, ok := ProjectTitleTarget(records[row.Address], *row, evidence[row.Address].Session, ordinal); ok {
			row.TitleTarget = &target
		}
	}
	return rows
}

// TitleTargetCurrent fences a queued optional effect against local owner
// replacement. It deliberately performs no session or client query.
func (c *Couch) TitleTargetCurrent(ctx context.Context, target TitleTarget) bool {
	if ctx.Err() != nil {
		return false
	}
	record, err := c.Threads.PreviewThreadContext(ctx, target.Address)
	if err != nil || record.SessionBinding == nil {
		return false
	}
	b := record.SessionBinding
	if b.Validate() != nil || b.Name != target.Session || b.StartNonce != target.StartNonce || b.ScopeKey != target.Address.RepoScope || b.Tag != string(target.Address.Tag) || launchProfileAgent(record) != target.Agent {
		return false
	}
	resolver, ok := c.Artifacts.(titleLaunchResolver)
	if !ok {
		return false
	}
	ordinal, err := resolver.CurrentTitleOrdinal(ctx, target.Address, target.Agent)
	return err == nil && ordinal == target.LaunchOrdinal && ordinal != 0 && ctx.Err() == nil
}
