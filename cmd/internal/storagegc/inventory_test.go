package storagegc

import (
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryExactOwnersAndNoPayloadReads(t *testing.T) {
	c, _ := coordinatorFixture(t)
	for _, scope := range []string{"one", "two"} {
		dir := filepath.Join(c.Root, "repos", scope)
		os.MkdirAll(dir, 0700)
		os.WriteFile(filepath.Join(dir, "draft-tag.md"), []byte("secret"), 0000)
		os.WriteFile(filepath.Join(dir, "scrollback-tag-codex.raw"), []byte("terminal"), 0000)
	}
	got, err := InventoryRoot(c.Root, nil, []string{"codex"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Groups) != 2 || len(got.Unknown) != 0 || !got.Complete {
		t.Fatalf("%+v", got)
	}
	for _, g := range got.Groups {
		if len(g.Members) != 2 {
			t.Fatalf("%+v", g)
		}
	}
}
func TestInventoryUnknownChildrenAndSymlinksBlock(t *testing.T) {
	c, _ := coordinatorFixture(t)
	os.WriteFile(filepath.Join(c.Root, "draft-tag.md"), nil, 0600)
	os.Mkdir(filepath.Join(c.Root, "queue-tag"), 0700)
	os.WriteFile(filepath.Join(c.Root, "queue-tag", "unexpected"), nil, 0600)
	os.Symlink(t.TempDir(), filepath.Join(c.Root, "repos"))
	got, err := InventoryRoot(c.Root, nil, []string{"codex"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Unknown) == 0 || len(got.Groups) != 1 || len(got.Groups[0].Blockers) == 0 {
		t.Fatalf("%+v", got)
	}
}
func TestInventoryBudgetBlocksIncompleteScan(t *testing.T) {
	c, _ := coordinatorFixture(t)
	os.WriteFile(filepath.Join(c.Root, "draft-tag.md"), nil, 0600)
	got, err := InventoryRoot(c.Root, []artifactpath.StorageOwner{}, []string{"codex"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Complete {
		t.Fatal("budget exhaustion reported complete")
	}
}

func TestInventoryDiagnosticLocksAreExactPersistentMetadata(t *testing.T) {
	for _, kind := range []string{"wrap-only", "adapt-only", "nonempty", "symlink", "arbitrary", "session"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := coordinatorFixture(t)
			name := "wrap-events-tag.jsonl.pair-diagnostics.lock"
			switch kind {
			case "adapt-only":
				name = "adapt-tag.jsonl.pair-diagnostics.lock"
			case "arbitrary":
				name = "unowned.pair-diagnostics.lock"
			case "session":
				name = "draft-tag.md.pair-diagnostics.lock"
			}
			path := filepath.Join(c.Root, name)
			if kind == "symlink" {
				if err := os.Symlink("/outside", path); err != nil {
					t.Fatal(err)
				}
			} else {
				var data []byte
				if kind == "nonempty" {
					data = []byte("do not ignore")
				}
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := InventoryRoot(c.Root, nil, []string{"codex"}, 100)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "wrap-only" || kind == "adapt-only" {
				if len(got.Unknown) != 0 || len(got.Groups) != 1 || len(got.Groups[0].Members) != 0 || len(got.Groups[0].Blockers) != 0 {
					t.Fatalf("valid lock blocks inventory: %+v", got)
				}
			} else if len(got.Unknown) == 0 {
				t.Fatalf("unsafe lock ignored %+v", got)
			}
		})
	}
}

func TestInventoryKnownOwnerWithoutPayloadNamespace(t *testing.T) {
	for _, kind := range []string{"absent-repos", "absent-scope", "scope-file", "scope-symlink", "repos-symlink"} {
		t.Run(kind, func(t *testing.T) {
			c, _ := coordinatorFixture(t)
			owner, err := artifactpath.NewStorageOwner(c.Root, "816fc349d3faebf8", "tag")
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "absent-scope":
				err = os.Mkdir(filepath.Join(c.Root, "repos"), 0700)
			case "scope-file":
				err = os.Mkdir(filepath.Join(c.Root, "repos"), 0700)
				if err == nil {
					err = os.WriteFile(owner.Directory(), nil, 0600)
				}
			case "scope-symlink":
				err = os.Mkdir(filepath.Join(c.Root, "repos"), 0700)
				if err == nil {
					err = os.Symlink(t.TempDir(), owner.Directory())
				}
			case "repos-symlink":
				err = os.Symlink(t.TempDir(), filepath.Join(c.Root, "repos"))
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := InventoryRoot(c.Root, []artifactpath.StorageOwner{owner}, []string{"codex"}, 100)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Groups) != 1 || got.Groups[0].Owner != owner || len(got.Groups[0].Members) != 0 {
				t.Fatalf("known owner omitted %+v", got)
			}
			unsafe := kind == "scope-file" || kind == "scope-symlink" || kind == "repos-symlink"
			if (len(got.Groups[0].Blockers) > 0) != unsafe {
				t.Fatalf("unsafe namespace disposition %+v", got)
			}
			if !unsafe {
				if _, err := os.Lstat(owner.Directory()); !os.IsNotExist(err) {
					t.Fatal("inventory created absent namespace", err)
				}
			}
		})
	}
}

func TestInventoryRejectsInvalidKnownOwners(t *testing.T) {
	c, _ := coordinatorFixture(t)
	for _, owner := range []artifactpath.StorageOwner{
		{DataDir: t.TempDir(), RepoScope: "816fc349d3faebf8", Tag: "tag"},
		{DataDir: c.Root, RepoScope: "../escape", Tag: "tag"},
		{DataDir: c.Root, RepoScope: "816fc349d3faebf8", Tag: "../escape"},
		{DataDir: c.Root, RepoScope: "", Tag: ""},
	} {
		if _, err := InventoryRoot(c.Root, []artifactpath.StorageOwner{owner}, nil, 100); err == nil {
			t.Fatalf("invalid known owner accepted %+v", owner)
		}
	}
}
