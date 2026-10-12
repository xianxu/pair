package webcast

import "sync"

// Remote driving in the cast Chrome (#467). While a viewer drives, the cast
// tab's page reaches nothing outside itself through Chrome: file choosers
// are intercepted (never shown, so cancelled), downloads are denied, and a
// page target the cast tab opens is closed before the cast can follow it.
// That holds for the operator's own pickers and downloads too, while
// someone drives: one rule, no guess at whose input opened what. The
// in-page control shows who drives. Everything is restored when nobody
// does.

// SetDriving tells Chrome who drives the cast tab ("" for nobody).
func (c *Chrome) SetDriving(driver string) error {
	ran := make(chan struct{})
	if !c.do(func(s *loopState) { c.setDriving(s, driver); close(ran) }) {
		return ErrEnded
	}
	<-ran
	return nil
}

func (c *Chrome) setDriving(s *loopState, driver string) {
	was := s.driver != ""
	s.driver = driver
	for session, ctx := range s.contexts {
		c.goSetControl(session, ctx, s.phase == PhaseLive, driver)
	}
	driving := driver != ""
	if was == driving {
		return
	}
	behavior := "default"
	if driving {
		behavior = "deny"
	}
	c.goCall("", "Browser.setDownloadBehavior", map[string]any{"behavior": behavior}, nil)
	for session := range s.sessions {
		c.goCall(session, "Page.setInterceptFileChooserDialog", map[string]any{"enabled": driving}, nil)
	}
}

// closeOpened closes a page the cast tab opened while a viewer drives, and
// reports whether it did: the tracker never sees it, so the cast doesn't
// follow it.
func (c *Chrome) closeOpened(s *loopState, in TargetCreated) bool {
	if s.driver == "" || in.Opener == "" || in.Opener != castTab(s) {
		return false
	}
	c.goCall("", "Target.closeTarget", map[string]any{"targetId": in.ID}, nil)
	return true
}

// dialogSignal is the cast tab's open JavaScript dialog as a channel the
// recorder selects on: closed while one is open.
type dialogSignal struct {
	mu sync.Mutex
	ch chan struct{}
	on bool
}

func (d *dialogSignal) signal() <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ch == nil {
		d.ch = make(chan struct{})
	}
	return d.ch
}

func (d *dialogSignal) open() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ch == nil {
		d.ch = make(chan struct{})
	}
	if !d.on {
		d.on = true
		close(d.ch)
	}
}

func (d *dialogSignal) closed() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.on {
		d.on, d.ch = false, make(chan struct{})
	}
}

// castTab is the tab being cast: the active one while live, else "".
func castTab(s *loopState) string {
	if tab, ok := s.tracker.Active(); ok && s.phase == PhaseLive {
		return tab.ID
	}
	return ""
}
