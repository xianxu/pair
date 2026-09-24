package couchtty

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// SlotGitProbe observes one checkout's git state (pair#317). Production wires
// couchcore.ProbeSlotGit; it must honour ctx.
type SlotGitProbe func(ctx context.Context, dir string) (couchcore.SlotGitStatus, error)

const (
	// defaultSlotGitInterval is how stale a glyph may get while nothing else
	// (a switch, an inventory landing) asks for a fresh pass.
	defaultSlotGitInterval = 10 * time.Second
	// slotGitProbeTimeout bounds one checkout's git status. A pass probes its
	// checkouts one at a time, so a pass costs at most N times this.
	slotGitProbeTimeout = 3 * time.Second
)

type slotGitResult struct {
	generation uint64
	observed   map[string]couchcore.SlotGitStatus
	failed     map[string]bool
}

// SetSlotGitProbe installs the probe and asks for a first pass.
func (c *Console) SetSlotGitProbe(probe SlotGitProbe) {
	c.mu.Lock()
	c.slotGitProbe = probe
	c.mu.Unlock()
	c.requestSlotGit()
}

// requestSlotGit never blocks: a request already queued covers this one, and
// one arriving while a pass runs becomes the schedule's single follow-up.
func (c *Console) requestSlotGit() {
	select {
	case c.slotGitRequests <- struct{}{}:
	default:
	}
}

// advanceSlotGit runs on Console.Run, the sole owner of the schedule. git runs
// on a worker outside c.mu; render and keystroke paths only ever read
// MenuState.SlotGit, so a slow or hung checkout can stale a glyph but never
// block a keypress.
func (c *Console) advanceSlotGit(event RefreshScheduleEvent) {
	if event.Kind == RefreshRequested {
		c.mu.Lock()
		installed := c.slotGitProbe != nil
		c.mu.Unlock()
		// Nothing can observe a checkout without a probe; a pass would only
		// churn the schedule and the trace.
		if !installed {
			return
		}
	}
	var effects []RefreshScheduleEffect
	c.slotGitSchedule, effects = AdvanceRefreshSchedule(c.slotGitSchedule, event)
	for _, effect := range effects {
		if effect.Kind != RefreshStart {
			continue
		}
		c.mu.Lock()
		probe := c.slotGitProbe
		paths := slotGitProbePaths(menuRows(c.menu))
		c.mu.Unlock()
		c.workers.Add(1)
		go func(generation uint64) {
			defer c.workers.Done()
			result := slotGitResult{generation: generation, observed: map[string]couchcore.SlotGitStatus{}, failed: map[string]bool{}}
			for _, path := range paths {
				if c.lifetime.Err() != nil {
					result.failed[path] = true
					continue
				}
				ctx, cancel := context.WithTimeout(c.lifetime, slotGitProbeTimeout)
				status, err := probe(ctx, path)
				cancel()
				if err != nil {
					result.failed[path] = true
					continue
				}
				result.observed[path] = status
			}
			select {
			case c.slotGitResults <- result:
			case <-c.stop:
			}
		}(effect.Generation)
	}
}

func (c *Console) finishSlotGit(result slotGitResult) {
	if c.slotGitSchedule.Running != result.generation {
		return
	}
	c.mu.Lock()
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventSlotGit, SlotGit: result.observed, SlotGitFailed: result.failed})
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if len(result.observed)+len(result.failed) > 0 {
		c.traceEvent(traceSlotGit, couchcore.ThreadAddress{}, "ok="+strconv.Itoa(len(result.observed))+" failed="+strconv.Itoa(len(result.failed)))
	}
	c.advanceSlotGit(RefreshScheduleEvent{Kind: RefreshFinished, Generation: result.generation})
	// Tabs render from the same state as the switcher, so a quiet child still
	// needs the reserved row repainted (lessons: #307).
	if panelFocused {
		c.showMenu()
	} else {
		c.repaint()
	}
}

// slotGitProbePaths is the pass's probe set: every checkout of a slot group,
// the primary (:0) included, once each and in a stable order.
func slotGitProbePaths(rows []couchcore.ActionableThreadSummary) []string {
	seen := map[string]bool{}
	var paths []string
	for _, row := range rows {
		if row.Target.Kind != couchcore.ThreadTargetSlot || row.Target.Validate() != nil {
			continue
		}
		for _, path := range []string{row.Target.Slot.WorktreeRoot, row.Target.Slot.PrimaryRoot} {
			if !seen[path] {
				seen[path] = true
				paths = append(paths, path)
			}
		}
	}
	sort.Strings(paths)
	return paths
}
