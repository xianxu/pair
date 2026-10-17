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
func TestWrapperInheritsTitleScopeOnCreateAndAttach(t *testing.T) {
	want := mustScope(t, "/home/u/work").Key // baseOpts' Cwd; RepoRoot unset

	t.Run("attach", func(t *testing.T) {
		rt := newFakeRuntime()
		opts := baseOpts(LaunchArgs{})
		if _, err := AttachExistingSession(opts, opts.Env, rt, "live", "📁work-live", "claude"); err != nil {
			t.Fatal(err)
		}
		assertWrapperTitleEnvironment(t, rt, want, opts.Env.DataDir)
	})

	t.Run("create", func(t *testing.T) {
		rt := newFakeRuntime()
		rt.uuids = []string{"MINTED-1"}
		opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"})
		if code, err := run(t, opts, rt); err != nil || code != 0 {
			t.Fatalf("create = %d, %v", code, err)
		}
		assertWrapperTitleEnvironment(t, rt, want, opts.Env.DataDir)
	})
}

// Attach's scope, one row per branch CONDITION rather than per fallback step
// (close review BR-3, then round 2): the root resolves or not; couch's record
// names this tag or not; that record's key validates or not; a key would be
// inherited or not. A compound condition is two branches, and the arm no row
// enters is where an unverified claim survives -- round 2 measured the
// validation arm deletable with the suite green.
//
// The answers: the repo root wins, being the key the ledger was written under;
// failing it, couch's record of THIS thread, if its key validates; failing
// that, an explicit empty key, never an inherited one -- that belongs to
// whatever ran pair, and from another thread's pane it would name the wrong
// session.
func TestTitlePollerScopeWhenAttachCannotResolveARoot(t *testing.T) {
	const couchScope = "c0ffee0123456789"
	rootScope := mustScope(t, "/home/u/work").Key
	for _, c := range []struct {
		name                           string
		cwd, couchScope, couchTag, inh string
		want                           string
	}{
		{"the resolved root outranks couch's record", "/home/u/work", couchScope, "live", "", rootScope},
		{"couch's record of this thread", "/", couchScope, "live", "", couchScope},
		{"couch's record of another thread is not borrowed", "/", couchScope, "other", "", ""},
		{"couch's record that does not validate is not trusted", "/", "../not a key", "live", "", ""},
		{"an inherited key is overridden, not trusted", "/", couchScope, "other", "f00dfeed01234567", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			rt := newFakeRuntime()
			if c.inh != "" {
				rt.env[contextcmd.EnvScopeKey] = c.inh
			}
			opts := baseOpts(LaunchArgs{})
			opts.Env.Cwd = c.cwd // ResolveRepoScope refuses "/"
			opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = c.couchScope, c.couchTag
			if _, err := AttachExistingSession(opts, opts.Env, rt, "live", "📁x-live", "claude"); err != nil {
				t.Fatal(err)
			}
			got, present := rt.env[contextcmd.EnvScopeKey]
			if !present {
				t.Fatalf("the poller starts with no %s at all -- the contract must say something", contextcmd.EnvScopeKey)
			}
			if got != c.want {
				t.Fatalf("scope = %q, want %q", got, c.want)
			}
		})
	}
}

func assertWrapperTitleEnvironment(t *testing.T, rt *fakeRuntime, scopeKey, dataDir string) {
	t.Helper()
	started := rt.env

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

func TestLauncherLeavesTitleLifetimeToWrapper(t *testing.T) {
	for _, mode := range []string{"create", "attach"} {
		t.Run(mode, func(t *testing.T) {
			rt := newFakeRuntime()
			opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "title-owner"})
			if mode == "create" {
				rt.uuids = []string{"MINTED-1"}
				if code, err := run(t, opts, rt); code != 0 || err != nil {
					t.Fatalf("create %d %v", code, err)
				}
			} else {
				if _, err := AttachExistingSession(opts, opts.Env, rt, "live", "fixture-session", "claude"); err != nil {
					t.Fatal(err)
				}
			}
			if len(rt.pollerEnvs) != 0 {
				t.Fatal("launcher detached a title outside the wrapper lease")
			}
		})
	}
}
