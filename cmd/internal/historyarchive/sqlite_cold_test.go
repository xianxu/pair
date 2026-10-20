package historyarchive

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// coldInputFixture advances source events at read/observation boundaries. Its
// bytes and evidence are independent; missing any fence admits a real bad case.
type coldInputFixture struct {
	*bytes.Reader
	body                     []byte
	stamp                    coldSourceStamp
	pass, readBytes, headers int
	observations             int
	onRead                   func(*coldInputFixture)
	onObserve                func(*coldInputFixture) error
}

func newColdInputFixture(size int) *coldInputFixture {
	body := bytes.Repeat([]byte{'x'}, size)
	copy(body, "SQLite format 3\x00")
	body[18], body[19] = 2, 2
	f := &coldInputFixture{body: body, Reader: bytes.NewReader(body)}
	f.stamp = coldSourceStamp{DB: coldFileStamp{Device: 1, Inode: 2, Size: int64(size), Mode: unix.S_IFREG | 0600, MtimeSec: 1, CtimeSec: 2}, Parent: coldFileStamp{Device: 1, Inode: 3, Mode: unix.S_IFDIR | 0700, MtimeSec: 3, CtimeSec: 4}}
	return f
}

func (f *coldInputFixture) Read(p []byte) (int, error) {
	n, err := f.Reader.Read(p)
	f.readBytes += n
	if n > 0 && f.onRead != nil {
		f.onRead(f)
	}
	return n, err
}
func (f *coldInputFixture) ReadAt(p []byte, off int64) (int, error) {
	f.headers++
	return f.Reader.ReadAt(p, off)
}
func (f *coldInputFixture) Seek(off int64, whence int) (int64, error) {
	if off == 0 && whence == io.SeekStart {
		f.pass++
	}
	return f.Reader.Seek(off, whence)
}
func (f *coldInputFixture) Observe() (coldSourceStamp, error) {
	f.observations++
	if f.onObserve != nil {
		if err := f.onObserve(f); err != nil {
			return coldSourceStamp{}, err
		}
	}
	return f.stamp, nil
}

func TestColdWALCopyCountedCost(t *testing.T) {
	for _, n := range []int{4, 8} {
		for _, size := range []int{128 << 10, 256 << 10} {
			t.Run(strings.Repeat("N", n)+"/"+strings.Repeat("B", size/(128<<10)), func(t *testing.T) {
				var totalBytes, totalObservations int
				for i := 0; i < n; i++ {
					f := newColdInputFixture(size)
					out := filepath.Join(t.TempDir(), "snapshot.sqlite")
					handled, err := copyColdWAL(context.Background(), f, out)
					if err != nil || !handled {
						t.Fatalf("stable capture: handled=%v err=%v", handled, err)
					}
					got, err := os.ReadFile(out)
					if err != nil || !bytes.Equal(got, f.body) {
						t.Fatalf("private bytes changed: %v", err)
					}
					info, err := os.Stat(out)
					if err != nil || info.Mode().Perm() != 0600 {
						t.Fatalf("unsafe private mode: %v %v", info, err)
					}
					if f.pass != 2 || f.headers != 1 {
						t.Fatalf("unexpected reads: %+v", f)
					}
					totalBytes += f.readBytes
					totalObservations += f.observations
				}
				if totalBytes != 2*n*size || totalObservations != 3*n {
					t.Fatalf("cost bytes=%d observations=%d, want %d/%d", totalBytes, totalObservations, 2*n*size, 3*n)
				}
			})
		}
	}
}

