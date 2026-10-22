package couchtty

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/clipcmd"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
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
	recoveryPreview *couchcore.RecoveryPreview
	generation      uint64
	prepared        *couchcore.PreparedStart
	switchPrepared  *couchcore.PreparedAgentSwitch
	err             error
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
		if p == nil || p.child.Endpoint().InputEnded() || p.process.PID <= 0 || p.process.Identity == "" {
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
	var effects []MenuEffect
	seeded, pending := false, 0
	var root couchcore.ThreadAddress
	c.mu.Lock()
	cancelActivity := c.invalidateActivityRowsLocked(result.inventory, result.err == nil)
	if result.err == nil {
		c.resourceRows = cloneResourceRows(result.inventory)
		c.resourceRowsKnown = true
		c.resourcesChangedLocked()
	}
	if c.menuReady {
		event := MenuEvent{Kind: MenuEventInventory, Inventory: result.inventory, Generation: result.generation}
		if result.err != nil {
			event.Error = result.err.Error()
		}
		armed := c.menu.Reattach.Phase == ReattachArmed
		c.menu, effects = ReduceMenu(c.menu, event)
		if armed && c.menu.Reattach.Phase != ReattachArmed {
			seeded, root, pending = true, c.menu.Reattach.Root, len(pendingPlaceholders(c.menu.Reattach))
		}
	}
	cancelPass := c.completedReattachPassLocked()
	c.mu.Unlock()
	if cancelActivity != nil {
		cancelActivity()
	}
	if result.err == nil {
		c.observeLifecycleInventory(result.inventory)
	}
	c.observeLifecyclePass()
	landed := "rows=" + strconv.Itoa(len(result.inventory))
	if cancelPass != nil {
		cancelPass()
	}
	if result.err != nil {
		landed = "error"
	}
	c.traceEvent(traceInventory, couchcore.ThreadAddress{}, landed)
	if seeded {
		c.traceEvent(tracePassSeeded, root, "pending="+strconv.Itoa(pending))
	}
	// An inventory can seed the reattach pass and start its first attempt
	// (pair#206). Nothing else this event produces is an effect. It is traced
	// above, before dispatch, so the seeding precedes the first attempt's start.
	c.dispatchMenuEffects(effects)
	c.advanceMenuRefresh(RefreshScheduleEvent{Kind: RefreshFinished, Generation: result.generation})
	// A landed inventory can add or remove slot checkouts; the switcher opening
	// (onHotkey) reaches the slot git refresh through here too (pair#317).
	if result.err == nil {
		c.requestSlotGit()
		// The first inventory is when live threads become known, so the idle
		// fades land without waiting a full activity interval (pair#247).
		c.requestActivity()
	}
	c.mu.Lock()
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	if panelFocused {
		c.showMenu()
	} else if result.err == nil {
		// Inventory now supplies tab grouping as well as switcher rows. Publish
		// accepted metadata even when the operator stays in a quiet child.
		c.repaint()
	}
}

// reduceMenu is the Console's single bridge from semantic events to the pure
// menu. Effects are dispatched only after releasing the state mutex; rendering
// takes another immutable snapshot so neither terminal IO nor operations hold
// the Console lock.
func (c *Console) reduceMenu(event MenuEvent) {
	event.At = c.now()
	c.mu.Lock()
	if event.Kind == MenuEventKey || event.Kind == MenuEventMouseSwitch || event.Kind == MenuEventParkHotkey {
		c.navigationGeneration++
	}
	c.ensureMenuLocked(couchcore.ThreadAddress{})
	var effects []MenuEffect
	c.menu, effects = ReduceMenu(c.menu, event)
	panelFocused := c.focus.IsPanel()
	cancelPass := c.completedReattachPassLocked()
	c.mu.Unlock()

	if cancelPass != nil {
		cancelPass()
	}
	if panelFocused {
		c.showMenu()
	}
	c.dispatchMenuEffects(effects)
}

func (c *Console) onMenuKey(key PanelKey) {
	if key.Kind == KeyCopyNotice {
		c.copyLastNotice()
		return
	}
	c.reduceMenu(MenuEvent{Kind: MenuEventKey, Key: key})
}

func (c *Console) onMenuInput(raw []byte) {
	buf := raw
	if len(c.menuHeld) > 0 {
		buf = append(append([]byte(nil), c.menuHeld...), raw...)
		c.menuHeld = nil
	}
	keys, held := DecodePanelKeys(buf)
	c.menuHeld = held
	for _, key := range keys {
		c.onMenuKey(key)
	}
}

