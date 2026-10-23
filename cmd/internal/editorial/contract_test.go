package editorial

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractPinnedSourceBytes(t *testing.T) {
	raw, err := os.ReadFile("../../../workshop/evidence/000823/contract-provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	var pin struct {
		Commit string
		Files  []struct {
			Source string
			Bytes  int
			SHA256 string
		}
	}
	if err := json.Unmarshal(raw, &pin); err != nil {
		t.Fatal(err)
	}
	if pin.Commit != "4f10a45d038acb2324c5a19e74a78f4a593571e6" || len(pin.Files) != 39 {
		t.Fatalf("unexpected source pin: %s %d", pin.Commit, len(pin.Files))
	}
	c, err := DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range pin.Files {
		rel := strings.TrimPrefix(f.Source, "construct/local/universal-review/")
		path := filepath.Join("testdata/contract", rel)
		if strings.HasPrefix(rel, "modes/") {
			path = rel
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Error(err)
			continue
		}
		digest := sha256.Sum256(data)
		if len(data) != f.Bytes || hex.EncodeToString(digest[:]) != f.SHA256 {
			t.Errorf("%s differs from pinned source", path)
		}
		if strings.HasPrefix(rel, "modes/") {
			mode, ok := c.Lookup(strings.TrimSuffix(filepath.Base(rel), ".md"))
			if !ok {
				t.Fatalf("mode missing: %s", rel)
			}
			_, body, ok := strings.Cut(string(data[4:]), "\n---\n")
			if !ok {
				t.Fatalf("bad source frontmatter: %s", rel)
			}
			if mode.Brief != body {
				t.Errorf("%s brief differs from exact pinned body", rel)
			}
		}
	}
}
