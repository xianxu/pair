package gccmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewDoesNotInitialize(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "draft-tag.md"), []byte("draft"), 0600)
	var out, errout bytes.Buffer
	if code := Run([]string{"--root", root, "--json"}, func(string) string { return "" }, &out, &errout); code != 0 {
		t.Fatalf("%d %s", code, &errout)
	}
	if !strings.Contains(out.String(), "migration_complete") {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".retention")); !os.IsNotExist(err) {
		t.Fatal("preview wrote metadata")
	}
}
func TestMigrationExplicitlyNamesAllStores(t *testing.T) {
	root := t.TempDir()
	store := t.TempDir()
	var out, errout bytes.Buffer
	if code := Run([]string{"--root", root, "--register-store", store}, func(string) string { return "" }, &out, &errout); code != 0 {
		t.Fatalf("%d %s", code, &errout)
	}
	if code := Run([]string{"--root", root, "--complete-migration"}, func(string) string { return "" }, &out, &errout); code == 0 {
		t.Fatal("acknowledged incomplete list")
	}
	if code := Run([]string{"--root", root, "--complete-migration", "--store", store}, func(string) string { return "" }, &out, &errout); code != 0 {
		t.Fatalf("%d %s", code, &errout)
	}
}
func TestInvalidArgumentsDoNotMutate(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	for _, args := range [][]string{{"--root", root, "extra"}, {"--root", root, "--store", "/not-a-store"}, {"--root", root, "--apply", "--complete-migration"}} {
		if code := Run(args, func(string) string { return "" }, &out, &out); code == 0 {
			t.Fatal("accepted", args)
		}
	}
	files, _ := os.ReadDir(root)
	if len(files) != 0 {
		t.Fatal("invalid arguments wrote storage")
	}
}
