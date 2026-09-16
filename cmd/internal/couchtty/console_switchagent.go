package couchtty

import (
	"context"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

type orientationWatch struct {
	request orientation.Request
	cancel  context.CancelFunc
	childID string
}
type orientationWatchResult struct {
	address couchcore.ThreadAddress
	request orientation.Request
	state   orientation.DeliveryState
	err     error
}

func (c *Console) watchOrientation(address couchcore.ThreadAddress, childID string, request orientation.Request) {
	c.mu.Lock()
	if c.orientationWatches == nil {
		c.orientationWatches = make(map[couchcore.ThreadAddress]orientationWatch)
	}
	if prior, ok := c.orientationWatches[address]; ok {
		prior.cancel()
	}
	ctx, cancel := context.WithTimeout(c.lifetime, 30*time.Second)
	c.orientationWatches[address] = orientationWatch{request: request, cancel: cancel, childID: childID}
	if c.menu.Orientation == nil {
		c.menu.Orientation = make(map[couchcore.ThreadAddress]orientation.Request)
	}
	c.menu.Orientation[address] = request
	fn := c.ops
	c.mu.Unlock()
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		defer cancel()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		result := orientationWatchResult{address: address, request: request}
		for {
			if fn == nil {
				result.err = fmt.Errorf("orientation status unavailable")
				break
			}
			value, err := fn(couchcore.OperationCall{Name: "orientation-status", Implicit: true, Context: ctx, Args: map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag), "agent": request.Agent, "attempt": request.Attempt}})
			if err != nil {
				result.err = err
				break
			}
			status, ok := value.(orientation.DeliveryState)
			if !ok {
				result.err = fmt.Errorf("invalid orientation status")
				break
			}
			result.state = status
			if status.Terminal() {
				break
			}
			c.mu.Lock()
			_, live := c.panes[childID]
			c.mu.Unlock()
			if !live {
				result.err = fmt.Errorf("target exited before orientation delivery")
				break
			}
			select {
			case <-ctx.Done():
				result.err = ctx.Err()
			case <-ticker.C:
				continue
			}
			break
		}
		select {
		case c.orientationResults <- result:
		case <-c.stop:
		}
	}()
}

func (c *Console) finishOrientation(result orientationWatchResult) {
	c.mu.Lock()
	watch, ok := c.orientationWatches[result.address]
	if !ok || watch.request.Attempt != result.request.Attempt {
		c.mu.Unlock()
		return
	}
	delete(c.orientationWatches, result.address)
	watch.cancel()
	text := "orientation prompt submitted"
	if result.err != nil || result.state.Phase != orientation.DeliverySubmitted {
		text = "Orientation delivery was not confirmed. Inspect the target before using Copy orientation prompt in this thread's actions."
		if result.state.BodyMayBePresent() {
			text = "Orientation text may already be in the composer; inspect it before submitting. Copy orientation prompt is available in actions."
		}
		if result.err != nil {
			text += " " + result.err.Error()
		} else if result.state.Reason != "" {
			text += " " + result.state.Reason
		}
	} else {
		delete(c.menu.Orientation, result.address)
	}
	text = string(result.address.Tag) + ": " + text
	setBookkeepingNotice(&c.menu, text)
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	if panel {
		c.showMenu()
	} else {
		c.setNotice(text)
	}
}

func (c *Console) copyOrientation(request orientation.Request) {
	if err := c.presenter.Copy(c.lifetime, []byte(request.Body)); err != nil {
		c.terminalError(err)
		return
	}

	c.mu.Lock()
	c.menu.Notice = infoMenuNotice("Clipboard copy requested; your terminal may not support it.")
	panel := c.focus.IsPanel()
	c.mu.Unlock()
	if panel {
		c.showMenu()
	}
}
