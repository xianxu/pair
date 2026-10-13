package webcast

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Driven is a browser a viewer can drive (#467): the cast Chrome, recording.
// Its recorder runs the driver's batches (drive.TabRunner) and takes the
// control records (ControlWriter); Opened says it has a recording to take them;
// SetDriving guards the cast tab and shows the driver.
type Driven interface {
	Emitter
	drive.ControlWriter
	drive.TabRunner
	Opened() bool
	SetDriving(driver string) error
}

// Opened: the recorder has opened its recording (the first go-live).
func (c *Chrome) Opened() bool { return c.rec != nil && c.rec.Opened() }

// WithTab runs a driver's batch on the recorder's queue.
func (c *Chrome) WithTab(ctx context.Context, by string, fn func(drive.Tab) error) error {
	if c.rec == nil {
		return drive.ErrNotLive // nothing recorded, nothing cast to drive (pair#505: no unmapped error)
	}
	return c.rec.WithTab(ctx, by, fn)
}

// liveGate adapts the controller's sink to the acknowledged recorder writer.
// Opening/closing and reducer acceptance are checked on that writer, so a
// cached Opened flag cannot acknowledge a closed or failed recorder.
type liveGate struct{ b Driven }

func (g liveGate) Put(r recording.Record) error {
	ctl, ok := r.(*recording.Ctl)
	if !ok {
		return fmt.Errorf("drive: control sink received %T", r)
	}
	return g.WriteControl(context.Background(), ctl)
}
func (g liveGate) WriteControl(ctx context.Context, ctl *recording.Ctl) error {
	return g.b.WriteControl(ctx, ctl)
}

// WriteControl acknowledges the recorder's ordered sink result.
func (c *Chrome) WriteControl(ctx context.Context, ctl *recording.Ctl) error {
	if c.rec == nil {
		return drive.ErrNotLive
	}
	return c.rec.WriteControl(ctx, ctl)
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
