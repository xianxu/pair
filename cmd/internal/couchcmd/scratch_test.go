package couchcmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchscratch"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestScratchRuntimeIsolationNeverUsesMountedGlobalVolume(t *testing.T) {
	rt := OSRuntime{isolatedRoot: t.TempDir()}
	if got := rt.scratchProbe(); got.Mounted || got.Root == couchscratch.DefaultRoot {
		t.Fatalf("isolated scratch escaped: %+v", got)
	}
}

func TestSlotScratchEnvironmentUsesRetainedIdentityAndPreservesSocket(t *testing.T) {
	root := t.TempDir()
	manager := couchscratch.New(couchscratch.Options{Probe: func() couchscratch.Probe {
		return couchscratch.Probe{Root: root, Mounted: true, FreeBytes: couchscratch.Floor}
	}})
	record := couchcore.ThreadRecord{StartingPath: "/missing/repo", WorkingPath: "/"}
	var received couchcore.ThreadRecord
	lookup := func(_ context.Context, got couchcore.ThreadRecord) (couchcore.ScratchSlotIdentity, error) {
		received = got
		return couchcore.ScratchSlotIdentity{Repo: "repo", RepoIdentity: "/stable/repo/.git", Number: 3}, nil
	}
	overlay := scratchSlotEnvironment(manager, lookup)(t.Context(), record)
	if received.StartingPath != record.StartingPath {
		t.Fatal("launch did not use retained record")
	}
	runner := couchcore.NewFakeRunner()
	decorated := configuredRunner{underlying: runner, environment: []string{"ZELLIJ_SOCKET_DIR=/original/zellij"}}
	for _, blocked := range []bool{false, true} {
		env := append([]string{"TMPDIR=/inherited", "ZELLIJ_SOCKET_DIR=/wrong"}, overlay...)
		var h couchcore.Handle
		var err error
		if blocked {
			h, err = decorated.StartBlocked(context.Background(), "/", []string{"pair"}, env, time.Second)
		} else {
			h, err = decorated.Start("/", []string{"pair"}, env)
		}
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, entry := range runner.Child(h.ID()).Env {
			key, value, _ := strings.Cut(entry, "=")
			got[key] = value
		}
		want := filepath.Join(root, "tmp", "repo-slot3")
		for _, key := range []string{"TMPDIR", "GOTMPDIR", "CLAUDE_CODE_TMPDIR"} {
			if got[key] != want {
				t.Fatalf("%s=%q", key, got[key])
			}
		}
		if got["ZELLIJ_SOCKET_DIR"] != "/original/zellij" {
			t.Fatal("socket namespace followed temp")
		}
	}
}

func TestRuntimeRunnerPinsSocketBeforeTempOverlay(t *testing.T) {
	t.Setenv("ZELLIJ_SOCKET_DIR", "/old/socket")
	root := isolatedCouchTestEnvironment(t)
	rt, lease, err := (OSRuntime{}).prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	// Explicit runtime namespace wins over arbitrary actor input.
	fake := couchcore.NewFakeRunner()
	h, err := rt.runtimeRunner(fake).Start(root, []string{"pair"}, []string{"ZELLIJ_SOCKET_DIR=/foreign", "TMPDIR=/ram"})
	if err != nil {
		t.Fatal(err)
	}
	var got string
	for _, e := range fake.Child(h.ID()).Env {
		if strings.HasPrefix(e, "ZELLIJ_SOCKET_DIR=") {
			got = strings.TrimPrefix(e, "ZELLIJ_SOCKET_DIR=")
		}
	}
	if got != "/old/socket" {
		t.Fatalf("socket=%q", got)
	}
	if os.Getenv("ZELLIJ_SOCKET_DIR") != "/old/socket" {
		t.Fatal("runtime changed process env")
	}
}

