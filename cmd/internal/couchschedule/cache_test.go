package couchschedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func cacheFixture(t *testing.T) (Builder, string) {
	t.Helper()
	root := t.TempDir()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\nfunc main(){}\n")
	b := Builder{CacheDir: t.TempDir(), CacheBytes: 4000}
	tick := int64(0)
	b.cacheNow = func() time.Time { tick++; return time.Unix(0, tick) }
	b.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		switch args[0] {
		case "env":
			return []byte(`{}`), nil
		case "tool":
			return []byte("fixture"), nil
		case "list":
			if args[1] == "-m" {
				return json.Marshal(buildModule{Path: "fixture.local/task", Dir: root, GoMod: filepath.Join(root, "go.mod"), Main: true})
			}
			return json.Marshal(buildPackage{Dir: filepath.Join(root, "scripts/scheduled/task"), Name: "main", Module: &buildModule{Path: "fixture.local/task", Dir: root}, GoFiles: []string{"main.go"}})
		case "mod":
			return []byte(`{}`), nil
		case "build":
			for i, v := range args {
				if v == "-o" {
					return nil, os.WriteFile(args[i+1], []byte(strings.Repeat("b", 1000)), 0700)
				}
			}
		}
		return nil, fmt.Errorf("unexpected command %v", args)
	}
	return b, root
}
func cacheBuild(t *testing.T, b Builder, root, version string) Artifact {
	t.Helper()
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\n// "+version+"\nfunc main(){}\n")
	a, e := b.Build(t.Context(), root, "task")
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func TestCacheLRUAndLiveProtection(t *testing.T) {
	b, root := cacheFixture(t)
	a := cacheBuild(t, b, root, "a")
	defer a.Close()
	if e := a.Close(); e != nil {
		t.Fatal(e)
	}
	c := cacheBuild(t, b, root, "c")
	defer c.Close()
	if e := c.Close(); e != nil {
		t.Fatal(e)
	}
	// Two entries fit; touching a makes c the eviction candidate.
	a = cacheBuild(t, b, root, "a")
	defer a.Close()
	d := cacheBuild(t, b, root, "d")
	defer d.Close()
	if _, e := os.Stat(c.Path); !os.IsNotExist(e) {
		t.Fatal("oldest unused entry retained", e)
	}
	if _, e := os.Stat(a.Path); e != nil {
		t.Fatal("live cache entry evicted", e)
	}
	if _, e := os.Stat(filepath.Join(b.CacheDir, c.Key+".lock")); e != nil {
		t.Fatal("publication lock inode removed", e)
	}
}
func TestCacheCapacityRefusesAllLive(t *testing.T) {
	b, root := cacheFixture(t)
	b.CacheBytes = 2500
	a := cacheBuild(t, b, root, "a")
	defer a.Close()
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\n// changed\nfunc main(){}\n")
	if _, e := b.Build(t.Context(), root, "task"); e == nil {
		t.Fatal("cache exceeded budget while all entries live")
	}
	if _, e := os.Stat(a.Path); e != nil {
		t.Fatal(e)
	}
}
func TestCacheUnsafeEntryRefused(t *testing.T) {
	for _, kind := range []string{"unknown", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			b, root := cacheFixture(t)
			a := cacheBuild(t, b, root, "a")
			if e := a.Close(); e != nil {
				t.Fatal(e)
			}
			dir := filepath.Dir(a.Path)
			if kind == "unknown" {
				if e := os.WriteFile(filepath.Join(dir, "unknown"), []byte("keep"), 0600); e != nil {
					t.Fatal(e)
				}
			} else {
				if e := os.Remove(a.Path); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(filepath.Join(root, "go.mod"), a.Path); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := b.Build(t.Context(), root, "task"); e == nil {
				t.Fatal("unsafe entry accepted")
			}
			if _, e := os.Lstat(a.Path); e != nil {
				t.Fatal("unsafe entry changed", e)
			}
		})
	}
}
func TestCacheFailedPublicationNotHitAndCancellation(t *testing.T) {
	b, root := cacheFixture(t)
	calls := 0
	run := b.Run
	b.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		if args[0] == "build" {
			calls++
			if calls == 1 {
				return nil, fmt.Errorf("compile failed")
			}
		}
		return run(ctx, dir, env, args)
	}
	if _, e := b.Build(t.Context(), root, "task"); e == nil {
		t.Fatal("failed build accepted")
	}
	a, e := b.Build(t.Context(), root, "task")
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if calls != 2 {
		t.Fatal("failed publication reused", calls)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, e = b.Build(ctx, root, "task"); e == nil {
		t.Fatal("cancel ignored")
	}
}
func TestCachePublicationFailureCannotBecomeHit(t *testing.T) {
	b, root := cacheFixture(t)
	calls := 0
	run := b.Run
	b.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		if args[0] == "build" {
			calls++
		}
		return run(ctx, dir, env, args)
	}
	b.cacheFault = func(step string) error {
		if step == "publication-ready" {
			return fmt.Errorf("publication failed")
		}
		return nil
	}
	if _, e := b.Build(t.Context(), root, "task"); e == nil {
		t.Fatal("publication fault ignored")
	}
	b.cacheFault = nil
	a, e := b.Build(t.Context(), root, "task")
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if calls != 2 {
		t.Fatal("failed publication became hit", calls)
	}
}
func TestCacheCancellationWhileWaitingMaintenance(t *testing.T) {
	b, root := cacheFixture(t)
	unlock, e := buildLock(t.Context(), filepath.Join(b.CacheDir, ".maintenance.lock"))
	if e != nil {
		t.Fatal(e)
	}
	defer unlock()
	compiled := make(chan struct{})
	run := b.Run
	b.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		out, e := run(ctx, dir, env, args)
		if args[0] == "build" {
			close(compiled)
		}
		return out, e
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { a, e := b.Build(ctx, root, "task"); _ = a.Close(); done <- e }()
	select {
	case <-compiled:
	case <-time.After(2 * time.Second):
		t.Fatal("build did not reach maintenance")
	}
	cancel()
	select {
	case e := <-done:
		if !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("maintenance wait ignored cancellation")
	}
}
func TestCacheOldestLiveEntryIsSkipped(t *testing.T) {
	b, root := cacheFixture(t)
	a := cacheBuild(t, b, root, "a")
	defer a.Close()
	c := cacheBuild(t, b, root, "c")
	if e := c.Close(); e != nil {
		t.Fatal(e)
	}
	d := cacheBuild(t, b, root, "d")
	defer d.Close()
	if _, e := os.Stat(a.Path); e != nil {
		t.Fatal("oldest live entry removed", e)
	}
	if _, e := os.Stat(c.Path); !os.IsNotExist(e) {
		t.Fatal("unused candidate not evicted", e)
	}
}
func TestCacheInvalidLiveBinaryIsNotReplaced(t *testing.T) {
	b, root := cacheFixture(t)
	a := cacheBuild(t, b, root, "a")
	defer a.Close()
	if e := os.WriteFile(a.Path, []byte("changed while owned"), 0700); e != nil {
		t.Fatal(e)
	}
	if _, e := b.Build(t.Context(), root, "task"); !errors.Is(e, errCacheLive) {
		t.Fatal("live invalid binary replaced", e)
	}
	raw, e := os.ReadFile(a.Path)
	if e != nil || string(raw) != "changed while owned" {
		t.Fatal(string(raw), e)
	}
}
