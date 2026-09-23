package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Real Git directories plus the stateful agent/process fake exercise isolation
// through the same preview, commit and recovery operations as the console.
func TestSlotAcceptanceTwoSlotsRecoverIndependently(t *testing.T) {
	env, f := managedStartFixture(t)
	ctx := context.Background()
	var actors []ActorRecord
	for _, agent := range []string{"claude", "codex"} {
		args := StartArgs{Cwd: f.Primary, Action: StartCreate, Stack: agent}
		preview, err := env.Couch.PrepareStart(ctx, args)
		if err != nil {
			t.Fatal(err)
		}
		actor, _, err := env.Couch.SpawnPrepared(ctx, args, preview.Resolution.Fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		actors = append(actors, actor)
	}
	snapshot, err := env.Couch.Threads.Snapshot()
	if err != nil || len(snapshot.Records) != 3 || len(snapshot.Slots) != 2 {
		t.Fatalf("inventory %+v %v", snapshot, err)
	}
	// Independent work in both hosts survives a conversation reset in :1.
	beforeHeads := map[string]string{}
	for i := 1; i <= 2; i++ {
		host := f.host(i)
		beforeHeads[host] = f.git(host, "rev-parse", "HEAD")
		if err := os.WriteFile(filepath.Join(host, "operator-work.txt"), []byte("keep local work\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	secondPath, err := env.Couch.Threads.RecordPath(actors[1].Thread)
	if err != nil {
		t.Fatal(err)
	}
	secondBefore, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	firstPath, err := env.Couch.Threads.RecordPath(actors[0].Thread)
	if err != nil {
		t.Fatal(err)
	}
	env.Proc.Kill(actors[0].PID)
	if err := os.WriteFile(firstPath, []byte("broken current metadata"), 0600); err != nil {
		t.Fatal(err)
	}
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	fresh, err := env.Couch.StartFreshSlot(ctx, f.host(1), "")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Record.Thread == actors[0].Thread || fresh.Record.Args.Stack != "claude" {
		t.Fatalf("fresh %+v", fresh.Record)
	}
	secondAfter, err := os.ReadFile(secondPath)
	if err != nil || string(secondBefore) != string(secondAfter) {
		t.Fatalf("other slot changed: %v", err)
	}
	for host, head := range beforeHeads {
		if got := f.git(host, "rev-parse", "HEAD"); got != head {
			t.Fatalf("HEAD changed: %s", host)
		}
		raw, err := os.ReadFile(filepath.Join(host, "operator-work.txt"))
		if err != nil || string(raw) != "keep local work\n" {
			t.Fatalf("work changed: %s %v", host, err)
		}
	}
	if f.WeaveCalls != 2 {
		t.Fatalf("ready slots unexpectedly compiled %d times", f.WeaveCalls)
	}
}
