package wrapcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestViewportDumpPreservesEmptyWithOneAttempt(t *testing.T) {
	f := zellijtest.New(t)
	empty := ""
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	p := &proxy{terminal: newTerminalModelForTest(t, 80, 24)}
	if !p.composerInView() {
		t.Fatal("empty viewport should preserve uncertain visibility")
	}
	if len(f.Queries()) != 1 || len(f.Effects()) != 0 {
		t.Fatalf("queries=%+v effects=%+v", f.Queries(), f.Effects())
	}
}
