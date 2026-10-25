package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchscratch"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// isolatedScratchOverlay accepts only one complete slot overlay beneath the
// injected private scratch root. Other actor paths retain isolation defaults.
func isolatedScratchOverlay(root string, env []string) []string {
	if root == "" {
		return nil
	}
	values := map[string]string{}
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		values[key] = value
	}
	path := values["TMPDIR"]
	if path == "" || values["GOTMPDIR"] != path || values["CLAUDE_CODE_TMPDIR"] != path || filepath.Clean(path) != path {
		return nil
	}
	name, ok := strings.CutPrefix(path, filepath.Join(root, "tmp")+string(filepath.Separator))
	if !ok || name == "" || strings.ContainsAny(name, "/\\") {
		return nil
	}
	return []string{"TMPDIR=" + path, "GOTMPDIR=" + path, "CLAUDE_CODE_TMPDIR=" + path}
}

// scratchProbe keeps isolated runtimes away from the machine's mounted volume.
func (r OSRuntime) scratchProbe() couchscratch.Probe {
	if r.scratchProbeFn != nil {
		p := r.scratchProbeFn()
		if r.isolatedRoot != "" {
			physical, err := canonicalFuture(p.Root)
			if err != nil || !confined(r.isolatedRoot, physical) {
				p.Mounted = false
				p.Err = errors.New("scratch probe escapes isolated Couch")
			} else {
				p.Root = physical
			}
		}
		return p
	}
	if r.isolatedRoot != "" {
		return couchscratch.Probe{Root: filepath.Join(r.isolatedRoot, "scratch")}
	}
	return couchscratch.ReadProbe(couchscratch.DefaultRoot)
}

// scratchSlotEnvironment binds temp roots to retained slot identity, never cwd.
func scratchSlotEnvironment(manager *couchscratch.Manager, lookup func(context.Context, couchcore.ThreadRecord) (couchcore.ScratchSlotIdentity, error)) func(context.Context, couchcore.ThreadRecord) []string {
	return func(ctx context.Context, record couchcore.ThreadRecord) []string {
		return manager.PrepareResolvedSlot(func() (couchscratch.Slot, error) {
			if err := ctx.Err(); err != nil {
				return couchscratch.Slot{}, err
			}
			slot, err := lookup(ctx, record)
			return couchscratch.Slot{Repo: slot.Repo, Identity: slot.RepoIdentity, Number: slot.Number}, err
		})
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
	c.ScratchDiagnostic = func(_ context.Context, record couchcore.ThreadRecord) string {
		_, status := r.scratchAdmission(record)
		return status
	}
	if !r.ownership.held() {
		return
	}
	// Startup installs this immutable callback before any launch or worker.
	notify := func(string) {}
	note := func(text string) { notify(text) }
	manager := couchscratch.New(couchscratch.Options{Probe: r.scratchProbe, Notify: note, Ticks: r.scratchTicks, Now: r.scratchNow})
	lookup := func(ctx context.Context, record couchcore.ThreadRecord) (couchcore.ScratchSlotIdentity, error) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		owner, err := c.Threads.ScratchSlot(ctx, record)
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
	prepare := scratchSlotEnvironment(manager, lookup)
	c.SlotEnvironment = func(ctx context.Context, record couchcore.ThreadRecord) []string {
		allowed, status := r.scratchAdmission(record)
		if !allowed {
			note(status)
			return nil
		}
		return prepare(ctx, record)
	}
	c.ScratchStartup = func(ctx context.Context, warn func(string)) {
		if warn != nil {
			notify = warn
		}
		// Tests with explicit isolation require an injected private scratch probe
		// before even an absent-RAM fallback may edit their native cache route.
		if r.isolatedRoot == "" || r.scratchProbeFn != nil {
			probe := r.scratchProbe()
			cacheHome := r.Getenv("XDG_CACHE_HOME")
			if r.isolatedRoot != "" {
				cacheHome = ""
			}
			cache, err := couchscratch.MetadataDirectory(r.Getenv("HOME"), cacheHome, runtime.GOOS)
			if err == nil {
				servers := r.scratchServers
				if servers == nil {
					servers = nativeScratchServers
				}
				bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
				err = couchscratch.ConfigureMetadata(bounded, couchscratch.MetadataOptions{CacheDir: cache, Root: probe.Root, UseRAM: couchscratch.ChooseScratch(probe), Notify: note, ServerPresence: servers})
				cancel()
			}
			if err != nil {
				if warning, ok := ctx.Value(scratchWarningContextKey{}).(func(string)); ok {
					warning("Zellij metadata unchanged: " + err.Error())
				} else {
					note("Zellij metadata unchanged: " + err.Error())
				}
			}
		}
		_, _ = manager.Sweep(ctx) // Manager reports one bounded aggregate itself.
	}
	c.ScratchMaintenance = manager.Maintain
}

// scratchAdmission shares the persistent declaration with launch and read-only show.
func (r OSRuntime) scratchAdmission(_ couchcore.ThreadRecord) (bool, string) {
	marker := couchscratch.ScratchOptInPath(r.Getenv("HOME"), r.Getenv("XDG_CONFIG_HOME"))
	if marker == "" {
		return false, "scratch: normal temp (opt-in location unavailable; set absolute HOME or XDG_CONFIG_HOME)"
	}
	if r.isolatedRoot != "" {
		physical, err := canonicalFuture(marker)
		if err != nil || !confined(r.isolatedRoot, physical) {
			return false, fmt.Sprintf("scratch: normal temp (opt-in outside isolated Couch: %q)", marker)
		}
	}
	info, err := os.Lstat(marker)
	optedIn := err == nil && info.Mode().IsRegular()
	if !optedIn {
		return false, fmt.Sprintf("scratch: normal temp (setup declaration required: %q)", marker)
	}
	if !couchscratch.ChooseSlotScratch(r.scratchProbe(), optedIn) {
		return false, "scratch: normal temp (RAM disk unavailable)"
	}
	return true, "scratch: RAM disk"
}

// nativeScratchServers observes every native server, regardless of Pair scope.
func nativeScratchServers(ctx context.Context) (bool, error) {
	states, err := (launcher.OSServerStates{}).ServerStates(ctx)
	return len(states) > 0, err
}

// Metadata setup has a typed error boundary separate from routine scratch
// fallback/sweep feedback. This startup-only context does not outlive setup.
type scratchWarningContextKey struct{}

func withScratchWarnings(ctx context.Context, console *couchtty.Console) context.Context {
	if console == nil {
		return ctx
	}
	return context.WithValue(ctx, scratchWarningContextKey{}, func(text string) {
		console.Warn(couchtty.Notice{Class: couchcore.NoticeScratchMetadataWarning, Kind: "status", Body: text})
	})
}
