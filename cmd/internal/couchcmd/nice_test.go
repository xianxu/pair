package couchcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestSlotNiceSettings(t *testing.T) {
	for raw, want := range map[string]int{"": 10, "0": 0, "5": 5, "20": 20} {
		got, err := slotNiceSettings(func(string) string { return raw })
		if err != nil || got != want {
			t.Errorf("COUCH_SLOT_NICE=%q = %d, %v; want %d", raw, got, err, want)
		}
	}
	for _, raw := range []string{"-1", "21", "ten", " 10", "1.5"} {
		if _, err := slotNiceSettings(func(string) string { return raw }); err == nil || !strings.Contains(err.Error(), "COUCH_SLOT_NICE") {
			t.Errorf("COUCH_SLOT_NICE=%q accepted or unnamed: %v", raw, err)
		}
	}
}

// Both production runners carry the configured niceness; a fake never does,
// because only the real runners reach the kernel.
func TestConsoleRunnerCarriesSlotNiceness(t *testing.T) {
	env := map[string]string{"COUCH_SLOT_NICE": "7"}
	config := consoleTraceConfig{getenv: func(k string) string { return env[k] }, root: t.TempDir()}
	_, runner, err := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	if pty, ok := runner.(*couchcore.PtyRunner); !ok || pty.Nice != 7 {
		t.Fatalf("console runner = %#v, want a PtyRunner at nice 7", runner)
	}
	_, runner, err = consoleRunnerFor("start", strings.NewReader(""), false, nil, nil, config)
	if err != nil {
		t.Fatal(err)
	}
	if exec, ok := runner.(couchcore.ExecRunner); !ok || exec.Nice != 7 {
		t.Fatalf("fallback runner = %#v, want an ExecRunner at nice 7", runner)
	}
	delete(env, "COUCH_SLOT_NICE")
	_, runner, _ = consoleRunnerFor("start", strings.NewReader(""), false, nil, nil, config)
	if exec, ok := runner.(couchcore.ExecRunner); !ok || exec.Nice != defaultSlotNice {
		t.Fatalf("default runner = %#v, want an ExecRunner at nice %d", runner, defaultSlotNice)
	}
	env["COUCH_SLOT_NICE"] = "loud"
	if _, _, err := consoleRunnerFor("start", strings.NewReader(""), true, nil, nil, config); err == nil {
		t.Fatal("an invalid COUCH_SLOT_NICE built a runner")
	}
}
