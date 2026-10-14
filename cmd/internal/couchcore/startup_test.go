package couchcore

import (
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
