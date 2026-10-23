package couchtty

// SetTabStyle configures chrome before Console adoption and its first geometry.
func (c *Console) SetTabStyle(style TabStyle) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tabStyle = style
}

// TabStyle returns the selected launch-local packing policy.
func (c *Console) TabStyle() TabStyle {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tabStyle
}
