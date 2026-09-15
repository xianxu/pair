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

func (c *Console) chrome() (RenderedStatusRow, []terminal.Cell, error) {
	c.mu.Lock()
	cols, model := int(c.size.Cols), c.statusModelLocked()
	c.mu.Unlock()
	row := RenderStatusRow(cols, model)
	cells, err := terminal.StyledRows(row.Body, cols, 1)
	return row, cells, err
}

func (c *Console) commitChrome(row RenderedStatusRow) {
	c.mu.Lock()
	c.statusChips = row.Chips
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
	c.mu.Lock()
	pane := c.panes[id]
	already := c.focus == FocusActor(id) && c.active == id && !force
	size := c.size
	c.mu.Unlock()
	if pane == nil {
		return false, fmt.Errorf("terminal %q is not attached", id)
	}
	if !already {
		c.mu.Lock()
		model := c.statusModelLocked()
		c.mu.Unlock()
		for i := range model.Actors {
			model.Actors[i].Active = model.Actors[i].Thread == pane.thread
		}
		row := RenderStatusRow(int(size.Cols), model)
		cells, err := terminal.StyledRows(row.Body, int(size.Cols), 1)
		if err != nil {
			return false, err
		}
		if err = c.presenter.Select(ctx, pane.child.Endpoint(), terminal.Geometry{Cols: int(size.Cols), Rows: int(size.Rows)}, cells); err != nil {
			c.traceTerminal("select", err)
			return false, err
		}
		c.commitChrome(row)

	}
	c.mu.Lock()
	c.active, c.focus = id, FocusActor(id)
	c.menu.ActiveAddress = pane.thread
	c.tracker.Switch(pane.thread, how.viaNotification())
	c.attention.Acknowledge(c.attention.Capture(pane.thread))
	c.syncAttentionLocked()
	c.mu.Unlock()
	c.traceTerminal("select", nil)
	// Selection changes the status model; updating chrome preserves the admitted
	// endpoint and any gesture when this was a same-actor acknowledgment.
	c.paintNow()
	return already, nil
}
