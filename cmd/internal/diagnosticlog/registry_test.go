package diagnosticlog

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestRegistryTwoRootsOneCanonicalLog(t *testing.T) {
	path, _, opts := fixture(t)
	roots := []string{t.TempDir(), t.TempDir()}
	for _, root := range roots {
		o := EnvironmentOptions(func(k string) string {
			if k == "PAIR_DATA_DIR" {
				return root
			}
			return ""
		})
		o.Now = opts.Now
		o.Proof = opts.Proof
		w, e := Open(path, o)
		if e != nil {
			t.Fatal(e)
		}
		w.Write([]byte("line\n"))
		w.Close()
		entries, e := registryEntriesForTest(root)
		if e != nil || len(entries) != 1 {
			t.Fatalf("entries=%v err=%v", entries, e)
		}
		canonicalPath, _ := canonical(path)
		if entries[0].Path != canonicalPath {
			t.Fatal(entries)
		}
	}
	s, e := load(path)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Writers) != 2 {
		t.Fatalf("root contributors lost %+v", s.Writers)
	}
}
func TestPreviewDoesNotInitializeOrRecover(t *testing.T) {
	path, now, opts := fixture(t)
	before, _ := os.ReadDir(filepath.Dir(path))
	if _, e := Preview(path, opts, 100); e == nil {
		t.Fatal("untracked preview accepted")
	}
	after, _ := os.ReadDir(filepath.Dir(path))
	if len(before) != len(after) {
		t.Fatal("preview created files")
	}
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("content"))
	w.Close()
	*now = now.Add(RetentionPeriod + 24*time.Hour)
	state, _ := os.ReadFile(filepath.Join(directory(path), "state.json"))
	rows, e := Preview(path, opts, 100)
	if e != nil || len(rows) != 1 || rows[0].Eligible {
		t.Fatalf("preview %v %v", rows, e)
	}
	stateAfter, _ := os.ReadFile(filepath.Join(directory(path), "state.json"))
	if !reflect.DeepEqual(state, stateAfter) {
		t.Fatal("preview changed state")
	}
}
func TestRegistryRejectsPathSubstitution(t *testing.T) {
	path, _, opts := fixture(t)
	root := t.TempDir()
	opts.Registry = func(_ context.Context, e RegistryEntry) error { return register(context.Background(), root, e) }
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Close()
	files, _ := filepath.Glob(filepath.Join(RegistryDirectory(root), "*.json"))
	var entry RegistryEntry
	b, _ := os.ReadFile(files[0])
	json.Unmarshal(b, &entry)
	entry.Path = filepath.Join(root, "unrelated")
	b, _ = json.Marshal(entry)
	os.WriteFile(files[0], b, 0600)
	if _, e = registryEntriesForTest(root); e == nil {
		t.Fatal("substituted path accepted")
	}
}
func TestPolicyBoundaryProperties(t *testing.T) {
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	for _, age := range []time.Duration{-time.Hour, 0, RetentionPeriod - time.Nanosecond, RetentionPeriod, RetentionPeriod + time.Nanosecond} {
		got := DecideSegment(now, now.Add(-age))
		if got != (age >= RetentionPeriod) {
			t.Fatalf("age %v eligible %v", age, got)
		}
	}
	if DecideSegment(now, time.Time{}) {
		t.Fatal("unknown age eligible")
	}
}

