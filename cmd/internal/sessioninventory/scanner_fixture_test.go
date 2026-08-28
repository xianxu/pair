package sessioninventory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func loadNativeFixture(t *testing.T, runtime *sessioninventorytest.FakeRuntime, agent sessioninventory.Agent, rootName, fixturePath string) sessioninventory.StorageRoot {
	t.Helper()
	root := sessioninventory.StorageRoot{Agent: agent, Name: rootName, Path: fixturePath}
	runtime.AddRoot(root)
	birth := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	err := filepath.WalkDir(fixturePath, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relativePath, err := filepath.Rel(fixturePath, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		runtime.PutFile(sessioninventory.FileEntry{
			Artifact:  sessioninventory.Artifact{StorageRoot: rootName, RelativePath: filepath.ToSlash(relativePath)},
			BirthTime: &birth,
			ModTime:   &birth,
		}, content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func inventoryFromScan(result sessioninventory.ScanResult) sessioninventory.Inventory {
	inventory := sessioninventory.BuildForest(result.Facts)
	inventory.Diagnostics = append(inventory.Diagnostics, result.Diagnostics...)
	return sessioninventory.SortInventory(inventory)
}

func diagnosticPresent(diagnostics []sessioninventory.Diagnostic, code sessioninventory.DiagnosticCode) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
