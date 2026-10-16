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

func TestResourceSnapshotStoreLoadsArtifactsIndependently(t *testing.T) {
	for _, bad := range []string{"checkpoint.json", "history.json"} {
		t.Run(bad, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			validate := func(raw []byte) error {
				if !json.Valid(raw) {
					return errors.New("malformed")
				}
				return nil
			}
			s := ResourceSnapshotStore{Dir: dir, ValidateCheckpoint: validate, ValidateHistory: validate}
			for _, name := range []string{"checkpoint.json", "history.json"} {
				raw := []byte(`{"valid":true}`)
				if name == bad {
					raw = []byte(`broken`)
				}
				if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			cp, h, err := s.Load()
			if err == nil {
				t.Fatal("missing corruption error")
			}
			if bad == "checkpoint.json" && string(h) != `{"valid":true}` || bad == "history.json" && string(cp) != `{"valid":true}` {
				t.Fatalf("valid sibling discarded: cp=%s history=%s err=%v", cp, h, err)
			}
		})
	}
}

func TestResourceSnapshotStoreSeparatesCorruptionFromUnsafePaths(t *testing.T) {
	for _, kind := range []string{"malformed", "oversized", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			s := ResourceSnapshotStore{Dir: dir, ValidateCheckpoint: func(b []byte) error {
				if !json.Valid(b) {
					return errors.New("bad JSON")
				}
				return nil
			}}
			p := filepath.Join(dir, "checkpoint.json")
			switch kind {
			case "malformed":
				err = os.WriteFile(p, []byte("broken"), 0600)
			case "oversized":
				err = os.WriteFile(p, make([]byte, ResourceCheckpointLimit+1), 0600)
			case "directory":
				err = os.Mkdir(p, 0700)
			case "symlink":
				err = os.Symlink(filepath.Join(dir, "missing"), p)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.LoadCheckpoint()
			var corrupt *ResourceSnapshotCorruptError
			if err == nil || errors.As(err, &corrupt) != (kind == "malformed" || kind == "oversized") {
				t.Fatalf("wrong recovery authority for %s: %v", kind, err)
			}
		})
	}
}

func TestResourceSnapshotStoreReadPermissionFailureIsNotCorruption(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permissions")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "checkpoint.json")
	if err := os.WriteFile(p, []byte(`{}`), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0600) })
	s := ResourceSnapshotStore{Dir: dir, ValidateCheckpoint: func([]byte) error { return nil }}
	_, err = s.LoadCheckpoint()
	var corrupt *ResourceSnapshotCorruptError
	if !errors.Is(err, os.ErrPermission) || errors.As(err, &corrupt) {
		t.Fatal("read failure granted rebuild permission", err)
	}
	if err := os.Chmod(p, 0600); err != nil {
		t.Fatal(err)
	}
	raw, err := s.LoadCheckpoint()
	if err != nil || string(raw) != `{}` {
		t.Fatal("read did not recover", string(raw), err)
	}
}
