package couchtty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTraceUsesManagedDiagnosticGeneration(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "trace")
	f, e := openTraceFile("TEST", path)
	if e != nil {
		t.Fatal(e)
	}
	f.writeLine("one")
	f.Close()
	if _, e = os.Stat(filepath.Join(path+".pair-diagnostics", "state.json")); e != nil {
		t.Fatal(e)
	}
}
