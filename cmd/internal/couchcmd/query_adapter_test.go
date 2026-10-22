package couchcmd

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestOSQueryRetriesEmptyWithoutEffects(t *testing.T) {
	f := zellijtest.New(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Clients: []zellijtest.Client{{ID: 7, PaneID: "terminal_0", RunningCommand: "sh"}}}}})
	empty := ""
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	raw, err := queryShortcutClients(context.Background(), "test")
	if err != nil || !strings.Contains(string(raw), "terminal_0") {
		t.Fatalf("clients=%q error=%v", raw, err)
	}
	if len(f.Queries()) != 2 || len(f.Effects()) != 0 {
		t.Fatalf("queries=%+v effects=%+v", f.Queries(), f.Effects())
	}
}

func TestShortcutQueryPreservesExitStderr(t *testing.T) {
	f := zellijtest.New(t)
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Error: "connection failed"})
	for _, env := range f.Env() {
		key, value, _ := strings.Cut(env, "=")
		t.Setenv(key, value)
	}
	_, err := queryShortcutClients(context.Background(), "test")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || !strings.Contains(string(exit.Stderr), "connection failed") {
		t.Fatalf("exit=%+v error=%v", exit, err)
	}
	if len(f.Queries()) != 1 || len(f.Effects()) != 0 {
		t.Fatalf("queries=%+v effects=%+v", f.Queries(), f.Effects())
	}
}
