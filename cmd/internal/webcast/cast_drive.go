package webcast

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Driven is a browser a viewer can drive (#467): the cast Chrome, recording.
// Its recorder runs the driver's batches (drive.TabRunner) and takes the
// control records (Emitter); Opened says it has a recording to take them;
// SetDriving guards the cast tab and shows the driver.
type Driven interface {
	Emitter
	drive.TabRunner
	Opened() bool
	SetDriving(driver string) error
}

// Opened: the recorder has opened its recording (the first go-live).
func (c *Chrome) Opened() bool { return c.rec != nil && c.rec.Opened() }

// WithTab runs a driver's batch on the recorder's queue.
func (c *Chrome) WithTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	if c.rec == nil {
		return ErrNotRecording
	}
	return c.rec.WithTab(ctx, by, fn)
}

// liveGate is the controller's sink in a cast: control records go into the
// recorder's queue once it has opened, and are refused before (the
// recorder would drop them, parting the controller from the stream). It
// asks the recorder, not Chrome's published phase: Chrome can publish live
// while the recorder still waits for the cast tab's session.
type liveGate struct{ b Driven }

func (g liveGate) Put(r recording.Record) error {
	if !g.b.Opened() {
		return drive.ErrNotLive
	}
	return g.b.Emit(r, 0)
}

func (liveGate) Close() error { return nil }

// followDriving tells d who drives, for as long as ctrl runs.
func followDriving(ctrl drive.Controller, d Driven) {
	states, cancel := ctrl.Subscribe()
	defer cancel()
	for st := range states {
		name := ""
		if st.Driver != nil {
			name = st.Driver.Name
		}
		d.SetDriving(name) // an ended browser has nothing to guard
	}
}
