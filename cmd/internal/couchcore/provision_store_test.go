package couchcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProvisionStoreStrictBoundedRecords(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "record.json")
	store := ProvisionStore{}
	value := SetupSuccess{SchemaVersion: 1, Host: "/fleet/worktree/repo-slot1/repo", Common: "/fleet/repo/.git", Admin: "/fleet/repo/.git/worktrees/repo", Slot: 1, BaselineSHA: strings.Repeat("a", 40)}
	if err := store.Write(path, value); err != nil {
		t.Fatal(err)
	}
	var got SetupSuccess
	exists, err := store.Read(path, &got)
	if err != nil || !exists || got != value {
		t.Fatalf("%+v %v %v", got, exists, err)
	}
	for _, raw := range []string{`{"schema_version":1,"schema_version":2}`, `{"extra":true}`, strings.Repeat("x", provisionRecordLimit+1), `{} {}`} {
		os.WriteFile(path, []byte(raw), 0600)
		if _, err := store.Read(path, &got); err == nil {
			t.Fatal("accepted malformed record")
		}
	}
	alias := filepath.Join(root, "alias.json")
	if err := os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(alias, &got); err == nil {
		t.Fatal("read symlink")
	}
	if err := store.Write(alias, value); err == nil {
		t.Fatal("write symlink")
	}
}
