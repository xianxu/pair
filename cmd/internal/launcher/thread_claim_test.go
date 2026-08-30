package launcher

import (
	"bytes"
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
	if established, err := ThreadAddressEstablished(global, scope, "couch-0001020304050607"); err != nil || established {
		t.Fatalf("reserved claim registration = %v, %v", established, err)
	}
	if err := EnsureThreadAddressForPair(global, scope, "couch-0001020304050607", false); !errors.Is(err, ErrThreadAddressClaimed) {
		t.Fatalf("direct Pair raced marker-only Couch claim: %v", err)
	}
	if err := EnsureThreadAddressForPair(global, scope, "couch-0001020304050607", true); err != nil {
		t.Fatalf("Couch child did not adopt parent claim: %v", err)
	}
	if established, err := ThreadAddressEstablished(global, scope, "couch-0001020304050607"); err != nil || !established {
		t.Fatalf("established claim registration = %v, %v", established, err)
	}
}

func TestReservedThreadAddressPublicationIsAtomicForConcurrentReaders(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	tag := "couch-0001020304050607"
	claim, err := ClaimNewThreadAddress(global, scope, tag)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = claim.Release() })

	ready := make(chan struct{})
	publish := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- establishReservedThreadAddressWithHook(NewScopedPaths(global, scope, tag), scope, tag, func() {
			close(ready)
			<-publish
		})
	}()
	<-ready
	for i := 0; i < 100; i++ {
		established, err := ThreadAddressEstablished(global, scope, tag)
		if err != nil || established {
			t.Fatalf("reader during publication = %v, %v; want complete reserved record", established, err)
		}
	}
	close(publish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if established, err := ThreadAddressEstablished(global, scope, tag); err != nil || !established {
		t.Fatalf("reader after publication = %v, %v", established, err)
	}
}

func TestRegisterExistingCouchThreadAcceptsOnlyExactEstablishedMarkerWithoutMutation(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	tag := "couch-0001020304050607"
	paths := NewScopedPaths(global, scope, tag)
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"schema":1,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"established"}` + "\n")
	if err := os.WriteFile(paths.ThreadClaim(), want, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RegisterExistingCouchThread(global, scope, tag); err != nil {
		t.Fatalf("register existing Couch thread: %v", err)
	}
	got, err := os.ReadFile(paths.ThreadClaim())
	if err != nil {
		t.Fatalf("established marker removed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("established marker rewritten:\n got %q\nwant %q", got, want)
	}
}

func TestRegisterExistingCouchThreadRejectsUntrustedMarkerWithoutMutation(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	tag := "couch-0001020304050607"
	paths := NewScopedPaths(global, scope, tag)
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"malformed":    "not-json\n",
		"wrong schema": `{"schema":2,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"established"}` + "\n",
		"wrong scope":  `{"schema":1,"scope":"fedcba9876543210","tag":"couch-0001020304050607","state":"established"}` + "\n",
		"wrong tag":    `{"schema":1,"scope":"0123456789abcdef","tag":"couch-ffffffffffffffff","state":"established"}` + "\n",
		"reserved":     `{"schema":1,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"reserved"}` + "\n",
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			want := []byte(raw)
			if err := os.WriteFile(paths.ThreadClaim(), want, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := RegisterExistingCouchThread(global, scope, tag); err == nil {
				t.Fatal("untrusted marker accepted")
			}
			got, err := os.ReadFile(paths.ThreadClaim())
			if err != nil {
				t.Fatalf("rejected marker removed: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("rejected marker rewritten:\n got %q\nwant %q", got, want)
			}
		})
	}
}

func TestRegisterExistingCouchThreadDoesNotCreateMissingScopeOrMarker(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef"}
	paths := NewScopedPaths(global, scope, "couch-0001020304050607")

	if err := RegisterExistingCouchThread(global, scope, "couch-0001020304050607"); err == nil {
		t.Fatal("missing marker accepted")
	}
	if _, err := os.Stat(paths.ScopeDir()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("registration created scope directory: %v", err)
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
	paths := NewScopedPaths(global, scope, "couch-0001020304050607")
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.resolved().SessionBindings(), []byte("not-json\n"), 0o600); err != nil {
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

func TestClaimNewThreadAddressRejectsLegacyGlobalBinding(t *testing.T) {
	global := t.TempDir()
	scope := RepoScope{Key: "0123456789abcdef", Root: "/repo", DisplayName: "repo"}
	tag := "couch-0001020304050607"
	entry := SessionNameEntry{SessionName: "📁repo-work", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: tag}
	line, err := BuildSessionNameIndexLine(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(global, "session-names.jsonl"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	claim, err := ClaimNewThreadAddress(global, scope, tag)
	if !errors.Is(err, ErrThreadAddressClaimed) || claim != nil {
		t.Fatalf("legacy global binding claim = %T, %v", claim, err)
	}
}

type recordingSessionDeleter struct {
	deleted []string
	err     error
}

func (d *recordingSessionDeleter) DeleteSession(name string) error {
	d.deleted = append(d.deleted, name)
	return d.err
}

func TestQuiesceThreadSessionDeletesOnlyExactIndexedPairSession(t *testing.T) {
	global := t.TempDir()
	paths := NewScopedPaths(global, RepoScope{Key: "0123456789abcdef"}, "couch-0001020304050607")
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	index := paths.resolved().SessionBindings()
	rows := "not-json\n"
	deleter := &recordingSessionDeleter{}
	if err := os.WriteFile(index, []byte(rows), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := QuiesceThreadSession(global, "0123456789abcdef", "couch-0001020304050607", deleter); err == nil || len(deleter.deleted) != 0 {
		t.Fatalf("malformed index quiescence = %v, deleted=%v", err, deleter.deleted)
	}

	entry := SessionNameEntry{SessionName: "📁pair-couch", ScopeKey: "0123456789abcdef", RepoRoot: "/repo", RepoName: "repo", Tag: "couch-0001020304050607"}
	line, err := BuildSessionNameIndexLine(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(index, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := QuiesceThreadSession(global, entry.ScopeKey, entry.Tag, deleter); err != nil {
		t.Fatal(err)
	}
	if len(deleter.deleted) != 1 || deleter.deleted[0] != entry.SessionName {
		t.Fatalf("deleted sessions = %v", deleter.deleted)
	}
}

func TestQuiesceThreadSessionUsesLegacyGlobalBinding(t *testing.T) {
	global := t.TempDir()
	entry := SessionNameEntry{
		SessionName: "📁repo-work", ScopeKey: "0123456789abcdef", RepoRoot: "/repo", RepoName: "repo", Tag: "work",
	}
	line, err := BuildSessionNameIndexLine(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(global, "session-names.jsonl"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	deleter := &recordingSessionDeleter{}
	if err := QuiesceThreadSession(global, entry.ScopeKey, entry.Tag, deleter); err != nil {
		t.Fatal(err)
	}
	if len(deleter.deleted) != 1 || deleter.deleted[0] != entry.SessionName {
		t.Fatalf("deleted = %v", deleter.deleted)
	}
}
