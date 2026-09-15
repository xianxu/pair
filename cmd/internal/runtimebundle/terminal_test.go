package runtimebundle

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestTerminalEnvironmentProvidesCompiledProfile(t *testing.T) {
	env, err := TerminalEnvironment(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 2 || env[0] != "TERM=pair-vt-256color" {
		t.Fatalf("env=%v", env)
	}
	command := exec.Command("infocmp", "-x", "pair-vt-256color")
	command.Env = append(os.Environ(), env...)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled profile not loadable: %v %s", err, out)
	}
	if !strings.Contains(string(out), "colors#256") || !strings.Contains(string(out), "Sync=") {
		t.Fatalf("wrong capabilities: %s", out)
	}
}
