package couchcmd

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
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
	script := "#!/bin/sh\necho x >> " + count + "\nsleep 1\n" + `case "$*" in
 'list-sessions --short') ` + zellijtest.ShellCase(t, "couch.message264.short") + ` ;;
 'list-sessions --no-formatting') ` + zellijtest.ShellCase(t, "couch.message264.full") + ` ;;
 *) exit 2 ;;
esac
`
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
	if polled != 1 {
		t.Fatalf("one refresh ran %d listings, want 1", polled)
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

// The wired resolver: every check the broker makes (sender, Observe, Reserve,
// and each 1 s promote retry of a queued message) reads the poller, so fifty
// checks against a 1 s-per-listing zellij run no zellij at all (pair#618).
func TestMessageThreadResolverChecksReadOnlyThePoller(t *testing.T) {
	rt := newRT(t, "/repo")
	record := seedDetachedThread(t, rt, "/repo")
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	count := filepath.Join(dir, "calls")
	stub := filepath.Join(dir, "zellij")
	script := "#!/bin/sh\necho x >> " + count + "\nsleep 1\n" + `case "$*" in
 'list-sessions --short') ` + zellijtest.ShellCase(t, "couch.message300.short") + ` ;;
 'list-sessions --no-formatting') ` + zellijtest.ShellCase(t, "couch.message300.full") + ` ;;
 *) exit 2 ;;
esac
`
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	c.Artifacts = couchcore.ScopedThreadArtifactCollisionChecker{GlobalDataDir: dir, Zellij: launcher.ZellijSource{Path: stub}}
	listings := launcher.NewListingPoller(launcher.ZellijSource{Path: stub}, time.Hour, 5*time.Second)
	resolve := messageThreadResolver(c, listings)
	binding := couchmessage.Binding{Scope: record.Address.RepoScope, Tag: string(record.Address.Tag), Session: "📁1-300"}
	if _, err := resolve(context.Background(), binding); err == nil || !strings.Contains(err.Error(), "not yet known") {
		t.Fatalf("before the first listing: %v", err)
	}
	listings.Refresh(context.Background())
	raw, _ := os.ReadFile(count)
	polled := strings.Count(string(raw), "x")
	start := time.Now()
	for range 50 {
		if root, err := resolve(context.Background(), binding); err != nil || root != "/repo" {
			t.Fatalf("check: %q %v", root, err)
		}
	}
	raw, _ = os.ReadFile(count)
	if n := strings.Count(string(raw), "x"); n != polled || time.Since(start) > time.Second {
		t.Fatalf("50 checks ran %d listings in %s", n-polled, time.Since(start))
	}
}
