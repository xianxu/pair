package wrapcmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugSpawnCmdUsesPairSubcommandWithAgent(t *testing.T) {
	t.Setenv("PAIR_HOME", "/opt/pair")
	cmd := slugSpawnCmd("codex")

	wantBin := filepath.Join("/opt/pair", "bin", "pair")
	if cmd.Path != wantBin {
		t.Errorf("cmd.Path = %q, want %q", cmd.Path, wantBin)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "slug" {
		t.Errorf("cmd.Args = %v, want [<pair> slug]", cmd.Args)
	}
	found := false
	for _, e := range cmd.Env {
		if e == "PAIR_AGENT=codex" {
			found = true
		}
	}
	if !found {
		t.Errorf("PAIR_AGENT=codex not in cmd.Env")
	}
}

func TestSlugSpawnCmdFallsBackToBarePairOnPath(t *testing.T) {
	t.Setenv("PAIR_HOME", "")
	cmd := slugSpawnCmd("claude")
	// exec.Command resolves a bare name via PATH; Args[0] stays the bare name.
	if cmd.Args[0] != "pair" || strings.ContainsRune(cmd.Args[0], filepath.Separator) {
		t.Errorf("Args[0] = %q, want bare 'pair'", cmd.Args[0])
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "slug" {
		t.Errorf("cmd.Args = %v, want [pair slug]", cmd.Args)
	}
}
