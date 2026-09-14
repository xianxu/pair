package launcher

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

// Keep the launch effects fake, but use the real durable address authority.
type warmLossRuntime struct {
	*fakeRuntime
	globalDataDir string
	claimErr      error
	couchOwned    bool
}

func (r *warmLossRuntime) EnsureThreadAddress(scope RepoScope, tag string, couchOwned bool) error {
	r.couchOwned = couchOwned
	r.claimErr = EnsureThreadAddressForPair(r.globalDataDir, scope, tag, couchOwned)
	return r.claimErr
}

func TestCouchWarmSessionLostBeforeLaunchCannotColdCreate(t *testing.T) {
	global := t.TempDir()
	tag := "couch-0001020304050607"
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: tag})
	opts.Env.Cwd = t.TempDir()
	opts.Env.DataDir = global
	scope, err := ResolveRepoScope(opts.Env.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = scope.Key, tag
	if _, err := ClaimNewThreadAddress(global, scope, tag); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThreadAddressForPair(global, scope, tag, true); err != nil {
		t.Fatal(err)
	}
	paths := NewScopedPaths(global, scope, tag)
	before, err := os.ReadFile(paths.ThreadClaim())
	if err != nil {
		t.Fatal(err)
	}

	// The session disappeared after Couch's warm check. No live session or
	// native profile is supplied, so RunLaunch itself chooses its create path.
	rt := &warmLossRuntime{fakeRuntime: newFakeRuntime(), globalDataDir: global}
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, rt, &stderr)
	if err != nil || code != 1 {
		t.Fatalf("RunLaunch = %d, %v; stderr: %s", code, err, stderr.String())
	}
	if !rt.couchOwned || !errors.Is(rt.claimErr, ErrThreadAddressClaimed) {
		t.Fatalf("address check: couchOwned=%v, error=%v", rt.couchOwned, rt.claimErr)
	}
	if len(rt.preparedLaunches) != 0 || rt.launchCount != 0 || len(rt.attached) != 0 ||
		len(rt.watchers) != 0 || len(rt.ledger) != 0 || len(rt.files) != 0 || len(rt.sessionIndex.Entries) != 0 {
		t.Fatalf("cold launch effects: prepared=%v launches=%d attached=%v watchers=%v ledger=%v files=%v index=%v",
			rt.preparedLaunches, rt.launchCount, rt.attached, rt.watchers, rt.ledger, rt.files, rt.sessionIndex)
	}
	after, err := os.ReadFile(paths.ThreadClaim())
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("established marker changed: before=%s after=%s error=%v", before, after, err)
	}
	entries, err := os.ReadDir(paths.ScopeDir())
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected durable artifacts: entries=%v error=%v", entries, err)
	}
}
