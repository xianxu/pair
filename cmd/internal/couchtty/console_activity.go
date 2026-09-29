package couchtty

import (
	"context"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// ActivityProbe reads one live thread's last activity (pair#247): the
// operator's input or its agent's work. Production wires threadactivity.Latest;
// it must honour ctx. A zero time means the thread left no signal at all.
type ActivityProbe func(ctx context.Context, row couchcore.ActionableThreadSummary) (time.Time, error)

const (
	// defaultActivityInterval is how stale an idle fade may get. The bands are a
	// day and three days wide, so a minute is far below what the eye resolves.
	defaultActivityInterval = 60 * time.Second
	// activityProbeTimeout bounds one thread's probe, most of which is listing
	// the agent's native session store (~40ms for 3,500 files, measured).
	activityProbeTimeout = 2 * time.Second
)

type activityResult struct {
	generation uint64
	observed   map[couchcore.ThreadAddress]time.Time
	failed     map[couchcore.ThreadAddress]bool
}

// SetActivityProbe installs the probe and asks for a first pass.
func (c *Console) SetActivityProbe(probe ActivityProbe) {
	c.mu.Lock()
	c.activityProbe = probe
	c.mu.Unlock()
	c.requestActivity()
}

// requestActivity never blocks: a queued request covers this one, and one
// arriving while a pass runs becomes the schedule's single follow-up.
func (c *Console) requestActivity() {
	select {
	case c.activityRequests <- struct{}{}:
	default:
	}
}

// advanceActivity runs on Console.Run, the schedule's sole owner, exactly like
// advanceSlotGit: probes run on a worker outside c.mu, and render paths only
// read MenuState.Activity, so a slow store can stale a fade but never block.
func (c *Console) advanceActivity(event RefreshScheduleEvent) {
	if event.Kind == RefreshRequested {
		c.mu.Lock()
		installed := c.activityProbe != nil
		c.mu.Unlock()
		if !installed {
			return
		}
	}
	var effects []RefreshScheduleEffect
	c.activitySchedule, effects = AdvanceRefreshSchedule(c.activitySchedule, event)
	for _, effect := range effects {
		if effect.Kind != RefreshStart {
			continue
		}
		c.mu.Lock()
		probe := c.activityProbe
		rows := activityProbeRows(menuRows(c.menu))
		c.mu.Unlock()
		c.workers.Add(1)
		go func(generation uint64) {
			defer c.workers.Done()
			result := activityResult{generation: generation, observed: map[couchcore.ThreadAddress]time.Time{}, failed: map[couchcore.ThreadAddress]bool{}}
			for _, row := range rows {
				if c.lifetime.Err() != nil {
					result.failed[row.Address] = true
					continue
				}
				ctx, cancel := context.WithTimeout(c.lifetime, activityProbeTimeout)
				at, err := probe(ctx, row)
				timedOut := ctx.Err() != nil
				cancel()
				switch {
				case err != nil || timedOut:
					result.failed[row.Address] = true
				case !at.IsZero():
					result.observed[row.Address] = at
				}
			}
			select {
			case c.activityResults <- result:
			case <-c.stop:
			}
		}(effect.Generation)
	}
}

func (c *Console) finishActivity(result activityResult) {
	if c.activitySchedule.Running != result.generation {
		return
	}
	c.mu.Lock()
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventActivity, Activity: result.observed, ActivityFailed: result.failed})
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if len(result.observed)+len(result.failed) > 0 {
		c.traceEvent(traceActivity, couchcore.ThreadAddress{}, "ok="+strconv.Itoa(len(result.observed))+" failed="+strconv.Itoa(len(result.failed)))
	}
	c.advanceActivity(RefreshScheduleEvent{Kind: RefreshFinished, Generation: result.generation})
	if panelFocused {
		c.showMenu()
	} else {
		c.repaint()
	}
}

// activityProbeRows is every live thread once. Idle fading is for live threads
// only (non-live rows keep their own age ramp), so nothing else is probed.
func activityProbeRows(rows []couchcore.ActionableThreadSummary) []couchcore.ActionableThreadSummary {
	seen := map[couchcore.ThreadAddress]bool{}
	var live []couchcore.ActionableThreadSummary
	for _, row := range rows {
		if !row.Live() || row.Address == (couchcore.ThreadAddress{}) || seen[row.Address] {
			continue
		}
		seen[row.Address] = true
		live = append(live, row)
	}
	return live
}
