package titlepoller

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestOSQueryRetriesEmptyWithoutEffects(t *testing.T) {
	f := zellijtest.New(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {}}})
	empty := ""
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	if !(OSRuntime{}).SessionAlive("test") {
		t.Fatal("session not observed after empty retry")
	}
	if len(f.Queries()) != 2 || len(f.Effects()) != 0 {
		t.Fatalf("queries=%+v effects=%+v", f.Queries(), f.Effects())
	}
}
