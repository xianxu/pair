package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseProfileRequest(t *testing.T) {
	for _, raw := range []string{"", "relative", "/absolute/../unclean", filepath.Join(t.TempDir(), "profile with spaces")} {
		got, err := ParseProfileRequest(raw)
		valid := raw == "" || (filepath.IsAbs(raw) && filepath.Clean(raw) == raw)
		if (err == nil) != valid || err == nil && got.Dir != raw {
			t.Fatalf("raw=%q request=%+v err=%v", raw, got, err)
		}
	}
}

type profileTestClock struct {
	entered chan time.Duration
	advance chan struct{}
}

func newProfileTestClock() *profileTestClock {
	return &profileTestClock{make(chan time.Duration, 8), make(chan struct{})}
}
func (c *profileTestClock) Wait(ctx context.Context, d time.Duration) bool {
	c.entered <- d
	select {
	case <-ctx.Done():
		return false
	case <-c.advance:
		return true
	}
}

type profileTestBackend struct {
	mu                       sync.Mutex
	writer                   io.Writer
	starts, stops, snapshots int
	busy                     bool
	startEntered, allowStart chan struct{}
}

func (b *profileTestBackend) StartCPU(w io.Writer) error {
	if b.startEntered != nil {
		close(b.startEntered)
		<-b.allowStart
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.starts++
	if b.busy {
		return errors.New("already owned")
	}
	if b.writer != nil {
		panic("double profile acquisition")
	}
	b.writer = w
	_, _ = w.Write([]byte("CPU"))
	return nil
}
func (b *profileTestBackend) StopCPU() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.writer == nil {
		panic("unowned StopCPU")
	}
	b.stops++
	_, _ = b.writer.Write([]byte("JOINED"))
	b.writer = nil
}
func (b *profileTestBackend) Allocations(w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.snapshots++
	_, err := w.Write([]byte("ALLOCS"))
	return err
}
func (b *profileTestBackend) counts() (int, int, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.starts, b.stops, b.snapshots
}

func awaitProfileWait(t *testing.T, c *profileTestClock, want time.Duration) {
	t.Helper()
	select {
	case got := <-c.entered:
		if got != want {
			t.Fatalf("wait=%s want%s", got, want)
		}
	case <-testwait.After(t):
		t.Fatal("profile never reached clock")
	}
}

