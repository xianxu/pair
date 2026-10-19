package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func failureNamespace(t *testing.T) CouchNamespace {
	t.Helper()
	n, e := ResolveCouchNamespace(t.TempDir(), "/")
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func TestLaunchFailureStoreReopenConcurrent(t *testing.T) {
	ns := failureNamespace(t)
	s := NewLaunchFailureStore(ns)
	if h, e := s.Read(); e != nil || len(h) != 0 {
		t.Fatalf("fresh: %v %v", h, e)
	}
	var wg sync.WaitGroup
	for i := 1; i <= 65; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if e := NewLaunchFailureStore(ns).Save(t.Context(), failureFixture(n)); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	h, e := NewLaunchFailureStore(ns).Read()
	if e != nil {
		t.Fatal(e)
	}
	if len(h) != 64 || h[0].Nonce != "65" || h[63].Nonce != "2" {
		t.Fatalf("retention %+v", h)
	}
	info, e := os.Stat(s.Path())
	if e != nil {
		t.Fatal(e)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %v", info.Mode())
	}
}

func TestLaunchFailureStoreWaitsBeyondFormerBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := NewLaunchFailureStore(failureNamespace(t))
		if err := s.Save(t.Context(), failureFixture(1)); err != nil {
			t.Fatal(err)
		}
		held, err := acquireLaunchFailureLock(filepath.Join(filepath.Dir(s.Path()), "store.lock"))
		if err != nil {
			t.Fatal(err)
		}
		defer held.Close()
		waits := 0
		s.waitForLock = func(context.Context) error {
			waits++
			if waits == 1 {
				// Only the virtual clock advances. The next real failed flock
				// occurs beyond the former deadline, before the owner releases.
				timer := time.NewTimer(2 * time.Second)
				<-timer.C
				return nil
			}
			return held.Close()
		}
		if err := s.Save(t.Context(), failureFixture(2)); err != nil {
			t.Fatal(err)
		}
		if waits != 2 {
			t.Fatalf("contended observations = %d, want 2", waits)
		}
		history, err := s.Read()
		if err != nil || len(history) != 2 || history[0].Nonce != "2" || history[1].Nonce != "1" {
			t.Fatalf("lost admitted history: %+v, %v", history, err)
		}
	})
}
func TestLaunchFailureStoreRejectsUnsafeAndCorrupt(t *testing.T) {
	for _, kind := range []string{"corrupt", "oversize", "symlink", "directory", "lock-symlink"} {
		t.Run(kind, func(t *testing.T) {
			s := NewLaunchFailureStore(failureNamespace(t))
			if e := s.Save(t.Context(), failureFixture(1)); e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "corrupt":
				os.WriteFile(s.Path(), []byte("bad"), 0600)
			case "oversize":
				os.WriteFile(s.Path(), bytes.Repeat([]byte(" "), (1<<20)+1), 0600)
			case "symlink":
				os.Remove(s.Path())
				os.Symlink(filepath.Join(t.TempDir(), "target"), s.Path())
			case "directory":
				os.Remove(s.Path())
				os.Mkdir(s.Path(), 0700)
			case "lock-symlink":
				os.Remove(filepath.Join(filepath.Dir(s.Path()), "store.lock"))
				os.Symlink(filepath.Join(t.TempDir(), "target"), filepath.Join(filepath.Dir(s.Path()), "store.lock"))
			}
			before, _ := os.ReadFile(s.Path())
			if e := s.Save(t.Context(), failureFixture(2)); e == nil {
				t.Fatal("unsafe/corrupt store replaced")
			}
			after, _ := os.ReadFile(s.Path())
			if !bytes.Equal(before, after) {
				t.Fatal("old evidence altered")
			}
		})
	}
}

// Admission follows context state, never an elapsed-time assertion.
func TestLaunchFailureStoreCanceledAdmissionPreservesEvidence(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			s := NewLaunchFailureStore(failureNamespace(t))
			if err := s.Save(t.Context(), failureFixture(1)); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(s.Path())
			if err != nil {
				t.Fatal(err)
			}
			var ctx context.Context
			var cancel context.CancelFunc
			want := error(context.Canceled)
			if expired {
				ctx, cancel = context.WithDeadline(t.Context(), time.Unix(0, 0))
				want = context.DeadlineExceeded
			} else {
				ctx, cancel = context.WithCancel(t.Context())
				cancel()
			}
			defer cancel()
			if err := s.Save(ctx, failureFixture(2)); !errors.Is(err, want) {
				t.Fatalf("canceled admission: %v, want %v", err, want)
			}
			after, err := os.ReadFile(s.Path())
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("refused admission changed evidence: %v", err)
			}
		})
	}
}

