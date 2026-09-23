package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type recoveryIOFunc func(context.Context, ProvisionCommand) ([]byte, error)

func (f recoveryIOFunc) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	return f(ctx, c)
}

type recoveryStorage struct {
	ProvisionStorage
	write  func(string, any) error
	remove func(string) error
}

func (s recoveryStorage) Write(path string, value any) error {
	if s.write != nil {
		return s.write(path, value)
	}
	return s.ProvisionStorage.Write(path, value)
}
func (s recoveryStorage) Remove(path string) error {
	if s.remove != nil {
		return s.remove(path)
	}
	return s.ProvisionStorage.Remove(path)
}
func recoverySuccessPath(f *ProvisionFixture) string {
	return filepath.Join(f.git(f.host(1), "rev-parse", "--absolute-git-dir"), "couch-setup-success.json")
}

func TestProvisionRecoveryLostSuccessPublication(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(fmt.Sprintf("published=%t", published), func(t *testing.T) {
			f := newProvisionFixture(t)
			p := NewWorkspaceProvisioner(f)
			req := ProvisionRequest{Path: f.Primary, Slot: 1}
			failed := false
			p.Store = recoveryStorage{ProvisionStorage: ProvisionStore{}, write: func(path string, value any) error {
				if strings.HasSuffix(path, "couch-setup-success.json") && !failed {
					failed = true
					if published {
						if err := (ProvisionStore{}).Write(path, value); err != nil {
							return err
						}
					}
					return errors.New("lost marker publication acknowledgment")
				}
				return (ProvisionStore{}).Write(path, value)
			}}
			if _, err := p.Ensure(context.Background(), req); err == nil {
				t.Fatal("lost publication accepted")
			}
			if !failed || f.WeaveCalls != 1 {
				t.Fatalf("injection=%t weave=%d", failed, f.WeaveCalls)
			}
			f.git(f.Primary, "commit", "--allow-empty", "-m", "advance remote")
			f.git(f.Primary, "push", "upstream", "main")
			r, err := p.Ensure(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			calls := 2
			if published {
				calls = 1
			}
			if f.WeaveCalls != calls || r.BaselineSHA != f.Base {
				t.Fatalf("calls=%d result=%+v", f.WeaveCalls, r)
			}
			if _, err := os.Stat(recoverySuccessPath(f)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProvisionRecoveryCanceledAfterSetup(t *testing.T) {
	f := newProvisionFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewWorkspaceProvisioner(recoveryIOFunc(func(ctx context.Context, c ProvisionCommand) ([]byte, error) {
		out, err := f.Run(ctx, c)
		if c.Program == "weave" && err == nil {
			cancel()
		}
		return out, err
	}))
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	if _, err := p.Ensure(ctx, req); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if _, err := os.Stat(recoverySuccessPath(f)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled setup published marker: %v", err)
	}
	p.IO = f
	if _, err := p.Ensure(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if f.WeaveCalls != 2 {
		t.Fatalf("unconfirmed setup was not repeated: %d", f.WeaveCalls)
	}
}

func TestProvisionRecoveryExternalHostPreservesFeature(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "branch", "main-slot1", f.Base)
	f.git(f.Primary, "worktree", "add", f.host(1), "main-slot1")
	f.git(f.host(1), "switch", "-c", "external-feature")
	f.git(f.host(1), "commit", "--allow-empty", "-m", "local feature")
	head := f.git(f.host(1), "rev-parse", "HEAD")
	dirty := filepath.Join(f.host(1), "keep.txt")
	if err := os.WriteFile(dirty, []byte("local work"), 0600); err != nil {
		t.Fatal(err)
	}
	p := NewWorkspaceProvisioner(f)
	r, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Disposition != "prepared" || r.BaselineSHA != f.Base || f.WeaveCalls != 1 {
		t.Fatalf("result=%+v calls=%d", r, f.WeaveCalls)
	}
	if f.git(f.host(1), "rev-parse", "HEAD") != head || f.git(f.host(1), "branch", "--show-current") != "external-feature" {
		t.Fatal("external feature changed")
	}
	if data, err := os.ReadFile(dirty); err != nil || string(data) != "local work" {
		t.Fatalf("dirty contents=%q err=%v", data, err)
	}
}

func TestProvisionRecoveryInterruptedUpstreamConfiguration(t *testing.T) {
	f := newProvisionFixture(t)
	interrupted := false
	f.AfterGit = func(c ProvisionCommand, _ []byte) error {
		if !interrupted && len(c.Args) == 4 && c.Args[0] == "config" && c.Args[1] == "--add" && c.Args[2] == "branch.main-slot1.remote" {
			interrupted = true
			return errors.New("interrupted after remote key")
		}
		return nil
	}
	p := NewWorkspaceProvisioner(f)
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	if _, err := p.Ensure(context.Background(), req); err == nil || !interrupted {
		t.Fatalf("injection=%t error=%v", interrupted, err)
	}
	f.git(f.Primary, "commit", "--allow-empty", "-m", "advance upstream")
	f.git(f.Primary, "push", "upstream", "main")
	r, err := p.Ensure(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if r.BaselineSHA != f.Base {
		t.Fatal("retry refreshed baseline")
	}
	if got := f.git(f.Primary, "config", "--get-all", "branch.main-slot1.remote"); got != "upstream" {
		t.Fatalf("remote duplicated or changed: %q", got)
	}
	if got := f.git(f.Primary, "config", "--get-all", "branch.main-slot1.merge"); got != "refs/heads/main" {
		t.Fatalf("merge not recovered: %q", got)
	}
	if got := f.git(f.host(1), "rev-parse", "--abbrev-ref", "@{upstream}"); got != "upstream/main" {
		t.Fatal(got)
	}
}

func awaitRecovery(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("recovery barrier timed out")
		return nil
	}
}
func awaitRecoveryBarrier(t *testing.T, ready <-chan struct{}) {
	t.Helper()
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("recovery barrier timed out")
	}
}

func TestProvisionRecoveryConcurrentWeaveBusy(t *testing.T) {
	f := newProvisionFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var mu sync.Mutex
	calls := 0
	busy := errors.New("weave setup already running")
	commandIO := recoveryIOFunc(func(ctx context.Context, c ProvisionCommand) ([]byte, error) {
		if c.Program == "weave" {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n == 1 {
				close(entered)
				select {
				case <-release:
					return nil, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, busy
		}
		mu.Lock()
		defer mu.Unlock()
		return f.Run(ctx, c)
	})
	p := NewWorkspaceProvisioner(commandIO)
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	done := make(chan error, 1)
	go func() { _, err := p.Ensure(context.Background(), req); done <- err }()
	awaitRecoveryBarrier(t, entered)
	if _, err := p.Ensure(context.Background(), req); !errors.Is(err, busy) {
		t.Fatalf("second caller error=%v", err)
	}
	if _, err := os.Stat(recoverySuccessPath(f)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("busy setup published success: %v", err)
	}
	releaseOnce.Do(func() { close(release) })
	if err := awaitRecovery(t, done); err != nil {
		t.Fatal(err)
	}
	r, err := p.Ensure(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	observed := calls
	mu.Unlock()
	if observed != 2 || r.Disposition != "reused" {
		t.Fatalf("calls=%d result=%+v", observed, r)
	}
	if got := f.git(f.Primary, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 2 {
		t.Fatalf("duplicate host: %s", got)
	}
}

func TestProvisionRecoveryMarkerPublicationExcludesCleanup(t *testing.T) {
	f := newProvisionFixture(t)
	f.FailWeave = true
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	p := NewWorkspaceProvisioner(f)
	if _, err := p.Ensure(context.Background(), req); err == nil {
		t.Fatal("expected setup failure")
	}
	f.FailWeave = false
	marker := recoverySuccessPath(f)
	tmp := filepath.Join(filepath.Dir(marker), ".couch-provision-live-writer")
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	common := filepath.Join(f.Primary, ".git")
	checkLocked := func() error {
		lease, err := AcquireHostCreationLease(common)
		if lease != nil {
			lease.Close()
		}
		if !errors.Is(err, ErrHostCreationBusy) {
			return fmt.Errorf("storage mutation without host lease: %v", err)
		}
		return nil
	}
	p.Store = recoveryStorage{ProvisionStorage: ProvisionStore{}, write: func(path string, value any) error {
		if err := checkLocked(); err != nil {
			return err
		}
		if path == marker {
			if err := os.WriteFile(tmp, []byte("live publication"), 0600); err != nil {
				return err
			}
			close(entered)
			<-release
		}
		return (ProvisionStore{}).Write(path, value)
	}, remove: func(path string) error {
		if err := checkLocked(); err != nil {
			return err
		}
		return (ProvisionStore{}).Remove(path)
	}}
	done := make(chan error, 1)
	go func() { _, err := p.Ensure(context.Background(), req); done <- err }()
	awaitRecoveryBarrier(t, entered)
	if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), req); !errors.Is(err, ErrHostCreationBusy) {
		t.Fatalf("publication did not exclude competing caller: %v", err)
	}
	if raw, err := os.ReadFile(tmp); err != nil || string(raw) != "live publication" {
		t.Fatalf("competing cleanup touched writer: %q %v", raw, err)
	}
	releaseOnce.Do(func() { close(release) })
	if err := awaitRecovery(t, done); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tmp); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed writer residue: %v", err)
	}
	if _, err := NewWorkspaceProvisioner(f).Ensure(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}

func TestProvisionRecoveryConcurrentPublishedMarkerWins(t *testing.T) {
	f := newProvisionFixture(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var mu sync.Mutex
	calls, writes := 0, 0
	commandIO := recoveryIOFunc(func(ctx context.Context, c ProvisionCommand) ([]byte, error) {
		if c.Program == "weave" {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n == 1 {
				close(entered)
				select {
				case <-release:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, nil
		}
		mu.Lock()
		defer mu.Unlock()
		return f.Run(ctx, c)
	})
	p := NewWorkspaceProvisioner(commandIO)
	p.Store = recoveryStorage{ProvisionStorage: ProvisionStore{}, write: func(path string, value any) error {
		if strings.HasSuffix(path, "couch-setup-success.json") {
			mu.Lock()
			writes++
			mu.Unlock()
		}
		return (ProvisionStore{}).Write(path, value)
	}}
	req := ProvisionRequest{Path: f.Primary, Slot: 1}
	done := make(chan error, 1)
	go func() { _, err := p.Ensure(context.Background(), req); done <- err }()
	awaitRecoveryBarrier(t, entered)
	second, err := p.Ensure(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if second.BaselineSHA != f.Base {
		t.Fatal(second)
	}
	marker := recoverySuccessPath(f)
	before, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	releaseOnce.Do(func() { close(release) })
	if err := awaitRecovery(t, done); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	observedCalls, observedWrites := calls, writes
	mu.Unlock()
	if observedCalls != 2 || observedWrites != 1 || string(before) != string(after) {
		t.Fatalf("calls=%d marker writes=%d marker changed=%t", observedCalls, observedWrites, string(before) != string(after))
	}
}
