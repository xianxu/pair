package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestThreadAddressClaimSerializesCurrentPairAndCouchProducers(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	claim, err := ClaimNewThreadAddress(global, scope, "couch-0001020304050607")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = claim.Release() })
	if err := EnsureThreadAddressForPair(global, scope, "couch-0001020304050607", false); !errors.Is(err, ErrThreadAddressClaimed) {
		t.Fatalf("direct Pair raced marker-only Couch claim: %v", err)
	}
	if err := EnsureThreadAddressForPair(global, scope, "couch-0001020304050607", true); err != nil {
		t.Fatalf("Couch child did not adopt parent claim: %v", err)
	}
}

func TestDirectPairAdoptsClaimBeforeHistoricalArtifacts(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	paths := NewScopedPaths(global, scope, "old-tag")
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Draft(), []byte("history"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureThreadAddressForPair(global, scope, "old-tag", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.ThreadClaim()); err != nil {
		t.Fatalf("direct Pair marker: %v", err)
	}
}

func TestClaimNewThreadAddressHasOneAtomicWinner(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	const contenders = 20
	var wg sync.WaitGroup
	wins := make(chan *ThreadAddressClaim, contenders)
	errs := make(chan error, contenders)
	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claim, err := ClaimNewThreadAddress(global, scope, "couch-0001020304050607")
			if err == nil {
				wins <- claim
				return
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(wins)
	close(errs)
	var winner *ThreadAddressClaim
	for claim := range wins {
		if winner != nil {
			t.Fatal("more than one atomic address-claim winner")
		}
		winner = claim
	}
	if winner == nil {
		t.Fatal("no address-claim winner")
	}
	t.Cleanup(func() { _ = winner.Release() })
	for err := range errs {
		if !errors.Is(err, ErrThreadAddressClaimed) {
			t.Errorf("loser error = %v", err)
		}
	}
}

func TestClaimNewThreadAddressFailsClosedOnMalformedSessionIndex(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	if err := os.WriteFile(filepath.Join(global, "session-names.jsonl"), []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	claim, err := ClaimNewThreadAddress(global, scope, "couch-0001020304050607")
	if err == nil || claim != nil {
		t.Fatalf("malformed index claim = %T, %v", claim, err)
	}
	if _, statErr := os.Stat(NewScopedPaths(global, scope, "couch-0001020304050607").ThreadClaim()); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed claim marker not rolled back: %v", statErr)
	}
}
