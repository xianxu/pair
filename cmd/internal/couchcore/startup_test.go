package couchcore

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func seedStartupParked(t *testing.T, env *testEnv, tag ThreadTag, path string, active ...time.Time) ThreadRecord {
	t.Helper()
	at := time.Unix(100, 0).UTC()
	if len(active) > 0 {
		at = active[0]
	}
	record := actionableTestThread(tag, at)
	record.StartingPath, record.WorkingPath = "/repo", path
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, record.LastActiveAt)
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-"+string(tag))
	return created
}

func TestStartupAttachesPreUpgradeHubWithoutChangingOtherSlots(t *testing.T) {
	env, address := startupFixture(t, 2, Layout3)
	current, err := env.Couch.Threads.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.Threads.updateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
		next.LatestLaunchProfile = nil
		next.Layout = ""
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := env.Couch.Threads.PreviewSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil || result.Handle == nil || result.Record.Thread != address || result.Record.Shape != StartWarmReattach || len(env.Runner.Ops) != 2 {
		t.Fatalf("legacy hub attachment: %+v err=%v ops=%v", result, err, env.Runner.Ops)
	}
	if len(env.Artifacts.Calls()) != 0 || env.Artifacts.DetachedQueries() != 0 {
		t.Fatal("legacy live hub required cold or agent ownership proof")
	}
	if strings.Contains(env.Runner.Ops[0], "--layout") || strings.Contains(env.Runner.Ops[0], "--claude") {
		t.Fatalf("warm startup selected layout or agent: %v", env.Runner.Ops)
	}
	after, err := env.Couch.Threads.PreviewSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("warm hub startup changed retained records")
	}
}

func TestStartupDoesNotResumeProvenHubConversation(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := seedStartupParked(t, env, "couch-0000000000000001", "/repo")
	env.Artifacts.SetPairSession(record.Address, "pair-"+string(record.Address.Tag), false)
	env.Couch.resumeRegistrationTimeout = 20 * time.Millisecond
	result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil || result.Handle != nil || len(env.Runner.Ops) != 0 || env.Artifacts.BindingResolutions() != 0 {
		t.Fatalf("proven hub resume: %+v err=%v ops=%v", result, err, env.Runner.Ops)
	}
}
