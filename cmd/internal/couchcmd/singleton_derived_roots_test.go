package couchcmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestIsolatedDerivedRootsRefuseUnsafeDirectoriesBeforePublication(t *testing.T) {
	for _, name := range []string{"home", "tmp", "data"} {
		for _, kind := range []string{"escaping-symlink", "dangling-symlink", "regular-file"} {
			for _, blocked := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/blocked=%v", name, kind, blocked), func(t *testing.T) {
					root, outside := derivedRootDirectory(t), derivedRootDirectory(t)
					if err := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("untouched\n"), 0600); err != nil {
						t.Fatal(err)
					}
					path, target := filepath.Join(root, name), outside
					switch kind {
					case "regular-file":
						if err := os.WriteFile(path, []byte("not a directory\n"), 0600); err != nil {
							t.Fatal(err)
						}
					default:
						if kind == "dangling-symlink" {
							target = filepath.Join(outside, "missing")
						}
						if err := os.Symlink(target, path); err != nil {
							t.Fatal(err)
						}
					}
					rt := derivedRootRuntime(root, outside)
					before, outsideBefore := derivedRootSnapshot(t, root), derivedRootSnapshot(t, outside)
					runner, _, err := attemptDerivedRootLaunch(rt, root, blocked)
					if err == nil {
						t.Errorf("accepted unsafe derived %s directory", name)
					}
					if len(runner.Ops) != 0 {
						t.Errorf("unsafe root reached runner: %v", runner.Ops)
					}
					// Explicit adoption cannot bypass the same derived-root gate.
					for _, apply := range []bool{false, true} {
						args := []string{"--adopt-store", filepath.Join(root, "store"), "--pair-data", filepath.Join(root, "selected-pair"), "--identity-dir", filepath.Join(root, "identity")}
						if apply {
							args = append(args, "--apply", strings.Repeat("a", 64))
						}
						var stdout, stderr strings.Builder
						if code := RunWithRuntime(args, strings.NewReader(""), &stdout, &stderr, rt); code == 0 {
							t.Errorf("adoption apply=%v accepted unsafe root: %s", apply, stdout.String())
						}
					}
					if _, err := os.Lstat(filepath.Join(root, "singleton", "selection.json")); !os.IsNotExist(err) {
						t.Errorf("refusal published selection: %v", err)
					}
					if after := derivedRootSnapshot(t, root); !reflect.DeepEqual(before, after) {
						t.Error("refusal changed isolated source tree")
					}
					if after := derivedRootSnapshot(t, outside); !reflect.DeepEqual(outsideBefore, after) {
						t.Error("refusal changed outside sentinel tree")
					}
				})
			}
		}
	}
}

func TestIsolatedDerivedRootsExportCanonicalConfinedDirectories(t *testing.T) {
	for _, mode := range []string{"outside-home-fallback", "explicit-confined-home", "internal-symlinks"} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/blocked=%v", mode, blocked), func(t *testing.T) {
				root, outside := derivedRootDirectory(t), derivedRootDirectory(t)
				home := outside
				want := map[string]string{"HOME": filepath.Join(root, "home"), "TMPDIR": filepath.Join(root, "tmp"), "XDG_DATA_HOME": filepath.Join(root, "data")}
				if mode == "explicit-confined-home" {
					home = filepath.Join(root, "explicit-home")
					want["HOME"] = home
				}
				if mode == "internal-symlinks" {
					for key, name := range map[string]string{"HOME": "home", "TMPDIR": "tmp", "XDG_DATA_HOME": "data"} {
						physical := filepath.Join(root, "physical-"+name)
						if err := os.Mkdir(physical, 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.Symlink(physical, filepath.Join(root, name)); err != nil {
							t.Fatal(err)
						}
						want[key] = physical
					}
				}
				_, env, err := attemptDerivedRootLaunch(derivedRootRuntime(root, home), root, blocked)
				if err != nil {
					t.Fatal(err)
				}
				for key, path := range want {
					if env[key] != path {
						t.Errorf("%s=%q, want confined physical path %q", key, env[key], path)
					}
				}
				entries, err := os.ReadDir(outside)
				if err != nil || len(entries) != 0 {
					t.Fatalf("outside HOME changed: %v %v", entries, err)
				}
			})
		}
	}
}

func derivedRootRuntime(root, home string) OSRuntime {
	// Explicit store/data/identity roots keep the derived XDG directory out of
	// the selection tuple: validating only selected roots cannot catch its escape.
	env := map[string]string{
		"COUCH_ISOLATED_ROOT": root, "HOME": home,
		"COUCH_STORE_DIR":     filepath.Join(root, "store"),
		"COUCH_PAIR_DATA_DIR": filepath.Join(root, "selected-pair"),
		"COUCH_IDENTITY_DIR":  filepath.Join(root, "identity"),
	}
	return OSRuntime{env: func(key string) string { return env[key] }, accountHome: func() (string, error) {
		return "", fmt.Errorf("isolated fixture must never look up account home")
	}}
}

func attemptDerivedRootLaunch(rt OSRuntime, root string, blocked bool) (*couchcore.FakeRunner, map[string]string, error) {
	runner := couchcore.NewFakeRunner()
	prepared, lease, err := rt.prepareSingleton(true)
	if err != nil {
		return runner, nil, err
	}
	defer lease.Close()
	var handle couchcore.Handle
	if blocked {
		child, startErr := prepared.runtimeRunner(runner).StartBlocked(context.Background(), root, []string{"pair", "resume", "tag"}, nil, time.Second)
		handle, err = child, startErr
		if child != nil {
			defer child.Cancel()
		}
	} else {
		handle, err = prepared.runtimeRunner(runner).Start(root, []string{"pair", "resume", "tag"}, nil)
	}
	if err != nil {
		return runner, nil, err
	}
	env := map[string]string{}
	for _, entry := range runner.Child(handle.ID()).Env {
		key, value, _ := strings.Cut(entry, "=")
		env[key] = value
	}
	runner.SetExited(handle.ID(), 0)
	return runner, env, nil
}

func derivedRootDirectory(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func derivedRootSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	entries := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			value += ":" + target
		case !entry.IsDir():
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += ":" + string(raw)
		}
		entries[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