func TestLaunchFailureStoreContendedAdmission(t *testing.T) {
	for _, outcome := range []string{"release", "cancel", "deadline", "release-cancel"} {
		t.Run(outcome, func(t *testing.T) {
			cancelWaiter := outcome != "release"
			s := NewLaunchFailureStore(failureNamespace(t))
			if err := s.Save(t.Context(), failureFixture(1)); err != nil {
				t.Fatal(err)
			}
			held, err := acquireLaunchFailureLock(filepath.Join(filepath.Dir(s.Path()), "store.lock"))
			if err != nil {
				t.Fatal(err)
			}
			defer held.Close()
			ctx, cancel := context.WithCancel(t.Context())
			wantError := error(context.Canceled)
			if outcome == "deadline" {
				cancel()
				budget := newSpendableBudget()
				ctx, _ = budget.bind(t.Context())
				cancel = budget.spend
				wantError = context.DeadlineExceeded
			}
			defer cancel()
			contended := make(chan struct{})
			proceed := make(chan struct{})
			s.waitForLock = func(ctx context.Context) error {
				close(contended) // Only reached after a real flock reports contention.
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-proceed:
					return nil
				}
			}
			done := make(chan error, 1)
			go func() { done <- s.Save(ctx, failureFixture(2)) }()
			select {
			case <-contended:
			case <-testwait.After(t):
				t.Fatal("no real contention observed")
			}
			if cancelWaiter {
				if outcome == "release-cancel" {
					cancel()
					if err := held.Close(); err != nil {
						t.Fatal(err)
					}
					close(proceed)
				}
				cancel()
			} else {
				if err := held.Close(); err != nil {
					t.Fatal(err)
				}
				close(proceed)
			}
			select {
			case err := <-done:
				if cancelWaiter && !errors.Is(err, wantError) {
					t.Fatalf("held lease not refused: %v", err)
				}
				if !cancelWaiter && err != nil {
					t.Fatal(err)
				}
			case <-testwait.After(t):
				t.Fatal("admission did not finish")
			}
			if err := held.Close(); err != nil {
				t.Fatal(err)
			}
			// Return has joined admission: a canceled waiter cannot acquire or write late.
			next, err := acquireLaunchFailureLock(filepath.Join(filepath.Dir(s.Path()), "store.lock"))
			if err != nil {
				t.Fatalf("lease leaked: %v", err)
			}
			next.Close()
			history, err := s.Read()
			if err != nil {
				t.Fatal(err)
			}
			want := 2
			if cancelWaiter {
				want = 1
			}
			if len(history) != want || history[0].Nonce != fmt.Sprint(want) {
				t.Fatalf("history after admission: %+v", history)
			}
		})
	}
}

func TestLaunchFailureStoreInterruptedPublicationKeepsOldEvidence(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	if err := s.Save(t.Context(), failureFixture(1)); err != nil {
		t.Fatal(err)
	}
	// A process killed before rename leaves an incomplete private staging file.
	staged := filepath.Join(filepath.Dir(s.Path()), ".launch-failure-interrupted")
	if err := os.WriteFile(staged, []byte("partial json"), 0600); err != nil {
		t.Fatal(err)
	}
	reopened := &LaunchFailureStore{path: s.Path()}
	h, err := reopened.Read()
	if err != nil || len(h) != 1 || h[0].Nonce != "1" {
		t.Fatalf("old evidence lost: %v %v", h, err)
	}
	if err := reopened.Save(t.Context(), failureFixture(2)); err != nil {
		t.Fatal(err)
	}
	h, err = reopened.Read()
	if err != nil || len(h) != 2 {
		t.Fatalf("recovery failed: %v %v", h, err)
	}
}

func TestLaunchFailureFakePreservesStateAndInjectsFailure(t *testing.T) {
	s := NewFakeLaunchFailureStore()
	f := failureFixture(1)
	f.Output = []byte("stderr")
	if err := s.Save(t.Context(), f); err != nil {
		t.Fatal(err)
	}
	f.Output[0] = 'X'
	h, err := s.Read()
	if err != nil || string(h[0].Output) != "stderr" {
		t.Fatalf("aliased write %v %v", h, err)
	}
	h[0].Output[0] = 'Y'
	h, err = s.Read()
	if err != nil || string(h[0].Output) != "stderr" {
		t.Fatal("aliased read")
	}
	s.SaveError = os.ErrPermission
	if err := s.Save(t.Context(), failureFixture(2)); err != os.ErrPermission {
		t.Fatalf("write injection %v", err)
	}
	s.SaveError = nil
	h, err = s.Read()
	if err != nil || len(h) != 1 || len(s.Writes) != 2 {
		t.Fatalf("failed write changed evidence %v %v", h, err)
	}
	s.ReadError = os.ErrPermission
	if _, err := s.Read(); err != os.ErrPermission {
		t.Fatalf("read injection %v", err)
	}
}

func TestLaunchFailureStoreEncodedLimitPreservesPreviousPublication(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	if err := s.Save(t.Context(), failureFixture(1)); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	f := failureFixture(2)
	f.Unknown = map[string]json.RawMessage{"future": append(append([]byte{'"'}, bytes.Repeat([]byte{'x'}, launchFailureEncodedLimit)...), '"')}
	if err := s.Save(t.Context(), f); err == nil {
		t.Fatal("single oversized entry accepted")
	}
	after, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("rejected oversize entry changed publication")
	}
}

func TestLaunchFailureStoreRawBytesAndSeparateExitObservations(t *testing.T) {
	ns := failureNamespace(t)
	s := NewLaunchFailureStore(ns)
	f := failureFixture(1)
	f.Output = []byte{0xff, 0xfe, '\n', 0}
	f.Stream = "pty-combined"
	if err := s.Save(t.Context(), f); err != nil {
		t.Fatal(err)
	}
	code := 23
	f.AfterExit = &code
	f.FinalError = "source parked"
	if err := s.Save(t.Context(), f); err != nil {
		t.Fatal(err)
	}
	h, err := NewLaunchFailureStore(ns).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 || !bytes.Equal(h[0].Output, f.Output) || h[0].BeforeExit != nil || h[0].AfterExit == nil || *h[0].AfterExit != 23 || h[0].FinalError != "source parked" {
		t.Fatalf("lost restart evidence %+v", h)
	}
}
