package couchcmd

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchscratch"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// scratchProbe keeps isolated runtimes away from the machine's mounted volume.
func (r OSRuntime) scratchProbe() couchscratch.Probe {
	if r.scratchProbeFn != nil {
		return r.scratchProbeFn()
	}
	if r.isolatedRoot != "" {
		return couchscratch.Probe{Root: filepath.Join(r.isolatedRoot, "scratch")}
	}
	return couchscratch.ReadProbe(couchscratch.DefaultRoot)
}

// scratchSlotEnvironment binds temp roots to retained slot identity, never cwd.
func scratchSlotEnvironment(manager *couchscratch.Manager, lookup func(couchcore.ThreadRecord) (couchcore.ScratchSlotIdentity, error), notify func(string)) func(couchcore.ThreadRecord) []string {
	return func(record couchcore.ThreadRecord) []string {
		slot, err := lookup(record)
		if err != nil {
			if notify != nil {
				notify("slot scratch keeps inherited temp: " + err.Error())
			}
			return nil
		}
		return manager.PrepareSlot(couchscratch.Slot{Repo: slot.Repo, Identity: slot.RepoIdentity, Number: slot.Number})
	}
}

// workspaceScratchOwner follows the host environment for dependency checkouts.
func workspaceScratchOwner(id couchcore.WorkspaceIdentity) (couchscratch.Slot, bool) {
	if h := id.EnvironmentHost; h != nil && h.Repo != "" && h.RepoIdentity != "" && h.Slot > 0 {
		return couchscratch.Slot{Repo: h.Repo, Identity: h.RepoIdentity, Number: h.Slot}, true
	}
	if id.Kind == "primary" && id.Slot != nil && *id.Slot == 0 && id.Repo != "" && id.RepoIdentity != "" {
		return couchscratch.Slot{Repo: id.Repo, Identity: id.RepoIdentity}, true
	}
	return couchscratch.Slot{}, false
}

// configureScratch composes startup effects only for the singleton owner. CLI
// readers cannot change native metadata or start a maintenance worker.
func (r OSRuntime) configureScratch(c *couchcore.Couch) {
	if !r.ownership.held() {
		return
	}
	// Startup installs this immutable callback before any launch or worker.
	notify := func(string) {}
	note := func(text string) { notify(text) }
	manager := couchscratch.New(couchscratch.Options{Probe: r.scratchProbe, Notify: note})
	lookup := func(record couchcore.ThreadRecord) (couchcore.ScratchSlotIdentity, error) {
		owner, err := c.Threads.ScratchSlot(record)
		if err == nil {
			return owner, nil
		}
		if !errors.Is(err, couchcore.ErrScratchAuthorityMissing) {
			return owner, err
		}
		// A primary's first thread can precede family enrollment. Use the
		// existing verified workspace resolver once, never the child's cwd.
		resolver, ok := c.Slots.(couchcore.SlotWorkspaceResolver)
		if !ok {
			return owner, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		id, resolveErr := resolver.ResolveWorkspace(ctx, record.StartingPath)
		if resolveErr != nil {
			return owner, err
		}
		if _, belongs, proofErr := couchcore.RecordCheckoutMembership(record, id.RepoIdentity, id.WorktreeRoot); proofErr != nil || !belongs {
			return owner, err
		}
		if slot, ok := workspaceScratchOwner(id); ok {
			return couchcore.ScratchSlotIdentity{Repo: slot.Repo, RepoIdentity: slot.Identity, Number: slot.Number}, nil
		}
		return owner, err
	}
	c.SlotEnvironment = scratchSlotEnvironment(manager, lookup, note)
	c.ScratchStartup = func(ctx context.Context, warn func(string)) {
		if warn != nil {
			notify = warn
		}
		// Tests with explicit isolation require an injected private scratch probe
		// before even an absent-RAM fallback may edit their native cache route.
		if r.isolatedRoot == "" || r.scratchProbeFn != nil {
			probe := r.scratchProbe()
			cache, err := couchscratch.MetadataDirectory(r.Getenv("HOME"), r.Getenv("XDG_CACHE_HOME"), runtime.GOOS)
			if err == nil {
				bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
				err = couchscratch.ConfigureMetadata(bounded, couchscratch.MetadataOptions{CacheDir: cache, Root: probe.Root, UseRAM: couchscratch.ChooseScratch(probe), Notify: note, ServerPresence: func(ctx context.Context) (bool, error) {
					states, err := (launcher.OSServerStates{}).ServerStates(ctx)
					return len(states) > 0, err
				}})
				cancel()
			}
			if err != nil {
				note("Zellij metadata unchanged: " + err.Error())
			}
		}
		_, _ = manager.Sweep(ctx) // Manager reports one bounded aggregate itself.
	}
	c.ScratchMaintenance = manager.Maintain
}
