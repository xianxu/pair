package couchtty

import "github.com/xianxu/pair/cmd/internal/couchcore"

// statusModelLocked joins attached and pending members to the same projection
// used by the switcher. Callers hold c.mu; no discovery or process IO occurs.
func (c *Console) statusModelLocked() StatusModel {
	model := StatusModel{TabStyle: c.tabStyle, Broadcast: c.broadcastCellLocked(), Pointer: c.pointerCellLocked(), Webcast: c.webcastCellLocked(), WebNotes: c.webNoteCount(), Notes: len(c.drawnNotes()), Notice: c.feed.Row().Body, Spinner: c.statusSpinner, Palette: c.menu.Palette, Capture: c.capture.Status()}
	if c.wcast.cast != nil {
		caps := c.wcast.cast.Capabilities()
		model.WebAnnotationOn = caps.Project("annotation").Enabled
		model.WebDrivingOn = caps.Project("drive").Enabled
		model.CapabilityNotice = capabilityNotice(caps)
	}

	c.attentionStatusLocked(&model)

	driver, asking := c.webDriveLabels()
	model.WebDrive = asking
	if model.WebDrivingOn {
		model.WebDriverName = driver
	}

	now := c.now()
	rows := menuRows(c.menu)
	byAddress := make(map[couchcore.ThreadAddress]int, len(rows))
	slotsByPath := make(map[string]couchcore.ThreadTarget)
	for i, row := range rows {
		if row.Address != (couchcore.ThreadAddress{}) {
			byAddress[row.Address] = i
		}
		if row.Target.Kind == couchcore.ThreadTargetSlot && row.Target.Validate() == nil {
			slotsByPath[row.Target.Slot.WorktreeRoot] = row.Target
		}
	}
	members := make(map[couchcore.ThreadAddress]StatusActor, len(c.panes))
	for _, id := range c.order {
		p := c.panes[id]
		at, known := c.menu.Activity[p.thread]
		actor := StatusActor{Thread: p.thread, Active: id == c.active, Bell: len(c.attention.Projection(p.thread)) > 0,
			Blocked: c.menu.Blocked[p.thread], Idle: IdleLevelFor(now, at, known)}
		index, found := byAddress[p.thread]
		if !found {
			// A slot's chip is its current row, as in the switcher (pair#607):
			// a pane left on a replaced address (a reboot's old thread) gets
			// none, so one slot never shows two chips.
			if _, slot := slotsByPath[string(p.tree)]; slot {
				continue
			}
			rows = append(rows, couchcore.ActionableThreadSummary{Address: p.thread, StartingPath: string(p.tree), WorkingPath: string(p.tree)})
			index = len(rows) - 1
		}
		members[p.thread] = actor
		// A snapshot can lag attachment. The pane's canonical tree supplies missing
		// context only; an observed slot target and its durable row key are retained.
		if rows[index].StartingPath == "" {
			rows[index].StartingPath = string(p.tree)
		}
		if rows[index].WorkingPath == "" {
			rows[index].WorkingPath = string(p.tree)
		}
	}
	for _, pending := range pendingPlaceholders(c.menu.Reattach) {
		if _, attached := members[pending.Address]; attached {
			continue
		}
		members[pending.Address] = StatusActor{Thread: pending.Address, Placeholder: true, Loading: pending.Loading}
		if index, found := byAddress[pending.Address]; !found {
			rows = append(rows, couchcore.ActionableThreadSummary{Address: pending.Address})
		} else if rows[index].WorkingPath == "" {
			// Labelled as the chip will be -- the starting path's repository
			// -- so the label does not change when the thread arrives.
			rows[index].WorkingPath = rows[index].StartingPath
		}
	}
	for _, entry := range PresentThreads(rows, c.menu.SlotGit, c.menu.Hub, c.menu.HubOwnership) {
		actor, found := members[entry.Row.Address]
		if !found {
			continue
		}
		actor.Label, actor.GroupKey, actor.SlotNumber, actor.Glyph, actor.Hub = entry.Label, entry.GroupKey, entry.SlotNumber, entry.Glyph, entry.Hub
		actor.OwnerHub = entry.OwnerHub
		model.Actors = append(model.Actors, actor)
	}
	activeHub := ""
	for _, a := range model.Actors {
		if a.Active {
			activeHub = a.OwnerHub
			break
		}
	}
	for i := range model.Actors {
		a := &model.Actors[i]
		a.OtherHub = activeHub != "" && a.OwnerHub != "" && a.OwnerHub != activeHub
	}
	return model
}

// SetHub records the hub slot's worktree root (pair#491): the checkout Couch
// started from. The tab bar and the switcher lead with it and put every other
// slot in its section. Couch sets it once, at startup.
func (c *Console) SetHub(root string) {
	c.mu.Lock()
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventHub, Hub: root})
	c.mu.Unlock()
	c.repaint()
}

// SetHubOwnership installs a routing snapshot without IO and keeps the menu and
// tabs on the same revision. The immutable value can safely outlive the caller.
func (c *Console) SetHubOwnership(ownership couchcore.HubOwnership) {
	c.mu.Lock()
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventHubOwnership, HubOwnership: ownership})
	c.mu.Unlock()
	c.repaint()
}
