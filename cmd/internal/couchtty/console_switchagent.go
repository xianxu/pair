package couchtty

import "github.com/xianxu/pair/cmd/internal/orientation"

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
