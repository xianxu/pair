package diagnosticlog

import (
	"encoding/json"
	"errors"
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
		entries, e := EnumerateRoot(root, 0, 100)
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
	*now = now.Add(8 * 24 * time.Hour)
	state, _ := os.ReadFile(filepath.Join(directory(path), "state.json"))
	rows, e := Preview(path, opts, 100)
	if e != nil || len(rows) != 1 || !rows[0].Eligible {
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
	opts.Registry = func(e RegistryEntry) error { return register(root, e) }
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
	if _, e = EnumerateRoot(root, 0, 100); e == nil {
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
	w.Write([]byte("expired"))
	w.Close()
	s, e := load(path)
	if e != nil {
		t.Fatal(e)
	}
	s.Writers = nil
	if e = save(path, s, true); e != nil {
		t.Fatal(e)
	}
	*now = now.Add(8 * 24 * time.Hour)
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
	entries, e := EnumerateRoot(root, 0, 100)
	if e != nil || len(entries) != 0 {
		t.Fatalf("registry=%v err=%v", entries, e)
	}
	if e = syscall.Flock(int(paused.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		t.Fatal(e)
	}
	if _, e = Open(path, opts); !errors.Is(e, ErrBusy) {
		t.Fatalf("opener ignored stable lock %v", e)
	}
	entries, e = EnumerateRoot(root, 0, 100)
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
	entries, e = EnumerateRoot(root, 0, 100)
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
			w.Write([]byte("expired"))
			w.Close()
			s, _ := load(path)
			s.Writers = nil
			save(path, s, true)
			*now = now.Add(8 * 24 * time.Hour)
			opts.Fault = func(step string) error {
				if step == point {
					return errors.New("crash")
				}
				return nil
			}
			if _, e = Collect(path, opts, 100); e == nil {
				t.Fatal("retirement fault ignored")
			}
			entries, e := EnumerateRoot(root, 0, 100)
			if e != nil || len(entries) != 1 {
				t.Fatal("recovery lost discovery", entries, e)
			}
			opts.Fault = nil
			if _, e = Collect(path, opts, 100); e != nil {
				t.Fatal(e)
			}
			entries, e = EnumerateRoot(root, 0, 100)
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
