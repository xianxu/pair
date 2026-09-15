package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandaloneCouchRegistersAllTracePathsWithoutPairEnvironment(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("COUCH_TRACE", "/unrequested/ambient-trace")
	home, data, traces := t.TempDir(), t.TempDir(), t.TempDir()
	root := launcher.ResolveDataDir(home, data)
	rt := testRT{env: map[string]string{"HOME": home, "XDG_DATA_HOME": data, "COUCH_TRACE": filepath.Join(traces, "timing"), "COUCH_INPUT_TRACE": filepath.Join(traces, "input"), "COUCH_MOUSE_TRACE": filepath.Join(traces, "mouse")}}
	console, _ := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil, tracesForRuntime(rt))
	if console == nil {
		t.Fatal("console missing")
	}
	defer console.SetInputTrace("")
	defer console.SetMouseTrace("")
	defer console.SetEventTrace("", processStartedAt)
	page, e := diagnosticlog.EnumerateRoot(context.Background(), root, 0, 100)
	entries := page.Entries
	if e != nil || len(entries) != 3 {
		t.Fatalf("registered traces=%v err=%v", entries, e)
	}
	for _, key := range []string{"COUCH_TRACE", "COUCH_INPUT_TRACE", "COUCH_MOUSE_TRACE"} {
		want, _ := filepath.EvalSymlinks(rt.env[key])
		found := false
		for _, entry := range entries {
			if entry.Path == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s not registered", key)
		}
	}
	b, e := os.ReadFile(rt.env["COUCH_TRACE"])
	if e != nil || !strings.Contains(string(b), "\tstartup\t") {
		t.Fatalf("startup trace lost %q %v", b, e)
	}
	if os.Getenv("PAIR_DATA_DIR") != "" {
		t.Fatal("composition mutated process environment")
	}
}
