package pairlifecycle

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPairLifecyclePackageRemainsLeaf(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"/launcher", "/couchcore", "/threadrecord", "/sessioninventory"}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			for _, suffix := range forbidden {
				if strings.Contains(spec.Path.Value, suffix) {
					t.Fatalf("%s imports forbidden higher-level package %s", entry.Name(), spec.Path.Value)
				}
			}
		}
	}
}
