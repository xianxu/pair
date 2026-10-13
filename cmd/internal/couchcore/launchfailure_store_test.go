package couchcore

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
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
	for i := 1; i <= 17; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if e := NewLaunchFailureStore(ns).Save(failureFixture(n)); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	h, e := NewLaunchFailureStore(ns).Read()
	if e != nil {
		t.Fatal(e)
	}
	if len(h) != 16 || h[0].Nonce != "17" || h[15].Nonce != "2" {
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
func TestLaunchFailureStoreRejectsUnsafeAndCorrupt(t *testing.T) {
	for _, kind := range []string{"corrupt", "oversize", "symlink", "directory", "lock-symlink"} {
		t.Run(kind, func(t *testing.T) {
			s := NewLaunchFailureStore(failureNamespace(t))
			if e := s.Save(failureFixture(1)); e != nil {
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
			if e := s.Save(failureFixture(2)); e == nil {
				t.Fatal("unsafe/corrupt store replaced")
			}
			after, _ := os.ReadFile(s.Path())
			if !bytes.Equal(before, after) {
				t.Fatal("old evidence altered")
			}
		})
	}
}
func TestLaunchFailureStoreLockTimeout(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	if e := s.Save(failureFixture(1)); e != nil {
		t.Fatal(e)
	}
	lock, e := acquireThreadStoreLockMode(filepath.Dir(s.Path()), false)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	start := time.Now()
	if e = s.Save(failureFixture(2)); e == nil {
		t.Fatal("lock bypassed")
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond || elapsed > 2*time.Second {
		t.Fatalf("unbounded lock wait %v", elapsed)
	}
}

func TestLaunchFailureStoreInterruptedPublicationKeepsOldEvidence(t *testing.T) {
	s := NewLaunchFailureStore(failureNamespace(t))
	if err := s.Save(failureFixture(1)); err != nil {
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
	if err := reopened.Save(failureFixture(2)); err != nil {
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
	if err := s.Save(f); err != nil {
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
	if err := s.Save(failureFixture(2)); err != os.ErrPermission {
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
	if err := s.Save(failureFixture(1)); err != nil {
		t.Fatal(err)
	}
	for i := 2; i <= 17; i++ {
		before, err := os.ReadFile(s.Path())
		if err != nil {
			t.Fatal(err)
		}
		f := failureFixture(i)
		f.StartingPath = string(bytes.Repeat([]byte{0}, 4096))
		f.Agent = f.StartingPath
		f.Stage = f.StartingPath
		f.Stream = f.StartingPath
		f.Operation = f.StartingPath
		if err := s.Save(f); err != nil {
			after, e := os.ReadFile(s.Path())
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("rejected oversize history replaced previous publication")
			}
			return
		}
	}
	t.Fatal("encoded limit never enforced")
}

func TestLaunchFailureStoreRawBytesAndSeparateExitObservations(t *testing.T) {
	ns := failureNamespace(t)
	s := NewLaunchFailureStore(ns)
	f := failureFixture(1)
	f.Output = []byte{0xff, 0xfe, '\n', 0}
	f.Stream = "pty-combined"
	if err := s.Save(f); err != nil {
		t.Fatal(err)
	}
	code := 23
	f.AfterExit = &code
	f.FinalError = "source parked"
	if err := s.Save(f); err != nil {
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
