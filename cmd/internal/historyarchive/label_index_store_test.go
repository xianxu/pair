package historyarchive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
)

func indexStoreFixture(t *testing.T, name string) LabelIndex {
	t.Helper()
	r := catalogFixture(t, name)
	return LabelIndex{Version: Version, Records: map[string]LabelIndexRecord{r.Label.ID: {Label: r.Label, Locations: map[string]BundleLocation{}}}}
}
func assertStoredIndex(t *testing.T, p string, want LabelIndex) {
	t.Helper()
	got, err := ReadLabelIndex(p + ".index.json")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(want)
	if string(a) != string(b) {
		t.Fatalf("snapshot mismatch: got %s want %s", a, b)
	}
}

func TestLabelIndexStoreAtomicSnapshots(t *testing.T) {
	p := catalogPath(t)
	store, err := OpenLabelIndexStore(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := store.Read()
	if err != nil || empty.Version != Version || empty.Records == nil || len(empty.Records) != 0 {
		t.Fatalf("initial read: %+v %v", empty, err)
	}
	if store.StageDir == nil || store.StagePath != p+".index-staging" {
		t.Fatal("missing owned stage")
	}
	for _, name := range []string{"one", "two"} {
		i := indexStoreFixture(t, name)
		if err := store.Commit(i); err != nil {
			t.Fatal(err)
		}
		assertStoredIndex(t, p, i)
		info, err := os.Stat(p + ".index.json")
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("snapshot mode: %v %v", info, err)
		}
		if _, err := os.Stat(filepath.Join(store.StagePath, "output")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("output survived publication: %v", err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p + ".index-staging"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stage survived close: %v", err)
	}
	if _, err := os.Stat(p + ".index.lock"); err != nil {
		t.Fatalf("lifetime lock should remain stable: %v", err)
	}
}

func TestLabelIndexStoreFaultMatrix(t *testing.T) {
	for _, phase := range []string{"write", "sync", "rename", "dirsync"} {
		t.Run(phase, func(t *testing.T) {
			p := catalogPath(t)
			store, err := OpenLabelIndexStore(context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			old, next := indexStoreFixture(t, "old"), indexStoreFixture(t, "new")
			if err := store.Commit(old); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(p + ".index.json")
			if err != nil {
				t.Fatal(err)
			}
			store.fault = func(at string) error {
				if at == phase {
					return os.ErrPermission
				}
				return nil
			}
			err = store.Commit(next)
			if err == nil {
				t.Fatal("injected failure reported success")
			}
			if phase == "dirsync" {
				if commitoutcome.Of(err) != commitoutcome.Indeterminate {
					t.Fatalf("lost publication uncertainty: %v", err)
				}
				assertStoredIndex(t, p, next)
			} else {
				if commitoutcome.Of(err) != commitoutcome.NotAuthoritative {
					t.Fatalf("prepublication failure uncertain: %v", err)
				}
				assertStoredIndex(t, p, old)
				after, readErr := os.ReadFile(p + ".index.json")
				if readErr != nil || !bytes.Equal(before, after) {
					t.Fatal("prepublication failure changed prior bytes")
				}
			}
			if _, err := os.Stat(filepath.Join(store.StagePath, "output")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed output survived: %v", err)
			}
			store.fault = nil
			// Retry must complete the intended snapshot even after uncertain publication.
			if err := store.Commit(next); err != nil {
				t.Fatal(err)
			}
			assertStoredIndex(t, p, next)
		})
	}
}

func TestLabelIndexStoreLifetimeLockAndCancellation(t *testing.T) {
	p := catalogPath(t)
	ctx, cancel := context.WithCancel(context.Background())
	store, err := OpenLabelIndexStore(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := OpenLabelIndexStore(context.Background(), p); err == nil {
		other.Close()
		t.Fatal("concurrent index writer admitted")
	}
	old := indexStoreFixture(t, "old")
	if err := store.Commit(old); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := store.Commit(indexStoreFixture(t, "new")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled commit: %v", err)
	}
	assertStoredIndex(t, p, old)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	other, err := OpenLabelIndexStore(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
	cancelled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	absent := filepath.Join(filepath.Dir(p), "missing", "catalog")
	if s, err := OpenLabelIndexStore(cancelled, absent); !errors.Is(err, context.Canceled) {
		if s != nil {
			s.Close()
		}
		t.Fatalf("cancelled open: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(absent)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled open created directories")
	}
}

func TestReadLabelIndexDoesNotCreateOrMutate(t *testing.T) {
	root := filepath.Dir(catalogPath(t))
	absent := filepath.Join(root, "absent", "snapshot.json")
	before := catalogReadSnapshot(t, root)
	if _, err := ReadLabelIndex(absent); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("absent read: %v", err)
	}
	if after := catalogReadSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("absent read mutated tree")
	}
	p := filepath.Join(root, "snapshot.json")
	raw, err := json.Marshal(indexStoreFixture(t, "one"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0400); err != nil {
		t.Fatal(err)
	}
	before = catalogReadSnapshot(t, root)
	if _, err := ReadLabelIndex(p); err != nil {
		t.Fatal(err)
	}
	if after := catalogReadSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("read changed snapshot")
	}
	if err := os.Symlink(p, p+".alias"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLabelIndex(p + ".alias"); err == nil {
		t.Fatal("reader followed symlink")
	}
}

func TestLabelIndexStorePreservesUnknownStaging(t *testing.T) {
	p := catalogPath(t)
	stage := p + ".index-staging"
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(stage, "mine")
	if err := os.WriteFile(sentinel, []byte("operator data"), 0600); err != nil {
		t.Fatal(err)
	}
	if store, err := OpenLabelIndexStore(context.Background(), p); err == nil {
		store.Close()
		t.Fatal("accepted unowned staging")
	}
	raw, err := os.ReadFile(sentinel)
	if err != nil || string(raw) != "operator data" {
		t.Fatal("unowned staging changed")
	}
}

func TestLabelIndexStoreBoundsAndMalformedSnapshot(t *testing.T) {
	p := catalogPath(t)
	store, err := OpenLabelIndexStore(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	valid := indexStoreFixture(t, "valid")
	if err := store.Commit(valid); err != nil {
		t.Fatal(err)
	}
	if err := store.Commit(LabelIndex{Version: Version}); err == nil {
		t.Fatal("accepted missing records")
	}
	assertStoredIndex(t, p, valid)
	// Sparse oversized input must be refused before decoding or loading it all.
	if err := os.Truncate(p+".index.json", MaxIndexBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(); err == nil {
		t.Fatal("accepted oversized reader snapshot")
	}
	if err := os.WriteFile(p+".index.json", []byte(`{"version":1,"records":null}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(); err == nil {
		t.Fatal("accepted malformed snapshot")
	}
}

func TestLabelIndexStoreWriterByteBoundPreservesOld(t *testing.T) {
	p := catalogPath(t)
	store, err := OpenLabelIndexStore(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	old := indexStoreFixture(t, "old")
	if err := store.Commit(old); err != nil {
		t.Fatal(err)
	}
	oversized := indexStoreFixture(t, "new")
	raw := bytes.Repeat([]byte{'x'}, MaxIndexBytes+2)
	raw[0], raw[len(raw)-1] = '"', '"'
	oversized.Extensions = map[string]json.RawMessage{"future": raw}
	if err := store.Commit(oversized); err == nil {
		t.Fatal("accepted oversized writer snapshot")
	}
	assertStoredIndex(t, p, old)
	if _, err := os.Stat(filepath.Join(store.StagePath, "output")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("oversized output staged: %v", err)
	}
}

func TestLabelIndexStoreCancellationBeforePublication(t *testing.T) {
	p := catalogPath(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := OpenLabelIndexStore(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	old := indexStoreFixture(t, "old")
	if err := store.Commit(old); err != nil {
		t.Fatal(err)
	}
	store.fault = func(phase string) error {
		if phase == "sync" {
			cancel()
		}
		return nil
	}
	if err := store.Commit(indexStoreFixture(t, "new")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled staged commit: %v", err)
	}
	assertStoredIndex(t, p, old)
	if _, err := os.Stat(filepath.Join(store.StagePath, "output")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled output survived: %v", err)
	}
}

func TestLabelIndexStoreCloseReportsRefusedCleanup(t *testing.T) {
	p := catalogPath(t)
	store, err := OpenLabelIndexStore(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(store.StagePath, "operator-file")
	if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err == nil {
		t.Fatal("cleanup refusal reported success")
	}
	if raw, err := os.ReadFile(sentinel); err != nil || string(raw) != "keep" {
		t.Fatal("cleanup refusal changed foreign file")
	}
}

func TestLabelIndexStoreRecoversInterruptedStage(t *testing.T) {
	if p := os.Getenv("PAIR_TEST_INDEX_STORE_CRASH_PATH"); p != "" {
		store, err := OpenLabelIndexStore(context.Background(), p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(store.StagePath, "output"), []byte("interrupted snapshot"), 0600); err != nil {
			t.Fatal(err)
		}
		// Exit releases the kernel flock but deliberately leaves owned stage entries.
		os.Exit(0)
	}
	p := catalogPath(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestLabelIndexStoreRecoversInterruptedStage$")
	cmd.Env = append(os.Environ(), "PAIR_TEST_INDEX_STORE_CRASH_PATH="+p)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("crash fixture: %v %s", err, output)
	}
	store, err := OpenLabelIndexStore(context.Background(), p)
	if err != nil {
		t.Fatalf("interrupted stage not recovered: %v", err)
	}
	defer store.Close()
	if _, err := os.Stat(filepath.Join(store.StagePath, "output")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old interrupted output survived recovery: %v", err)
	}
	intended := indexStoreFixture(t, "recovered")
	if err := store.Commit(intended); err != nil {
		t.Fatal(err)
	}
	assertStoredIndex(t, p, intended)
}

func TestLabelIndexStoreClosePreservesPublicationOutcome(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(fmt.Sprint(uncertain), func(t *testing.T) {
			p := catalogPath(t)
			store, err := OpenLabelIndexStore(context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			want := commitoutcome.Committed
			if uncertain {
				want = commitoutcome.Indeterminate
				store.fault = func(phase string) error {
					if phase == "dirsync" {
						return os.ErrPermission
					}
					return nil
				}
			}
			intended := indexStoreFixture(t, "published")
			if err := store.Commit(intended); commitoutcome.Of(err) != want {
				t.Fatalf("publication: %v", err)
			}
			if err := os.WriteFile(filepath.Join(store.StagePath, "operator-file"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err == nil || commitoutcome.Of(err) != want {
				t.Fatalf("cleanup lost publication outcome: %v", err)
			}
			assertStoredIndex(t, p, intended)
		})
	}
}

func TestLabelIndexStoreRefusesSymlinkAuthority(t *testing.T) {
	for _, component := range []string{"parent", "lock", "stage", "snapshot"} {
		t.Run(component, func(t *testing.T) {
			p := catalogPath(t)
			root := filepath.Dir(p)
			target := filepath.Join(root, "operator-data")
			if component == "parent" || component == "stage" {
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				raw, err := json.Marshal(indexStoreFixture(t, "sentinel"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			link := p + ".index.lock"
			switch component {
			case "parent":
				link = filepath.Join(root, "alias")
			case "stage":
				link = p + ".index-staging"
			case "snapshot":
				link = p + ".index.json"
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			if component == "parent" {
				p = filepath.Join(link, "nested", "catalog")
			}
			before := catalogReadSnapshot(t, target)
			store, err := OpenLabelIndexStore(context.Background(), p)
			if component == "snapshot" {
				if err != nil {
					t.Fatal(err)
				}
				if err := store.Commit(indexStoreFixture(t, "new")); err == nil {
					t.Fatal("commit replaced snapshot symlink")
				}
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				store.Close()
				t.Fatal("opened symlink authority")
			}
			if after := catalogReadSnapshot(t, target); !reflect.DeepEqual(before, after) {
				t.Fatal("followed or changed authority symlink target")
			}
			info, err := os.Lstat(link)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("authority symlink replaced: %v", err)
			}
		})
	}
}
