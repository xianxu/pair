package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// freshClaimInput is everything claimFreshRecord needs to mint a fresh,
// start-claimed record and publish it. Commit is the one store write that
// publishes it -- a slot's replaceSlotCurrent, or the main store's
// ReplaceThreadExpected -- and Store is the store that write lands in, which
// decides whether a refused commit may release the address claim.
type freshClaimInput struct {
	ScopeKey, Cwd, RepoIdentity, TagPrefix string
	Profile                                LaunchProfileResolution
	Used                                   map[ThreadAddress]bool
	Store                                  *ThreadStore
	Commit                                 func(ThreadRecord) error
}

// claimFreshRecord is the claim loop fresh slot and reboot share: pick a tag,
// build the start-claimed record, claim the address against every Pair
// artifact producer, and commit -- all inside one attempt, so a collision at
// either the artifact claim or the store moves to the next tag rather than
// failing the start.
//
// The record starts UNNAMED (#363). A fresh agent is a fresh conversation: the
// stored name and description belong to the conversation they described, and
// they leave with it, into the archive.
func (c *Couch) claimFreshRecord(ctx context.Context, in freshClaimInput) (ThreadRecord, string, error) {
	if in.Commit == nil || in.Store == nil {
		return ThreadRecord{}, "", errors.New("fresh claim has no commit")
	}
	owner, err := c.Proc.Current()
	if err != nil {
		return ThreadRecord{}, "", err
	}
	nonce, err := allocateStartNonce(c.Entropy)
	if err != nil {
		return ThreadRecord{}, "", err
	}
	for attempt := 0; attempt < threadTagAttempts; attempt++ {
		tag, err := c.allocateConversationTag(ctx, in.TagPrefix)
		if err != nil {
			return ThreadRecord{}, "", err
		}
		record := ThreadRecord{SchemaVersion: ThreadSchemaVersion, Address: ThreadAddress{RepoScope: in.ScopeKey, Tag: ThreadTag(tag)}, StartingPath: in.Cwd, WorkingPath: in.Cwd, CreatedAt: c.Clock.Now(), Revision: 1}
		if in.Used[record.Address] {
			continue
		}
		record.Incarnations = []ThreadIncarnation{{State: IncarnationCreating, StartedAt: c.Clock.Now(), RepoIdentity: in.RepoIdentity}}
		record, err = AdvanceStartTransaction(record, StartEvent{Kind: StartClaimed, Nonce: nonce, Owner: SupervisorOwner{PID: owner.PID, Identity: owner.Identity}, Profile: &in.Profile.Profile})
		if err != nil {
			return ThreadRecord{}, "", err
		}
		claim, err := c.Artifacts.Claim(record.Address)
		if errors.Is(err, launcher.ErrThreadAddressClaimed) {
			continue
		}
		if err != nil {
			return ThreadRecord{}, "", err
		}
		if err := ctx.Err(); err != nil {
			return ThreadRecord{}, "", errors.Join(err, claim.Release())
		}
		if err := in.Commit(record); err != nil {
			var exists *ThreadExistsError
			if errors.As(err, &exists) {
				if releaseErr := claim.Release(); releaseErr != nil {
					return ThreadRecord{}, "", errors.Join(err, releaseErr)
				}
				continue
			}
			return ThreadRecord{}, "", in.Store.releaseRefusedClaim(record.Address, claim, err)
		}
		return record, nonce, nil
	}
	return ThreadRecord{}, "", fmt.Errorf("fresh start exhausted %d native address collision attempts", threadTagAttempts)
}

// releaseRefusedClaim decides what a commit the caller saw fail did to the
// address it claimed.
//
// Once a journal exists, replay may publish this address even though the
// caller saw an error, so the native ownership marker is kept until
// reconciliation. It is kept too when the record file already names this
// address (a slot's thread.json envelope, or the main store's record file).
// Only a refusal that provably published nothing releases the claim.
func (s *ThreadStore) releaseRefusedClaim(address ThreadAddress, claim ThreadArtifactClaim, cause error) error {
	if _, err := os.Lstat(s.journalPath()); !errors.Is(err, os.ErrNotExist) {
		return cause
	}
	raw, err := s.readRetentionFile(s.recordPath(address))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Join(cause, err)
	}
	var envelope struct {
		Address ThreadAddress `json:"address"`
	}
	if err == nil {
		if json.Unmarshal(raw, &envelope) != nil || envelope.Address == address {
			return cause
		}
	}
	return errors.Join(cause, claim.Release())
}
