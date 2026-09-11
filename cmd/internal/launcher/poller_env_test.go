package launcher

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
)

// pair#183: every reattached thread lost its context meter because attach
// spawned the title poller without PAIR_SCOPE_KEY, and an empty key matches no
// session. Asserted as what the poller STARTS WITH, on both launch paths, so the
// two cannot drift again -- and against the names the contract itself declares,
// never a list kept here: two hand-kept lists are what drifted.
func TestTitlePollerStartsWithItsWholeContractOnBothPaths(t *testing.T) {
	want := mustScope(t, "/home/u/work").Key // baseOpts' Cwd; RepoRoot unset

	t.Run("attach", func(t *testing.T) {
		rt := newFakeRuntime()
		opts := baseOpts(LaunchArgs{})
		if _, err := AttachExistingSession(opts, opts.Env, rt, "live", "📁work-live", "claude"); err != nil {
			t.Fatal(err)
		}
		assertPollerStartsWith(t, rt, want, opts.Env.DataDir)
	})

	t.Run("create", func(t *testing.T) {
		rt := newFakeRuntime()
		rt.uuids = []string{"MINTED-1"}
		opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"})
		if code, err := run(t, opts, rt); err != nil || code != 0 {
			t.Fatalf("create = %d, %v", code, err)
		}
		assertPollerStartsWith(t, rt, want, opts.Env.DataDir)
	})
}

func assertPollerStartsWith(t *testing.T, rt *fakeRuntime, scopeKey, dataDir string) {
	t.Helper()
	if len(rt.pollerEnvs) != 1 {
		t.Fatalf("title pollers spawned = %d, want 1", len(rt.pollerEnvs))
	}
	started := rt.pollerEnvs[0]

	// Every name the contract declares arrives non-empty. The names come from
	// the contract, so a new field joins this check without anyone listing it,
	// and an explicit empty value -- which would override a good inherited one --
	// fails here too. (The arguments only have to be non-empty for Environ to
	// render the names; the names are what is being read.)
	for _, kv := range titlepoller.NewSessionEnv("probe", "probe").Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if started[name] == "" {
			t.Errorf("the poller starts without %s -- its context meter cannot resolve", name)
		}
	}
	// And the values are the right ones, read through the poller's own reader.
	got := contextcmd.EnvFrom(func(name string) string { return started[name] })
	if got.PairScopeKey != scopeKey || got.PairDataDir != dataDir {
		t.Errorf("the poller reads scope %q / data dir %q, want %q / %q", got.PairScopeKey, got.PairDataDir, scopeKey, dataDir)
	}
}
