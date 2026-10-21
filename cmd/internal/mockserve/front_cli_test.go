package mockserve

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHybridCommandParsingAndReport(t *testing.T) {
	for _, args := range [][]string{{"serve", "mocks", "--real", "http://localhost:3000"}, {"serve", "mocks", "--real", "http://localhost:3000", "--port", "0"}, {"serve", "mocks", "--port", "0", "--real", "http://localhost:3000"}, {"check", "mocks"}} {
		if _, err := ParseCommand(args); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"serve", "mocks", "--real", "http://elsewhere:3000"}, {"serve", "mocks", "--real"}, {"serve", "mocks", "--port", "1", "--port", "2"}, {"check", "mocks", "--real", "http://localhost:3000"}} {
		if _, err := ParseCommand(args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "screens.json"), []byte(frontCatalogue), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if code := Run(t.Context(), []string{"check", dir}, &out, &errs); code != 0 || !strings.Contains(out.String(), "detail /detail") || errs.Len() != 0 {
		t.Fatalf("check: %d %s %s", code, &out, &errs)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("check wrote files: %v %v", entries, err)
	}
}
