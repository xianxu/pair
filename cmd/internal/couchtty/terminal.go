package couchtty

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

type terminalCommand struct {
	ctx  context.Context
	run  func() error
	done chan error
}

// Operation workers join the same ordering domain as operator input. The
// buffered completion lets cancellation abandon a caller without blocking Run.
func (c *Console) runTerminalCommand(ctx context.Context, run func() error) error {
	if ctx == nil {
		ctx = c.lifetime
	}
	c.mu.Lock()
	started := c.started
	c.mu.Unlock()
	if !started {
		return run()
	}
	command := terminalCommand{ctx: ctx, run: run, done: make(chan error, 1)}
	select {
	case c.terminalCommands <- command:
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop:
		return context.Canceled
	}
	select {
	case err := <-command.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop:
		return context.Canceled
	}
}

// shutdownCancellation accepts only error trees whose every leaf is the
// owner's cancellation. errors.Is would also match a joined physical failure.
func shutdownCancellation(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !shutdownCancellation(child) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return shutdownCancellation(wrapped.Unwrap())
	}
	return err == context.Canceled
}

// TerminalFailure is why the console's terminal failed, as teardown classified
// it: the same failure Run reported, with an owner shutdown's cancellation
// filtered out (pair#409 reads it after Run). It is nil before Run returns.
func (c *Console) TerminalFailure() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.exitFailure
}

func (c *Console) terminalError(err error) {
	if err != nil && !(c.lifetime.Err() != nil && shutdownCancellation(err)) {
		c.mu.Lock()
		if c.terminalFailure == nil {
			c.terminalFailure = err
		}
		c.mu.Unlock()
		c.Stop()
	}
}

func (c *Console) chrome() (RenderedStatusBar, []terminal.Cell, error) {
	c.mu.Lock()
	size, model := c.size, c.statusModelLocked()
	c.mu.Unlock()
	return renderChrome(size, model)
}

func (c *Console) commitChrome(bar RenderedStatusBar) {
	state, err := c.presenter.ChromeState(c.lifetime)
	if err != nil {
		c.terminalError(err)
		return
	}
	var row RenderedStatusRow
	if len(bar.Rows) > 0 {
		row = bar.Rows[0]
	}
	c.mu.Lock()
	c.statusRows, c.statusRevision = bar.Height(), state.Revision
	c.statusChips = bar.Chips
	c.statusControl = row.Control
	c.statusRemoteSession = row.RemoteSession
	c.statusPointer, c.statusNotes = row.Pointer, row.Notes
	c.statusWebcast, c.statusWebFocus, c.statusWebNotes, c.statusWebDrive = row.Webcast, row.WebFocus, row.WebNotes, row.WebDrive
	c.statusWebPointerLink, c.statusWebDriveLink = row.WebPointerLink, row.WebDriveLink
	c.statusAttention = row.Attention
	first := !c.framePainted
	c.framePainted = true
	var shown couchcore.ThreadAddress
	if p := c.panes[c.active]; p != nil {
		shown = p.thread
	}
	c.mu.Unlock()
	if first {
		c.traceEvent(traceFirstFrame, shown, "")
	}
}

// publishSelectionPublishing admits child geometry and its matching hit map.
// The caller holds publishMu.
func (c *Console) publishSelectionPublishing(ctx context.Context, pane *pane) error {
	c.mu.Lock()
	size, model := c.size, c.statusModelLocked()
	c.mu.Unlock()
	for i := range model.Actors {
		model.Actors[i].Active = model.Actors[i].Thread == pane.thread
	}
	bar, cells, err := renderChrome(size, model)
	if err != nil {
		return err
	}
	desired := chromeChildSize(size, bar.Height())
	if pane.child.Size() != desired && !pane.child.Endpoint().InputEnded() {
		if err = pane.child.Resize(desired); err != nil {
			return err
		}
	}
	c.traceCaptureTransition("select-start", pane.child.Endpoint().ID(), nil)
	err = c.presenter.Select(ctx, pane.child.Endpoint(), terminal.Geometry{Cols: int(size.Cols), Rows: int(size.Rows)}, cells)
	c.traceCaptureTransition("select-end", pane.child.Endpoint().ID(), err)
	if err != nil {
		c.traceTerminal("select", err)
		return err
	}
	c.commitChrome(bar)
	return c.resizeBackgroundChildren(desired, pane.child)
}

func (c *Console) selectActor(id string, force bool, how arrival) (bool, error) {
	return c.selectActorContext(c.lifetime, id, force, how)
}

func (c *Console) selectActorContext(ctx context.Context, id string, force bool, how arrival) (bool, error) {
	if ctx == nil {
		ctx = c.lifetime
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	// The presenter's selection and the console's focus change as one step
	// under publishMu: a paint between them would read the old focus and
	// re-panel over the endpoint just selected (pair#484).
	c.publishMu.Lock()
	c.mu.Lock()
	pane := c.panes[id]
	already := c.focus == FocusActor(id) && c.active == id && !force
	c.mu.Unlock()
	if pane == nil {
		c.publishMu.Unlock()
		return false, fmt.Errorf("terminal %q is not attached", id)
	}
	if !already {
		if err := c.publishSelectionPublishing(ctx, pane); err != nil {
			c.publishMu.Unlock()
			return false, err
		}
	}
	c.beforePublish("select")
	c.mu.Lock()
	c.menu.LandingAttempt = 0
	c.navigationGeneration++
	c.active, c.focus = id, FocusActor(id)
	c.menu.ActiveAddress = pane.thread
	c.tracker.Switch(pane.thread, how.viaNotification())
	c.attention.Acknowledge(c.attention.Capture(pane.thread))
	c.syncAttentionLocked()
	c.mu.Unlock()
	c.publishMu.Unlock()
	c.traceTerminal("select", nil)
	// Selection changes the status model; updating chrome preserves the admitted
	// endpoint and any gesture when this was a same-actor acknowledgment.
	c.paintNow()
	return already, nil
}