func TestColdWALCopyRefusesMutations(t *testing.T) {
	mutations := map[string]func(*coldInputFixture){
		"writer-starts":      func(f *coldInputFixture) { f.stamp.Sidecars = 3; f.stamp.Parent.CtimeNsec++ },
		"transient-sidecars": func(f *coldInputFixture) { f.stamp.Sidecars = 0; f.stamp.Parent.CtimeNsec++ },
		"restored-mtime":     func(f *coldInputFixture) { f.stamp.DB.CtimeNsec++ },
		"database-replaced":  func(f *coldInputFixture) { f.stamp.DB.Inode++ },
		"parent-replaced":    func(f *coldInputFixture) { f.stamp.Parent.Inode++ },
		"digest-mismatch":    func(f *coldInputFixture) { f.body[100] ^= 1 },
	}
	for name, mutate := range mutations {
		for _, pass := range []int{1, 2} {
			t.Run(name+"/pass"+strings.Repeat("I", pass), func(t *testing.T) {
				f := newColdInputFixture(128 << 10)
				fired := false
				f.onRead = func(f *coldInputFixture) {
					if f.pass == pass && !fired {
						fired = true
						mutate(f)
					}
				}
				// Mutating already-read bytes in pass 2 cannot change the
				// captured state. Change the upcoming bytes for this oracle.
				if name == "digest-mismatch" && pass == 2 {
					f.onObserve = func(f *coldInputFixture) error {
						if f.observations == 2 {
							fired = true
							mutate(f)
						}
						return nil
					}
					f.onRead = nil
				}
				out := filepath.Join(t.TempDir(), "snapshot.sqlite")
				handled, err := copyColdWAL(context.Background(), f, out)
				if !fired || !handled || !errors.Is(err, errColdCapture) {
					t.Fatalf("mutation admitted: fired=%v handled=%v err=%v", fired, handled, err)
				}
				if _, err := os.Stat(out); !os.IsNotExist(err) {
					t.Fatalf("uncertain candidate retained: %v", err)
				}
			})
		}
	}
}

func TestColdWALCopyBoundsAndCancellation(t *testing.T) {
	for _, shape := range []string{"cancel", "deadline", "observation-error", "short-read", "oversize", "collision"} {
		t.Run(shape, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f := newColdInputFixture(128 << 10)
			out := filepath.Join(t.TempDir(), "snapshot.sqlite")
			switch shape {
			case "cancel":
				f.onRead = func(*coldInputFixture) { cancel() }
			case "deadline":
				f.onObserve = func(*coldInputFixture) error { return context.DeadlineExceeded }
			case "observation-error":
				f.onObserve = func(f *coldInputFixture) error {
					if f.observations == 2 {
						return os.ErrPermission
					}
					return nil
				}
			case "short-read":
				f.stamp.DB.Size++
			case "oversize":
				f.stamp.DB.Size = MaxSnapshotRawBytes + 1
			case "collision":
				if err := os.WriteFile(out, []byte("previous"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			handled, err := copyColdWAL(ctx, f, out)
			wantUncertain := shape != "oversize" && shape != "collision"
			if !handled || err == nil || errors.Is(err, errColdCapture) != wantUncertain {
				t.Fatalf("wrong capture failure classification: %v %v", handled, err)
			}
			if shape == "collision" {
				got, _ := os.ReadFile(out)
				if string(got) != "previous" {
					t.Fatalf("existing output replaced: %q", got)
				}
			} else if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("partial candidate retained: %v", err)
			}
		})
	}
}

func TestColdWALCopyLeavesActiveAndOtherFormatsToSQLite(t *testing.T) {
	for _, shape := range []string{"wal", "shm", "journal", "rollback", "other", "short-header"} {
		t.Run(shape, func(t *testing.T) {
			f := newColdInputFixture(128 << 10)
			switch shape {
			case "wal":
				f.stamp.Sidecars = 1
			case "shm":
				f.stamp.Sidecars = 2
			case "journal":
				f.stamp.Sidecars = 4
			case "rollback":
				f.body[18], f.body[19] = 1, 1
			case "other":
				f.body[0] = 'X'
			case "short-header":
				f.Reader = bytes.NewReader(f.body[:50])
			}
			out := filepath.Join(t.TempDir(), "snapshot.sqlite")
			handled, err := copyColdWAL(context.Background(), f, out)
			if handled || err != nil || f.readBytes != 0 {
				t.Fatalf("stole native route: %v %v reads=%d", handled, err, f.readBytes)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("unexpected private output: %v", err)
			}
		})
	}
}

func TestColdSnapshotOutputPreservesLocalWriteFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readonly.sqlite")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n, err := (coldSnapshotOutput{writer: f}).Write([]byte("replacement"))
	if n != 0 || !errors.Is(err, errColdOutput) || errors.Is(err, errColdCapture) {
		t.Fatalf("local write failure lost its category: n=%d err=%v", n, err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "original" {
		t.Fatalf("read-only fixture changed: %q %v", got, err)
	}
}

func TestObserveColdSourceNeverTreatsUnknownAsAbsent(t *testing.T) {
	for _, shape := range []string{"stable", "present", "transient", "replace", "symlink", "missing-parent"} {
		t.Run(shape, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "closed.sqlite")
			if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			source, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer source.Close()
			parent, err := os.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer parent.Close()
			before, err := observeColdSource(source, parent, path)
			if err != nil {
				t.Fatal(err)
			}
			switch shape {
			case "present", "transient":
				if err := os.WriteFile(path+"-wal", []byte("writer"), 0600); err != nil {
					t.Fatal(err)
				}
				if shape == "transient" {
					if err := os.Remove(path + "-wal"); err != nil {
						t.Fatal(err)
					}
				}
			case "replace":
				if err := os.Rename(path, path+".old"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(path, path+"-journal"); err != nil {
					t.Fatal(err)
				}
			case "missing-parent":
				if err := os.Rename(dir, dir+".moved"); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.RemoveAll(dir + ".moved") })
			}
			after, err := observeColdSource(source, parent, path)
			if shape == "stable" {
				if err != nil || !sameColdSource(before, after) {
					t.Fatalf("stable evidence refused: %+v %v", after, err)
				}
			} else if err == nil && sameColdSource(before, after) {
				t.Fatalf("unknown/changed source accepted: %s", shape)
			}
		})
	}
}

