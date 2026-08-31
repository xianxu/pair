package couchtty

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
)

// ActionableThreadProvider is the Console's only inventory I/O seam. The
// Console supplies an immutable snapshot of exact hosted-process evidence.
type ActionableThreadProvider func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error)

type menuRefreshResult struct {
	generation uint64
	inventory  []couchcore.ActionableThreadSummary
	err        error
}

type menuPreviewResult struct {
	generation uint64
	prepared   *couchcore.PreparedStart
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
		c.showMenu()
	}
}

// reduceMenu is the Console's single bridge from semantic events to the pure
// menu. Effects are dispatched only after releasing the state mutex; rendering
// takes another immutable snapshot so neither terminal IO nor operations hold
// the Console lock.
func (c *Console) reduceMenu(event MenuEvent) {
	c.mu.Lock()
	if !c.menuReady {
		c.menu = NewMenuState(nil, couchcore.ThreadAddress{})
		c.menu.Notice = infoMenuNotice("thread inventory unavailable")
		c.menuReady = true
	}
	var effects []MenuEffect
	c.menu, effects = ReduceMenu(c.menu, event)
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()

	if panelFocused {
		c.showMenu()
	}
	c.dispatchMenuEffects(effects)
}

func (c *Console) onMenuKey(key PanelKey) {
	c.reduceMenu(MenuEvent{Kind: MenuEventKey, Key: key})
}

func (c *Console) onMenuInput(raw []byte) {
	buf := raw
	if len(c.panelHeld) > 0 {
		buf = append(append([]byte(nil), c.panelHeld...), raw...)
		c.panelHeld = nil
	}
	keys, held := DecodePanelKeys(buf)
	c.panelHeld = held
	for _, key := range keys {
		c.onMenuKey(key)
	}
}

func (c *Console) showMenu() {
	c.mu.Lock()
	state := cloneMenuState(c.menu)
	size := c.size
	c.mu.Unlock()
	height := int(size.Rows) - 1
	if height < 1 {
		height = 1
	}
	view := RenderMenuView(state, int(size.Cols), height, time.Now(), true)
	_, _ = c.host.Write([]byte(hostty.HideCursor))
	c.takeOverScreen([]byte(view.Body))
	c.paintNow()
	if view.Cursor == nil {
		_, _ = c.host.Write([]byte(hostty.HideCursor))
		return
	}
	_, _ = io.WriteString(c.host, hostty.MoveTo(view.Cursor.Row, view.Cursor.Col)+hostty.ShowCursor)
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

// dispatchMenuEffects is the thin stateful shell around the pure menu. Preview
// requests enter the bounded preview scheduler; declared operations reuse the
// Console's one sequential operation queue.
func (c *Console) dispatchMenuEffects(effects []MenuEffect) {
	for _, effect := range effects {
		if effect.Preview != nil {
			c.advanceMenuPreview(PreviewScheduleEvent{Kind: PreviewRequested, Request: *effect.Preview})
			continue
		}
		if effect.Operation != "" {
			c.runMenuOperation(effect)
		}
	}
}

func (c *Console) advanceMenuPreview(event PreviewScheduleEvent) {
	c.mu.Lock()
	var effects []PreviewScheduleEffect
	c.previewSchedule, effects = AdvancePreviewSchedule(c.previewSchedule, event)
	c.mu.Unlock()
	for _, effect := range effects {
		switch effect.Kind {
		case PreviewCancel:
			c.mu.Lock()
			cancel := c.previewCancel
			matches := c.previewRunning == effect.Generation
			c.mu.Unlock()
			if matches && cancel != nil {
				cancel()
			}
		case PreviewStart:
			c.startMenuPreview(effect.Request)
		}
	}
}

func (c *Console) startMenuPreview(request PreviewRequest) {
	ctx, cancel := context.WithCancel(c.lifetime)
	c.mu.Lock()
	c.previewCancel = cancel
	c.previewRunning = request.Generation
	fn := c.ops
	c.mu.Unlock()
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		args := map[string]string{"path": request.Path}
		if request.Agent != "" {
			args["agent"] = request.Agent
		}
		var prepared *couchcore.PreparedStart
		var err error
		if fn == nil {
			err = errors.New("no action dispatcher wired")
		} else {
			var value any
			value, err = fn(couchcore.OperationCall{
				Name: "prepare-start", Args: args, Implicit: true, Context: ctx,
			})
			if err == nil {
				accepted, ok := value.(couchcore.PreparedStart)
				if !ok {
					err = errors.New("prepare-start returned an invalid result")
				} else {
					prepared = &accepted
				}
			}
		}
		result := menuPreviewResult{generation: request.Generation, prepared: prepared, err: err}
		select {
		case c.previewResults <- result:
		case <-c.stop:
		}
	}()
}

func (c *Console) finishMenuPreview(result menuPreviewResult) {
	c.mu.Lock()
	if c.previewRunning == result.generation {
		if c.previewCancel != nil {
			c.previewCancel()
		}
		c.previewCancel = nil
		c.previewRunning = 0
	}
	var menuEffects []MenuEffect
	if c.menuReady {
		event := MenuEvent{Kind: MenuEventPreviewResult, Generation: result.generation, Prepared: result.prepared}
		if result.err != nil {
			event.Error = result.err.Error()
		}
		c.menu, menuEffects = ReduceMenu(c.menu, event)
	}
	c.mu.Unlock()
	c.advanceMenuPreview(PreviewScheduleEvent{Kind: PreviewFinished, Generation: result.generation})
	c.dispatchMenuEffects(menuEffects)
	c.mu.Lock()
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if panelFocused {
		c.showMenu()
	}
}
