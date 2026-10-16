package diagnosticlog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeletionReplayAfterParentCleanupPreservesIntentUntilBackup(t *testing.T) {
	path, p, opts := deletionIntentFixture(t)
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p + ".json"); err != nil {
		t.Fatal(err)
	}
	if err := removeEmptyParents(filepath.Dir(p), directory(path), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := Collect(path, opts, 100); !errors.Is(err, ErrBackupRequired) {
		t.Fatal(err)
	}
	state, err := load(path)
	if err != nil || state.Deleting == nil {
		t.Fatal("completed historical effect lost recovery authority", err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "new" {
		t.Fatal(string(body), err)
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

func TestBackupFenceStopsHistoricalDeletionAtEveryEffectBoundary(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		for _, stage := range []string{"intent", "payload-gone", "metadata-gone", "parents-gone"} {
			t.Run(fmt.Sprintf("%s/cancel=%t", stage, cancelled), func(t *testing.T) {
				path, p, opts := deletionIntentFixture(t)
				if stage != "intent" {
					if err := os.Remove(p); err != nil {
						t.Fatal(err)
					}
				}
				if stage == "metadata-gone" || stage == "parents-gone" {
					if err := os.Remove(p + ".json"); err != nil {
						t.Fatal(err)
					}
				}
				if stage == "parents-gone" {
					if err := removeEmptyParents(filepath.Dir(p), directory(path), opts); err != nil {
						t.Fatal(err)
					}
				}
				before, err := os.ReadFile(filepath.Join(directory(path), "state.json"))
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				opts.Context = ctx
				if cancelled {
					cancel()
				}
				_, err = Collect(path, opts, 100)
				want := ErrBackupRequired
				if cancelled {
					want = context.Canceled
				}
				if !errors.Is(err, want) {
					t.Fatal(err)
				}
				after, err := os.ReadFile(filepath.Join(directory(path), "state.json"))
				if err != nil || string(before) != string(after) {
					t.Fatal("deletion authority changed", err)
				}
				if stage == "intent" {
					body, err := os.ReadFile(p)
					if err != nil || string(body) != "old" {
						t.Fatal(string(body), err)
					}
				}
				if stage == "intent" || stage == "payload-gone" {
					if _, err := os.Stat(p + ".json"); err != nil {
						t.Fatal("remaining metadata evicted", err)
					}
				}
				if body, err := os.ReadFile(path); err != nil || string(body) != "new" {
					t.Fatal(string(body), err)
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