func TestSQLiteSnapshotColdAndActiveRoutes(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "cold", true: "active"}[active], func(t *testing.T) {
			src, stage, scratch := t.TempDir(), t.TempDir(), t.TempDir()
			path := filepath.Join(src, "db.sqlite")
			if err := os.WriteFile(path, newColdInputFixture(512).body, 0600); err != nil {
				t.Fatal(err)
			}
			if active {
				if err := os.WriteFile(path+"-wal", []byte("writer WAL"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := cold701SourceImage(t, src)
			fake := &fakeSQLiteRunner{}
			result, err := (SQLiteSnapshotter{Scratch: scratch, Runner: fake}).Snapshot(context.Background(), src, "db.sqlite", stage)
			wantReads := 0
			if active {
				wantReads = 1
			}
			if err != nil || result.Status != SnapshotReady || fake.reads != wantReads || fake.validations != 1 {
				t.Fatalf("wrong lane: %+v %v fake=%+v", result, err, fake)
			}
			cold701AssertSourceImage(t, src, before)
			if names := dirNames(t, scratch); len(names) != 0 {
				t.Fatalf("scratch leaked: %v", names)
			}
		})
	}
}

func TestSQLiteColdRefusalPreservesPreviousStage(t *testing.T) {
	for _, shape := range []string{"integrity", "tool", "deadline", "unsafe-sidecar"} {
		t.Run(shape, func(t *testing.T) {
			src, stage, scratch := t.TempDir(), t.TempDir(), t.TempDir()
			path := filepath.Join(src, "db.sqlite")
			if err := os.WriteFile(path, newColdInputFixture(512).body, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(stage, "db.sqlite"), []byte("last verified candidate"), 0600); err != nil {
				t.Fatal(err)
			}
			fake := &fakeSQLiteRunner{}
			want := SnapshotRejected
			validations := 1
			switch shape {
			case "integrity":
				fake.corrupt = true
			case "tool":
				fake.fail = errors.New("validator exit 1")
			case "deadline":
				fake.fail, want = ErrSQLiteDeadline, SnapshotDeferred
			case "unsafe-sidecar":
				if err := os.Symlink(path, path+"-journal"); err != nil {
					t.Fatal(err)
				}
				want, validations = SnapshotRejected, 0
			}
			var before map[string]cold701ImageEntry
			if shape != "unsafe-sidecar" {
				before = cold701SourceImage(t, src)
			}
			result, err := (SQLiteSnapshotter{Scratch: scratch, Runner: fake}).Snapshot(context.Background(), src, "db.sqlite", stage)
			if err != nil || result.Status != want || fake.reads != 0 || fake.validations != validations {
				t.Fatalf("bad candidate published or retried: %+v %v fake=%+v", result, err, fake)
			}
			if before != nil {
				cold701AssertSourceImage(t, src, before)
			} else {
				body, readErr := os.ReadFile(path)
				link, linkErr := os.Readlink(path + "-journal")
				if readErr != nil || linkErr != nil || link != path || !bytes.Equal(body, newColdInputFixture(512).body) {
					t.Fatalf("unsafe fixture changed: read=%v link=%q/%v", readErr, link, linkErr)
				}
			}
			got, err := os.ReadFile(filepath.Join(stage, "db.sqlite"))
			if err != nil || string(got) != "last verified candidate" {
				t.Fatalf("previous candidate lost: %q %v", got, err)
			}
			if names := dirNames(t, scratch); len(names) != 0 {
				t.Fatalf("scratch leaked: %v", names)
			}
		})
	}
}
