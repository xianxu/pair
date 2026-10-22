package layoutcmd

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestOSQueryRetriesEmptyBeforeWorkbenchConfirmation(t *testing.T) {
	f := zellijtest.New(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {
		Panes: []zellijtest.Pane{zellijtest.DefaultPane(0, "nvim")}, Tab: zellijtest.DefaultTab(),
	}}})
	empty := ""
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	panes, tab, position, err := ConfirmWorkbenchPanesContext(context.Background(), OSRuntime{}, "0", nil)
	if err != nil || len(panes) != 1 || tab != 0 || position != 0 {
		t.Fatalf("confirmation panes=%+v tab=%d position=%d error=%v", panes, tab, position, err)
	}
	if got := len(f.Queries()); got != 3 {
		t.Fatalf("query invocations=%d, want empty retry plus two confirmations", got)
	}
	if effects := f.Effects(); len(effects) != 0 {
		t.Fatalf("query caused effects: %+v", effects)
	}
}

func TestNativeQueryObservesRetriesAndFinalLogicalOutcome(t *testing.T) {
	f := zellijtest.New(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{zellijtest.DefaultPane(0, "sh")}, Tab: zellijtest.DefaultTab()}}})
	empty := ""
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	var logs []string
	live := &fullscreenLiveRuntime{session: "test", caller: "0", queryLog: func(format string, fields ...any) { logs = append(logs, fmt.Sprintf(format, fields...)) }}
	raw, err := live.ListPanesJSONContext(context.Background())
	if err != nil || len(raw) == 0 || len(logs) != 2 {
		t.Fatalf("raw=%q error=%v logs=%v", raw, err, logs)
	}
	// The higher-level conformance observer records the final logical outcome.
	f.SetFaults(zellijtest.Fault{Ordinal: 3, Reply: &empty})
	native := &nativeReplyRuntime{fullscreenLiveRuntime: live}
	_, err = native.ListPanesJSONContext(context.Background())
	if err != nil || len(native.events) != 1 || native.events[0].Error != "" || native.events[0].Unknown {
		t.Fatalf("error=%v events=%+v", err, native.events)
	}
	if len(f.Queries()) != 4 || len(f.Effects()) != 0 {
		t.Fatalf("queries=%+v effects=%+v", f.Queries(), f.Effects())
	}
}
