package wrapcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDebugUsesManagedDiagnosticGeneration(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "trace")
	p := &proxy{debugLogPath: path}
	p.debug("one", "detail")
	if _, e := os.Stat(filepath.Join(path+".pair-diagnostics", "state.json")); e != nil {
		t.Fatal(e)
	}
}
