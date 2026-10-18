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

// The turn-end slug refresh is off by default when Couch hosts the session
// (#650): no `pair slug` process, no small-model call. PAIR_SLUG=on is the
// explicit re-enable; outside Couch the spawn is unchanged.
func TestMaybeSpawnSlugOffByDefaultUnderCouch(t *testing.T) {
	cases := []struct {
		name   string
		hosted bool
		slug   string
		want   int
	}{
		{"couch hosted spawns nothing", true, "", 0},
		{"couch hosted with PAIR_SLUG=on spawns", true, "on", 1},
		{"outside couch spawns as before", false, "", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Pin all three in every case: the test process itself may run
			// hosted under Couch, so the ambient env must not leak in.
			scope := ""
			if tc.hosted {
				scope = "pair:8"
			}
			t.Setenv("COUCH_THREAD_SCOPE", scope)
			t.Setenv("COUCH_THREAD_TAG", "")
			t.Setenv("PAIR_SLUG", tc.slug)
			spawns := 0
			p := &proxy{spawnSlug: func() { spawns++ }}
			p.maybeSpawnSlug()
			if spawns != tc.want {
				t.Fatalf("spawns = %d, want %d", spawns, tc.want)
			}
		})
	}
}
