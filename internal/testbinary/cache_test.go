package testbinary

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCacheBuildsOnceAndOwnsArtifactBeyondIndividualTests(t *testing.T) {
	var cache Cache
	t.Cleanup(func() {
		if err := cache.Close(); err != nil {
			t.Error(err)
		}
	})
	var builds atomic.Int32
	build := func(dir string) error {
		builds.Add(1)
		return os.WriteFile(filepath.Join(dir, "binary"), []byte("fixture artifact"), 0700)
	}
	paths := make([]string, 8)
	t.Run("concurrent consumers", func(t *testing.T) {
		var workers sync.WaitGroup
		for i := range paths {
			workers.Go(func() {
				dir, err := cache.Directory(build)
				if err != nil {
					t.Error(err)
					return
				}
				paths[i] = dir
			})
		}
		workers.Wait()
	})
	if builds.Load() != 1 {
		t.Fatalf("builds=%d, want1", builds.Load())
	}
	for _, path := range paths {
		if path != paths[0] {
			t.Fatalf("artifacts differ: %q", paths)
		}
		if _, err := os.Stat(filepath.Join(path, "binary")); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := cache.Directory(build)
	if err != nil || dir != paths[0] || builds.Load() != 1 {
		t.Fatalf("lookup after consumer cleanup: %q %v builds=%d", dir, err, builds.Load())
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("suite cleanup left artifact: %v", err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Directory(build); err == nil || builds.Load() != 1 {
		t.Fatalf("closed cache rebuilt: %v builds=%d", err, builds.Load())
	}
}

func TestCacheFailureIsRetainedUntilSuiteCleanup(t *testing.T) {
	var cache Cache
	defer cache.Close()
	failure := errors.New("compiler failed")
	builds := 0
	var artifact string
	for range 2 {
		_, err := cache.Directory(func(dir string) error {
			builds++
			artifact = dir
			if err := os.WriteFile(filepath.Join(dir, "partial"), nil, 0600); err != nil {
				return err
			}
			return failure
		})
		if !errors.Is(err, failure) {
			t.Fatalf("build error=%v", err)
		}
	}
	if builds != 1 {
		t.Fatalf("failed build retried %d times", builds)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("failed artifact retained: %v", err)
	}
}
