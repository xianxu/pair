package historyarchive_test

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchcoretest"
	archive "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Inventory patterns select a matching file or a whole matching directory tree.
// The future collector owns traversal and admission; this oracle only checks
// whether a produced file or one of its directory ancestors has a declaration.
func inventorySelects(t *testing.T, kind, root, file string) bool {
	t.Helper()
	inv, err := archive.Defaults()
	if err != nil {
		t.Fatal(err)
	}
	rule, err := inv.Resolve(kind)
	if err != nil {
		t.Error(err)
		return false
	}
	relative, err := filepath.Rel(root, file)
	if err != nil {
		t.Fatal(err)
	}
	for candidate := filepath.ToSlash(relative); candidate != "."; candidate = path.Dir(candidate) {
		for _, pattern := range rule.Patterns {
			match, err := path.Match(pattern, candidate)
			if err != nil {
				t.Fatal(err)
			}
			if match {
				return true
			}
		}
	}
	return false
}
func requireInventoryPath(t *testing.T, kind, root, file string) {
	t.Helper()
	if !inventorySelects(t, kind, root, file) {
		t.Errorf("%s does not declare produced path %s relative to %s", kind, file, root)
	}
}
func TestInventoryPairProducerPathsAllNamespaces(t *testing.T) {
	root := t.TempDir()
	scoped, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := artifactpath.ResolveScoped(root, "couch-0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []artifactpath.Paths{scoped, legacy} {
		parked, err := p.ParkedScrollbackArtifacts("20261015T120000Z")
		if err != nil {
			t.Fatal(err)
		}
		scroll, err := p.ScrollbackArtifacts("codex")
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			kind  string
			files []string
		}{{"pair-terminal-events", []string{p.WrapEvents()}}, {"pair-parked-captures", []string{p.Parked(), parked.Raw, parked.Events, parked.Metadata}}, {"pair-scrollback", []string{scroll.Raw, scroll.Events, scroll.ANSI, scroll.Viewport}}, {"couch-sessions", []string{p.SessionBindings()}}, {"couch-ledgers", []string{p.Ledger()}}, {"pair-adapt-diagnostics", []string{p.AdaptLog()}}, {"pair-fullscreen-diagnostics", []string{p.FullscreenDiagnostics()}}} {
			for _, file := range tc.files {
				requireInventoryPath(t, tc.kind, root, file)
			}
		}
		for _, tc := range []struct{ kind, file string }{{"pair-scrollback", scroll.OpenLock}, {"pair-fullscreen-diagnostics", p.FullscreenLock()}, {"pair-fullscreen-diagnostics", p.FullscreenReturn()}} {
			if inventorySelects(t, tc.kind, root, tc.file) {
				t.Errorf("%s selects transient control %s", tc.kind, tc.file)
			}
		}
	}
}
func TestInventoryCouchProducedGlobalAndSlotHistory(t *testing.T) {
	fleet, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	storeRoot := filepath.Join(fleet, "custom-couch")
	ns, err := couchcore.ResolveCouchNamespace(storeRoot, fleet)
	if err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewThreadStore(ns)
	primary := filepath.Join(fleet, "sample")
	layout := couchcore.NewSlotLayout(primary, filepath.Join(primary, ".git"), 1)
	if err := os.MkdirAll(layout.Host(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(primary, 0700); err != nil {
		t.Fatal(err)
	}
	makeRecord := func(root, tag string) couchcore.ThreadRecord {
		scope, err := launcher.ResolveRepoScope(root)
		if err != nil {
			t.Fatal(err)
		}
		return couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: couchcore.ThreadTag(tag)}, StartingPath: root, WorkingPath: root, CreatedAt: time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC), Revision: 1}
	}
	global, err := store.CreateThread(makeRecord(primary, "couch-0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	globalPath, err := store.RecordPath(global.Address)
	if err != nil {
		t.Fatal(err)
	}
	requireInventoryPath(t, "couch-threads", storeRoot, globalPath)
	if err := store.ArchiveThread(global.Address, couchcoretest.RetireProof(t, store, global.Address)); err != nil {
		t.Fatal(err)
	}
	zero := 0
	address := "sample:0"
	resting := "main"
	repository := couchcore.SlotRepository{Identity: couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "sample", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: fleet, EnvironmentRoot: fleet, WorktreeRoot: primary, Kind: "primary", Slot: &zero, Address: &address, RestingBranch: &resting}, Slots: []couchcore.SlotCandidate{{Identity: couchcore.SlotIdentity{Repo: "sample", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, EnvironmentRoot: layout.Env(), WorktreeRoot: layout.Host(), Number: 1}, Verified: true}}}
	if err := store.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local, err := store.CreateThread(makeRecord(layout.Host(), "couch-1111111111111111"))
	if err != nil {
		t.Fatal(err)
	}
	localPath, err := store.RecordPath(local.Address)
	if err != nil {
		t.Fatal(err)
	}
	requireInventoryPath(t, "couch-slot-threads", layout.Store(), localPath)
	if err := store.ArchiveThread(local.Address, couchcoretest.RetireProof(t, store, local.Address)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ kind, root, walk string }{{"couch-threads", storeRoot, filepath.Join(storeRoot, "threadstore")}, {"couch-slot-threads", layout.Store(), layout.Store()}} {
		err := filepath.WalkDir(tc.walk, func(file string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && filepath.Ext(file) == ".json" {
				requireInventoryPath(t, tc.kind, tc.root, file)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestInventoryWebcastProducedSubtree(t *testing.T) {
	root := t.TempDir()
	recordings := filepath.Join(root, "webcast", "recordings")
	if err := os.MkdirAll(recordings, 0700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(recordings, "capture")
	writer, err := recording.NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	hash, err := writer.PutAsset([]byte("asset body"))
	if err != nil {
		t.Fatal(err)
	}
	requireInventoryPath(t, "webcast-recordings", root, filepath.Join(dir, recording.AssetName(hash)))
	if err := writer.Put(&recording.Stop{Header: recording.Header{T: 1}, Why: recording.StopEnd}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	err = filepath.WalkDir(dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			requireInventoryPath(t, "webcast-recordings", root, file)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{filepath.Join(root, "webcast", "origin-privacy.json"), filepath.Join(root, "webcast", "chrome-profile", "Cookies")} {
		if inventorySelects(t, "webcast-recordings", root, file) {
			t.Errorf("selected nonrecording %s", file)
		}
	}
}
func TestInventoryCouchExcludesControlState(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		kind  string
		files []string
	}{{"couch-threads", []string{"threadstore/store.lock", "threadstore/.thread-store-publication/temporary", "registry.json", "desc/actor"}}, {"couch-slot-threads", []string{"store.lock", ".thread-store-publication/temporary", "saved-work/check-out/file", "manifest.json"}}} {
		for _, file := range tc.files {
			if inventorySelects(t, tc.kind, root, filepath.Join(root, filepath.FromSlash(file))) {
				t.Errorf("%s selected excluded control %s", tc.kind, file)
			}
		}
	}
}