func TestWorkspaceScratchOwnerUsesEnvironmentHost(t *testing.T) {
	zero, three := 0, 3
	for _, tc := range []struct {
		workspace couchcore.WorkspaceIdentity
		want      couchscratch.Slot
	}{
		{couchcore.WorkspaceIdentity{Repo: "repo", RepoIdentity: "/repo/.git", Slot: &zero, Kind: "primary"}, couchscratch.Slot{Repo: "repo", Identity: "/repo/.git", Number: 0}},
		{couchcore.WorkspaceIdentity{Repo: "dependency", RepoIdentity: "/dep/.git", EnvironmentHost: &couchcore.EnvironmentHost{Repo: "repo", RepoIdentity: "/repo/.git", Slot: three}}, couchscratch.Slot{Repo: "repo", Identity: "/repo/.git", Number: 3}},
	} {
		got, ok := workspaceScratchOwner(tc.workspace)
		if !ok || got != tc.want {
			t.Fatalf("owner=%+v,%v want %+v", got, ok, tc.want)
		}
	}
	if _, ok := workspaceScratchOwner(couchcore.WorkspaceIdentity{}); ok {
		t.Fatal("unknown workspace qualified")
	}
}

func TestScratchCompositionOnlyStartsForSupervisorOwner(t *testing.T) {
	root := isolatedCouchTestEnvironment(t)
	rt, lease, err := (OSRuntime{}).prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	c, err := rt.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	if c.ScratchStartup == nil || c.ScratchMaintenance == nil || c.SlotEnvironment == nil {
		t.Fatal("owner lacks scratch composition")
	}
	c.ScratchStartup(t.Context(), func(string) {})
	if _, err := os.Stat(filepath.Join(root, "scratch")); !os.IsNotExist(err) {
		t.Fatalf("isolated runtime created global-style scratch: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c.ScratchMaintenance(ctx)
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	read, err := rt.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	if read.ScratchStartup != nil || read.ScratchMaintenance != nil || read.SlotEnvironment != nil {
		t.Fatal("reader gained maintenance authority")
	}
}

func TestOwnedChildWritesOnlyInsidePrivateSlotTemp(t *testing.T) {
	root := t.TempDir()
	manager := couchscratch.New(couchscratch.Options{Probe: func() couchscratch.Probe {
		return couchscratch.Probe{Root: root, Mounted: true, FreeBytes: couchscratch.Floor}
	}})
	overlay := manager.PrepareSlot(couchscratch.Slot{Repo: "private", Identity: "/fixture/private/.git", Number: 3})
	output := filepath.Join(root, "observed")
	runner := configuredRunner{underlying: couchcore.ExecRunner{}, environment: []string{"ZELLIJ_SOCKET_DIR=/private/old/socket"}}
	h, err := runner.Start(root, []string{"/bin/sh", "-c", `printf '%s\n' "$TMPDIR" "$GOTMPDIR" "$CLAUDE_CODE_TMPDIR" "$ZELLIJ_SOCKET_DIR" > "$1"; printf fixture > "$TMPDIR/child-file"`, "sh", output}, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if code := h.Wait(); code != 0 {
		t.Fatalf("child exit %d", code)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "tmp", "private-slot3")
	if string(raw) != want+"\n"+want+"\n"+want+"\n/private/old/socket\n" {
		t.Fatalf("child env=%q", raw)
	}
	if raw, err := os.ReadFile(filepath.Join(want, "child-file")); err != nil || string(raw) != "fixture" {
		t.Fatalf("child temp=%q %v", raw, err)
	}
}

func TestInjectedScratchConstructorLaunchesWithRAMAndFallback(t *testing.T) {
	root := isolatedCouchTestEnvironment(t)
	socketRoot := filepath.Join(root, "native-sockets")
	t.Setenv("ZELLIJ_SOCKET_DIR", socketRoot)
	ram := filepath.Join(root, "ram")
	if err := os.Mkdir(ram, 0700); err != nil {
		t.Fatal(err)
	}
	ticks := make(chan time.Time)
	serverCalls := 0
	mounted := true
	rt := OSRuntime{
		scratchProbeFn: func() couchscratch.Probe {
			return couchscratch.Probe{Root: ram, Mounted: mounted, FreeBytes: couchscratch.Floor}
		},
		scratchServers: func(context.Context) (bool, error) { serverCalls++; return false, nil },
		scratchTicks:   ticks,
		scratchNow:     func() time.Time { return time.Unix(200000, 0) },
	}
	rt, lease, err := rt.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	runner := couchcore.NewFakeRunner()
	c, err := rt.NewCouchWith(runner, ns)
	if err != nil {
		t.Fatal(err)
	}
	c.ScratchStartup(t.Context(), func(string) {})
	if serverCalls != 1 {
		t.Fatalf("startup native reads=%d", serverCalls)
	}
	checkout := filepath.Join(root, "fixture")
	scope, err := launcher.ResolveRepoScope(checkout)
	if err != nil {
		t.Fatal(err)
	}
	zero := 0
	c.Slots = &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{checkout: {Repo: "fixture", RepoIdentity: filepath.Join(checkout, ".git"), WorktreeRoot: checkout, PrimaryRoot: checkout, Kind: "primary", Slot: &zero}}}
	record := couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0123456789abcdef"}, StartingPath: checkout, WorkingPath: checkout, CreatedAt: time.Unix(100, 0), Revision: 1, LatestLaunchProfile: &couchcore.LaunchProfile{Agent: "codex"}}
	for _, enabled := range []bool{true, false} {
		mounted = enabled
		env := append([]string{"ZELLIJ_SOCKET_DIR=/wrong", "TMPDIR=/outside", "GOTMPDIR=/outside", "CLAUDE_CODE_TMPDIR=/outside"}, c.SlotEnvironment(t.Context(), record)...)
		h, err := c.Runner.Start("/", []string{"pair"}, env)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, e := range runner.Child(h.ID()).Env {
			k, v, _ := strings.Cut(e, "=")
			got[k] = v
		}
		want := rt.isolatedPaths.temporary
		if enabled {
			want = filepath.Join(ram, "tmp", "fixture-slot0")
		}
		for _, key := range []string{"TMPDIR", "GOTMPDIR", "CLAUDE_CODE_TMPDIR"} {
			if got[key] != want {
				t.Fatalf("enabled=%v %s=%q want %q", enabled, key, got[key], want)
			}
		}
		if got["ZELLIJ_SOCKET_DIR"] != socketRoot {
			t.Fatal("constructor lost the pinned native namespace")
		}
	}
}

func TestIsolatedScratchOverlayRefusesIncompleteOrForeignPaths(t *testing.T) {
	const root = "/private/fixture/ram"
	makeEnv := func(path string) []string {
		return []string{"TMPDIR=" + path, "GOTMPDIR=" + path, "CLAUDE_CODE_TMPDIR=" + path}
	}
	for _, path := range []string{"/foreign/tmp/repo-slot0", root + "/tmp/repo-slot0/../../escape", root + "/tmp/repo-slot0/child", root + "/tmp/"} {
		if got := isolatedScratchOverlay(root, makeEnv(path)); len(got) != 0 {
			t.Fatalf("unsafe path accepted: %q", path)
		}
	}
	if got := isolatedScratchOverlay(root, []string{"TMPDIR=" + root + "/tmp/repo-slot0"}); len(got) != 0 {
		t.Fatal("incomplete overlay accepted")
	}
	if got := isolatedScratchOverlay(root, makeEnv(root+"/tmp/repo-slot0")); len(got) != 3 {
		t.Fatal("complete private overlay refused")
	}
}

func TestIsolatedScratchProbeRejectsForeignBackend(t *testing.T) {
	root := t.TempDir()
	foreign := t.TempDir()
	rt := OSRuntime{isolatedRoot: root, scratchProbeFn: func() couchscratch.Probe {
		return couchscratch.Probe{Root: foreign, Mounted: true, FreeBytes: couchscratch.Floor}
	}}
	if p := rt.scratchProbe(); p.Mounted || p.Err == nil {
		t.Fatalf("foreign backend accepted: %+v", p)
	}
}
