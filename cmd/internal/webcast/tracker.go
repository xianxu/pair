package webcast

// Tracker decides which tab is cast: the tab active in the cast Chrome.
//
// CDP has no "tab activated" event, so activity is inferred from what each
// page reports (visibility and focus, from the injected script) and from
// target lifecycle. Inputs are applied in arrival order and ordered by a
// sequence number, never wall time:
//
//   - a page's activation is the sequence of its creation, of a report that
//     shows it visible and focused, or of a report that shows it becoming
//     visible; a page whose first report says hidden was opened in the
//     background, so its creation is withdrawn;
//   - the active tab is the most recently activated page among those that are
//     visible or have never reported. Pages that never report are the ones the
//     script can't run in (chrome://newtab): Cmd+T opens one in the
//     foreground, so it counts as active until another page says otherwise;
//   - when every page is hidden (the window minimized), the most recently
//     activated page stays active, so a cast doesn't drop its tab.
//
// So there is an active tab exactly when a page is open.
//
// A background-opened chrome:// page is wrongly taken as active until another
// page reports; accepted (see atlas/webcast.md).
type Tracker struct {
	seq   uint64
	pages map[string]*trackedPage
}

type trackedPage struct {
	tab       Tab
	reported  bool
	visible   bool
	activated uint64
}

// TrackerInput is one of TargetCreated, TargetChanged, TargetDestroyed,
// PageReport.
type TrackerInput interface{ trackerInput() }

// TargetCreated: a page target appeared.
type TargetCreated struct{ ID, URL, Title string }

// TargetChanged: a page's URL or title changed.
type TargetChanged struct{ ID, URL, Title string }

// TargetDestroyed: a page closed.
type TargetDestroyed struct{ ID string }

// PageReport: the page's top document reported its visibility and focus.
type PageReport struct {
	ID               string
	Visible, Focused bool
}

func (TargetCreated) trackerInput()   {}
func (TargetChanged) trackerInput()   {}
func (TargetDestroyed) trackerInput() {}
func (PageReport) trackerInput()      {}

// Apply folds one input in. Inputs for unknown pages are ignored, except
// TargetCreated; a repeated TargetCreated is a change.
func (t *Tracker) Apply(in TrackerInput) {
	if t.pages == nil {
		t.pages = map[string]*trackedPage{}
	}
	t.seq++
	switch in := in.(type) {
	case TargetCreated:
		if p, ok := t.pages[in.ID]; ok {
			p.tab.URL, p.tab.Title = in.URL, in.Title
			return
		}
		t.pages[in.ID] = &trackedPage{tab: Tab(in), activated: t.seq}
	case TargetChanged:
		if p, ok := t.pages[in.ID]; ok {
			p.tab.URL, p.tab.Title = in.URL, in.Title
		}
	case TargetDestroyed:
		delete(t.pages, in.ID)
	case PageReport:
		p, ok := t.pages[in.ID]
		if !ok {
			return
		}
		becameVisible := in.Visible && (!p.reported || !p.visible)
		switch {
		case becameVisible || (in.Visible && in.Focused):
			p.activated = t.seq
		case !p.reported:
			// Opened in the background: its creation was never an activation.
			p.activated = 0
		}
		p.reported, p.visible = true, in.Visible
	}
}

// Active is the cast tab; false only when no page is open.
func (t *Tracker) Active() (Tab, bool) {
	var best, fallback *trackedPage
	for _, p := range t.pages {
		if fallback == nil || p.activated > fallback.activated {
			fallback = p
		}
		if p.reported && !p.visible {
			continue
		}
		if best == nil || p.activated > best.activated {
			best = p
		}
	}
	if best == nil {
		best = fallback
	}
	if best == nil {
		return Tab{}, false
	}
	return best.tab, true
}

// Pages is how many page targets are open.
func (t *Tracker) Pages() int { return len(t.pages) }
