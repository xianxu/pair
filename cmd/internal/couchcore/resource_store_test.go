package couchcore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func resourceHistoryStoreFixture(t *testing.T) ResourceHistoryStore {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return ResourceHistoryStore{Dir: filepath.Join(dir, "resources")}
}
func TestResourceHistoryStoreValidatesBeforePublication(t *testing.T) {
	s := resourceHistoryStoreFixture(t)
	good := []byte(`{"version":1,"frames":[]}`)
	if err := s.Save(good); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{[]byte(`broken`), []byte(`{"version":99,"frames":[]}`), bytes.Repeat([]byte(" "), ResourceHistoryLimit+1)} {
		if err := s.Save(bad); err == nil {
			t.Fatal("invalid history accepted")
		}
	}
	raw, err := s.Load()
	if err != nil || !bytes.Equal(raw, good) {
		t.Fatal("failed save replaced good history", string(raw), err)
	}
	info, err := os.Stat(filepath.Join(s.Dir, "history.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("history permissions", info, err)
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "history.json" {
		t.Fatal("history store created other artifacts", entries, err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "history.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Fatal("corrupt history accepted")
	}
}
func TestResourceHistoryStoreRejectsSymlinksAndNonregularFiles(t *testing.T) {
	for _, target := range []string{"history.json", ".history-stage"} {
		t.Run(target, func(t *testing.T) {
			s := resourceHistoryStoreFixture(t)
			if err := os.Mkdir(s.Dir, 0700); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(filepath.Dir(s.Dir), "outside")
			original := []byte(`{"version":1,"frames":[]}`)
			if err := os.WriteFile(outside, original, 0600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(s.Dir, target)
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
			if target == "history.json" {
				if _, err := s.Load(); err == nil {
					t.Fatal("load followed symlink")
				}
			}
			if err := s.Save(original); err == nil {
				t.Fatal("save followed symlink")
			}
			got, err := os.ReadFile(outside)
			if err != nil || !bytes.Equal(got, original) {
				t.Fatal("outside target changed")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			if err := s.Save(original); err == nil {
				t.Fatal("nonregular path accepted")
			}
		})
	}
}
func TestResourceHistoryStoreFixedStageAndByteBound(t *testing.T) {
	s := resourceHistoryStoreFixture(t)
	header := []byte(`{"version":1,"frames":[]}`)
	bounded := append(header, bytes.Repeat([]byte(" "), ResourceHistoryLimit-len(header))...)
	if err := s.Save(bounded); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(append(bounded, ' ')); err == nil {
		t.Fatal("over-cap history accepted")
	}
	stage := filepath.Join(s.Dir, ".history-stage")
	if err := os.WriteFile(stage, bounded, 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil || len(entries) != 2 {
		t.Fatal("unbounded stage names", entries, err)
	}
	var total int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		total += info.Size()
	}
	if total != 2*ResourceHistoryLimit {
		t.Fatal("disk envelope", total)
	}
	// An owned regular crash residue is reused at the same fixed stage name.
	if err := s.Save(header); err != nil {
		t.Fatal("stale regular stage blocked publication", err)
	}
	entries, err = os.ReadDir(s.Dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "history.json" {
		t.Fatal("stage residue accumulated", entries, err)
	}
}
