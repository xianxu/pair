package testcouch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type ownerReporter struct{ logs []string }

func (r *ownerReporter) Logf(format string, args ...any) { r.logs = append(r.logs, format) }
func (*ownerReporter) Failed() bool                      { return false }

func ownerShortScratch(t *testing.T) string {
	t.Helper()
	base := "/tmp"
	if info, err := os.Stat("/Volumes/TestScratch"); err == nil && info.IsDir() {
		base = "/Volumes/TestScratch"
	}
	physical, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	return physical
}

func ownerParent(t *testing.T) string {
	t.Helper()
	// Short socket placement is part of the native contract, not a test bypass.
	base := os.TempDir()
	if info, err := os.Stat("/Volumes/TestScratch"); err == nil && info.IsDir() {
		base = "/Volumes/TestScratch"
	}
	physical, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(physical, "own-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(dir); err != nil {
			t.Error(err)
		}
	})
	return dir
}

func closeOwned(t *testing.T, f *Fleet) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := f.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.AssertCleaned(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.Roots.Root); !os.IsNotExist(err) {
		t.Fatalf("owned root survived cleanup: %v", err)
	}
}

func TestOwnedConstructorRetainsReporterAndCallerLifetime(t *testing.T) {
	parent := ownerShortScratch(t)
	b, _, _ := runtimeFilesFixture(t)
	b.Source = runtimeSourceForTest(t)
	reporter := &ownerReporter{}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f, err := NewOwned(ctx, b, Options{Workers: 1}, OwnerConfig{ScratchParent: parent, Reporter: reporter})
	if err != nil {
		if f != nil {
			closeOwned(t, f)
		}
		t.Fatal(err)
	}
	defer closeOwned(t, f)
	if f.Options.AgentKind != "claude" || f.Options.RuntimeBudget != 120*time.Second {
		t.Fatalf("lost default options: %+v", f.Options)
	}
	if filepath.Dir(f.Roots.Root) != parent || f.RunID == "" || len(f.Environment) == 0 {
		t.Fatalf("incomplete isolated owner: %+v", f.Roots)
	}
	child, done, err := f.begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-child.Done():
	case <-time.After(time.Second):
		t.Fatal("caller lifetime did not cancel admitted work")
	}
	done()
	_, laterDone, laterErr := f.begin(context.Background())
	if laterDone != nil {
		laterDone()
	}
	if !errors.Is(laterErr, context.Canceled) {
		t.Fatalf("expired owner lifetime admitted new work: %v", laterErr)
	}
	if len(reporter.logs) == 0 {
		t.Fatal("minimal reporter received no diagnostics")
	}
}

func TestOwnedConstructorRefusesInvalidInputsBeforeAllocation(t *testing.T) {
	parent := ownerParent(t)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(parent, alias); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"relative", "missing", "alias", "denied", "ambient-live", "invalid-options", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			cfg := OwnerConfig{ScratchParent: parent}
			opts := Options{Workers: 1}
			ctx := context.Background()
			switch scenario {
			case "relative":
				cfg.ScratchParent = "relative"
			case "missing":
				cfg.ScratchParent = filepath.Join(parent, "missing")
			case "alias":
				cfg.ScratchParent = alias
			case "denied":
				cfg.Denied = []string{parent}
			case "ambient-live":
				t.Setenv("COUCH_STORE_DIR", parent)
			case "invalid-options":
				opts.Workers = 0
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			f, err := NewOwned(ctx, Binaries{}, opts, cfg)
			if err == nil || f != nil {
				t.Fatalf("invalid constructor retained an owner: %v %v", f, err)
			}
			entries, readErr := os.ReadDir(parent)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("refusal created effects: %v %v", entries, readErr)
			}
		})
	}
}

func TestOwnedConstructorFailureReturnsCleanupOwner(t *testing.T) {
	parent := ownerShortScratch(t)
	f, err := NewOwned(t.Context(), Binaries{}, Options{Workers: 1}, OwnerConfig{ScratchParent: parent})
	if err == nil || f == nil {
		t.Fatalf("runtime failure hid partial ownership: f=%v err=%v", f, err)
	}
	if f.RunID == "" || f.Roots.Root == "" {
		t.Fatalf("partial owner lost cleanup identity: %+v", f)
	}
	closeOwned(t, f)
}

func TestOwnedConstructorSocketOverflowDoesNotSilentlyRelocate(t *testing.T) {
	parent := filepath.Join(ownerParent(t), strings.Repeat("long", 20))
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(parent); err != nil {
			t.Error(err)
		}
	})
	f, err := NewOwned(t.Context(), Binaries{}, Options{Workers: 1}, OwnerConfig{ScratchParent: parent})
	if err == nil || !strings.Contains(err.Error(), "socket") {
		t.Fatalf("overflow not refused: %v", err)
	}
	if f != nil {
		if filepath.Dir(f.Roots.Root) != parent {
			t.Fatal("explicit scratch silently relocated")
		}
		closeOwned(t, f)
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed placement leaked root: %v %v", entries, err)
	}
}

func TestOwnedConstructorRejectsNilLifetime(t *testing.T) {
	f, err := NewOwned(nil, Binaries{}, Options{Workers: 1}, OwnerConfig{ScratchParent: ownerParent(t)})
	if f != nil || err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("nil lifetime accepted: %v %v", f, err)
	}
}

func TestOwnedDoneExposesOnlyCurrentConsole(t *testing.T) {
	f := &Fleet{}
	if f.Done() != nil {
		t.Fatal("absent console advertised completion")
	}
	done := make(chan struct{})
	f.console = &terminalProcess{done: done}
	if f.Done() != done {
		t.Fatal("completion does not follow owned console")
	}
	close(done)
	select {
	case <-f.Done():
	default:
		t.Fatal("owned completion not observable")
	}
	f.console = nil
	if f.Done() != nil {
		t.Fatal("retired console remained observable")
	}
}

func TestOwnedConstructorAllowsOnlyExplicitSocketFallback(t *testing.T) {
	parent := filepath.Join(ownerParent(t), strings.Repeat("long", 20))
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(parent); err != nil {
			t.Error(err)
		}
	})
	f, err := NewOwned(t.Context(), Binaries{}, Options{Workers: 1}, OwnerConfig{ScratchParent: parent, AllowScratchFallback: true})
	// Runtime validation still fails, but ownership must identify the selected
	// fallback and the original unpublished lease must have been removed.
	if err == nil || f == nil || strings.Contains(err.Error(), "socket") {
		t.Fatalf("authorized fallback failed before runtime validation: %v %v", f, err)
	}
	physical, err := filepath.EvalSymlinks("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(f.Roots.Root) != physical {
		t.Fatalf("wrong fallback root: %s", f.Roots.Root)
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatalf("original lease survived fallback: %v %v", entries, err)
	}
	closeOwned(t, f)
}
