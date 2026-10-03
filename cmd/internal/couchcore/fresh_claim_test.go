package couchcore

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// recordingClaims observes what the production claim loop does with each
// address claim, which the shared fake's no-op claim cannot show.
type recordingClaims struct {
	ThreadArtifactController
	mu       sync.Mutex
	released []ThreadAddress
}

type recordingClaim struct {
	owner   *recordingClaims
	address ThreadAddress
}

func (r *recordingClaims) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	if _, err := r.ThreadArtifactController.Claim(address); err != nil {
		return nil, err
	}
	return recordingClaim{owner: r, address: address}, nil
}

func (c recordingClaim) Release() error {
	c.owner.mu.Lock()
	defer c.owner.mu.Unlock()
	c.owner.released = append(c.owner.released, c.address)
	return nil
}

func (r *recordingClaims) releasedAddresses() []ThreadAddress {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ThreadAddress(nil), r.released...)
}

// A commit the caller saw fail did not necessarily publish nothing. Once the
// replace's journal is durable, replay will publish the new address, so its
// claim must survive the error; a refusal before any journal published
// nothing, and holding its claim would leak it.
func TestRefusedReplaceKeepsTheClaimOnceTheJournalIsDurable(t *testing.T) {
	claimFor := func(t *testing.T, env *testEnv, old ThreadRecord, revision uint64) (*recordingClaims, error) {
		claims := &recordingClaims{ThreadArtifactController: env.Couch.Artifacts}
		env.Couch.Artifacts = claims
		_, _, err := env.Couch.claimFreshRecord(context.Background(), freshClaimInput{
			ScopeKey: old.Address.RepoScope, Cwd: old.StartingPath, RepoIdentity: "/repo/.git", TagPrefix: "repo",
			Profile: LaunchProfileResolution{Profile: LaunchProfile{Agent: "claude", Argv: []string{}}},
			Store:   env.Couch.Threads,
			Commit: func(next ThreadRecord) error {
				return env.Couch.Threads.ReplaceThreadExpected(old.Address, revision, next)
			},
		})
		return claims, err
	}

	t.Run("journal durable", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		old := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
		env.Couch.Threads.hooks.AfterJournal = func() error { return errors.New("interrupted") }
		claims, err := claimFor(t, env, old, old.Revision)
		if err == nil {
			t.Fatal("the interrupted commit reported success")
		}
		if released := claims.releasedAddresses(); len(released) != 0 {
			t.Fatalf("released %v although the journal will publish it", released)
		}
		env.Couch.Threads.hooks = threadStoreHooks{}
		if err := env.Couch.Threads.RecoverStoreJournal(); err != nil {
			t.Fatal(err)
		}
		snapshot, err := env.Couch.Threads.Snapshot()
		if err != nil || len(snapshot.Records) != 1 || snapshot.Records[0].Address == old.Address {
			t.Fatalf("replay did not publish the claimed record: %+v, %v", snapshot.Records, err)
		}
	})

	t.Run("refused before the journal", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		old := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
		claims, err := claimFor(t, env, old, old.Revision+1)
		var revision *ThreadRevisionError
		if !errors.As(err, &revision) {
			t.Fatalf("err = %v, want a revision refusal", err)
		}
		if released := claims.releasedAddresses(); len(released) != 1 {
			t.Fatalf("released %v, want exactly the refused address", released)
		}
	})
}
