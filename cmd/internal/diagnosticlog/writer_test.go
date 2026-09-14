package diagnosticlog

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T) (string, *time.Time, Options) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trace.log")
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	return path, &now, Options{SynchronousMaintenance: true, Now: func() time.Time { return now }, Proof: func(string, []Registration) error { return nil }}
}
func TestGenerationAgeSurvivesRestart(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("first\n"))
	w.Close()
	*now = now.Add(23 * time.Hour)
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("second\n"))
	w.Close()
	*now = now.Add(2 * time.Hour)
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if _, e = w.Write([]byte("third\n")); e != nil {
		t.Fatal(e)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "third\n" {
		t.Fatalf("current=%q", got)
	}
	rows, e := Preview(path, opts, 100)
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%+v", rows)
	}
}
func TestUnknownWriterBlocksRotationAndCollection(t *testing.T) {
	path, now, opts := fixture(t)
	opts.Proof = nil
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	w.Write([]byte("first\n"))
	*now = now.Add(8 * 24 * time.Hour)
	w.Write([]byte("second\n"))
	got, _ := os.ReadFile(path)
	if string(got) != "first\nsecond\n" {
		t.Fatalf("unknown writer lost data %q", got)
	}
	if _, e = Collect(path, opts, 100); e == nil {
		t.Fatal("unknown proof permitted collection")
	}
}
func TestRotationRecovery(t *testing.T) {
	for _, point := range []string{"intent", "rename", "state"} {
		t.Run(point, func(t *testing.T) {
			path, now, opts := fixture(t)
			w, e := Open(path, opts)
			if e != nil {
				t.Fatal(e)
			}
			w.Write([]byte("before\n"))
			w.Close()
			*now = now.Add(25 * time.Hour)
			opts.Fault = func(step string) error {
				if step == point {
					return errors.New("crash")
				}
				return nil
			}
			w, e = Open(path, opts)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = w.Write([]byte("lost\n")); e == nil {
				t.Fatal("fault ignored")
			}
			w.Close()
			opts.Fault = nil
			w, e = Open(path, opts)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			if _, e = w.Write([]byte("after\n")); e != nil {
				t.Fatal(e)
			}
			files, _ := allSegments(path)
			if len(files) != 1 {
				t.Fatalf("segments=%v", files)
			}
			got, _ := os.ReadFile(files[0])
			if string(got) != "before\n" {
				t.Fatalf("segment=%q", got)
			}
			got, _ = os.ReadFile(path)
			if string(got) != "after\n" {
				t.Fatalf("current=%q", got)
			}
		})
	}
}
func TestConcurrentWritersReopenCurrent(t *testing.T) {
	path, _, opts := fixture(t)
	opts.MaxBytes = 8
	writers := make([]*Writer, 4)
	for i := range writers {
		var e error
		writers[i], e = Open(path, opts)
		if e != nil {
			t.Fatal(e)
		}
		defer writers[i].Close()
	}
	var wg sync.WaitGroup
	for _, w := range writers {
		wg.Go(func() {
			for range 25 {
				_, e := w.Write([]byte("line\n"))
				if e != nil && !errors.Is(e, ErrBusy) {
					t.Error(e)
				}
			}
		})
	}
	wg.Wait()
	files, _ := allSegments(path)
	files = append(files, path)
	for _, f := range files {
		b, e := os.ReadFile(f)
		if e != nil {
			t.Fatal(e)
		}
		if len(b)%5 != 0 {
			t.Fatalf("torn append %q", b)
		}
	}
}
func TestExpiryAndStableLock(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("young\n"))
	w.Close()
	lock, e := os.Stat(path + ".pair-diagnostics.lock")
	if e != nil {
		t.Fatal(e)
	}
	*now = now.Add(7*24*time.Hour - time.Nanosecond)
	rows, e := Collect(path, opts, 100)
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 0 {
		t.Fatal(rows)
	}
	*now = now.Add(time.Nanosecond)
	rows, e = Collect(path, opts, 100)
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 1 {
		t.Fatal(rows)
	}
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatalf("current remains %v", e)
	}
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	w.Write([]byte("new\n"))
	after, _ := os.Stat(path + ".pair-diagnostics.lock")
	if !os.SameFile(lock, after) {
		t.Fatal("lock inode replaced")
	}
}
func TestRejectSubstitution(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("original\n"))
	w.Close()
	*now = now.Add(8 * 24 * time.Hour)
	os.Rename(path, path+".old")
	os.WriteFile(path, []byte("replacement\n"), 0600)
	if _, e = Collect(path, opts, 100); e == nil {
		t.Fatal("replacement accepted")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "replacement\n" {
		t.Fatal("replacement removed")
	}
}

