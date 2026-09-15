package diagnosticlog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeletionReplayAfterParentCleanup(t *testing.T) {
	path, now, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("old")); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(25 * time.Hour)
	if _, err = w.Write([]byte("new")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	names, _, _, err := generationPage(path, "", 100, false, opts)
	if err != nil || len(names) != 1 {
		t.Fatalf("generation %v %v", names, err)
	}
	p := segmentPath(path, names[0])
	var g generation
	if err = readJSON(p+".json", &g); err != nil {
		t.Fatal(err)
	}
	s, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Deleting = &g
	if err = save(path, s, true, Options{}); err != nil {
		t.Fatal(err)
	}
	// Exact durable state after recoverDeletion removes its empty parent tree,
	// before it clears the durable Deleting intent.
	if err = os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(p + ".json"); err != nil {
		t.Fatal(err)
	}
	if err = removeEmptyParents(filepath.Dir(p), directory(path), opts); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(8 * 24 * time.Hour)
	if _, err = Collect(path, opts, 100); err != nil {
		t.Fatalf("cannot replay completed deletion: %v", err)
	}
}

func deletionIntentFixture(t *testing.T) (string, string, Options) {
	t.Helper()
	path, now, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("old")); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(25 * time.Hour)
	if _, err = w.Write([]byte("new")); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	names, _, _, err := generationPage(path, "", 100, false, opts)
	if err != nil || len(names) != 1 {
		t.Fatalf("generation %v %v", names, err)
	}
	p := segmentPath(path, names[0])
	var g generation
	if err = readJSON(p+".json", &g); err != nil {
		t.Fatal(err)
	}
	s, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Deleting = &g
	if err = save(path, s, true, Options{}); err != nil {
		t.Fatal(err)
	}
	return path, p, opts
}

func TestDeletionReplayEveryEffectBoundary(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		for _, boundary := range []string{"delete-payload", "delete-metadata", "parent-1", "parent-2", "parent-3", "parent-4", "parent-5", "delete-retire-intent"} {
			t.Run(fmt.Sprintf("%s/cancel=%v", boundary, cancelled), func(t *testing.T) {
				path, p, opts := deletionIntentFixture(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				opts.Context = ctx
				hit := false
				parents := 0
				opts.Fault = func(step string) error {
					if strings.HasPrefix(step, "delete-parent:") {
						parents++
						step = fmt.Sprintf("parent-%d", parents)
					}
					if step != boundary {
						return nil
					}
					hit = true
					if cancelled {
						cancel()
						return nil
					}
					return errors.New("crash")
				}
				if _, err := Collect(path, opts, 100); err == nil || !hit {
					t.Fatalf("boundary not stopped: hit=%v err=%v", hit, err)
				}
				opts.Context = context.Background()
				opts.Fault = nil
				if _, err := Collect(path, opts, 100); err != nil {
					t.Fatal(err)
				}
				s, err := load(path)
				if err != nil || s.Deleting != nil {
					t.Fatalf("intent not retired: %+v %v", s.Deleting, err)
				}
				if _, err := os.Stat(p); !os.IsNotExist(err) {
					t.Fatalf("payload survived: %v", err)
				}
				if got, err := os.ReadFile(path); err != nil || string(got) != "new" {
					t.Fatalf("current changed: %q %v", got, err)
				}
				if _, err := Collect(path, opts, 100); err != nil {
					t.Fatalf("repeat replay: %v", err)
				}
			})
		}
	}
}

func TestDeletionReplayRejectsReplacement(t *testing.T) {
	for _, kind := range []string{"payload", "metadata", "ancestor-symlink", "ancestor-file", "unknown-sibling"} {
		t.Run(kind, func(t *testing.T) {
			path, p, opts := deletionIntentFixture(t)
			switch kind {
			case "payload":
				if err := os.WriteFile(p, []byte("replacement"), 0600); err != nil {
					t.Fatal(err)
				}
			case "metadata":
				if err := os.WriteFile(p+".json", []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "ancestor-symlink", "ancestor-file":
				dir := filepath.Dir(p)
				if err := os.Rename(dir, dir+"-saved"); err != nil {
					t.Fatal(err)
				}
				if kind == "ancestor-symlink" {
					if err := os.Symlink(dir+"-saved", dir); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.WriteFile(dir, []byte("keep"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			case "unknown-sibling":
				if err := os.WriteFile(filepath.Join(filepath.Dir(p), "unknown"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := Collect(path, opts, 100)
			if kind == "unknown-sibling" {

				if b, e := os.ReadFile(filepath.Join(filepath.Dir(p), "unknown")); e != nil || string(b) != "keep" {
					t.Fatalf("unknown changed: %q %v", b, e)
				}
			} else if err == nil {
				t.Fatal("replacement accepted")
			}
			if kind != "unknown-sibling" {
				state, loadErr := load(path)
				if loadErr != nil || state.Deleting == nil {
					t.Fatalf("replacement lost durable intent: %v", loadErr)
				}
			}
			if kind == "ancestor-symlink" || kind == "ancestor-file" {
				saved := filepath.Join(filepath.Dir(p)+"-saved", filepath.Base(p))
				if body, e := os.ReadFile(saved); e != nil || string(body) != "old" {
					t.Fatalf("replacement ancestry touched: %q %v", body, e)
				}
			}
			if kind == "metadata" {
				if b, e := os.ReadFile(p); e != nil || string(b) != "old" {
					t.Fatalf("payload removed before metadata validation: %q %v", b, e)
				}
			}
		})
	}
}