func (c *Console) showMenu() {
	c.publishMu.Lock()
	defer c.publishMu.Unlock()
	c.showMenuPublishing()
}

// showMenuPublishing is showMenu for a caller already holding publishMu.
func (c *Console) showMenuPublishing() {
	c.mu.Lock()
	state, size := cloneMenuState(c.menu), c.size
	c.mu.Unlock()
	bar, bottom, err := c.chrome()
	if err != nil {
		c.terminalError(err)
		return
	}
	height := max(1, int(size.Rows)-bar.Height())
	view := RenderMenuView(state, int(size.Cols), height, c.now(), true)
	cells, err := terminal.StyledRows(view.Body, int(size.Cols), height)
	if err != nil {
		c.terminalError(err)
		return
	}
	cells = append(cells, bottom...)
	cursor := terminal.Cursor{}
	if view.Cursor != nil {
		cursor = terminal.Cursor{X: view.Cursor.Col - 1, Y: view.Cursor.Row - 1, Visible: true}
	}
	frame, err := terminal.PanelFrame(terminal.Geometry{Cols: int(size.Cols), Rows: int(size.Rows)}, cells, cursor)
	if err == nil {
		frame.ChromeRows = bar.Height()
		c.beforePublish("panel")
		c.traceCaptureTransition("panel-start", "", nil)
		// Private: the switcher lists the whole fleet; a broadcast shows a
		// placeholder instead (#395).
		err = c.presenter.Panel(c.lifetime, frame, terminal.FramePrivate)
		c.traceCaptureTransition("panel-end", "", err)
	}
	if err != nil {
		c.terminalError(err)
		return
	}
	c.mu.Lock()
	c.menuExtents = view.Extents
	c.focus = FocusPanel()
	c.mu.Unlock()
	c.commitChrome(bar)
	c.terminalError(c.resizeBackgroundChildren(chromeChildSize(size, bar.Height()), nil))
}

