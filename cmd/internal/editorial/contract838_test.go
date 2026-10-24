package editorial

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"
)

// These are frozen upstream contract inputs, not a second implementation of
// universal-review. Runtime behavior is exercised by the editorial tests.
func TestContract838PinnedExport(t *testing.T) {
	const revision = "5a17f789619d3dcaafec62ae51b350833836a3e7"
	const sourceRoot = "construct/local/universal-review/"
	const manifestDigest = "85e60265e25341b7351ddabdadd288d9fc209a254a70249b8aacf18bf410eee0"
	type filePin struct {
		Source string
		Path   string
		Bytes  int
		SHA256 string
	}
	var provenance struct {
		Repository string
		Commit     string
		Manifest   filePin
		Files      []filePin
	}
	readJSON := func(filename string, dst any) {
		t.Helper()
		raw, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("%s: %v", filename, err)
		}
	}
	readJSON("../../../workshop/evidence/000838/contract-provenance.json", &provenance)
	if provenance.Repository != "github.com/xianxu/ariadne" || provenance.Commit != revision {
		t.Fatalf("unexpected upstream provenance: repository=%q commit=%q", provenance.Repository, provenance.Commit)
	}
	if provenance.Manifest != (filePin{Source: sourceRoot + "export.json", Path: "export.json", Bytes: 19783, SHA256: manifestDigest}) {
		t.Fatalf("unexpected export manifest pin: %+v", provenance.Manifest)
	}
	root := "testdata/contract838"
	verify := func(f filePin) {
		t.Helper()
		if !fs.ValidPath(f.Path) {
			t.Fatalf("invalid exported path: %q", f.Path)
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Path)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(raw)
		if len(raw) != f.Bytes || hex.EncodeToString(digest[:]) != f.SHA256 {
			t.Errorf("%s differs from landed upstream source %s", f.Path, f.Source)
		}
	}
	verify(provenance.Manifest)
	var manifest struct {
		Schema  string
		Version int
		Members []struct {
			Role, Name, Path, Source, SHA256 string
			Bytes                            int
		}
	}
	readJSON(filepath.Join(root, "export.json"), &manifest)
	if manifest.Schema != "ariadne/universal-review-export" || manifest.Version != 4 || len(manifest.Members) != 84 {
		t.Fatalf("unexpected export contract: %s v%d, %d members", manifest.Schema, manifest.Version, len(manifest.Members))
	}
	pins := make(map[string]filePin)
	for _, f := range provenance.Files {
		if _, exists := pins[f.Path]; exists {
			t.Fatalf("duplicate provenance path %q", f.Path)
		}
		pins[f.Path] = f
	}
	expected := map[string]bool{"export.json": true}
	for _, member := range manifest.Members {
		if expected[member.Path] {
			t.Fatalf("duplicate export member %q", member.Path)
		}
		expected[member.Path] = true
		source := member.Source
		if source == "" {
			source = sourceRoot + member.Path
		}
		want := filePin{Source: source, Path: member.Path, Bytes: member.Bytes, SHA256: member.SHA256}
		if got, ok := pins[member.Path]; !ok || got != want {
			t.Fatalf("provenance for %s: got %+v, want %+v", member.Path, got, want)
		}
		delete(pins, member.Path)
		verify(want)
	}
	if len(pins) != 0 {
		t.Fatalf("provenance contains unexported members: %+v", pins)
	}
	if err := filepath.WalkDir(root, func(filename string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			rel, err := filepath.Rel(root, filename)
			if err != nil {
				return err
			}
			if rel = filepath.ToSlash(rel); !expected[rel] || path.Clean(rel) != rel {
				t.Errorf("unexported fixture file %s", rel)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
