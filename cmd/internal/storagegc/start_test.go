package storagegc

import (
	"context"
	"testing"
	"time"
)

func TestStartReservationSurvivesParentDeathUntilBothChildrenAcknowledge(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	parent := ProcessIdentity{PID: 1, Birth: "parent"}
	wrapper := ProcessIdentity{PID: 2, Birth: "wrapper"}
	editor := ProcessIdentity{PID: 3, Birth: "editor"}
	probe := &FakeProcessProbe{Processes: map[int]string{1: "parent", 2: "wrapper", 3: "editor"}}
	c.Probe = probe
	id, err := c.ReserveStart(ctx, o, parent, []string{"wrapper", "draft-editor"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MarkStartSpawned(ctx, o, id, parent); err != nil {
		t.Fatal(err)
	}
	delete(probe.Processes, 1)
	c.Now = func() time.Time { return time.Date(2027, 9, 13, 0, 0, 0, 0, time.UTC) }
	if err := c.RecoverUse(ctx, o); err != nil {
		t.Fatal(err)
	}
	state, err := c.ReadOwner(o)
	if err != nil || len(state.Starts) != 1 {
		t.Fatalf("parent death expired pending start %+v %v", state, err)
	}
	if _, err := c.RegisterProcess(ctx, o, wrapper, "wrapper"); err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(ctx, func(l *Locked) error { return l.AcknowledgeStart(o, id, "wrapper", wrapper) }); err != nil {
		t.Fatal(err)
	}
	state, _ = c.ReadOwner(o)
	if len(state.Starts) != 1 {
		t.Fatal("one child cleared reservation")
	}
	if _, err := c.RegisterProcess(ctx, o, editor, "draft-editor"); err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(ctx, func(l *Locked) error { return l.AcknowledgeStart(o, id, "draft-editor", editor) }); err != nil {
		t.Fatal(err)
	}
	state, _ = c.ReadOwner(o)
	if len(state.Starts) != 0 || len(state.Processes) != 2 || !state.Activity.LastUse.Equal(c.Now()) {
		t.Fatalf("handoff lost child lifetime/use %+v", state)
	}
}

func TestStartCancelRequiresKnownPreSpawnBoundary(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p := ProcessIdentity{PID: 1, Birth: "parent"}
	c.Probe = &FakeProcessProbe{Processes: map[int]string{1: "parent"}}
	id, err := c.ReserveStart(ctx, o, p, []string{"wrapper"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.CancelStartBeforeSpawn(ctx, o, id, p); err != nil {
		t.Fatal(err)
	}
	id, err = c.ReserveStart(ctx, o, p, []string{"wrapper"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MarkStartSpawned(ctx, o, id, p); err != nil {
		t.Fatal(err)
	}
	if err := c.CancelStartBeforeSpawn(ctx, o, id, p); err == nil {
		t.Fatal("uncertain spawned effects canceled")
	}
	if err := c.WithLock(ctx, func(l *Locked) error { return l.AcknowledgeStart(o, id, "wrapper", p) }); err == nil {
		t.Fatal("unregistered child acknowledged")
	}
}
