package sessionsync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// memoryOperatorFiles models visibility separately from confirmed directory
// durability. A failed effect may already be visible; crash restores durability.
type memoryOperatorFiles struct {
	visible, durable       map[string]string
	failPublish, failSync  bool
	failRemoveAt, removals int
	syncs                  int
}

func newMemoryOperatorFiles() *memoryOperatorFiles {
	return &memoryOperatorFiles{visible: map[string]string{}, durable: map[string]string{}}
}
func (f *memoryOperatorFiles) Read(path string) ([]byte, error) {
	if raw, ok := f.visible[path]; ok {
		return []byte(raw), nil
	}
	return nil, os.ErrNotExist
}
func (f *memoryOperatorFiles) Publish(path string, raw []byte) error {
	f.visible[path] = string(raw)
	if f.failPublish {
		f.failPublish = false
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (f *memoryOperatorFiles) Remove(path string) error {
	delete(f.visible, path)
	f.removals++
	if f.failRemoveAt == f.removals {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (f *memoryOperatorFiles) SyncDir(dir string) error {
	f.syncs++
	if f.failSync {
		f.failSync = false
		return io.ErrUnexpectedEOF
	}
	for path := range f.durable {
		if filepath.Dir(path) == dir {
			delete(f.durable, path)
		}
	}
	for path, raw := range f.visible {
		if filepath.Dir(path) == dir {
			f.durable[path] = raw
		}
	}
	return nil
}
func (f *memoryOperatorFiles) crash() {
	f.visible = map[string]string{}
	for path, raw := range f.durable {
		f.visible[path] = raw
	}
}

func TestOperatorLockUncertainPublicationRecoversExactTurn(t *testing.T) {
	r, req := operatorRig(t)
	files := newMemoryOperatorFiles()
	files.failPublish = true
	r.sync.Registry = &registryStore{Dir: r.sync.registryStore().Dir, OperatorLocks: files}
	_, err := r.sync.AcquireOperator(req)
	wantRefusal(t, err, 1, "operator-lock-uncertain")
	text := files.visible[req.File+".lock"]
	if text == "" || len(files.durable) != 0 {
		t.Fatal("fake did not model uncertain publication")
	}
	result, err := r.sync.AcquireOperator(req)
	if err != nil || result.State != OperatorAlreadyHeld || result.LockText != text {
		t.Fatalf("retry %+v %v", result, err)
	}
	files.crash()
	if files.visible[result.LockPath] != text {
		t.Fatal("ack preceded confirmed protection durability")
	}
}

func TestOperatorLockAbsentReleaseRetryConfirmsDeletion(t *testing.T) {
	r, req := operatorRig(t)
	files := newMemoryOperatorFiles()
	r.sync.Registry = &registryStore{Dir: r.sync.registryStore().Dir, OperatorLocks: files}
	owned, err := r.sync.AcquireOperator(req)
	if err != nil {
		t.Fatal(err)
	}
	files.failSync = true
	_, err = r.sync.ReleaseOperator(req)
	wantRefusal(t, err, 1, "operator-lock-uncertain")
	if _, exists := files.visible[owned.LockPath]; exists {
		t.Fatal("unlink did not become visible")
	}
	if _, exists := files.durable[owned.LockPath]; !exists {
		t.Fatal("fake incorrectly confirmed failed fsync")
	}
	before := files.syncs
	result, err := r.sync.ReleaseOperator(req)
	if err != nil || result.State != OperatorAlreadyReleased || files.syncs <= before {
		t.Fatalf("absent retry %+v %v syncs=%d", result, err, files.syncs)
	}
	files.crash()
	if _, exists := files.visible[owned.LockPath]; exists {
		t.Fatal("acknowledged removal resurrected on crash")
	}
}

func TestOperatorLockPartialAliasReleaseReconciles(t *testing.T) {
	r, req := operatorRig(t)
	dir := filepath.Join(filepath.Dir(req.File), "alias-dir")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias.md")
	if err := os.Symlink(req.File, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Register(alias, "ops:0"); err != nil {
		t.Fatal(err)
	}
	files := newMemoryOperatorFiles()
	r.sync.Registry = &registryStore{Dir: r.sync.registryStore().Dir, OperatorLocks: files}
	owned, err := r.sync.AcquireOperator(req)
	if err != nil {
		t.Fatal(err)
	}
	files.visible[alias+".lock"] = owned.LockText
	if err := files.SyncDir(dir); err != nil {
		t.Fatal(err)
	}
	files.failRemoveAt = 1
	_, err = r.sync.ReleaseOperator(req)
	wantRefusal(t, err, 1, "operator-lock-uncertain")
	if len(files.visible) != 1 || len(files.durable) != 2 {
		t.Fatalf("partial effect lost: visible=%v durable=%v", files.visible, files.durable)
	}
	if _, err := r.sync.ReleaseOperator(req); err != nil {
		t.Fatal(err)
	}
	files.crash()
	if len(files.visible) != 0 {
		t.Fatalf("partial removal resurrected: %v", files.visible)
	}
}

type countedOperatorFiles struct {
	operatorLockFiles
	reads                  map[string]int
	publications, removals int
}

func (f *countedOperatorFiles) Read(path string) ([]byte, error) {
	if strings.HasSuffix(path, ".lock") {
		f.reads[path]++
	}
	return f.operatorLockFiles.Read(path)
}
func (f *countedOperatorFiles) Publish(path string, raw []byte) error {
	f.publications++
	return f.operatorLockFiles.Publish(path, raw)
}
func (f *countedOperatorFiles) Remove(path string) error {
	if strings.HasSuffix(path, ".lock") {
		f.removals++
	}
	return f.operatorLockFiles.Remove(path)
}

func TestOperatorLockAliasOperationCounts(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			r, req := operatorRig(t)
			var aliases []string
			for i := 0; i < n; i++ {
				alias := filepath.Join(filepath.Dir(req.File), fmt.Sprintf("alias-%d.md", i))
				if err := os.Symlink(req.File, alias); err != nil {
					t.Fatal(err)
				}
				if _, err := r.sync.Register(alias, "ops:0"); err != nil {
					t.Fatal(err)
				}
				aliases = append(aliases, alias)
			}
			files := &countedOperatorFiles{operatorLockFiles: osOperatorLockFiles{}, reads: map[string]int{}}
			transactions := 0
			r.sync.Registry = &registryStore{Dir: r.sync.registryStore().Dir, OperatorLocks: files, TryLock: func(fd int) error { transactions++; return syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB) }}
			owned, err := r.sync.AcquireOperator(req)
			if err != nil {
				t.Fatal(err)
			}
			checkReads := func() {
				t.Helper()
				if len(files.reads) != n+1 {
					t.Fatalf("read %d paths for %d aliases", len(files.reads), n)
				}
				for path, count := range files.reads {
					if count != 1 {
						t.Fatalf("rescanned %s %d times", path, count)
					}
				}
			}
			checkReads()
			if transactions != 1 || files.publications != 1 {
				t.Fatalf("transactions=%d publications=%d", transactions, files.publications)
			}
			for _, alias := range aliases {
				if err := os.WriteFile(alias+".lock", []byte(owned.LockText), 0600); err != nil {
					t.Fatal(err)
				}
			}
			files.reads = map[string]int{}
			if _, err := r.sync.ReleaseOperator(req); err != nil {
				t.Fatal(err)
			}
			checkReads()
			if transactions != 2 || files.removals != n+1 {
				t.Fatalf("transactions=%d removals=%d aliases=%d", transactions, files.removals, n)
			}
		})
	}
}

func TestOperatorLockRefusesUnsafePathsAndReclaimsStage(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "fifo", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			r, req := operatorRig(t)
			lock := req.File + ".lock"
			var err error
			switch kind {
			case "symlink":
				err = os.Symlink(req.File, lock)
			case "directory":
				err = os.Mkdir(lock, 0700)
			case "fifo":
				err = syscall.Mkfifo(lock, 0600)
			case "oversize":
				err = os.WriteFile(lock, []byte(strings.Repeat("x", 4097)), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := r.read(req.File)
			_, err = r.sync.AcquireOperator(req)
			wantRefusal(t, err, 1, "operator-path")
			if r.read(req.File) != before {
				t.Fatal("unsafe lock path changed document")
			}
		})
	}
	r, req := operatorRig(t)
	stage := operatorStagePath(req.File + ".lock")
	if err := os.WriteFile(stage, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.AcquireOperator(req); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stage); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphan stage remained: %v", err)
	}
}

func TestOperatorLockAvailabilityIsNotAuthority(t *testing.T) {
	r, req := operatorRig(t)
	root := r.sync.registryStore().Dir
	r.sync.Registry = &registryStore{Dir: root, TryLock: func(int) error { return syscall.EIO }}
	_, err := r.sync.AcquireOperator(req)
	wantRefusal(t, err, ExitUnavailable, "session-sync-unavailable")
	r.sync.Registry = &registryStore{Dir: root}
	if err := os.WriteFile(filepath.Join(root, "registry.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = r.sync.AcquireOperator(req)
	wantRefusal(t, err, 1, "registry-invalid")
}

func TestOperatorLockInvalidStorePathIsNotUnavailable(t *testing.T) {
	r, req := operatorRig(t)
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("authority corruption"), 0600); err != nil {
		t.Fatal(err)
	}
	r.sync.Registry = &registryStore{Dir: filepath.Join(blocked, "documents")}
	_, err := r.sync.AcquireOperator(req)
	wantRefusal(t, err, 1, "registry-invalid")
}
