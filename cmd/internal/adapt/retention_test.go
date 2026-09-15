package adapt

import (
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenUsesManagedDiagnosticGeneration(t *testing.T) {
	d := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", d)
	t.Setenv("PAIR_TAG", "test")
	l := Open("test", "codex")
	if l == nil {
		t.Fatal("logger unavailable")
	}
	l.Log(1, "test", Fired, "")
	l.Close()
	p, e := artifactpath.ResolveScoped(d, "test")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(p.AdaptLog()+".pair-diagnostics", "state.json")); e != nil {
		t.Fatal(e)
	}
}