func TestProfileWindowCompletionAndPrivateOutputs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profiles")
	clock := newProfileTestClock()
	backend := &profileTestBackend{}
	complete := make(chan string, 8)
	window, err := StartProfileWindow(context.Background(), ProfileRequest{Dir: dir}, profileDependencies{clock: clock, backend: backend}, func(s string) { complete <- s })
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	awaitProfileWait(t, clock, time.Minute)
	clock.advance <- struct{}{}
	awaitProfileWait(t, clock, 30*time.Second)
	clock.advance <- struct{}{}
	select {
	case <-window.done:
	case <-testwait.After(t):
		t.Fatal("capture did not complete")
	}
	if err := window.Close(); err != nil {
		t.Fatal(err)
	}
	if a, b, c := backend.counts(); a != 1 || b != 1 || c != 2 {
		t.Fatalf("backend=%d/%d/%d", a, b, c)
	}
	info, err := os.Stat(dir)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("private directory: %v %v", info, err)
	}
	for _, name := range []string{"cpu.pprof", "allocs-before.pprof", "allocs-after.pprof", "metadata.json"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.Mode().Perm() != 0600 || info.Size() == 0 {
			t.Fatalf("private output %s: %v %v", name, info, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil || !bytes.Contains(raw, []byte(`"status":"complete"`)) {
		t.Fatalf("manifest=%s err=%v", raw, err)
	}
	if _, err := StartProfileWindow(context.Background(), ProfileRequest{Dir: dir}, profileDependencies{}, nil); err == nil {
		t.Fatal("existing artifacts were reused")
	}
}

func TestProfileWindowCancellationOwnership(t *testing.T) {
	for _, phase := range []string{"warmup", "acquiring", "capturing", "busy"} {
		t.Run(phase, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "profiles")
			clock := newProfileTestClock()
			backend := &profileTestBackend{busy: phase == "busy"}
			if phase == "acquiring" {
				backend.startEntered = make(chan struct{})
				backend.allowStart = make(chan struct{})
			}
			window, err := StartProfileWindow(context.Background(), ProfileRequest{Dir: dir}, profileDependencies{clock: clock, backend: backend}, nil)
			if err != nil {
				t.Fatal(err)
			}
			awaitProfileWait(t, clock, time.Minute)
			if phase != "warmup" {
				clock.advance <- struct{}{}
				if phase == "acquiring" {
					<-backend.startEntered
				} else if phase == "capturing" {
					awaitProfileWait(t, clock, 30*time.Second)
				} else {
					<-window.done
				}
			}
			closed := make(chan error, 1)
			go func() { closed <- window.Close() }()
			if phase == "acquiring" {
				select {
				case <-closed:
					t.Fatal("Close failed to join held acquisition")
				default:
				}
				window.cancel()
				close(backend.allowStart)
			}
			var closeErr error
			select {
			case closeErr = <-closed:
			case <-testwait.After(t):
				t.Fatal("Close did not join")
			}
			if (closeErr != nil) != (phase == "busy") {
				t.Fatalf("Close=%v", closeErr)
			}
			a, b, _ := backend.counts()
			if phase == "warmup" && (a != 0 || b != 0) || phase == "busy" && (a != 1 || b != 0) || (phase == "acquiring" || phase == "capturing") && (a != 1 || b != 1) {
				t.Fatalf("starts=%d stops=%d", a, b)
			}
			for range 3 {
				if err := window.Close(); (err != nil) != (phase == "busy") {
					t.Fatalf("repeat Close=%v", err)
				}
			}
		})
	}
}

type profileFaultFile struct {
	io.WriteCloser
	writeErr, closeErr error
}

func (f profileFaultFile) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.WriteCloser.Write(p)
}
func (f profileFaultFile) Close() error { return errors.Join(f.WriteCloser.Close(), f.closeErr) }

type profileFaultStorage struct {
	osProfileStorage
	name                          string
	createErr, writeErr, closeErr error
}

func (s profileFaultStorage) Create(dir, name string) (io.WriteCloser, error) {
	if name == s.name && s.createErr != nil {
		return nil, s.createErr
	}
	f, err := s.osProfileStorage.Create(dir, name)
	if err != nil {
		return nil, err
	}
	if name == s.name {
		return profileFaultFile{f, s.writeErr, s.closeErr}, nil
	}
	return f, nil
}

