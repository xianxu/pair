package gccmd

import (
	"bytes"
	"encoding/json"
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

func TestForgetMissingStoreResetsMigration(t *testing.T) {
	root, store := t.TempDir(), t.TempDir()
	var out bytes.Buffer
	run := func(args ...string) int {
		t.Helper()
		out.Reset()
		return Run(append([]string{"--root", root}, args...), func(string) string { return "" }, &out, &out)
	}
	if code := run("--complete-migration", "--store", store); code != 0 {
		t.Fatal(out.String())
	}
	if err := os.Remove(store); err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(filepath.Dir(store))
	if err != nil {
		t.Fatal(err)
	}
	store = filepath.Join(canonicalRoot, filepath.Base(store))
	if code := run("--forget-missing-store", store); code != 0 {
		t.Fatalf("missing store recovery failed: %s", out.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, ".retention", "stores.json"))
	var registry struct {
		MigrationComplete bool     `json:"migration_complete"`
		Stores            []string `json:"stores"`
	}
	decodeErr := json.Unmarshal(raw, &registry)
	if err != nil || decodeErr != nil || registry.MigrationComplete || len(registry.Stores) != 0 {
		t.Fatalf("migration not reset: %s %v", raw, err)
	}
	if code := run("--forget-missing-store", store); code == 0 {
		t.Fatal("duplicate abandonment accepted")
	}
	if code := run("--complete-migration"); code != 0 {
		t.Fatal(out.String())
	}
}

func TestForgetMissingStoreRejectsMixedOperationsBeforeMutation(t *testing.T) {
	for _, other := range [][]string{{"--apply"}, {"--complete-migration"}, {"--register-store", "/missing"}, {"--store", "/missing"}} {
		root := t.TempDir()
		var out bytes.Buffer
		args := append([]string{"--root", root, "--forget-missing-store", "/missing"}, other...)
		if Run(args, func(string) string { return "" }, &out, &out) == 0 {
			t.Fatal("mixed mutation accepted")
		}
		entries, err := os.ReadDir(root)
		if err != nil || len(entries) != 0 {
			t.Fatalf("invalid invocation mutated root: %v %v", entries, err)
		}
	}
}
