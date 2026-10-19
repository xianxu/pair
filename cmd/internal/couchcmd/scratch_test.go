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
	lookup := func(got couchcore.ThreadRecord) (couchcore.ScratchSlotIdentity, error) {
		received = got
		return couchcore.ScratchSlotIdentity{Repo: "repo", RepoIdentity: "/stable/repo/.git", Number: 3}, nil
	}
	overlay := scratchSlotEnvironment(manager, lookup, nil)(record)
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
