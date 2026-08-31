package couchtty

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// ActionableThreadProvider is the Console's only inventory I/O seam. The
// Console supplies an immutable snapshot of exact hosted-process evidence.
type ActionableThreadProvider func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error)

type menuRefreshResult struct {
	generation uint64
	inventory  []couchcore.ActionableThreadSummary
	err        error
}

func (c *Console) SetActionableProvider(provider ActionableThreadProvider) {
	c.mu.Lock()
	c.actionable = provider
	c.mu.Unlock()
	c.requestMenuRefresh()
}

func (c *Console) ActionableProvider() ActionableThreadProvider {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.actionable
}

func (c *Console) menuSnapshot() MenuState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneMenuState(c.menu)
}

func (c *Console) snapshotMenuObservations() []couchcore.LiveTTYObservation {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotMenuObservationsLocked()
}

func (c *Console) snapshotMenuObservationsLocked() []couchcore.LiveTTYObservation {
	observations := make([]couchcore.LiveTTYObservation, 0, len(c.order))
	for _, id := range c.order {
		p := c.panes[id]
		if p == nil || p.child.Done() || p.process.PID <= 0 || p.process.Identity == "" {
			continue
		}
		observations = append(observations, couchcore.LiveTTYObservation{Address: p.thread, Process: p.process})
	}
	return observations
}

func (c *Console) requestMenuRefresh() {
	c.mu.Lock()
	available := c.actionable != nil
	c.mu.Unlock()
	if !available {
		return
	}
	select {
	case c.refreshRequests <- struct{}{}:
	default:
	}
}

// advanceMenuRefresh runs on Console.Run, which is the sole owner of schedule
// ordering. Provider I/O runs outside the Console mutex and reports one typed
// terminal result back to the same loop.
func (c *Console) advanceMenuRefresh(event RefreshScheduleEvent) {
	var effects []RefreshScheduleEffect
	c.refreshSchedule, effects = AdvanceRefreshSchedule(c.refreshSchedule, event)
	for _, effect := range effects {
		if effect.Kind != RefreshStart {
			continue
		}
		c.mu.Lock()
		provider := c.actionable
		observations := c.snapshotMenuObservationsLocked()
		if c.menuReady {
			c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventRefreshStarted})
		}
		c.mu.Unlock()
		if provider == nil {
			select {
			case c.refreshResults <- menuRefreshResult{generation: effect.Generation, err: context.Canceled}:
			case <-c.stop:
			}
			continue
		}
		c.workers.Add(1)
		go func(generation uint64, observations []couchcore.LiveTTYObservation) {
			defer c.workers.Done()
			inventory, err := provider(c.lifetime, observations)
			result := menuRefreshResult{generation: generation, inventory: inventory, err: err}
			select {
			case c.refreshResults <- result:
			case <-c.stop:
			}
		}(effect.Generation, observations)
	}
}

func (c *Console) finishMenuRefresh(result menuRefreshResult) {
	if c.refreshSchedule.Running != result.generation {
		return
	}
	c.mu.Lock()
	if c.menuReady {
		event := MenuEvent{Kind: MenuEventInventory, Inventory: result.inventory}
		if result.err != nil {
			event.Error = result.err.Error()
		}
		c.menu, _ = ReduceMenu(c.menu, event)
	}
	c.mu.Unlock()
	c.advanceMenuRefresh(RefreshScheduleEvent{Kind: RefreshFinished, Generation: result.generation})
	c.mu.Lock()
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if panelFocused {
		c.rebuildPanel()
		c.showPanel()
	}
}

func panelSummariesFromActionable(rows []couchcore.ActionableThreadSummary) []couchcore.ThreadSummary {
	summaries := make([]couchcore.ThreadSummary, len(rows))
	for i, row := range rows {
		summaries[i] = couchcore.ThreadSummary{
			Address: row.Address, StartingPath: row.StartingPath, WorkingPath: row.WorkingPath,
			Name: row.Name, Description: row.Description, PublishedSummary: row.PublishedSummary,
		}
		if row.Live() {
			summaries[i].Incarnations = []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}
		}
	}
	return summaries
}

func actionableMemoryResolver(rows []couchcore.ActionableThreadSummary) func(string) ([]couchcore.ThreadAddress, error) {
	fields := make([]couchcore.ThreadReferenceFields, len(rows))
	for i, row := range rows {
		fields[i] = couchcore.ThreadReferenceFields{Address: row.Address, Name: row.Name, WorkingPath: row.WorkingPath}
	}
	return func(ref string) ([]couchcore.ThreadAddress, error) {
		addresses, err := couchcore.MatchThreadReferenceFields(fields, ref)
		if errors.Is(err, couchcore.ErrThreadReferenceNotFound) {
			return nil, nil
		}
		return addresses, err
	}
}
