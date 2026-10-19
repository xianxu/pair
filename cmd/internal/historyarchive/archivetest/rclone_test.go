package archivetest

import (
	"context"
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