func TestPausedOpenerRepublishesAfterRetirement(t *testing.T) {
	path, now, base := fixture(t)
	root := t.TempDir()
	opts := EnvironmentOptions(func(k string) string {
		if k == "PAIR_DATA_DIR" {
			return root
		}
		return ""
	})
	opts.Now = base.Now
	opts.Proof = base.Proof
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	// This fixture retires only an abandoned, empty current reservation.
	w.Close()
	s, e := load(path)
	if e != nil {
		t.Fatal(e)
	}
	s.Writers = nil
	s.Current = generation{}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if e = save(path, s, true, Options{}); e != nil {
		t.Fatal(e)
	}
	*now = now.Add(RetentionPeriod + 24*time.Hour)
	paused, e := openRegular(lockPath(path), syscall.O_RDWR)
	if e != nil {
		t.Fatal(e)
	}
	defer paused.Close()
	before, _ := paused.Stat()
	if _, e = Collect(path, opts, 100); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(directory(path)); !os.IsNotExist(e) {
		t.Fatal("state directory retained", e)
	}
	entries, e := registryEntriesForTest(root)
	if e != nil || len(entries) != 0 {
		t.Fatalf("registry=%v err=%v", entries, e)
	}
	if e = syscall.Flock(int(paused.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	if _, e = Open(path, opts); !errors.Is(e, ErrBusy) {
		t.Fatalf("opener ignored stable lock %v", e)
	}
	entries, e = registryEntriesForTest(root)
	if e != nil || len(entries) != 0 {
		t.Fatal("busy opener published before lock", entries, e)
	}
	syscall.Flock(int(paused.Fd()), syscall.LOCK_UN)
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	w.Write([]byte("new"))
	entries, e = registryEntriesForTest(root)
	if e != nil || len(entries) != 1 {
		t.Fatalf("opener did not republish %v %v", entries, e)
	}
	after, _ := os.Stat(lockPath(path))
	if !os.SameFile(before, after) {
		t.Fatal("stable coordination inode replaced")
	}
}

func TestRetirementCrashKeepsDiscoverableRecovery(t *testing.T) {
	for _, point := range []string{"retire-state", "retire-directory"} {
		t.Run(point, func(t *testing.T) {
			path, now, base := fixture(t)
			root := t.TempDir()
			opts := EnvironmentOptions(func(k string) string {
				if k == "PAIR_DATA_DIR" {
					return root
				}
				return ""
			})
			opts.Now = base.Now
			opts.Proof = base.Proof
			w, e := Open(path, opts)
			if e != nil {
				t.Fatal(e)
			}
			// This fixture retires only an abandoned, empty current reservation.
			w.Close()
			s, _ := load(path)
			s.Writers = nil
			s.Current = generation{}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			save(path, s, true, Options{})
			*now = now.Add(RetentionPeriod + 24*time.Hour)
			opts.Fault = func(step string) error {
				if step == point {
					return errors.New("crash")
				}
				return nil
			}
			if _, e = Collect(path, opts, 100); e == nil {
				t.Fatal("retirement fault ignored")
			}
			entries, e := registryEntriesForTest(root)
			if e != nil || len(entries) != 1 {
				t.Fatal("recovery lost discovery", entries, e)
			}
			opts.Fault = nil
			if _, e = Collect(path, opts, 100); e != nil {
				t.Fatal(e)
			}
			entries, e = registryEntriesForTest(root)
			if e != nil || len(entries) != 0 {
				t.Fatal("retirement not recovered", entries, e)
			}
		})
	}
}

func FuzzDecideSegmentAge(f *testing.F) {
	for _, delta := range []int64{0, -int64(RetentionPeriod), -int64(RetentionPeriod) + 1, -int64(RetentionPeriod) - 1, int64(time.Hour)} {
		f.Add(delta)
	}
	f.Fuzz(func(t *testing.T, delta int64) {
		now := time.Unix(0, 0).UTC()
		stamp := now.Add(time.Duration(delta))
		if got := DecideSegment(now, stamp); got != (delta <= -int64(RetentionPeriod)) {
			t.Fatalf("delta=%d eligible=%v", delta, got)
		}
	})
}

func registryEntriesForTest(root string) ([]RegistryEntry, error) {
	page, err := EnumerateRoot(context.Background(), root, 0, 100)
	return page.Entries, err
}

func TestRegistryRepeatedOpenPreservesDeclaration(t *testing.T) {
	path, _, opts := fixture(t)
	root := t.TempDir()
	opts.Registry = func(ctx context.Context, entry RegistryEntry) error { return register(ctx, root, entry) }
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	canonicalPath, err := canonical(path)
	if err != nil {
		t.Fatal(err)
	}
	entry := RegistryEntry{Version: 1, Path: canonicalPath, Directory: directory(canonicalPath), Lock: lockPath(canonicalPath)}
	sum := sha256.Sum256([]byte(entry.Path))
	declaration := filepath.Join(RegistryDirectory(root), fmt.Sprintf("%x.json", sum))
	before, err := os.Stat(declaration)
	if err != nil {
		t.Fatal(err)
	}
	staged := false
	if err := registerWithHook(context.Background(), root, entry, func() error { staged = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if staged {
		t.Error("unchanged registry declaration staged a replacement")
	}
	for range 3 {
		w, err = Open(path, opts)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	after, err := os.Stat(declaration)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Error("repeated Open rewrote unchanged registry declaration")
	}
	if err := os.Remove(declaration); err != nil {
		t.Fatal(err)
	}
	staged = false
	if err := registerWithHook(context.Background(), root, entry, func() error { staged = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !staged {
		t.Error("missing registry declaration was not republished")
	}
	if _, err := os.Stat(declaration); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryRegisterRefusesExistingChangedDeclaration(t *testing.T) {
	for _, kind := range []string{"altered", "malformed", "unknown-field", "symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path, err := canonical(filepath.Join(root, "trace.log"))
			if err != nil {
				t.Fatal(err)
			}
			entry := RegistryEntry{Version: 1, Path: path, Directory: directory(path), Lock: lockPath(path)}
			if err := register(context.Background(), root, entry); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256([]byte(path))
			declaration := filepath.Join(RegistryDirectory(root), fmt.Sprintf("%x.json", sum))
			switch kind {
			case "altered":
				changed := entry
				changed.Lock += ".changed"
				b, err := json.Marshal(changed)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(declaration, b, 0600); err != nil {
					t.Fatal(err)
				}
			case "malformed":
				if err := os.WriteFile(declaration, []byte("{"), 0600); err != nil {
					t.Fatal(err)
				}
			case "unknown-field":
				b, err := json.Marshal(entry)
				if err != nil {
					t.Fatal(err)
				}
				b = append(b[:len(b)-1], []byte(",\"unknown\":true}")...)
				if err := os.WriteFile(declaration, b, 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink", "directory":
				if err := os.Remove(declaration); err != nil {
					t.Fatal(err)
				}
				if kind == "symlink" {
					target := filepath.Join(root, "target")
					b, err := json.Marshal(entry)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(target, b, 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(target, declaration); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Mkdir(declaration, 0700); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.Lstat(declaration)
			if err != nil {
				t.Fatal(err)
			}
			staged := false
			err = registerWithHook(context.Background(), root, entry, func() error { staged = true; return nil })
			if err == nil {
				t.Error("unsafe or changed declaration was overwritten")
			}
			if staged {
				t.Error("replacement was staged before declaration validation")
			}
			after, err := os.Lstat(declaration)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
				t.Error("refusal mutated existing declaration")
			}
		})
	}
}

func TestRegistryRetryReconfirmsFailedPublicationDurability(t *testing.T) {
	path, _, _ := fixture(t)
	path, err := canonical(path)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	entry := RegistryEntry{Version: 1, Path: path, Directory: directory(path), Lock: lockPath(path)}
	failed := errors.New("directory sync failed after rename")
	syncs, stages := 0, 0
	fail := true
	syncDirectory := func(dir string) error {
		syncs++
		if dir != RegistryDirectory(root) {
			t.Fatalf("unexpected sync directory %q", dir)
		}
		if fail {
			return failed
		}
		return syncDir(dir)
	}
	stage := func() error { stages++; return nil }
	if err := registerWithDirectorySync(context.Background(), root, entry, stage, syncDirectory); !errors.Is(err, failed) {
		t.Fatalf("publication sync error=%v", err)
	}
	sum := sha256.Sum256([]byte(path))
	declaration := filepath.Join(RegistryDirectory(root), fmt.Sprintf("%x.json", sum))
	before, err := os.Stat(declaration)
	if err != nil {
		t.Fatalf("rename did not publish declaration: %v", err)
	}
	body, err := os.ReadFile(declaration)
	if err != nil {
		t.Fatal(err)
	}
	if err := registerWithDirectorySync(context.Background(), root, entry, stage, syncDirectory); !errors.Is(err, failed) {
		t.Fatalf("unchanged retry forgot failed sync: %v", err)
	}
	fail = false
	if err := registerWithDirectorySync(context.Background(), root, entry, stage, syncDirectory); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(declaration)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(declaration)
	if err != nil || string(got) != string(body) || !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("retry replaced published declaration: %q %v", got, err)
	}
	if syncs != 3 || stages != 1 {
		t.Fatalf("syncs=%d stages=%d want3/1", syncs, stages)
	}
}
