// Package gcruntime composes retention with the owners of external evidence.
package gcruntime

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type CouchReferences struct{ Coordinator *storagegc.Coordinator }

func (r CouchReferences) Snapshot(ctx context.Context, held *storagegc.Locked, paths []string) (storagegc.References, error) {
	var refs storagegc.References
	if r.Coordinator == nil {
		return refs, errors.New("missing Pair coordinator")
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return refs, err
		}
		namespace, err := couchcore.ExistingCouchNamespace(path)
		if err != nil {
			return refs, err
		}
		snapshot, err := couchcore.ReadStoreRetention(namespace, r.Coordinator, held)
		if err != nil {
			return refs, err
		}
		for _, a := range snapshot.Visible {
			if err := ctx.Err(); err != nil {
				return refs, err
			}
			owner, err := artifactpath.NewStorageOwner(r.Coordinator.Root, a.RepoScope, string(a.Tag))
			if err != nil {
				return refs, err
			}
			refs.Visible = append(refs.Visible, owner)
		}
		for _, a := range snapshot.Archives {
			if err := ctx.Err(); err != nil {
				return refs, err
			}
			owner, err := artifactpath.NewStorageOwner(r.Coordinator.Root, a.Address.RepoScope, string(a.Address.Tag))
			if err != nil {
				return refs, err
			}
			refs.Archives = append(refs.Archives, storagegc.ArchiveReference{Store: path, Owner: owner, RecordHash: a.RecordHash, ArchivedAt: a.ArchivedAt, ClockError: a.ClockError})
		}
	}
	return refs, nil
}
func archiveRequest(id string, ref storagegc.ArchiveReference) couchcore.ArchiveDetachRequest {
	return couchcore.ArchiveDetachRequest{OperationID: id, Address: couchcore.ThreadAddress{RepoScope: ref.Owner.RepoScope, Tag: couchcore.ThreadTag(ref.Owner.Tag)}, RecordHash: ref.RecordHash, ArchivedAt: ref.ArchivedAt}
}
func (r CouchReferences) Detach(held *storagegc.Locked, id string, ref storagegc.ArchiveReference) error {
	ns, err := couchcore.ExistingCouchNamespace(ref.Store)
	if err != nil {
		return err
	}
	return couchcore.DetachStoreArchive(ns, r.Coordinator, held, archiveRequest(id, ref))
}
func (r CouchReferences) Forget(held *storagegc.Locked, id string, ref storagegc.ArchiveReference) error {
	ns, err := couchcore.ExistingCouchNamespace(ref.Store)
	if err != nil {
		return err
	}
	return couchcore.ForgetStoreArchiveReceipt(ns, r.Coordinator, held, archiveRequest(id, ref))
}

// Onboard is apply-only and bounded by the collector's selected owner batch.
func (r CouchReferences) Onboard(ctx context.Context, held *storagegc.Locked, paths []string, owners []artifactpath.StorageOwner) error {
	if r.Coordinator == nil {
		return errors.New("missing Pair coordinator")
	}
	if len(owners) == 0 {
		return nil
	}
	addresses := make([]couchcore.ThreadAddress, 0, len(owners))
	for _, owner := range owners {
		expected, err := artifactpath.NewStorageOwner(r.Coordinator.Root, owner.RepoScope, owner.Tag)
		if err != nil {
			return err
		}
		if expected != owner {
			return errors.New("archive onboarding owner belongs to another root")
		}
		// Flat standalone owners share the Pair root but cannot address Couch
		// archives. Keep owner validation above this scope filter.
		if owner.RepoScope == "" {
			continue
		}
		addresses = append(addresses, couchcore.ThreadAddress{RepoScope: owner.RepoScope, Tag: couchcore.ThreadTag(owner.Tag)})
	}
	if len(addresses) == 0 {
		return nil
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		ns, err := couchcore.ExistingCouchNamespace(path)
		if err != nil {
			return err
		}
		if err := couchcore.OnboardStoreArchiveGrace(ctx, ns, r.Coordinator, held, addresses); err != nil {
			return err
		}
	}
	return nil
}

// Recover settles pending Couch journals before apply's read-only inventory.
func (r CouchReferences) Recover(ctx context.Context, held *storagegc.Locked, paths []string) error {
	if r.Coordinator == nil {
		return errors.New("missing Pair coordinator")
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		ns, err := couchcore.ExistingCouchNamespace(path)
		if err != nil {
			return err
		}
		if err := couchcore.RecoverStoreRetention(ctx, ns, r.Coordinator, held); err != nil {
			return err
		}
	}
	return nil
}
