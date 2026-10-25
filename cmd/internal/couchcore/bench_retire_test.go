package couchcore

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func registerOrphan(t *testing.T, c *Couch, scope string, pid int) {
	t.Helper()
	entry := LeftRunningOrphan{Origin: "slot:bench", Thread: ThreadAddress{RepoScope: scope, Tag: "earlier"}, Process: ProcessIdentity{PID: pid, Identity: "id-orphan"}, Command: "runner", LeftAt: time.Now()}
	if err := NewLeftRunningRegistry(c.Namespace).Add(context.Background(), []LeftRunningOrphan{entry}); err != nil {
		t.Fatal(err)
	}
}

func survivorCode(err error) string {
	var refusal *SlotOperationError
	if errors.As(err, &refusal) {
		return refusal.Code
	}
	return ""
}

// Survivors are every not-provably-dead orphan of this scope, from this
// teardown or any earlier generation; other scopes and the dead never count.
func TestBenchSurvivors(t *testing.T) {
	env := newTestEnv(t, "/repo")
	c := env.Couch
	registerOrphan(t, c, "bench", 701) // alive
	registerOrphan(t, c, "bench", 702) // dead: never set
	registerOrphan(t, c, "bench", 703) // liveness unknown
	registerOrphan(t, c, "production", 704)
	env.Proc.Set(701, "id-orphan")
	env.Proc.SetUnknown(703)
	env.Proc.Set(704, "id-orphan")
	env.Proc.Set(705, "id-left")
	left := []LeftRunningOrphan{{Thread: ThreadAddress{RepoScope: "bench", Tag: "now"}, Process: ProcessIdentity{PID: 705, Identity: "id-left"}, Command: "npm"}}
	got, err := c.benchSurvivors("bench", left)
	if err != nil {
		t.Fatal(err)
	}
	pids := map[int]bool{}
	for _, s := range got {
		pids[s.Process.PID] = true
	}
	if len(got) != 3 || !pids[701] || !pids[703] || !pids[705] {
		t.Fatalf("survivors %+v", got)
	}
}

func parkedBenchThread(t *testing.T, env *testEnv) ThreadRecord {
	t.Helper()
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	return parked
}

// A live registered orphan of the slot refuses retirement and keeps the
// record; once it exits, the same request archives.
func TestRetireBenchSlotRefusesWhileAnOrphanLivesThenArchives(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := parkedBenchThread(t, env)
	registerOrphan(t, env.Couch, parked.Address.RepoScope, 777)
	env.Proc.Set(777, "id-orphan")
	for attempt := 0; attempt < 2; attempt++ {
		_, err := env.Couch.RetireBenchSlot(context.Background(), RebootTarget{Address: parked.Address})
		if survivorCode(err) != SlotOpBenchSurvivors {
			t.Fatalf("attempt %d: want survivor refusal, got %v", attempt, err)
		}
		if _, err := env.Couch.Threads.GetThread(parked.Address); err != nil {
			t.Fatalf("attempt %d: refused retire lost the record: %v", attempt, err)
		}
	}
	env.Proc.Kill(777)
	got, err := env.Couch.RetireBenchSlot(context.Background(), RebootTarget{Address: parked.Address})
	if err != nil || got.Archived != parked.Address || got.Vacant {
		t.Fatalf("retire after the survivor exited: %+v %v", got, err)
	}
	if _, err := env.Couch.Threads.GetThread(parked.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("record still current: %v", err)
	}
}

// A slot with no thread is vacant, unless an earlier generation's orphan lives.
func TestRetireBenchSlotVacant(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	got, err := env.Couch.RetireBenchSlot(context.Background(), RebootTarget{Path: local.slot.WorktreeRoot})
	if err != nil || !got.Vacant {
		t.Fatalf("vacant slot: %+v %v", got, err)
	}
	scope, err := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	if err != nil {
		t.Fatal(err)
	}
	registerOrphan(t, env.Couch, scope.Key, 778)
	env.Proc.Set(778, "id-orphan")
	if _, err := env.Couch.RetireBenchSlot(context.Background(), RebootTarget{Path: local.slot.WorktreeRoot}); survivorCode(err) != SlotOpBenchSurvivors {
		t.Fatalf("vacant slot with a live orphan: %v", err)
	}
}

// BR-3: survivors this teardown named are recorded before the judgement, so a
// retry after the agent stopped (no LeftRunning any more) still refuses; a
// registry that cannot record them refuses too.
func TestBenchSurvivorsAreRecordedForTheRetry(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Proc.Set(790, "id-left")
	left := []LeftRunningOrphan{{Origin: "slot:bench", Thread: ThreadAddress{RepoScope: "bench", Tag: "now"}, Process: ProcessIdentity{PID: 790, Identity: "id-left"}, Command: "npm", LeftAt: time.Now()}}
	if err := env.Couch.refuseBenchSurvivors("bench", left); survivorCode(err) != SlotOpBenchSurvivors {
		t.Fatalf("first attempt: %v", err)
	}
	if err := env.Couch.refuseBenchSurvivors("bench", nil); survivorCode(err) != SlotOpBenchSurvivors {
		t.Fatalf("retry lost the survivor: %v", err)
	}

	broken := newTestEnv(t, "/repo")
	broken.Proc.Set(790, "id-left")
	registry := NewLeftRunningRegistry(broken.Couch.Namespace)
	if err := os.MkdirAll(registry.dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registry.Path(), []byte(`{"schema_version":2,"orphans":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := broken.Couch.refuseBenchSurvivors("bench", left); err == nil {
		t.Fatal("an unrecordable survivor did not refuse")
	}
}
