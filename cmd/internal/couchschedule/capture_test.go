package couchschedule

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestBuildUsesPrivateRecoverableCapture(t *testing.T) {
	root := t.TempDir()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
	cache := filepath.Join(t.TempDir(), "cache")
	sentinel := errors.New("stop before discovery")
	b := Builder{CacheDir: cache, Run: func(context.Context, string, []string, []string) ([]byte, error) { return nil, sentinel }}
	if _, err := b.Build(context.Background(), root, "task"); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	names, err := os.ReadDir(cache + ".captures")
	if err != nil {
		t.Fatalf("private capture pool missing: %v", err)
	}
	if len(names) != 5 {
		t.Fatalf("fixed pool has %d names, want allocation lock, two leases and two slots", len(names))
	}
	for _, name := range []string{"slot-0", "slot-1"} {
		children, err := os.ReadDir(filepath.Join(cache+".captures", name))
		if err != nil || len(children) != 0 {
			t.Fatalf("capture cleanup %s: %v %v", name, children, err)
		}
	}
}

func TestCaptureSlotsBoundAndCancel(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	a, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	defer a.close()
	b, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	defer b.close()
	if a.path == b.path {
		t.Fatal("shared capture generation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, e = acquireBuildScratch(ctx, cache); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("third capture: %v", e)
	}
}

func TestCaptureReplacementAndUnknownProtected(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	a, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	original := a.path + "-moved"
	if e = os.Rename(a.path, original); e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(a.path, 0700); e != nil {
		t.Fatal(e)
	}
	marker := filepath.Join(a.path, "capture.mod")
	if e = os.WriteFile(marker, []byte("replacement"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = a.close(); e == nil {
		t.Fatal("replacement accepted")
	}
	if got, e := os.ReadFile(marker); e != nil || string(got) != "replacement" {
		t.Fatalf("replacement lost: %q %v", got, e)
	}
	// The ambiguous second occupant refuses this slot; another slot still works.
	b, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	defer b.close()
	if b.slot == a.slot {
		t.Fatal("ambiguous slot reused")
	}
}

func TestCaptureRecoveryAbruptDeath(t *testing.T) {
	for _, mode := range []string{"capture", "default-command", "owned-command"} {
		t.Run(mode, func(t *testing.T) {
			cache := filepath.Join(t.TempDir(), "cache")
			ready := filepath.Join(t.TempDir(), "ready")
			release := ready + "-release"
			cmd := exec.Command(os.Args[0], "-test.run=^TestCaptureDeathHelper$")
			cmd.Env = append(os.Environ(), "PAIR_CAPTURE_HELPER="+mode, "PAIR_CAPTURE_CACHE="+cache, "PAIR_CAPTURE_READY="+ready, "PAIR_CAPTURE_RELEASE="+release)
			if out, e := cmd.CombinedOutput(); e == nil {
				t.Fatalf("helper did not die: %s", out)
			} else if exit, ok := e.(*exec.ExitError); !ok || exit.ExitCode() != 73 {
				t.Fatalf("helper failure: %v %s", e, out)
			}
			var old string
			slots, _ := os.ReadDir(cache + ".captures")
			for _, slot := range slots {
				if !slot.IsDir() {
					continue
				}
				children, _ := os.ReadDir(filepath.Join(cache+".captures", slot.Name()))
				if len(children) > 0 {
					old = filepath.Join(cache+".captures", slot.Name(), children[0].Name())
					break
				}
			}
			if old == "" {
				t.Fatal("missing abandoned capture")
			}
			if mode != "capture" {
				data, e := os.ReadFile(ready)
				if e != nil {
					t.Fatal(e)
				}
				pid, e := strconv.Atoi(string(data))
				if e != nil {
					t.Fatal(e)
				}
				defer syscall.Kill(pid, syscall.SIGKILL)
				other, e := acquireBuildScratch(context.Background(), cache)
				if e != nil {
					t.Fatal(e)
				}
				if other.slot == filepath.Dir(old) {
					t.Fatal("live Go command lost lease")
				}
				if _, e = os.Stat(filepath.Join(old, "capture.mod")); e != nil {
					t.Fatalf("live scratch removed: %v", e)
				}
				other.close()
				if e = os.WriteFile(release, nil, 0600); e != nil {
					t.Fatal(e)
				}
				end := time.Now().Add(5 * time.Second)
				for {
					lease, e := cacheFileLease(filepath.Dir(old)+".lock", unix.LOCK_EX)
					if e == nil {
						lease.Close()
						break
					}
					if time.Now().After(end) {
						t.Fatal("Go command did not release lease")
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
			next, e := acquireBuildScratch(context.Background(), cache)
			if e != nil {
				t.Fatal(e)
			}
			defer next.close()
			if _, e = os.Stat(old); !os.IsNotExist(e) {
				t.Fatalf("abandoned generation remains: %v", e)
			}
		})
	}
}

func TestCaptureDeathHelper(t *testing.T) {
	mode := os.Getenv("PAIR_CAPTURE_HELPER")
	if mode == "" {
		return
	}
	if mode == "go" {
		f := os.NewFile(3, "capture-lease")
		if _, e := f.Stat(); e != nil {
			os.Exit(75)
		}
		if e := os.WriteFile(os.Getenv("PAIR_CAPTURE_READY"), []byte(strconv.Itoa(os.Getpid())), 0600); e != nil {
			os.Exit(76)
		}
		for {
			if e := os.WriteFile(filepath.Join(os.Getenv("PAIR_CAPTURE_PATH"), "task"), []byte("live writer"), 0600); e != nil {
				os.Exit(77)
			}
			if _, e := os.Stat(os.Getenv("PAIR_CAPTURE_RELEASE")); e == nil {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		f.Close()
		return
	}
	cache := os.Getenv("PAIR_CAPTURE_CACHE")
	root := filepath.Join(filepath.Dir(cache), "root")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture.local/task\n\ngo 1.23\n"), 0600); e != nil {
		t.Fatal(e)
	}
	b := Builder{CacheDir: cache}
	b.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		if mode == "capture" {
			os.Exit(73)
		}
		paths, e := filepath.Glob(cache + ".captures/slot-*/capture-*")
		if e != nil || len(paths) != 1 {
			os.Exit(78)
		}
		env = append(os.Environ(), "PAIR_CAPTURE_HELPER=go", "PAIR_CAPTURE_PATH="+paths[0])
		go func() {
			end := time.Now().Add(5 * time.Second)
			for time.Now().Before(end) {
				if _, e := os.Stat(os.Getenv("PAIR_CAPTURE_READY")); e == nil {
					os.Exit(73)
				}
				time.Sleep(5 * time.Millisecond)
			}
			os.Exit(74)
		}()
		if mode == "default-command" {
			return (Builder{GoBinary: os.Args[0]}).command(ctx, dir, env, "-test.run=^TestCaptureDeathHelper$")
		}
		_, e = ownedGroupCommand(ctx, os.Args[0], dir, env, []string{"-test.run=^TestCaptureDeathHelper$"}, io.Discard, io.Discard, false)
		return nil, e
	}
	if _, e := b.Build(context.Background(), root, "task"); e != nil {
		t.Fatal(e)
	}
	t.Fatal("build unexpectedly completed")
}

func TestCaptureRecoveryOversizedKnownAndUnsafeMembers(t *testing.T) {
	for _, shape := range []string{"oversized-task", "symlink", "hardlink", "unknown"} {
		t.Run(shape, func(t *testing.T) {
			cache := filepath.Join(t.TempDir(), "cache")
			a, e := acquireBuildScratch(context.Background(), cache)
			if e != nil {
				t.Fatal(e)
			}
			task := filepath.Join(a.path, "task")
			switch shape {
			case "oversized-task":
				f, e := os.Create(task)
				if e != nil {
					t.Fatal(e)
				}
				e = f.Truncate(300 << 20)
				f.Close()
				if e != nil {
					t.Fatal(e)
				}
			case "symlink":
				if e = os.Symlink(t.TempDir(), task); e != nil {
					t.Fatal(e)
				}
			case "hardlink":
				other := filepath.Join(t.TempDir(), "task")
				if e = os.WriteFile(other, []byte("external"), 0600); e != nil {
					t.Fatal(e)
				}
				if e = os.Link(other, task); e != nil {
					t.Fatal(e)
				}
			case "unknown":
				if e = os.WriteFile(filepath.Join(a.path, "unknown"), nil, 0600); e != nil {
					t.Fatal(e)
				}
			}
			// Simulate death: release descriptors without performing cleanup.
			a.lease.Close()
			unix.Close(a.retainedFD)
			b, e := acquireBuildScratch(context.Background(), cache)
			if e != nil {
				t.Fatal(e)
			}
			defer b.close()
			if shape == "oversized-task" {
				if _, e = os.Lstat(a.path); !os.IsNotExist(e) {
					t.Fatalf("oversized known task not reclaimed: %v", e)
				}
			} else {
				if _, e = os.Lstat(a.path); e != nil {
					t.Fatalf("unsafe capture deleted: %v", e)
				}
				if b.slot == a.slot {
					t.Fatal("unsafe capture slot reused")
				}
			}
		})
	}
}

func TestCaptureSnapshotReplacementProtected(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	a, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	defer a.close()
	task := filepath.Join(a.path, "task")
	if e = os.WriteFile(task, []byte("original"), 0600); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open(a.path)
	if e != nil {
		t.Fatal(e)
	}
	tree, e := snapshotCapture(context.Background(), f, 0, &captureBounds{})
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(task, filepath.Join(t.TempDir(), "moved")); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(task, []byte("replacement"), 0600); e != nil {
		t.Fatal(e)
	}
	parent, e := os.Open(a.slot)
	if e != nil {
		t.Fatal(e)
	}
	defer parent.Close()
	if e = unlinkCapture(context.Background(), parent, filepath.Base(a.path), tree); e == nil {
		t.Fatal("replacement accepted")
	}
	if got, e := os.ReadFile(task); e != nil || string(got) != "replacement" {
		t.Fatalf("replacement lost: %q %v", got, e)
	}
}

func TestBuildUncertainCaptureJoinAndGarbageCollection(t *testing.T) {
	for _, ignored := range []bool{false, true} {
		t.Run(strconv.FormatBool(ignored), func(t *testing.T) {
			cache := filepath.Join(t.TempDir(), "cache")
			root := t.TempDir()
			writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
			cmd := exec.Command("sleep", "30")
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if e := cmd.Start(); e != nil {
				t.Fatal(e)
			}
			defer func() { cmd.Process.Kill(); cmd.Wait() }()
			b := Builder{CacheDir: cache, Run: func(context.Context, string, []string, []string) ([]byte, error) {
				return nil, &groupUncertain{pid: cmd.Process.Pid, err: errors.New("injected descendant uncertainty")}
			}}
			_, err := b.Build(context.Background(), root, "task")
			var cleanup BuildCleanupError
			if !errors.As(err, &cleanup) {
				t.Fatalf("missing cleanup owner: %v", err)
			}
			var unknown *groupUncertain
			errors.As(err, &unknown)
			raw := unknown.capture.retainedFD
			old := unknown.capture.path
			// Explicit close simulates collection of the managed descriptor. The raw
			// duplicate must retain the open-file-description lock independently.
			unknown.capture.lease.Close()
			if ignored {
				cleanup = nil
				unknown = nil
				err = nil
			}
			runtime.GC()
			runtime.GC()
			other, e := acquireBuildScratch(context.Background(), cache)
			if e != nil {
				t.Fatal(e)
			}
			defer other.close()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			if _, e = acquireBuildScratch(ctx, cache); !errors.Is(e, context.DeadlineExceeded) {
				t.Fatalf("uncertain lease lost: %v", e)
			}
			if ignored {
				if e = unix.Close(raw); e != nil {
					t.Fatal(e)
				}
			} else {
				cmd.Process.Kill()
				cmd.Wait()
				joinCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if e = cleanup.Join(joinCtx); e != nil {
					t.Fatal(e)
				}
				if _, e = os.Stat(old); !os.IsNotExist(e) {
					t.Fatalf("joined capture retained: %v", e)
				}
			}
		})
	}
}

func TestCaptureMaterializationBounds(t *testing.T) {
	mapped := map[string]string{"module": "m0000"}
	if e := validateCaptureInputs([]buildInput{{Module: "module", Path: strings.Repeat("a/", 129) + "file"}}, mapped); e == nil {
		t.Fatal("deep capture accepted")
	}
	var inputs []buildInput
	for i := 0; i < 10000; i++ {
		inputs = append(inputs, buildInput{Module: "module", Path: fmt.Sprintf("%d/a/b/file", i)})
	}
	if e := validateCaptureInputs(inputs, mapped); e == nil {
		t.Fatal("unrecoverable directory count accepted")
	}
}

func TestCaptureRefusesSymlinkAncestorBeforeCreation(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	if e := os.Symlink(outside, link); e != nil {
		t.Fatal(e)
	}
	if _, e := acquireBuildScratch(context.Background(), filepath.Join(link, "missing", "tail", "cache")); e == nil {
		t.Fatal("symlink ancestor accepted")
	}
	if names, e := os.ReadDir(outside); e != nil || len(names) != 0 {
		t.Fatalf("created captures through symlink: %v %v", names, e)
	}
}

func TestCaptureRootUsesCanonicalCachePath(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	a, e := acquireBuildScratch(context.Background(), cache+string(filepath.Separator))
	if e != nil {
		t.Fatal(e)
	}
	defer a.close()
	if a.root != cache+".captures" {
		t.Fatalf("capture root %s, want %s", a.root, cache+".captures")
	}
}

func TestBuildCleanupConcurrentJoinWaitsForGroup(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	a, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	unknown := &groupUncertain{pid: cmd.Process.Pid, err: errors.New("cleanup uncertain"), capture: a}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	if e = unknown.Join(ctx); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatalf("Join stopped live group or released lease: %v", e)
	}
	cancel()
	if e = cmd.Process.Signal(syscall.Signal(0)); e != nil {
		t.Fatalf("Join signalled unrelated or live group: %v", e)
	}
	results := make(chan error, 8)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for i := 0; i < 8; i++ {
		go func() { results <- unknown.Join(ctx) }()
	}
	cmd.Process.Kill()
	cmd.Wait()
	for i := 0; i < 8; i++ {
		if e = <-results; e != nil {
			t.Fatal(e)
		}
	}
	if _, e = os.Stat(a.path); !os.IsNotExist(e) {
		t.Fatalf("joined capture retained: %v", e)
	}
}

func TestBuildCleanupJoinCancellationWhileAllocationHeld(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	a, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if e = cmd.Run(); e != nil {
		t.Fatal(e)
	}
	unknown := &groupUncertain{pid: cmd.Process.Pid, capture: a}
	unlock, e := buildLock(context.Background(), filepath.Join(a.root, "allocation.lock"))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- unknown.Join(ctx) }()
	select {
	case e = <-done:
		unlock()
		if !errors.Is(e, context.DeadlineExceeded) {
			t.Fatalf("cleanup cancellation: %v", e)
		}
	case <-time.After(200 * time.Millisecond):
		unlock()
		<-done
		t.Fatal("Join ignored context behind allocation lock")
	}
	lease, e := cacheFileLease(a.slot+".lock", unix.LOCK_EX)
	if e != nil {
		t.Fatalf("dead-writer slot remained locked: %v", e)
	}
	lease.Close()
	if _, e = os.Stat(a.path); e != nil {
		t.Fatalf("recoverable residue lost: %v", e)
	}
}

func TestBuildDefaultCommandCancellationJoinsWriter(t *testing.T) {
	root := t.TempDir()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
	cache := filepath.Join(t.TempDir(), "cache")
	ready := filepath.Join(t.TempDir(), "ready")
	b := Builder{CacheDir: cache, Run: func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		return (Builder{GoBinary: "sh"}).command(ctx, dir, env, "-c", `trap '' TERM; echo $$ > "$1"; while :; do sleep 1; done`, "sh", ready)
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, e := b.Build(ctx, root, "task"); done <- e }()
	end := time.Now().Add(5 * time.Second)
	var pid int
	for {
		data, e := os.ReadFile(ready)
		if e == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
			if pid > 0 {
				break
			}
		}
		if time.Now().After(end) {
			t.Fatal("writer did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case e := <-done:
		if !errors.Is(e, context.Canceled) {
			t.Fatalf("cancel: %v", e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("writer not joined")
	}
	if gone, e := ownedGroupGone(pid); e != nil || !gone {
		t.Fatalf("writer group alive after Build returned: %v %v", gone, e)
	}
	for _, slot := range []string{"slot-0", "slot-1"} {
		names, e := os.ReadDir(filepath.Join(cache+".captures", slot))
		if e != nil || len(names) != 0 {
			t.Fatalf("capture cleanup: %v %v", names, e)
		}
	}
}

func TestBuildCleanupConcurrentJoinHonorsEachContext(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	a, e := acquireBuildScratch(context.Background(), cache)
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command("true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if e = cmd.Run(); e != nil {
		t.Fatal(e)
	}
	unknown := &groupUncertain{pid: cmd.Process.Pid, capture: a}
	unlock, e := buildLock(context.Background(), filepath.Join(a.root, "allocation.lock"))
	if e != nil {
		t.Fatal(e)
	}
	firstCtx, firstCancel := context.WithTimeout(context.Background(), time.Second)
	defer firstCancel()
	first := make(chan error, 1)
	go func() { first <- unknown.Join(firstCtx) }()
	time.Sleep(20 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	second := make(chan error, 1)
	go func() { second <- unknown.Join(ctx) }()
	timedOut := false
	select {
	case e = <-second:
		if !errors.Is(e, context.DeadlineExceeded) {
			unlock()
			<-first
			t.Fatalf("second Join: %v", e)
		}
	case <-time.After(200 * time.Millisecond):
		timedOut = true
	}
	unlock()
	<-first
	if timedOut {
		<-second
		t.Fatal("concurrent Join ignored its own context")
	}
}
