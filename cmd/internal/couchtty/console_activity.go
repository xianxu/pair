package couchtty

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/threadactivity"
)

// ActivityProbe reads one live thread's last activity (pair#247): the
// operator's input or its agent's work. Production wires threadactivity.Latest;
// it must honour ctx. A zero time means the thread left no signal at all.
type ActivityProbe func(ctx context.Context, row couchcore.ActionableThreadSummary) (time.Time, error)

// ActivityPass shares optional title and activity work on the existing schedule.
// The bool map reports failed/unknown observations, preserving the previous fade.
type ActivityPass func(context.Context, []couchcore.ActionableThreadSummary) (map[couchcore.ThreadAddress]time.Time, map[couchcore.ThreadAddress]bool)

// SetActivityPass replaces per-row native probes with one combined display pass.
func (c *Console) SetActivityPass(pass ActivityPass) {
	c.mu.Lock()
	c.activityPass = pass
	c.mu.Unlock()
	c.requestActivity()
}

// invalidateActivityRowsLocked fences a worker using an obsolete target set.
// The caller invokes the returned cancel after releasing c.mu.
func (c *Console) invalidateActivityRowsLocked(rows []couchcore.ActionableThreadSummary, known bool) context.CancelFunc {
	previousKnown := c.activityRowsKnown
	c.activityRowsKnown = known
	if known && previousKnown && sameActivityTargets(c.resourceRows, rows) {
		return nil
	}
	c.activityRowsGeneration++
	return c.activityCancel
}

func sameActivityTargets(a, b []couchcore.ActionableThreadSummary) bool {
	a, b = activityProbeRows(a), activityProbeRows(b)
	if len(a) != len(b) {
		return false
	}
	byAddress := map[couchcore.ThreadAddress]couchcore.ActionableThreadSummary{}
	for _, row := range a {
		byAddress[row.Address] = row
	}
	for _, row := range b {
		old, ok := byAddress[row.Address]
		if !ok || old.Agent != row.Agent || !reflect.DeepEqual(old.TitleTarget, row.TitleTarget) {
			return false
		}
	}
	return true
}

// ThreadActivityProbe adapts the shared thread-activity definition to Console
// rows. Its runtime owns caches across rows and passes; each query stays scoped
// to that row's repository and exact thread address.
func ThreadActivityProbe(dataDir string, runtime threadactivity.Runtime) ActivityProbe {
	return func(ctx context.Context, row couchcore.ActionableThreadSummary) (time.Time, error) {
		thread, err := threadactivity.InScope(dataDir, row.Address.RepoScope, string(row.Address.Tag), row.Agent)
		if err != nil {
			return time.Time{}, err
		}
		return threadactivity.Latest(ctx, runtime, thread), nil
	}
}

const (
	// defaultActivityInterval is how stale an idle fade may get. The bands are a
	// day and three days wide, so a minute is far below what the eye resolves.
	defaultActivityInterval = 60 * time.Second
	// activityProbeTimeout bounds one thread's probe, most of which is listing
	// the agent's native session store (~40ms for 3,500 files, measured).
	activityProbeTimeout = 2 * time.Second
)

type activityResult struct {
	generation     uint64
	rowsGeneration uint64
	observed       map[couchcore.ThreadAddress]time.Time
	failed         map[couchcore.ThreadAddress]bool
}

// SetActivityProbe installs the probe and asks for a first pass.
func (c *Console) SetActivityProbe(probe ActivityProbe) {
	c.mu.Lock()
	c.activityProbe = probe
	c.mu.Unlock()
	c.requestActivity()
}

// SetWrapperActivity adds the admitted wrapper's in-memory work evidence to
// the same activity pass. It adds no IO probe or polling schedule.
func (c *Console) SetWrapperActivity(view func(couchcore.ActionableThreadSummary) time.Time) {
	c.mu.Lock()
	c.wrapperActivity = view
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
		installed := c.activityProbe != nil || c.activityPass != nil
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
		pass := c.activityPass
		rowsGeneration := c.activityRowsGeneration
		rowsKnown := c.activityRowsKnown
		passContext, passCancel := context.WithTimeout(c.lifetime, activityProbeTimeout)
		c.activityCancel = passCancel
		wrapper := c.wrapperActivity
		previous := c.menu.Activity
		rows := activityProbeRows(menuRows(c.menu))
		c.mu.Unlock()
		c.workers.Add(1)
		go func(generation uint64) {
			defer c.workers.Done()
			defer passCancel()
			result := activityResult{generation: generation, rowsGeneration: rowsGeneration, observed: map[couchcore.ThreadAddress]time.Time{}, failed: map[couchcore.ThreadAddress]bool{}}
			if pass != nil && rowsKnown && c.lifetime.Err() == nil {
				result.observed, result.failed = pass(passContext, rows)
				if result.observed == nil {
					result.observed = map[couchcore.ThreadAddress]time.Time{}
				}
				if result.failed == nil {
					result.failed = map[couchcore.ThreadAddress]bool{}
				}
			}
			for _, row := range rows {
				if c.lifetime.Err() != nil {
					result.failed[row.Address] = true
					continue
				}
				at, failed := result.observed[row.Address], result.failed[row.Address]
				if pass == nil {
					ctx, cancel := context.WithTimeout(c.lifetime, c.probeTimeout)
					if probe == nil {
						failed = true
					} else {
						var err error
						at, err = probe(ctx, row)
						failed = err != nil || ctx.Err() != nil
					}
					cancel()
				} else if !rowsKnown || passContext.Err() != nil {
					failed = true
				}
				if wrapper != nil {
					if work := wrapper(row); !work.IsZero() {
						if failed && previous[row.Address].After(at) {
							at = previous[row.Address]
						}
						if work.After(at) {
							at = work
						}
						failed = false
					}
				}
				if failed {
					result.failed[row.Address] = true
				} else {
					delete(result.failed, row.Address)
					if !at.IsZero() {
						result.observed[row.Address] = at
					}
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
	if result.rowsGeneration == c.activityRowsGeneration {
		c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventActivity, Activity: result.observed, ActivityFailed: result.failed})
	}
	c.mu.Unlock()
	if len(result.observed)+len(result.failed) > 0 {
		c.traceEvent(traceActivity, couchcore.ThreadAddress{}, "ok="+strconv.Itoa(len(result.observed))+" failed="+strconv.Itoa(len(result.failed)))
	}
	c.advanceActivity(RefreshScheduleEvent{Kind: RefreshFinished, Generation: result.generation})
	c.repaintVisible()
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
