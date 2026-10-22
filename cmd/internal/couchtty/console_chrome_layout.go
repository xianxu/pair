package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func renderChrome(size ptychild.Size, model StatusModel) (RenderedStatusBar, []terminal.Cell, error) {
	bar := RenderStatusBar(int(size.Cols), min(2, max(0, int(size.Rows)-1)), model)
	if bar.Height() == 0 {
		return bar, nil, nil
	}
	cells, err := terminal.StyledRows(bar.Body(), int(size.Cols), bar.Height())
	return bar, cells, err
}
func chromeChildSize(size ptychild.Size, rows int) ptychild.Size {
	return ptychild.Size{Cols: size.Cols, Rows: uint16(max(1, int(size.Rows)-rows))}
}

// childSizeLocked predicts birth geometry with the same pure planner as paint.
func (c *Console) childSizeLocked() ptychild.Size {
	bar := RenderStatusBar(int(c.size.Cols), min(2, max(0, int(c.size.Rows)-1)), c.statusModelLocked())
	return chromeChildSize(c.size, bar.Height())
}

// chromeConfirmed fences geometry-dependent input against the physical paint.
// Never call while holding c.mu: the presenter can call overlay readers.
func (c *Console) chromeConfirmed() bool {
	_, ok := c.confirmedChromeState()
	return ok
}
func (c *Console) confirmedChromeState() (terminal.ChromeState, bool) {
	c.publishMu.Lock()
	defer c.publishMu.Unlock()
	return c.confirmedChromeStatePublishing()
}

// confirmedChromeStatePublishing reads confirmation and the matching hit map
// while the caller owns publication, including admission through metadata commit.
func (c *Console) confirmedChromeStatePublishing() (terminal.ChromeState, bool) {
	state, err := c.presenter.ChromeState(c.lifetime)
	if err != nil {
		return state, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return state, state.Confirmed && state.Revision == c.statusRevision && state.Rows == c.statusRows && state.Host == (terminal.Geometry{Cols: int(c.size.Cols), Rows: int(c.size.Rows)})
}
func (c *Console) resizeBackgroundChildren(size ptychild.Size, selected *ptychild.Child) error {
	c.mu.Lock()
	children := make([]*ptychild.Child, 0, len(c.panes))
	for _, p := range c.panes {
		children = append(children, p.child)
	}
	c.mu.Unlock()
	for _, child := range children {
		if child == selected || child.Endpoint().InputEnded() || child.Size() == size {
			continue
		}
		if err := child.Resize(size); err != nil {
			return err
		}
	}
	return nil
}

// paintActorPublishing holds publishMu across layout admission and hit maps.
func (c *Console) paintActorPublishing(site string) {
	bar, cells, err := c.chrome()
	c.mu.Lock()
	size, selected := c.size, c.panes[c.active]
	c.mu.Unlock()
	desired := chromeChildSize(size, bar.Height())
	c.beforePublish(site)
	if err == nil {
		state, stateErr := c.presenter.ChromeState(c.lifetime)
		err = stateErr
		if err == nil {
			host := terminal.Geometry{Cols: int(size.Cols), Rows: int(size.Rows)}
			if state.Host != host || state.Rows != bar.Height() {
				err = c.presenter.ResizeLayout(c.lifetime, host, cells, func(g terminal.Geometry) error {
					if selected == nil {
						return nil
					}
					return selected.child.ResizePTY(ptychild.Size{Cols: uint16(g.Cols), Rows: uint16(g.Rows)})
				})
			} else {
				err = c.presenter.UpdateChrome(c.lifetime, cells)
			}
		}
	}
	if terminal.IsRoutingAnswer(err) {
		c.traceDropped(site, err)
		c.terminalError(c.resizeBackgroundChildren(desired, nil))
		return
	}
	if err != nil {
		c.terminalError(err)
		return
	}
	c.commitChrome(bar)
	var child *ptychild.Child
	if selected != nil {
		child = selected.child
	}
	c.terminalError(c.resizeBackgroundChildren(desired, child))
}
