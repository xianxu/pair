package couchcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceSnapshotStoreValidatesBeforePublication(t *testing.T) {
	validate := func(raw []byte) error {
		if !json.Valid(raw) {
			return errors.New("invalid JSON")
		}
		return nil
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := ResourceSnapshotStore{Dir: filepath.Join(root, "resources"), ValidateCheckpoint: validate, ValidateHistory: validate}
	if err := s.SaveCheckpoint([]byte(`{"generation":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveCheckpoint([]byte(`broken`)); err == nil {
		t.Fatal("accepted corrupt checkpoint")
	}
	if err := s.SaveCheckpoint(bytes.Repeat([]byte(" "), ResourceCheckpointLimit+1)); err == nil {
		t.Fatal("accepted oversized checkpoint")
	}
	c, h, err := s.Load()
	if err != nil || string(c) != `{"generation":1}` || len(h) != 0 {
		t.Fatalf("load %q %q %v", c, h, err)
	}
	info, err := os.Stat(filepath.Join(s.Dir, "checkpoint.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("permissions %v %v", info, err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "checkpoint.json"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Load(); err == nil {
		t.Fatal("admitted corrupt provenance")
	}
}

func TestResourceSnapshotStoreRejectsSymlinksAndNonregularFiles(t *testing.T) {
	for _, target := range []string{"checkpoint.json", "history.json", ".checkpoint-stage"} {
		t.Run(target, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(t.TempDir(), "outside")
			if err := os.WriteFile(outside, []byte(`{}`), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(dir, target)); err != nil {
				t.Fatal(err)
			}
			s := ResourceSnapshotStore{Dir: dir, ValidateCheckpoint: func([]byte) error { return nil }, ValidateHistory: func([]byte) error { return nil }}
			if target == ".checkpoint-stage" {
				if err := s.SaveCheckpoint([]byte(`{}`)); err == nil {
					t.Fatal("followed stage link")
				}
			} else {
				if _, _, err := s.Load(); err == nil {
					t.Fatal("followed link")
				}
			}
		})
	}
}
