package sessioninventory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogInvalidationRemovesOnlyDerivedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.json")
	os.WriteFile(path, []byte("stale"), 0600)
	other := filepath.Join(dir, "other")
	os.WriteFile(other, []byte("keep"), 0600)
	store := CatalogStore{Runtime: CatalogOSRuntime{}}
	if err := store.Invalidate(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("catalog retained")
	}
	if raw, err := os.ReadFile(other); err != nil || string(raw) != "keep" {
		t.Fatal("unrelated file changed")
	}
	if err := store.Invalidate(path); err != nil {
		t.Fatal("retry", err)
	}
}
