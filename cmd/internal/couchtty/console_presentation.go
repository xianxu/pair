package couchtty

import "github.com/xianxu/pair/cmd/internal/couchcore"

// statusModelLocked joins attached and pending members to the same projection
// used by the switcher. Callers hold c.mu; no discovery or process IO occurs.
func (c *Console) statusModelLocked() StatusModel {
	model := StatusModel{Notice: c.feed.Row().Body, Spinner: c.statusSpinner}
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
		actor := StatusActor{Thread: p.thread, Active: id == c.active, Bell: len(c.attention.Projection(p.thread)) > 0}
		members[p.thread] = actor
		index, found := byAddress[p.thread]
		if !found {
			row := couchcore.ActionableThreadSummary{Address: p.thread, StartingPath: string(p.tree), WorkingPath: string(p.tree), Name: p.label}
			if target, ok := slotsByPath[string(p.tree)]; ok {
				row.Target = target
				row.RowKey, _ = target.RowKey()
			}
			rows = append(rows, row)
			index = len(rows) - 1
		}
		// A snapshot can lag attachment. The pane's canonical tree supplies missing
		// context only; an observed slot target and its durable row key are retained.
		if rows[index].StartingPath == "" {
			rows[index].StartingPath = string(p.tree)
		}
		if rows[index].WorkingPath == "" {
			rows[index].WorkingPath = string(p.tree)
		}
		// Ordinary tabs retain their attached label. Workspace labels for a
		// verified slot group are supplied by PresentThreads instead.
		rows[index].Name = p.label
	}
	for _, pending := range pendingPlaceholders(c.menu.Reattach) {
		if _, attached := members[pending.Address]; attached {
			continue
		}
		members[pending.Address] = StatusActor{Thread: pending.Address, Placeholder: true, Loading: pending.Loading}
		if index, found := byAddress[pending.Address]; !found {
			rows = append(rows, couchcore.ActionableThreadSummary{Address: pending.Address})
		} else if rows[index].Name == "" && rows[index].StartingPath != "" {
			rows[index].Name = couchcore.Worktree(rows[index].StartingPath).Repo()
		}
	}
	for _, entry := range PresentThreads(rows, c.menu.SlotGit) {
		actor, found := members[entry.Row.Address]
		if !found {
			continue
		}
		actor.Label, actor.GroupKey, actor.SlotNumber, actor.Glyph = entry.Label, entry.GroupKey, entry.SlotNumber, entry.Glyph
		model.Actors = append(model.Actors, actor)
	}
	return model
}