func TestProfileWindowRetainsOutputFailures(t *testing.T) {
	cause := errors.New("held output failed")
	for _, fault := range []string{"create", "cpu write", "cpu close", "snapshot write", "metadata write"} {
		t.Run(fault, func(t *testing.T) {
			storage := profileFaultStorage{name: "cpu.pprof"}
			switch fault {
			case "create":
				storage.createErr = cause
			case "cpu write":
				storage.writeErr = cause
			case "cpu close":
				storage.closeErr = cause
			case "snapshot write":
				storage.name = "allocs-before.pprof"
				storage.writeErr = cause
			case "metadata write":
				storage.name = "metadata.json"
				storage.writeErr = cause
			}
			clock := newProfileTestClock()
			backend := &profileTestBackend{}
			window, err := StartProfileWindow(context.Background(), ProfileRequest{Dir: filepath.Join(t.TempDir(), "profile")}, profileDependencies{clock: clock, backend: backend, storage: storage}, nil)
			if fault == "create" {
				if !errors.Is(err, cause) {
					t.Fatalf("create=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			awaitProfileWait(t, clock, time.Minute)
			clock.advance <- struct{}{}
			if fault != "snapshot write" {
				awaitProfileWait(t, clock, 30*time.Second)
				clock.advance <- struct{}{}
			}
			<-window.done
			if err := window.Close(); !errors.Is(err, cause) {
				t.Fatalf("lost %s: %v", fault, err)
			}
		})
	}
}

func TestProfileWriterBoundsAndFirstFailure(t *testing.T) {
	var dst bytes.Buffer
	writer := &profileWriter{file: nopProfileCloser{&dst}, remaining: 3}
	n, err := writer.Write([]byte("1234"))
	if n != 3 || err == nil || dst.String() != "123" {
		t.Fatalf("write=%d,%v %q", n, err, dst.String())
	}
	first := err
	n, err = writer.Write([]byte("more"))
	if n != 0 || err != first || dst.String() != "123" {
		t.Fatal("write escaped bound or lost cause")
	}
	if err := writer.Close(); err != first {
		t.Fatalf("Close lost first failure: %v", err)
	}
}

type nopProfileCloser struct{ io.Writer }

func (nopProfileCloser) Close() error { return nil }

func TestProfileOptInDisabled(t *testing.T) {
	window, err := StartProfileWindow(context.Background(), ProfileRequest{}, profileDependencies{}, nil)
	if err != nil || window != nil {
		t.Fatalf("disabled=%v %v", window, err)
	}
	rt := newRT(t)
	dir := filepath.Join(t.TempDir(), "profile")
	rt.env["COUCH_PROFILE_DIR"] = dir
	for _, args := range [][]string{{"--help"}, {"--skill"}, {}} {
		var out, stderr bytes.Buffer
		RunWithRuntime(args, strings.NewReader(""), &out, &stderr, rt)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("non-launch or terminal-invalid command created profiles: %v", err)
	}
}

func TestProfileLaunchJoinsAndPreservesExit(t *testing.T) {
	for _, raw := range []string{"", "relative", filepath.Join(t.TempDir(), "profile")} {
		t.Run(raw, func(t *testing.T) {
			var notices []string
			calls := 0
			code, profileErr := runProfiledConsole(func(key string) string {
				if key != "COUCH_PROFILE_DIR" {
					t.Fatalf("unexpected env read %s", key)
				}
				return raw
			}, func(s string) { notices = append(notices, s) }, func() int { calls++; return 17 })
			if profileErr != nil {
				t.Fatal(profileErr)
			}
			if code != 17 || calls != 1 {
				t.Fatalf("console exit=%d calls=%d", code, calls)
			}
			if filepath.IsAbs(raw) {
				data, err := os.ReadFile(filepath.Join(raw, "metadata.json"))
				if err != nil || !bytes.Contains(data, []byte(`"status":"cancelled"`)) {
					t.Fatalf("joined launch=%s %v", data, err)
				}
			}
			if raw == "relative" && (len(notices) != 1 || !strings.Contains(notices[0], "requires")) {
				t.Fatalf("missing diagnostic: %v", notices)
			}
		})
	}
}

// Only the clock is controlled here: the actual process-wide profiler and
// allocation snapshots must produce files understood by the real pprof reader.
type profileConformanceClock struct {
	warm bool
	work uint64
}

func (c *profileConformanceClock) Wait(ctx context.Context, _ time.Duration) bool {
	if !c.warm {
		c.warm = true
		return true
	}
	until := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(until) {
		for range 1000 {
			c.work = c.work*1664525 + 1013904223
		}
		if ctx.Err() != nil {
			return false
		}
	}
	return true
}
func TestProfileRuntimeConformance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profiles")
	window, err := StartProfileWindow(context.Background(), ProfileRequest{Dir: dir}, profileDependencies{clock: &profileConformanceClock{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	select {
	case <-window.done:
	case <-testwait.After(t):
		t.Fatal("runtime profile did not finish")
	}
	if err := window.Close(); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cpu.pprof", "allocs-before.pprof", "allocs-after.pprof"} {
		ctx, cancel := context.WithTimeout(context.Background(), testwait.Budget(t))
		output, err := exec.CommandContext(ctx, "go", "tool", "pprof", "-top", executable, filepath.Join(dir, name)).CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("pprof %s: %v: %s", name, err, output)
		}
		if !bytes.Contains(output, []byte("Type:")) {
			t.Fatalf("profile reader omitted sample type: %s", output)
		}
	}
}
