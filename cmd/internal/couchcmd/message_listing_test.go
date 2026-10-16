package couchcmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// The message authority's terminal check reads the polled listing (pair#618):
// with a zellij that takes 1 s per listing, three checks (sender, recipient
// Observe, recipient Reserve) finish inside the 2 s admission budget and run
// no zellij of their own.
func TestMessageAuthorityListedCheckReadsThePolledListing(t *testing.T) {
	dir := t.TempDir()
	count := filepath.Join(dir, "calls")
	stub := filepath.Join(dir, "zellij")
	script := "#!/bin/sh\necho x >> " + count + "\nsleep 1\nprintf '\\360\\237\\223\\2011-264\\n'\n"
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := func() int {
		raw, _ := os.ReadFile(count)
		return strings.Count(string(raw), "x")
	}
	checker := couchcore.ScopedThreadArtifactCollisionChecker{GlobalDataDir: dir, Zellij: launcher.ZellijSource{Path: stub}}
	listings := launcher.NewListingPoller(checker.Zellij, time.Hour, 5*time.Second)
	listings.Refresh(context.Background())
	polled := calls()
	if polled != 2 {
		t.Fatalf("one refresh ran %d listings, want 2", polled)
	}
	observer, ok := messageListedObserver(checker, listings)
	if !ok {
		t.Fatal("checker not observable")
	}
	address := couchcore.ThreadAddress{RepoScope: "scope", Tag: "1-pair-58"}
	start := time.Now()
	for range 3 {
		session, err := observer.ListedPairSessionContext(context.Background(), address, "📁1-264")
		if err != nil || !session.Present {
			t.Fatalf("listed check: %+v %v", session, err)
		}
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("three checks took %s", elapsed)
	}
	if calls() != polled {
		t.Fatalf("checks ran zellij: %d listings after the poll", calls()-polled)
	}
	if _, err := (messageListedObserverOrFail(t, &checker, listings)).ListedPairSessionContext(context.Background(), address, "📁1-264"); err != nil {
		t.Fatalf("pointer checker: %v", err)
	}
	if calls() != polled {
		t.Fatal("the pointer checker ran zellij")
	}
	if checker.Zellij.Listings != nil {
		t.Fatal("the supervisor's own checker was given the poller")
	}
}

func messageListedObserverOrFail(t *testing.T, artifacts couchcore.ThreadArtifactController, listings *launcher.ListingPoller) listedSessionObserver {
	t.Helper()
	observer, ok := messageListedObserver(artifacts, listings)
	if !ok {
		t.Fatal("not observable")
	}
	return observer
}
