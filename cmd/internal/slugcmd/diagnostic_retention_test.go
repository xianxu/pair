package slugcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugUsesManagedDiagnosticGeneration(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "trace")
	t.Setenv("PAIR_SLUG_LOG", path)
	logf("one")
	if _, e := os.Stat(filepath.Join(path+".pair-diagnostics", "state.json")); e != nil {
		t.Fatal(e)
	}
}
