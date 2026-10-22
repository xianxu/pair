package couchtty

// openGroupSwitcher opens the existing root panel without moving actor focus.
// A zero identity denotes global scarcity rather than one presentation group.
// Runs on Console.Run, like onHotkey and mouse routing.
func (c *Console) openGroupSwitcher(group StatusGroupID) {
	c.mu.Lock()
	c.menu.LandingAttempt = 0
	c.navigationGeneration++
	c.focus = FocusPanel()
	if len(c.menu.Frames) == 0 {
		appendMenuFrame(&c.menu, MenuFrame{Kind: MenuFrameRoot})
	}
	c.menu.Frames = c.menu.Frames[:1]
	root := &c.menu.Frames[0]
	root.Filter = ""
	root.View = MenuViewNormal
	root.GroupScope = nil
	if group != (StatusGroupID{}) {
		root.GroupScope = &group
	}
	reconcileRootSelection(&c.menu, c.menu.ActiveAddress)
	c.mu.Unlock()
	c.requestMenuRefresh()
	c.showMenu()
}
