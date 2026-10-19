package historyarchive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func existingCatalogFixture(t *testing.T) (string, CatalogRecord) {
	t.Helper()
	p := catalogPath(t)
	r := catalogFixture(t, "existing")
	c, err := OpenCatalog(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Append(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	return p, r
}

type catalogReadState struct {
	Bytes      string
	Mode       os.FileMode
	ModTime    time.Time
	ChangeTime string
}

func catalogReadSnapshot(t *testing.T, root string) map[string]catalogReadState {
	t.Helper()
	result := map[string]catalogReadState{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		var stat unix.Stat_t
		if err := unix.Lstat(path, &stat); err != nil {
			return err
		}
		state := catalogReadState{Mode: info.Mode(), ModTime: info.ModTime(), ChangeTime: fmt.Sprint(reflect.ValueOf(stat).FieldByName("Ctim"), reflect.ValueOf(stat).FieldByName("Ctimespec"))}
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			state.Bytes = string(raw)
		}
		result[path] = state
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestOpenExistingCatalogReadsWithoutMutation(t *testing.T) {
	p, r := existingCatalogFixture(t)
	before := catalogReadSnapshot(t, filepath.Dir(p))
	c, err := OpenExistingCatalog(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	c.fault = func(string) error { t.Error("read attempted catalog sync"); return os.ErrPermission }
	flags, err := unix.FcntlInt(c.lock.Fd(), unix.F_GETFL, 0)
	if err != nil || flags&unix.O_ACCMODE != unix.O_RDONLY {
		t.Fatalf("lock must be read-only: flags=%d err=%v", flags, err)
	}
	got, found, err := c.Find(context.Background(), r.Label.ID)
	if err != nil || !found || !sameCatalogRecord(got, r) {
		t.Fatalf("find: found=%v err=%v", found, err)
	}
	var all []CatalogRecord
	if err := c.Each(context.Background(), func(r CatalogRecord) error { all = append(all, r); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || !sameCatalogRecord(all[0], r) {
		t.Fatalf("each: %v", all)
	}
	if err := c.Append(context.Background(), catalogFixture(t, "new")); err == nil {
		t.Fatal("read-only catalog admitted append")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if after := catalogReadSnapshot(t, filepath.Dir(p)); !reflect.DeepEqual(before, after) {
		t.Fatalf("read changed catalog tree: before=%v after=%v", before, after)
	}
}

func TestOpenExistingCatalogMissingDoesNotCreate(t *testing.T) {
	for _, layout := range []string{"parent", "catalog", "lock", "shard"} {
		t.Run(layout, func(t *testing.T) {
			root := filepath.Dir(catalogPath(t))
			p := filepath.Join(root, "missing", "history.jsonl")
			if layout != "parent" {
				if err := os.Mkdir(filepath.Dir(p), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if layout == "lock" {
				if err := os.WriteFile(p, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if layout == "shard" {
				if err := os.WriteFile(p+".lock", nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := catalogReadSnapshot(t, root)
			c, err := OpenExistingCatalog(context.Background(), p)
			if c != nil {
				c.Close()
				t.Fatal("opened missing catalog")
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("want not-exist, got %v", err)
			}
			if after := catalogReadSnapshot(t, root); !reflect.DeepEqual(before, after) {
				t.Fatal("missing read created or changed metadata")
			}
		})
	}
}

func TestOpenExistingCatalogRefusesTornTailWithoutRepair(t *testing.T) {
	p, _ := existingCatalogFixture(t)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"torn":`); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	before := catalogReadSnapshot(t, filepath.Dir(p))
	c, err := OpenExistingCatalog(context.Background(), p)
	if c != nil {
		c.Close()
	}
	if err == nil {
		t.Fatal("accepted uncommitted catalog tail")
	}
	if after := catalogReadSnapshot(t, filepath.Dir(p)); !reflect.DeepEqual(before, after) {
		t.Fatal("read repaired torn tail")
	}
	// A refused reader must release its lock so the writer can perform recovery.
	c, err = OpenCatalog(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}

func TestOpenExistingCatalogRefusesWriterAndAllowsReaders(t *testing.T) {
	p, _ := existingCatalogFixture(t)
	writer, err := OpenCatalog(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if c, err := OpenExistingCatalog(context.Background(), p); err == nil {
		c.Close()
		t.Fatal("reader admitted during write")
	}
	writer.Close()
	first, err := OpenExistingCatalog(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := OpenExistingCatalog(context.Background(), p)
	if err != nil {
		t.Fatalf("second reader refused: %v", err)
	}
	defer second.Close()
	if c, err := OpenCatalog(context.Background(), p); err == nil {
		c.Close()
		t.Fatal("writer admitted during reads")
	}
}

func TestOpenExistingCatalogRefusesSymlinks(t *testing.T) {
	for _, target := range []string{"parent", "lock", "shard"} {
		t.Run(target, func(t *testing.T) {
			p, _ := existingCatalogFixture(t)
			switch target {
			case "parent":
				alias := filepath.Join(filepath.Dir(p), "alias")
				if err := os.Symlink(filepath.Dir(p), alias); err != nil {
					t.Fatal(err)
				}
				p = filepath.Join(alias, filepath.Base(p))
			default:
				name := p
				if target == "lock" {
					name += ".lock"
				}
				if err := os.Rename(name, name+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(name+".original", name); err != nil {
					t.Fatal(err)
				}
			}
			c, err := OpenExistingCatalog(context.Background(), p)
			if c != nil {
				c.Close()
			}
			if err == nil {
				t.Fatal("accepted symlink")
			}
		})
	}
}

func TestOpenExistingCatalogPreservesShardValidation(t *testing.T) {
	for _, failure := range []string{"gap", "corrupt", "oversized"} {
		t.Run(failure, func(t *testing.T) {
			p, _ := existingCatalogFixture(t)
			switch failure {
			case "gap":
				if err := os.WriteFile(p+".000002", nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "corrupt":
				if err := os.WriteFile(p, []byte("{broken}\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				if err := os.Truncate(p, catalogShardBytes+1); err != nil {
					t.Fatal(err)
				}
			}
			c, err := OpenExistingCatalog(context.Background(), p)
			if c != nil {
				c.Close()
			}
			if err == nil {
				t.Fatal("accepted invalid catalog")
			}
		})
	}
}

func TestOpenExistingCatalogReadsAllShardsOnReadOnlyTree(t *testing.T) {
	p, first := existingCatalogFixture(t)
	second := catalogFixture(t, "second")
	raw, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p+".000001", append(raw, '\n'), 0400); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{p, p + ".lock"} {
		if err := os.Chmod(name, 0400); err != nil {
			t.Fatal(err)
		}
	}
	root := filepath.Dir(p)
	if err := os.Chmod(root, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, 0700) })
	before := catalogReadSnapshot(t, root)
	c, err := OpenExistingCatalog(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, want := range []CatalogRecord{first, second} {
		got, found, err := c.Find(context.Background(), want.Label.ID)
		if err != nil || !found || !sameCatalogRecord(got, want) {
			t.Fatalf("find shard: found=%v err=%v", found, err)
		}
	}
	count := 0
	if err := c.Each(context.Background(), func(CatalogRecord) error { count++; return nil }); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("each returned %d records", count)
	}
	if after := catalogReadSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("read-only tree changed")
	}
}

func TestCatalogScanReportCountsActualRepeatedTraversals(t *testing.T) {
	p, record := existingCatalogFixture(t)
	var report catalogScanReport
	catalog, err := openCatalogMode(context.Background(), p, catalogShardBytes, true, &report)
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	if report.ShardScans != 1 || report.RecordVisits != 1 || report.DirectoryScans != 1 {
		t.Fatalf("open traversal: %+v", report)
	}
	for k := 0; k < 3; k++ {
		if _, found, err := catalog.Find(context.Background(), record.Label.ID); err != nil || !found {
			t.Fatalf("find: found=%v err=%v", found, err)
		}
	}
	if err := catalog.Each(context.Background(), func(CatalogRecord) error { return nil }); err != nil {
		t.Fatal(err)
	}
	// Counts grow for every actual scan, including repeated lookups. They are
	// independent of observation cardinality and therefore expose N rescans.
	if report.ShardScans != 5 || report.RecordVisits != 5 || report.DirectoryScans != 1 {
		t.Fatalf("repeated traversals were not counted: %+v", report)
	}
}
