package clipcmd

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestOSQueryRetriesEmptyWithoutEffects(t *testing.T) {
	f := zellijtest.New(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{zellijtest.DefaultPane(0, "sh")}}}})
	empty := ""
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	raw, err := (OSRuntime{}).ListPanesContext(context.Background())
	if err != nil || len(raw) == 0 {
		t.Fatalf("panes=%q error=%v", raw, err)
	}
	if len(f.Queries()) != 2 || len(f.Effects()) != 0 {
		t.Fatalf("queries=%+v effects=%+v", f.Queries(), f.Effects())
	}
}
