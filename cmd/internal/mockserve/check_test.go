package mockserve

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCatalogueOnly(t *testing.T) {
	d := t.TempDir()
	writeProjectFile(t, d, "screens.json", testCatalogue)
	// These unusable inputs must not be read or repaired by check.
	writeProjectFile(t, d, "fixtures.json", "invalid JSON")
	writeProjectFile(t, d, ".mock-current", "../bad")
	before, _ := os.ReadDir(d)
	var out bytes.Buffer
	if err := Check(d, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Screen progress: 1/3 real\nMissing: runs /runs\nMissing: detail /runs/detail\n" {
		t.Fatal(out.String())
	}
	after, _ := os.ReadDir(d)
	if len(before) != len(after) {
		t.Fatal("check wrote files")
	}
	b, _ := os.ReadFile(filepath.Join(d, "screens.json"))
	if string(b) != testCatalogue {
		t.Fatal("check rewrote catalogue")
	}
}
func TestCheckInvalidInputs(t *testing.T) {
	for _, kind := range []string{"missing", "invalid", "directory", "symlink", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			d := t.TempDir()
			p := filepath.Join(d, "screens.json")
			switch kind {
			case "invalid":
				writeProjectFile(t, d, "screens.json", "null")
			case "directory":
				if err := os.Mkdir(p, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				outside := t.TempDir()
				writeProjectFile(t, outside, "data", testCatalogue)
				if err := os.Symlink(filepath.Join(outside, "data"), p); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if err := os.WriteFile(p, bytes.Repeat([]byte(" "), 256*1024+1), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			if err := Check(d, &out); err == nil {
				t.Fatal("accepted", kind)
			}
			if out.Len() != 0 {
				t.Fatal("reported invalid catalogue")
			}
		})
	}
}

type failingCatalogueWriter struct{}

func (failingCatalogueWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestCheckWriterFailure(t *testing.T) {
	d := t.TempDir()
	writeProjectFile(t, d, "screens.json", testCatalogue)
	if err := Check(d, failingCatalogueWriter{}); err == nil {
		t.Fatal("ignored writer error")
	}
}
