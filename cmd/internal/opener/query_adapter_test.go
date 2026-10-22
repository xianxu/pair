package opener

import (
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
	if id := (OSRuntime{}).AgentPaneID(); id != "0" {
		t.Fatalf("agent pane=%q", id)
	}
	if len(f.Queries()) != 2 || len(f.Effects()) != 0 {
		t.Fatalf("queries=%+v effects=%+v", f.Queries(), f.Effects())
	}
}

func TestDumpScreenPreservesEmptyWithOneAttempt(t *testing.T) {
	f := zellijtest.New(t)
	empty := ""
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	raw, err := (OSRuntime{}).DumpScreen("0")
	if err != nil || raw != "" || len(f.Queries()) != 1 || len(f.Effects()) != 0 {
		t.Fatalf("raw=%q error=%v queries=%+v effects=%+v", raw, err, f.Queries(), f.Effects())
	}
}
