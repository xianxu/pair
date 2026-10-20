package couchcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchscratch"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func scratchWiringRuntime(t *testing.T) (OSRuntime, string, time.Time) {
	t.Helper()
	root := isolatedCouchTestEnvironment(t)
	ram := filepath.Join(root, "ram")
	if err := os.Mkdir(ram, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(root, "private-zellij-sockets"))
	t.Setenv("COUCH_LIFECYCLE_NOTICES", "off")
	// A future clock makes the injected Now observable: without that wiring,
	// the fixture's 25-hour-old entry is still in the host's future and kept.
	now := time.Now().Add(7 * 24 * time.Hour)
	return OSRuntime{
		scratchProbeFn: func() couchscratch.Probe {
			return couchscratch.Probe{Root: ram, Mounted: true, FreeBytes: couchscratch.Floor}
		},
		scratchServers: func(context.Context) (bool, error) { return false, nil },
		scratchNow:     func() time.Time { return now },
	}, ram, now
}

func scratchWiringOldEntry(t *testing.T, ram, name string, now time.Time) string {
	t.Helper()
	manager := couchscratch.New(couchscratch.Options{Probe: func() couchscratch.Probe {
		return couchscratch.Probe{Root: ram, Mounted: true, FreeBytes: couchscratch.Floor}
	}})
	overlay := manager.PrepareSlot(couchscratch.Slot{Repo: "fixture", Identity: "/fixture/.git", Number: 3})
	if len(overlay) == 0 {
		t.Fatal("private scratch fixture failed")
	}
	path := filepath.Join(ram, "tmp", "fixture-slot3", name)
	if err := os.WriteFile(path, []byte("old temp"), 0600); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-25 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScratchStartupRunsBeforeMissingResumeFailure(t *testing.T) {
	rt, ram, now := scratchWiringRuntime(t)
	var observations atomic.Int32
	rt.scratchServers = func(context.Context) (bool, error) { observations.Add(1); return false, nil }
	old := scratchWiringOldEntry(t, ram, "startup-old", now)
	op, ok := Resolve("resume")
	if !ok {
		t.Fatal("resume operation missing")
	}
	var out, errw bytes.Buffer
	code := runTypedOperationWithConsole(op, map[string]string{"ref": "scratch-missing-target"}, nil, false, "", nil, nil, bytes.NewReader(nil), &out, &errw, rt, nil)
	if code == 0 {
		t.Fatalf("missing resume unexpectedly succeeded: %s", out.String())
	}
	if observations.Load() != 1 {
		t.Fatalf("startup observer calls=%d; stderr=%s", observations.Load(), errw.String())
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("startup did not sweep aged temp before operation failure: %v; stderr=%s", err, errw.String())
	}
	cache, err := couchscratch.MetadataDirectory(os.Getenv("HOME"), "", runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(filepath.Join(cache, "session_info"))
	if err != nil || link != filepath.Join(ram, "zellij-session-info") {
		t.Fatalf("startup metadata route=%q,%v; stderr=%s", link, err, errw.String())
	}
}

func TestScratchServiceDrivesMaintenanceAndJoinsOnClose(t *testing.T) {
	raw, ram, now := scratchWiringRuntime(t)
	ticks := make(chan time.Time)
	raw.scratchTicks = ticks
	rt, lease, err := raw.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	c, err := rt.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	old := scratchWiringOldEntry(t, ram, "maintenance-old", now)
	entered, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	actual := c.ScratchMaintenance
	c.ScratchMaintenance = func(ctx context.Context) { close(entered); actual(ctx); close(cancelled); <-release }
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), bytes.NewReader(nil))
	defer func() { console.Stop(); console.Run() }()
	service, err := startMessageService(console, c)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	if service == nil {
		close(release)
		t.Fatal("production service was not composed")
	}
	closed := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		service.Close()
	}()
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("service did not start scratch maintenance")
	}
	select {
	case ticks <- now:
	case <-testwait.After(t):
		t.Fatal("maintenance did not receive configured tick")
	}
	// Taking the next unbuffered tick proves the first synchronous sweep finished.
	select {
	case ticks <- now:
	case <-testwait.After(t):
		t.Fatal("maintenance did not finish its sweep")
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("service maintenance did not sweep private aged entry: %v", err)
	}
	go func() { service.Close(); close(closed) }()
	select {
	case <-cancelled:
	case <-testwait.After(t):
		t.Fatal("service close did not cancel maintenance")
	}
	select {
	case <-closed:
		t.Fatal("service close returned before maintenance worker ended")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-closed:
	case <-testwait.After(t):
		t.Fatal("service did not join maintenance after release")
	}
}