func TestCollectionRecoveryBeforeNextAppend(t *testing.T) {
	for _, point := range []string{"delete-intent", "delete-payload"} {
		t.Run(point, func(t *testing.T) {
			path, now, opts := fixture(t)
			w, e := Open(path, opts)
			if e != nil {
				t.Fatal(e)
			}
			w.Write([]byte("expired\n"))
			w.Close()
			*now = now.Add(8 * 24 * time.Hour)
			opts.Fault = func(step string) error {
				if step == point {
					return errors.New("crash")
				}
				return nil
			}
			if _, e = Collect(path, opts, 100); e == nil {
				t.Fatal("fault ignored")
			}
			opts.Fault = nil
			w, e = Open(path, opts)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			if _, e = w.Write([]byte("new\n")); e != nil {
				t.Fatal(e)
			}
			got, _ := os.ReadFile(path)
			if string(got) != "new\n" {
				t.Fatalf("current %q", got)
			}
		})
	}
}
func TestSameSizeExternalModificationBlocks(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("old\n"))
	w.Close()
	st, _ := os.Stat(path)
	os.WriteFile(path, []byte("new\n"), 0600)
	os.Chtimes(path, st.ModTime().Add(time.Minute), st.ModTime().Add(time.Minute))
	*now = now.Add(8 * 24 * time.Hour)
	if _, e = Collect(path, opts, 100); e == nil {
		t.Fatal("same-size outside write accepted")
	}
}

func TestOpenCreatesCurrentAndRejectsReadOnlyTarget(t *testing.T) {
	path, _, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if _, e = os.Stat(path); e != nil {
		t.Fatal("open did not create current:", e)
	}
}
func TestMalformedDeletionIntentCannotRemoveCurrent(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("young\n"))
	w.Close()
	s, e := load(path)
	if e != nil {
		t.Fatal(e)
	}
	s.Deleting = &generation{}
	save(path, s, true)
	*now = now.Add(8 * 24 * time.Hour)
	if _, e = Collect(path, opts, 100); e == nil {
		t.Fatal("malformed deletion accepted")
	}
	if got, _ := os.ReadFile(path); string(got) != "young\n" {
		t.Fatal("current damaged")
	}
}

func TestProductionWriteNeverWaitsForInspection(t *testing.T) {
	path, _, opts := fixture(t)
	opts.SynchronousMaintenance = false
	opts.MaxBytes = 4
	entered := make(chan struct{})
	release := make(chan struct{})
	opts.Proof = func(string, []Registration) error { close(entered); <-release; return nil }
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("first"))
	returned := make(chan error, 1)
	go func() { _, e := w.Write([]byte("second")); returned <- e }()
	select {
	case e := <-returned:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("append waited for inspection")
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("maintenance not scheduled")
	}
	if _, e = w.Write([]byte("optional")); !errors.Is(e, ErrBusy) {
		t.Fatalf("contended optional record %v", e)
	}
	close(release)
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
}

func TestRotationRecoveryRejectsReplacementCurrent(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("old"))
	w.Close()
	*now = now.Add(25 * time.Hour)
	opts.Fault = func(step string) error {
		if step == "rename" {
			return errors.New("crash")
		}
		return nil
	}
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("attempt"))
	w.Close()
	os.WriteFile(path, []byte("replacement"), 0600)
	opts.Fault = nil
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if _, e = w.Write([]byte("append")); e == nil {
		t.Fatal("replacement accepted during recovery")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "replacement" {
		t.Fatal("replacement changed")
	}
}

func TestAbandonedMetadataTempHasBoundedLifetime(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("old"))
	w.Close()
	s, _ := load(path)
	s.Writers = nil
	save(path, s, true)
	abandoned := filepath.Join(directory(path), ".pending-1234")
	os.WriteFile(abandoned, []byte("interrupted metadata"), 0600)
	os.Chtimes(abandoned, *now, *now)
	*now = now.Add(8 * 24 * time.Hour)
	if _, e = Collect(path, opts, 100); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(abandoned); !os.IsNotExist(e) {
		t.Fatal("old abandoned metadata retained")
	}
	if _, e = os.Stat(directory(path)); !os.IsNotExist(e) {
		t.Fatal("metadata did not retire")
	}
}

func allSegments(path string) ([]string, error) {
	var names []string
	e := filepath.WalkDir(directory(path), func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && validSegment(d.Name()) {
			names = append(names, p)
		}
		return nil
	})
	return names, e
}

func TestRotationRecoveryRetiresAbandonedLeafMetadataTemp(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("old"))
	w.Close()
	*now = now.Add(25 * time.Hour)
	opts.Fault = func(step string) error {
		if step == "rename" {
			return errors.New("crash")
		}
		return nil
	}
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("attempt"))
	w.Close()
	s, _ := load(path)
	temp := filepath.Join(filepath.Dir(segmentPath(path, s.Pending.Name)), ".pending-5678")
	os.WriteFile(temp, []byte("partial generation metadata"), 0600)
	opts.Fault = nil
	w, e = Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if _, e = w.Write([]byte("after")); e != nil {
		t.Fatal(e)
	}
	if _, e = Preview(path, opts, 100); e != nil {
		t.Fatalf("recovered generation blocked by abandoned temp: %v", e)
	}
	if _, e = os.Stat(temp); !os.IsNotExist(e) {
		t.Fatal("abandoned leaf temp survived recovery")
	}
}
