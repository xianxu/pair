package couchcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// pair#847: bench-start takes an exact agent and explicit argv, both required;
// bench-retire takes neither. Both require --confirm.
func TestParseBenchCLI(t *testing.T) {
	ops := couchcore.Operations()
	inv, err := ParseCLI([]string{"--bench-start", "fixture:2", "--agent", "claude", "--agent-args", "--model x", "--confirm", "--json"}, ops)
	if err != nil || inv.messageOp != couchcore.OpBenchStart || inv.ref != "fixture:2" || inv.messageAgent != "claude" || !inv.confirmed || !inv.jsonOutput {
		t.Fatalf("bench-start: %+v %v", inv, err)
	}
	if argv, present := inv.switchArgs.Argv(); !present || strings.Join(argv, " ") != "--model x" {
		t.Fatalf("argv %v present %v", argv, present)
	}
	if inv, err := ParseCLI([]string{"--bench-start", "fixture:2", "--agent", "codex", "--agent-args", "", "--confirm"}, ops); err != nil {
		t.Fatalf("explicit empty argv: %+v %v", inv, err)
	} else if argv, present := inv.switchArgs.Argv(); !present || len(argv) != 0 {
		t.Fatalf("explicit empty argv lost: %v %v", argv, present)
	}
	if inv, err := ParseCLI([]string{"--bench-retire", "fixture:2", "--confirm"}, ops); err != nil || inv.messageOp != couchcore.OpBenchRetire {
		t.Fatalf("bench-retire: %+v %v", inv, err)
	}
	for _, args := range [][]string{
		{"--bench-start", "fixture:2", "--agent", "claude", "--confirm"},
		{"--bench-start", "fixture:2", "--agent-args", "", "--confirm"},
		{"--bench-start", "fixture:2", "--agent", "nope", "--agent-args", "", "--confirm"},
		{"--bench-start", "fixture:2", "--agent", "claude", "--agent-args", ""},
		{"--bench-retire", "fixture:2", "--agent", "claude", "--confirm"},
		{"--bench-retire", "fixture:2", "--agent-args", "", "--confirm"},
		{"--bench-retire", "fixture:2"},
	} {
		if _, err := ParseCLI(args, ops); err == nil {
			t.Errorf("ParseCLI(%q) accepted", args)
		}
	}
}
