package couchschedule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func cacheFixture(t *testing.T) (Builder, string) {
	t.Helper()
	return cacheFixtureDirs(t, t.TempDir(), t.TempDir())
}
func cacheFixtureDirs(t *testing.T, root, cache string) (Builder, string) {
	t.Helper()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\nfunc main(){}\n")
	b := Builder{CacheDir: cache, CacheBytes: 4000}
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
	b.AfterCapture = func() { close(compiled) }
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { a, e := b.Build(ctx, root, "task"); _ = a.Close(); done <- e }()
	select {
	case <-compiled:
	case <-time.After(2 * time.Second):
		t.Fatal("build did not reach cache admission")
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
func TestCacheFutureManifestMetadataSurvivesReuseAndRebuild(t *testing.T) {
	b, root := cacheFixture(t)
	a := cacheBuild(t, b, root, "a")
	_ = a.Close()
	path := filepath.Join(filepath.Dir(a.Path), "manifest.json")
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	raw = bytes.Replace(raw, []byte(`"Inputs":[{`), []byte(`"future_metadata":{ "enabled": true },"Inputs":[{"future_input":{ "rev": 2 },`), 1)
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	for _, rebuild := range []bool{false, true} {
		if rebuild {
			if e = os.WriteFile(a.Path, []byte("corrupt"), 0700); e != nil {
				t.Fatal(e)
			}
		}
		before := append([]byte(nil), raw...)
		next, e := b.Build(t.Context(), root, "task")
		if e != nil {
			t.Fatal("future metadata blocked build", e)
		}
		_ = next.Close()
		raw, e = os.ReadFile(path)
		if !rebuild && e == nil && !bytes.Equal(raw, before) {
			t.Fatal("cache touch rewrote manifest bytes")
		}
		if e != nil || !bytes.Contains(raw, []byte(`{ "enabled": true }`)) || !bytes.Contains(raw, []byte(`{ "rev": 2 }`)) {
			t.Fatal("future metadata lost", string(raw), e)
		}
	}
}

func TestCacheCrashHelper(t *testing.T) {
	step := os.Getenv("PAIR_CACHE_CRASH_STEP")
	if step == "" {
		return
	}
	b, root := cacheFixtureDirs(t, os.Getenv("PAIR_CACHE_CRASH_ROOT"), os.Getenv("PAIR_CACHE_CRASH_CACHE"))
	if step == "legacy-manifest-temp" || step == "legacy-binary-temp" || step == "legacy-half-eviction" {
		a := cacheBuild(t, b, root, "a")
		dir := filepath.Dir(a.Path)
		if step == "legacy-half-eviction" {
			if e := os.Remove(filepath.Join(dir, "manifest.json")); e != nil {
				t.Fatal(e)
			}
		} else {
			prefix := ".manifest-123456789"
			if step == "legacy-binary-temp" {
				prefix = ".binary-123456789"
			}
			if e := os.WriteFile(filepath.Join(dir, prefix), []byte("interrupted writer"), 0600); e != nil {
				t.Fatal(e)
			}
		}
		os.Exit(73)
	}
	b.cacheFault = func(at string) error {
		if at == step {
			os.Exit(73)
		}
		return nil
	}
	a := cacheBuild(t, b, root, os.Getenv("PAIR_CACHE_CRASH_VERSION"))
	_ = a.Close()
	t.Fatalf("checkpoint %s was never reached", step)
}
func crashCache(t *testing.T, b Builder, root, version, step string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCacheCrashHelper$")
	cmd.Env = append(os.Environ(), "PAIR_CACHE_CRASH_STEP="+step, "PAIR_CACHE_CRASH_ROOT="+root, "PAIR_CACHE_CRASH_CACHE="+b.CacheDir, "PAIR_CACHE_CRASH_VERSION="+version)
	out, e := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(e, &exit) || exit.ExitCode() != 73 {
		t.Fatalf("crash helper %s: %s %v", step, out, e)
	}
}
func TestCacheAbruptMutationRecovery(t *testing.T) {
	for _, scenario := range []struct {
		step, version string
		seed          bool
	}{{"publication-created", "a", false}, {"publication-task-written", "a", false}, {"publication-manifest-written", "a", false}, {"publication-ready", "a", false}, {"publication-renamed", "a", false}, {"manifest-update-ready", "a", true}, {"eviction-reserved", "d", true}, {"eviction-renamed", "d", true}, {"cleanup-task", "d", true}, {"cleanup-manifest.json", "d", true}, {"cleanup-removed", "d", true}} {
		t.Run(scenario.step, func(t *testing.T) {
			b, root := cacheFixture(t)
			if scenario.seed {
				a := cacheBuild(t, b, root, "a")
				_ = a.Close()
				c := cacheBuild(t, b, root, "c")
				_ = c.Close()
			}
			crashCache(t, b, root, scenario.version, scenario.step)
			b.cacheRecoveryNow = func() time.Time { return time.Now().Add(2 * time.Minute) }
			a := cacheBuild(t, b, root, scenario.version)
			defer a.Close()
			if _, e := inspectCache(t.Context(), b.CacheDir); e != nil {
				t.Fatal("abrupt mutation blocked namespace", e)
			}
			ns, e := cacheNames(b.CacheDir)
			if e != nil {
				t.Fatal(e)
			}
			for _, name := range ns {
				if privateCacheName.MatchString(name) {
					t.Fatal("aged mutation residue retained", name)
				}
			}
		})
	}
}
func TestCacheKnownCorruptEntryRebuilds(t *testing.T) {
	for _, kind := range []string{"missing-manifest", "malformed-manifest"} {
		t.Run(kind, func(t *testing.T) {
			b, root := cacheFixture(t)
			a := cacheBuild(t, b, root, "a")
			_ = a.Close()
			path := filepath.Join(filepath.Dir(a.Path), "manifest.json")
			var e error
			if kind == "missing-manifest" {
				e = os.Remove(path)
			} else {
				e = os.WriteFile(path, []byte(`{"broken":`), 0600)
			}
			if e != nil {
				t.Fatal(e)
			}
			rebuilt, e := b.Build(t.Context(), root, "task")
			if e != nil {
				t.Fatal("corrupt known cache cannot rebuild", e)
			}
			defer rebuilt.Close()
		})
	}
}
func TestCacheResidueReplacementRefused(t *testing.T) {
	b, _ := cacheFixture(t)
	path := filepath.Join(b.CacheDir, ".publication-123456789")
	if e := os.Mkdir(path, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(path, "task"), []byte("partial"), 0700); e != nil {
		t.Fatal(e)
	}
	old := time.Now().Add(-2 * time.Minute)
	if e := os.Chtimes(path, old, old); e != nil {
		t.Fatal(e)
	}
	replaced := false
	b.cacheFault = func(step string) error {
		if step == "cleanup-observed" && !replaced {
			replaced = true
			if e := os.Rename(path, path+"-saved"); e != nil {
				return e
			}
			if e := os.Mkdir(path, 0700); e != nil {
				return e
			}
			return os.WriteFile(filepath.Join(path, "task"), []byte("replacement"), 0700)
		}
		return nil
	}
	if e := b.recoverCache(t.Context(), b.CacheDir); e == nil {
		t.Fatal("replacement accepted")
	}
	raw, e := os.ReadFile(filepath.Join(path, "task"))
	if e != nil || string(raw) != "replacement" {
		t.Fatal(string(raw), e)
	}
	raw, e = os.ReadFile(filepath.Join(path+"-saved", "task"))
	if e != nil || string(raw) != "partial" {
		t.Fatal("observed source changed", string(raw), e)
	}
}
func TestCacheFreshResidueAccountedAndUnknownShapeRetained(t *testing.T) {
	b, root := cacheFixture(t)
	path := filepath.Join(b.CacheDir, fmt.Sprintf(".publication-%d-123", time.Now().UnixNano()))
	if e := os.Mkdir(path, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(path, "task"), []byte("partial"), 0700); e != nil {
		t.Fatal(e)
	}
	a := cacheBuild(t, b, root, "a")
	_ = a.Close()
	if _, e := os.Stat(path); e != nil {
		t.Fatal("fresh residue removed", e)
	}
	if e := os.WriteFile(filepath.Join(path, "unknown"), []byte("operator data"), 0600); e != nil {
		t.Fatal(e)
	}
	b.cacheRecoveryNow = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if e := b.recoverCache(t.Context(), b.CacheDir); e == nil {
		t.Fatal("unknown residue shape accepted")
	}
	if raw, e := os.ReadFile(filepath.Join(path, "unknown")); e != nil || string(raw) != "operator data" {
		t.Fatal(string(raw), e)
	}
}

func TestCacheAbruptLegacyMutationRecovery(t *testing.T) {
	for _, step := range []string{"legacy-manifest-temp", "legacy-binary-temp", "legacy-half-eviction"} {
		t.Run(step, func(t *testing.T) {
			b, root := cacheFixture(t)
			a := cacheBuild(t, b, root, "a")
			_ = a.Close()
			crashCache(t, b, root, "a", step)
			b.cacheRecoveryNow = func() time.Time { return time.Now().Add(2 * time.Minute) }
			other := cacheBuild(t, b, root, "d")
			defer other.Close()
			rebuilt := cacheBuild(t, b, root, "a")
			defer rebuilt.Close()
			ns, e := os.ReadDir(filepath.Dir(rebuilt.Path))
			if e != nil || len(ns) != 2 {
				t.Fatal("legacy mutation survived", ns, e)
			}
		})
	}
}
func TestCacheRecoveryProtectsNewLiveKey(t *testing.T) {
	b, root := cacheFixture(t)
	a := cacheBuild(t, b, root, "a")
	_ = a.Close()
	c := cacheBuild(t, b, root, "c")
	_ = c.Close()
	crashCache(t, b, root, "d", "eviction-renamed")
	b.CacheBytes = 12000
	replacement := cacheBuild(t, b, root, "a")
	defer replacement.Close()
	b.cacheRecoveryNow = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if e := b.recoverCache(t.Context(), b.CacheDir); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(replacement.Path); e != nil {
		t.Fatal("new live source removed while reclaiming old trash", e)
	}
}
func TestCacheFreshResidueCountsAgainstCapacity(t *testing.T) {
	b, root := cacheFixture(t)
	a := cacheBuild(t, b, root, "a")
	_ = a.Close()
	entries, e := inspectCache(t.Context(), b.CacheDir)
	if e != nil || len(entries) != 1 {
		t.Fatal(entries, e)
	}
	b.CacheBytes = entries[0].bytes + 500
	stage, e := os.MkdirTemp(b.CacheDir, fmt.Sprintf(".publication-%d-", time.Now().UnixNano()))
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(stage, "task"), bytes.Repeat([]byte("x"), 1000), 0700); e != nil {
		t.Fatal(e)
	}
	if _, e = b.Build(t.Context(), root, "task"); !errors.Is(e, ErrCacheCapacity) {
		t.Fatal("fresh residue ignored in budget", e)
	}
	if _, e = os.Stat(stage); e != nil {
		t.Fatal("fresh residue removed before grace", e)
	}
}

func TestCacheZeroBytePartialEntriesRebuild(t *testing.T) {
	for _, members := range [][]string{{"task"}, {"manifest.json"}, {"task", "manifest.json"}} {
		t.Run(strings.Join(members, "+"), func(t *testing.T) {
			b, root := cacheFixture(t)
			a := cacheBuild(t, b, root, "a")
			if e := a.Close(); e != nil {
				t.Fatal(e)
			}
			dir := filepath.Dir(a.Path)
			for _, name := range []string{"task", "manifest.json"} {
				if e := os.Remove(filepath.Join(dir, name)); e != nil {
					t.Fatal(e)
				}
			}
			for _, name := range members {
				if e := os.WriteFile(filepath.Join(dir, name), nil, 0600); e != nil {
					t.Fatal(e)
				}
			}
			rebuilt, e := b.Build(t.Context(), root, "task")
			if e != nil {
				t.Fatal("zero-byte partial entry could not rebuild", e)
			}
			defer rebuilt.Close()
			st, e := os.Stat(rebuilt.Path)
			if e != nil || st.Size() == 0 {
				t.Fatal("empty artifact retained", st, e)
			}
		})
	}
}

func TestCachePublicationPreservesReplacements(t *testing.T) {
	for _, target := range []string{"directory", "task", "manifest.json"} {
		for _, returnError := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/error=%t", target, returnError), func(t *testing.T) {
				b, root := cacheFixture(t)
				saved := filepath.Join(t.TempDir(), "observed")
				var foreign string
				b.cacheFault = func(step string) error {
					if step != "publication-ready" {
						return nil
					}
					stages, err := filepath.Glob(filepath.Join(b.CacheDir, ".publication-*"))
					if err != nil || len(stages) != 1 {
						return fmt.Errorf("publication inventory: %v %v", stages, err)
					}
					stage := stages[0]
					if target == "directory" {
						if err = os.Rename(stage, saved); err != nil {
							return err
						}
						if err = os.Mkdir(stage, 0700); err != nil {
							return err
						}
						foreign = filepath.Join(stage, "task")
					} else {
						foreign = filepath.Join(stage, target)
						if err = os.Rename(foreign, saved); err != nil {
							return err
						}
					}
					if err = os.WriteFile(foreign, []byte("foreign replacement"), 0600); err != nil {
						return err
					}
					if returnError {
						return errors.New("returned publication fault")
					}
					return nil
				}
				if _, err := b.Build(t.Context(), root, "task"); err == nil {
					t.Fatal("publication fault ignored")
				}
				raw, err := os.ReadFile(foreign)
				if err != nil || string(raw) != "foreign replacement" {
					t.Fatalf("replacement lost: %q %v", raw, err)
				}
			})
		}
	}
}
