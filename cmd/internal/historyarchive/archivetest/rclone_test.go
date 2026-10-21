package archivetest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	archive "github.com/xianxu/pair/cmd/internal/historyarchive"
	"os"
	"path/filepath"
	"testing"
)

func TestFakeRcloneBatchRetainsPartialAndDuplicateObjects(t *testing.T) {
	f := &FakeRclone{Objects: map[string][]FakeRemoteObject{"old.gpg": {{ID: "1", Data: []byte("old")}, {ID: "2", Data: []byte("other")}}}, FailAfterObjects: 1}
	r := archive.Rclone{Runner: f}
	pin, err := r.ResolveDestination(context.Background(), archive.DriveDestination{Remote: "pair-backup", Folder: "backups/pair-history"})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := r.ObserveFlatInventory(context.Background(), pin)
	if err != nil || len(inventory["old.gpg"]) != 2 {
		t.Fatalf("inventory %v %v", inventory, err)
	}
	stage := t.TempDir()
	for _, name := range []string{"a.gpg", "b.gpg"} {
		if err := os.WriteFile(filepath.Join(stage, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.CopyBatch(context.Background(), pin, stage); err == nil {
		t.Fatal("missing partial batch fault")
	}
	if len(f.Objects["a.gpg"]) != 1 || len(f.Objects["b.gpg"]) != 0 {
		t.Fatal("wrong retained partial batch")
	}
	f.FailAfterObjects = 0
	if err := r.CopyBatch(context.Background(), pin, stage); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(list, []byte("a.gpg\nb.gpg\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hashes, err := r.DownloadSHA256Batch(context.Background(), pin, list)
	if err != nil || len(hashes) != 2 {
		t.Fatalf("hashes %v %v", hashes, err)
	}
	if f.Invocations != 5 || f.Lists != 1 || f.CopyBatches != 2 || f.HashBatches != 1 || f.Copies != 2 || f.Downloads != 2 {
		t.Fatalf("counts %+v", f)
	}
	f.AfterCopyError = os.ErrPermission
	if err := r.CopyBatch(context.Background(), pin, stage); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("late error %v", err)
	}
}

func TestFakeRclonePinnedMkdirRetainsHierarchy(t *testing.T) {
	f := &FakeRclone{}
	ctx := context.Background()
	if _, err := f.Run(ctx, []string{"mkdir", "pair-backup:2026", "--drive-root-folder-id", "folder-1"}, 1024); err != nil {
		t.Fatal(err)
	}
	raw, err := f.Run(ctx, []string{"lsjson", "pair-backup:", "--drive-root-folder-id", "folder-1"}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	var entries []archive.RemoteEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "2026" || !entries[0].IsDir || entries[0].ID == "" {
		t.Fatalf("year inventory: %s", raw)
	}
	yearID := entries[0].ID
	if _, err := f.Run(ctx, []string{"mkdir", "pair-backup:10", "--drive-root-folder-id", yearID}, 1024); err != nil {
		t.Fatal(err)
	}
	raw, err = f.Run(ctx, []string{"lsjson", "pair-backup:", "--drive-root-folder-id", yearID}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "10" || !entries[0].IsDir || entries[0].ID == yearID {
		t.Fatalf("month inventory: %s", raw)
	}
}

func TestFakeRcloneFolderAmbiguityAndUncertainMkdir(t *testing.T) {
	ctx := context.Background()
	late := errors.New("mkdir acknowledgement lost")
	f := &FakeRclone{FolderID: "configured-root", AfterMkdirError: late,
		Objects: map[string][]FakeRemoteObject{"file": {{ID: "file-1", Data: []byte("file")}}},
		ChildFolders: map[string]map[string][]FakeRemoteObject{"configured-root": {
			"duplicate":  {{ID: "dir-a", IsDir: true}, {ID: "dir-b", IsDir: true}},
			"wrong-type": {{ID: "file-2", Data: []byte("file")}},
		}},
	}
	mkdir := func(name string) error {
		_, err := f.Run(ctx, []string{"mkdir", "remote:" + name, "--drive-root-folder-id", "configured-root"}, 1024)
		return err
	}
	if err := mkdir("2026"); !errors.Is(err, late) {
		t.Fatalf("late mkdir: %v", err)
	}
	first := f.ChildFolders["configured-root"]["2026"][0].ID
	f.AfterMkdirError = nil
	if err := mkdir("2026"); err != nil {
		t.Fatal(err)
	}
	if got := f.ChildFolders["configured-root"]["2026"]; len(got) != 1 || got[0].ID != first {
		t.Fatalf("retry lost retained identity: %+v", got)
	}
	for _, name := range []string{"file", "wrong-type", "../escape", "2026/10"} {
		if err := mkdir(name); err == nil {
			t.Fatalf("mkdir accepted %q", name)
		}
	}
	raw, err := f.Run(ctx, []string{"lsjson", "remote:", "--drive-root-folder-id", "configured-root"}, 4096)
	if err != nil {
		t.Fatal(err)
	}
	inv, err := archive.DecodeFlatInventory(raw)
	if err != nil || len(inv["duplicate"]) != 2 || inv["wrong-type"][0].IsDir {
		t.Fatalf("lost ambiguity: %s %v", raw, err)
	}
	if f.FolderLists["configured-root"] != 1 || f.Lists != 1 || f.Mkdirs != 2 {
		t.Fatalf("counts: %+v", f)
	}
	raw, err = f.Run(ctx, []string{"lsjson", "remote:", "--dirs-only", "--drive-root-folder-id", "configured-root"}, 4096)
	if err != nil {
		t.Fatal(err)
	}
	inv, err = archive.DecodeFlatInventory(raw)
	if err != nil || len(inv["duplicate"]) != 2 || len(inv["file"]) != 0 || len(inv["wrong-type"]) != 0 {
		t.Fatalf("dirs-only: %s %v", raw, err)
	}
}

func TestFakeRclonePinnedTransfersRetainPartialBytes(t *testing.T) {
	ctx := context.Background()
	name := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.json"
	f := &FakeRclone{Objects: map[string][]FakeRemoteObject{name: {{ID: "root-object", Data: []byte("root bytes")}}}, PartialCopyBytes: 3, AfterCopyError: os.ErrPermission, FolderObjects: map[string]map[string][]FakeRemoteObject{}}
	r := archive.Rclone{Runner: f}
	pin := archive.PinnedDrive{Remote: "pair-backup", Root: "pair-backup:", FolderID: "month-id"}
	stage := t.TempDir()
	if err := os.WriteFile(filepath.Join(stage, name), []byte("month bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.CopyBatch(ctx, pin, stage); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("late copy: %v", err)
	}
	retained := f.FolderObjects["month-id"][name][0]
	if string(retained.Data) != "mon" || string(f.Objects[name][0].Data) != "root bytes" {
		t.Fatalf("wrong folder bytes: %+v", f)
	}
	f.PartialCopyBytes, f.AfterCopyError = 0, nil
	if err := r.CopyBatch(ctx, pin, stage); err != nil {
		t.Fatal(err)
	}
	if got := f.FolderObjects["month-id"][name][0]; got.ID != retained.ID || string(got.Data) != "mon" {
		t.Fatalf("immutable retry changed bytes: %+v", got)
	}
	list := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(list, []byte(name+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	hashes, err := r.DownloadSHA256Batch(ctx, pin, list)
	sum := sha256.Sum256([]byte("mon"))
	if err != nil || hashes[name] != hex.EncodeToString(sum[:]) {
		t.Fatalf("pinned hash: %v %v", hashes, err)
	}
	destination := t.TempDir()
	if err := r.DownloadBatch(ctx, pin, []string{name}, destination, filepath.Join(t.TempDir(), "filter"), 100, 100); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, name))
	if err != nil || string(data) != "mon" {
		t.Fatalf("pinned download %q %v", data, err)
	}
	if f.Copies != 1 || f.HashBatches != 1 || f.DownloadBatches != 1 {
		t.Fatalf("counts: %+v", f)
	}
}

func TestFakeRcloneFolderHookMutatesAndCancelsBeforeEffect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	legacyCalls, folderCalls := 0, 0
	f := &FakeRclone{}
	f.Before = func(_ context.Context, operation, name string) error {
		legacyCalls++
		if operation != "mkdir" || name != "" {
			t.Fatalf("legacy hook changed: %q %q", operation, name)
		}
		return nil
	}
	f.BeforeFolder = func(_ context.Context, operation, folderID string) error {
		folderCalls++
		if operation != "mkdir" || folderID != "year-id" {
			t.Fatalf("folder hook: %s %s", operation, folderID)
		}
		cancel()
		return nil
	}
	_, err := f.Run(ctx, []string{"mkdir", "remote:10", "--drive-root-folder-id", "year-id"}, 1024)
	if !errors.Is(err, context.Canceled) || f.Mkdirs != 0 || len(f.ChildFolders) != 0 || legacyCalls != 1 || folderCalls != 1 {
		t.Fatalf("canceled effect: %+v %v", f, err)
	}
}
