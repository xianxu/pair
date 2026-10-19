package archivetest

import (
	"context"
	archive "github.com/xianxu/pair/cmd/internal/historyarchive"
	"os"
	"path/filepath"
	"testing"
)

func TestFakeRcloneRetainsLateCopyAndDuplicateObjects(t *testing.T) {
	f := &FakeRclone{Objects: map[string][]FakeRemoteObject{"old.gpg": {{ID: "1", Data: []byte("old")}, {ID: "2", Data: []byte("other")}}}}
	r := archive.Rclone{Runner: f}
	pin, err := r.ResolveDestination(context.Background(), archive.DriveDestination{Remote: "pair-backup", Folder: "backups/pair-history"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ObserveFlatInventory(context.Background(), pin)
	if err != nil || len(got["old.gpg"]) != 2 {
		t.Fatalf("inventory: %v %v", got, err)
	}
	name := filepath.Join(t.TempDir(), "new.gpg")
	if err := os.WriteFile(name, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	f.AfterCopyError = os.ErrPermission
	if err := r.Copy(context.Background(), pin, name, "new.gpg"); err == nil {
		t.Fatal("late fault hidden")
	}
	if len(f.Objects["new.gpg"]) != 1 {
		t.Fatal("late result lost")
	}
	if f.Lists != 1 || f.Copies != 1 {
		t.Fatalf("counts: %+v", f)
	}
}
