package keyscmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrintsRealBindings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"Alt+⏎", "send buffer + clear", "Alt+h", "Alt+x"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	// The bug in its exact form: the help must not point back at the help key.
	if strings.Contains(out, "keybindings are on Alt+h") {
		t.Error("help refers the reader back to Alt+h (#132)")
	}
	// Nor should it be the CLI synopsis.
	if strings.Contains(out, "pair resume <tag>") {
		t.Error("help is showing CLI usage instead of keybindings")
	}
}

func TestRunCentersWhenAsked(t *testing.T) {
	var plain, centered, stderr bytes.Buffer
	Run(nil, &plain, &stderr)
	Run([]string{"--center", "120"}, &centered, &stderr)
	if plain.String() == centered.String() {
		t.Fatal("--center had no effect")
	}
	if !strings.HasPrefix(centered.String(), " ") {
		t.Error("centered output should be indented")
	}
}

func TestRunIgnoresGarbageCenterValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--center", "not-a-number"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if stdout.Len() == 0 {
		t.Error("a bad --center must not suppress the help")
	}
}