// dispatchMenuEffects is the thin stateful shell around the pure menu. Preview
// requests enter the bounded preview scheduler; declared operations reuse the
// Console's bounded lifecycle and hosted-navigation lanes.
func (c *Console) dispatchMenuEffects(effects []MenuEffect) {
	for _, effect := range effects {
		if effect.CancelPreview != 0 {
			c.mu.Lock()
			if c.previewSchedule.Pending != nil && c.previewSchedule.Pending.Generation == effect.CancelPreview {
				c.previewSchedule.Pending = nil
			}
			cancel := c.previewCancel
			matches := c.previewRunning == effect.CancelPreview
			c.mu.Unlock()
			if matches && cancel != nil {
				cancel()
			}
			continue
		}
		if effect.Completion != nil {
			c.advanceMenuCompletion(latestScheduleEvent[CompletionRequest, CompletionIdentity]{Kind: latestRequested, Request: *effect.Completion})
			continue
		}
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
	if request.RecoveryTarget != nil {
		cancel()
		ctx, cancel = context.WithTimeout(c.lifetime, 35*time.Second)
	}
	c.mu.Lock()
	c.previewCancel = cancel
	c.previewRunning = request.Generation
	fn := c.ops
	c.mu.Unlock()
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		args := map[string]string{"path": request.Path}
		if request.Action != "" {
			args["action"] = string(request.Action)
		}
		if request.Agent != "" {
			args["agent"] = request.Agent
		}
		var switchPrepared *couchcore.PreparedAgentSwitch
		operation := "prepare-start"
		if request.SwitchAddress != (couchcore.ThreadAddress{}) {
			operation = "prepare-switch-agent"
			args = map[string]string{"repo-scope": request.SwitchAddress.RepoScope, "tag": string(request.SwitchAddress.Tag), "agent": request.Agent}
			if request.SwitchArgv != "" {
				args["argv"] = request.SwitchArgv
			}
		}
		if request.RecoveryTarget != nil {
			operation = "resume-candidates"
			args = map[string]string{}
			if request.RecoveryTarget.Path != "" {
				args["path"] = request.RecoveryTarget.Path
			} else {
				args["repo-scope"] = request.RecoveryTarget.Address.RepoScope
				args["tag"] = string(request.RecoveryTarget.Address.Tag)
			}
		}
		var recoveryPreview *couchcore.RecoveryPreview
		var prepared *couchcore.PreparedStart
		var err error
		if fn == nil {
			err = errors.New("no action dispatcher wired")
		} else {
			var value any
			value, err = fn(couchcore.OperationCall{
				Name: operation, Args: args, Implicit: true, Context: ctx,
			})
			if err == nil && operation == "resume-candidates" {
				accepted, ok := value.(couchcore.RecoveryPreview)
				if !ok {
					err = errors.New("invalid recovery preview result")
				} else {
					recoveryPreview = &accepted
				}
			} else if err == nil && operation == "prepare-switch-agent" {
				accepted, ok := value.(couchcore.PreparedAgentSwitch)
				if !ok {
					err = errors.New("invalid switch preview result")
				} else {
					switchPrepared = &accepted
				}
			} else if err == nil {
				accepted, ok := value.(couchcore.PreparedStart)
				if !ok {
					err = errors.New("prepare-start returned an invalid result")
				} else {
					prepared = &accepted
				}
			}
		}
		result := menuPreviewResult{generation: request.Generation, prepared: prepared, switchPrepared: switchPrepared, recoveryPreview: recoveryPreview, err: err}
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
		event := MenuEvent{Kind: MenuEventPreviewResult, Generation: result.generation, Prepared: result.prepared, SwitchPrepared: result.switchPrepared, RecoveryPreview: result.recoveryPreview}
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

// CopyObservation is content-free: native stderr remains on the local UI only.
// OSC acceptance is explicitly unconfirmed; it never proves clipboard delivery.
type CopyObservation struct {
	Outcome string                        `json:"outcome"`
	Native  clipcmd.ClipboardCopyMetadata `json:"native"`
	OSC     string                        `json:"osc"`
}

func (c *Console) SetCopyObserver(observer func(CopyObservation)) {
	c.mu.Lock()
	c.copyObserver = observer
	c.mu.Unlock()
}

// copyLastNotice is owned by the Console input loop. IO stays outside mu and
// a held failure keeps the original notice even when newer feed items arrive.
func (c *Console) copyLastNotice() {
	if c.lifetime.Err() != nil {
		return
	}
	c.mu.Lock()
	feedback := c.menu.CopyFeedback
	text, found := feedback.payload, feedback.held()
	if !found {
		text, found = selectedMenuOperationDetail(c.menu)
	}
	if !found && c.menu.Notice.Level == MenuNoticeError && c.menu.Notice.Text != "" {
		text, found = c.menu.Notice.Text, true
	}
	if !found {
		messages := c.feed.Messages()
		if len(messages) > 0 {
			text, found = messages[len(messages)-1].Body, true
		}
	}
	observer := c.copyObserver
	c.mu.Unlock()

	observation := CopyObservation{Outcome: "empty", OSC: "not-requested"}
	next := CopyFeedback{Kind: CopyFeedbackEmpty, Text: "No notice to copy."}
	if found {
		ctx, cancel := context.WithTimeout(c.lifetime, time.Second)
		result, localErr := clipcmd.NewOSRuntime().ClipboardCopyLocalResult(ctx, text)
		cancel()
		observation.Native = result.Metadata
		if localErr == nil {
			observation.Outcome = "copied"
			next = CopyFeedback{Kind: CopyFeedbackSuccess, Text: "Full notice copied to clipboard."}
		} else {
			next = CopyFeedback{Kind: CopyFeedbackUnconfirmed, payload: text}
			fallbackErr := c.presenter.Copy(c.lifetime, []byte(text))
			if fallbackErr != nil {
				observation.Outcome, observation.OSC = "failed", "failed"
				next.Kind = CopyFeedbackFailed
				next.Text = "Copy failed: " + localErr.Error() + "; terminal fallback: " + fallbackErr.Error()
			} else {
				observation.Outcome, observation.OSC = "unconfirmed", "unconfirmed"
				next.Text = "Local copy failed: " + localErr.Error() + "; terminal copy requested; delivery is unconfirmed."
			}
		}
	}
	if observer != nil {
		observer(observation)
	}
	c.mu.Lock()
	if c.lifetime.Err() != nil {
		c.mu.Unlock()
		return
	}
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventCopyResult, CopyFeedback: next})
	c.mu.Unlock()
	c.showMenu()
}
