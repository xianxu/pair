package storagegc

import (
	"context"
	"os/exec"
	"reflect"
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

func TestRecoverDeadUnspawnedStartsPreservesUnknownAndSpawned(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	probe := &FakeProcessProbe{Processes: map[int]string{1: "dead", 2: "live", 3: "unknown", 4: "spawned"}}
	c.Probe = probe
	for pid := 1; pid <= 4; pid++ {
		parent := ProcessIdentity{PID: pid, Birth: probe.Processes[pid]}
		id, err := c.ReserveStart(ctx, o, parent, []string{"wrapper"})
		if err != nil {
			t.Fatal(err)
		}
		if pid == 4 {
			if err := c.MarkStartSpawned(ctx, o, id, parent); err != nil {
				t.Fatal(err)
			}
		}
	}
	before, _ := c.ReadOwner(o)
	delete(probe.Processes, 1)
	delete(probe.Processes, 4)
	probe.Unknown = map[int]bool{3: true}
	if err := c.RecoverUse(ctx, o); err != nil {
		t.Fatal(err)
	}
	after, _ := c.ReadOwner(o)
	if len(after.Starts) != 3 || after.Starts[0].Parent.PID != 2 || after.Starts[1].Parent.PID != 3 || after.Starts[2].Parent.PID != 4 {
		t.Fatalf("wrong recovered starts: %+v", after.Starts)
	}
	if !after.Activity.LastUse.Equal(before.Activity.LastUse) {
		t.Fatal("unspawned failure invented use")
	}
	writes := 0
	c.BeforePersist = func() error { writes++; return nil }
	if err := c.RecoverUse(ctx, o); err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Fatalf("unchanged recovery rewrote owner %d times", writes)
	}
}

func TestReserveStartReclaimsDeadPreSpawnReservationsBeforeCap(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	probe := &FakeProcessProbe{Processes: map[int]string{1: "parent"}}
	c.Probe = probe
	for i := 0; i < 32; i++ {
		if _, err := c.ReserveStart(ctx, o, ProcessIdentity{PID: 1, Birth: "parent"}, []string{"wrapper"}); err != nil {
			t.Fatal(err)
		}
	}
	delete(probe.Processes, 1)
	probe.Processes[2] = "new"
	if _, err := c.ReserveStart(ctx, o, ProcessIdentity{PID: 2, Birth: "new"}, []string{"wrapper"}); err != nil {
		t.Fatalf("dead pre-spawn reservations blocked launch: %v", err)
	}
	state, _ := c.ReadOwner(o)
	if len(state.Starts) != 1 || state.Starts[0].Parent.PID != 2 {
		t.Fatalf("abandoned starts survived: %+v", state.Starts)
	}
}

func TestExplicitStartResolutionRetainsLiveUnknownEvidence(t *testing.T) {
	for _, mode := range []string{"no-confirmation", "wrong-parent", "parent-unknown", "registered-live", "registered-unknown", "acknowledged-live", "intent-live", "all-dead"} {
		t.Run(mode, func(t *testing.T) {
			c, o := coordinatorFixture(t)
			ctx := context.Background()
			parent := ProcessIdentity{PID: 1, Birth: "parent"}
			child := ProcessIdentity{PID: 2, Birth: "child"}
			probe := &FakeProcessProbe{Processes: map[int]string{1: "parent", 2: "child"}, Unknown: map[int]bool{}}
			c.Probe = probe
			id, err := c.ReserveStart(ctx, o, parent, []string{"wrapper", "draft-editor"})
			if err != nil {
				t.Fatal(err)
			}
			if err := c.MarkStartSpawned(ctx, o, id, parent); err != nil {
				t.Fatal(err)
			}
			if mode == "registered-live" || mode == "registered-unknown" || mode == "acknowledged-live" {
				if _, err := c.RegisterProcess(ctx, o, child, "wrapper"); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "acknowledged-live" {
				if err := c.WithLock(ctx, func(l *Locked) error { return l.AcknowledgeStart(o, id, "wrapper", child) }); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "intent-live" {
				if _, err := c.BeginUse(ctx, o, child, "draft"); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := c.ReadOwner(o)
			delete(probe.Processes, 1)
			if mode == "parent-unknown" {
				probe.Unknown[1] = true
			}
			if mode == "registered-unknown" {
				probe.Unknown[2] = true
			}
			given := parent
			if mode == "wrong-parent" {
				given.Birth = "foreign"
			}
			later := c.Now().Add(time.Hour)
			c.Now = func() time.Time { return later }
			err = c.ResolveAbandonedStart(ctx, o, id, given, mode != "no-confirmation")
			after, _ := c.ReadOwner(o)
			if mode == "all-dead" {
				if err != nil || len(after.Starts) != 0 || !after.Activity.LastUse.Equal(later) {
					t.Fatalf("resolution=%+v %v", after, err)
				}
				return
			}
			if err == nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("unsafe resolution changed state: %s err=%v", mode, err)
			}
		})
	}
}

func TestUnspawnedStartRecoversAfterActualParentDeath(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	child := exec.Command("sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	parent, err := CurrentProcessIdentity(child.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReserveStart(ctx, o, parent, []string{"wrapper"}); err != nil {
		t.Fatal(err)
	}
	before, _ := c.ReadOwner(o)
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	if err := c.RecoverUse(ctx, o); err != nil {
		t.Fatal(err)
	}
	after, err := c.ReadOwner(o)
	if err != nil || len(after.Starts) != 0 || !after.Activity.LastUse.Equal(before.Activity.LastUse) {
		t.Fatalf("unspawned death not reconciled: %+v %v", after, err)
	}
}

func TestCanRetireUnspawnedStart(t *testing.T) {
	for _, liveness := range []Liveness{ProcessAlive, ProcessUnknown, ProcessDead} {
		for _, spawned := range []bool{false, true} {
			for _, ack := range []bool{false, true} {
				start := StartReservation{Spawned: spawned}
				if ack {
					start.Acknowledged = map[string]ProcessIdentity{"wrapper": {PID: 42, Birth: "child"}}
				}
				want := liveness == ProcessDead && !spawned && !ack
				if got := CanRetireUnspawnedStart(start, liveness); got != want {
					t.Fatalf("liveness=%s spawned=%v ack=%v got=%v", liveness, spawned, ack, got)
				}
			}
		}
	}
}
