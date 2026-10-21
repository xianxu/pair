package testcouch

import (
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestOwnershipPublicationKeepsPreviousReaderComplete(t *testing.T) {
	f := &Fleet{Roots: RootsPlan{Root: t.TempDir()}}
	first := Resource{ID: "123", Generation: "first", Kind: "agent"}
	f.record(RunEvent{Kind: "own", Resource: first})
	path := filepath.Join(f.Roots.Root, "ownership.json")
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	before, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	f.record(RunEvent{Kind: "own", Resource: Resource{ID: "456", Generation: "second", Kind: "agent"}})
	if _, err := reader.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	held, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != string(before) {
		t.Fatal("publication rewrote the previous snapshot in place")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var current OwnedRun
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatal(err)
	}
	if len(current.Resources) != 2 {
		t.Fatalf("current snapshot lost ownership: %+v", current)
	}
}

func TestOwnershipPublicationFailureSurfacesAndRetainsMemory(t *testing.T) {
	root := t.TempDir()
	f := &Fleet{Roots: RootsPlan{Root: root}, ctx: context.Background()}
	first := Resource{ID: "123", Generation: "first", Kind: "agent"}
	f.record(RunEvent{Kind: "own", Resource: first})
	before, err := os.ReadFile(filepath.Join(root, "ownership.json"))
	if err != nil {
		t.Fatal(err)
	}
	// A read-only directory prevents staging a replacement while the complete
	// existing snapshot remains readable. No injected writer or environment seam.
	if err := os.Chmod(root, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(root, 0700)
	second := Resource{ID: "456", Generation: "second", Kind: "agent"}
	f.record(RunEvent{Kind: "own", Resource: second})
	ctx, done, err := f.begin(context.Background())
	if err == nil {
		done()
		t.Fatalf("publication failure was silently discarded: %v", ctx)
	}
	after, readErr := os.ReadFile(filepath.Join(root, "ownership.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(before) != string(after) {
		t.Fatal("failed publication damaged previous complete evidence")
	}
	if len(f.model.Resources) != 2 || f.model.Resources[second.ID+"\x00"+second.Generation] != second {
		t.Fatalf("failed publication erased cleanup authority: %+v", f.model)
	}
}

func TestOwnershipFailureStillCleansExactOwnedProcess(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	joined := make(chan struct{})
	go func() { _ = cmd.Wait(); close(joined) }()
	generation := procutil.Identity(strconv.Itoa(cmd.Process.Pid))
	if generation == "" {
		t.Fatal("child generation unavailable")
	}
	fctx, fcancel := context.WithCancel(context.Background())
	defer fcancel()
	idle := make(chan struct{})
	close(idle)
	f := &Fleet{Roots: RootsPlan{Root: root}, RunID: "manifest-failure-owned-child", t: t, ctx: fctx, cancel: fcancel, idle: idle}
	f.record(RunEvent{Kind: "ready"})
	if err := os.Chmod(root, 0500); err != nil {
		t.Fatal(err)
	}
	f.record(RunEvent{Kind: "own", Resource: Resource{ID: strconv.Itoa(cmd.Process.Pid), Generation: generation, Kind: "test-child"}})
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(ctx); err == nil || !strings.Contains(err.Error(), "publish native ownership evidence") {
		t.Fatalf("cleanup discarded publication failure: %v", err)
	}
	select {
	case <-joined:
	case <-ctx.Done():
		t.Fatal("owned child not joined")
	}
	if err := f.AssertCleaned(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("clean root retained solely for evidence failure: %v", err)
	}
}

func TestUncertainLeaderDescendantIsCollectedByExistingClose(t *testing.T) {
	if os.Getenv("PAIR_TESTCOUCH_ORPHAN_CHILD") == "1" {
		child, stop := context.WithTimeout(t.Context(), 30*time.Second)
		defer stop()
		<-child.Done()
		return
	}
	root := t.TempDir()
	runID := "retained-cli-" + filepath.Base(root)
	cmd := exec.Command("/bin/sh", "-c", `"$1" -test.run=^TestUncertainLeaderDescendantIsCollectedByExistingClose$ </dev/null >/dev/null 2>&1 & printf '%s' "$!"`, "orphan", os.Args[0])
	cmd.Env = []string{"PAIR_TESTCOUCH_RUN_ID=" + runID, "PAIR_TESTCOUCH_ORPHAN_CHILD=1", "PATH=/usr/bin:/bin"}
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(output))
	if err != nil {
		t.Fatal(err)
	}
	generation := procutil.Identity(strconv.Itoa(pid))
	if generation == "" {
		t.Fatal("descendant generation unavailable")
	}
	t.Cleanup(func() {
		if procutil.Identity(strconv.Itoa(pid)) == generation {
			if e := (nativeProcesses{}).Signal(pid, generation, syscall.SIGKILL); e != nil && e != syscall.ESRCH {
				t.Error(e)
			}
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	fctx, fcancel := context.WithCancel(t.Context())
	defer fcancel()
	idle := make(chan struct{})
	close(idle)
	f := &Fleet{Roots: RootsPlan{Root: root}, RunID: runID, t: t, ctx: fctx, cancel: fcancel, idle: idle}
	f.record(RunEvent{Kind: "unknown", Error: "fixture CLI cleanup uncertain"})
	if err := f.captureOwned(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.model.Resources[strconv.Itoa(pid)+"\x00"+generation]; !ok {
		t.Fatal("private-run census lost reparented descendant")
	}
	if err := f.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.AssertCleaned(); err != nil {
		t.Fatal(err)
	}
	rows, err := (nativeProcesses{}).Table(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rows {
		if p.PID == pid && p.Identity == generation && !p.Zombie {
			t.Fatal("Close left owned descendant executing")
		}
	}
}
