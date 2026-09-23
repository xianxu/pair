package couchcore

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// ArchiveDetachRequest is the root transaction's immutable archive identity.
// OperationID is unique per root collection and survives cross-store recovery.
type ArchiveDetachRequest struct {
	SlotEnvironment string        `json:"slot_environment,omitempty"`
	OperationID     string        `json:"operation_id"`
	Address         ThreadAddress `json:"address"`
	RecordHash      string        `json:"record_hash"`
	ArchivedAt      time.Time     `json:"archived_at"`
}

var archiveOperationPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func (r ArchiveDetachRequest) validate() error {
	if r.SlotEnvironment != "" && !workspaceAbsolute(r.SlotEnvironment) {
		return errors.New("invalid archive slot locator")
	}
	if !archiveOperationPattern.MatchString(r.OperationID) || r.ArchivedAt.IsZero() {
		return errors.New("invalid archive detach operation")
	}
	hash, err := hex.DecodeString(r.RecordHash)
	if err != nil || len(hash) != 32 || strings.ToLower(r.RecordHash) != r.RecordHash {
		return errors.New("invalid archive identity hash")
	}
	return validateThreadAddress(r.Address)
}
func (r ArchiveDetachRequest) matches(other ArchiveDetachRequest) bool {
	return r.SlotEnvironment == other.SlotEnvironment && r.OperationID == other.OperationID && r.Address == other.Address && r.RecordHash == other.RecordHash && r.ArchivedAt.Equal(other.ArchivedAt)
}
func (s *ThreadStore) archiveReceiptPath(operationID string) string {
	return filepath.Join(s.root, "gc-receipts", operationID+".json")
}

func (s *ThreadStore) withRetentionWrite(held *storagegc.Locked, fn func() error) error {
	if s == nil || s.coordinator == nil || !held.Writable(s.coordinator.Root) {
		return errors.New("archive mutation requires this root's writable lock")
	}
	if err := held.CheckContext(); err != nil {
		return err
	}
	if err := held.RegisterStore(s.namespace.Dir()); err != nil {
		return err
	}
	return retentionLockError(s.withStoreLockChecked(fn, held.CheckContext))
}
func (s *ThreadStore) readArchiveReceiptLocked(request ArchiveDetachRequest) ([]byte, bool, error) {
	raw, err := s.readRetentionFile(s.archiveReceiptPath(request.OperationID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var receipt ArchiveDetachRequest
	if err := strictThreadStoreJSON(raw, &receipt); err != nil {
		return nil, false, err
	}
	if err := receipt.validate(); err != nil {
		return nil, false, err
	}
	if !receipt.matches(request) {
		return nil, false, errors.New("archive receipt belongs to a different operation identity")
	}
	return raw, true, nil
}

// DetachArchive writes its receipt before removing archive and grace through one
// store journal. Recovery completes that journal before accepting any replay.
// A matching receipt is completion evidence; replay never inspects a newer
// archive at the same address after the original operation finished.
func (s *ThreadStore) DetachArchive(held *storagegc.Locked, request ArchiveDetachRequest) error {
	if !s.layout.Local {
		if err := request.validate(); err != nil {
			return err
		}
		if err := s.withRetentionWrite(held, func() error { return nil }); err != nil {
			return err
		}
		backend, err := s.retentionArchiveBackend(held, request)
		if err != nil {
			return err
		}
		if backend != s {
			return backend.DetachArchive(held, request)
		}
	}
	if s.layout.Local && (s.slot == nil || request.SlotEnvironment != s.slot.EnvironmentRoot) {
		return errors.New("archive request backing store mismatch")
	}
	if err := request.validate(); err != nil {
		return err
	}
	return s.withRetentionWrite(held, func() error {
		if s.layout.Local {
			if err := s.requireLocalRetentionCurrent(); err != nil {
				return err
			}
		}
		if _, exists, err := s.readArchiveReceiptLocked(request); err != nil {
			return err
		} else if exists {
			return nil
		}
		manifest, _, _, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		if manifestContains(manifest, request.Address) {
			return errors.New("visible thread cannot be detached from archive")
		}
		evidence, err := s.readArchiveGraceLocked(request.Address)
		if err != nil {
			return err
		}
		if evidence.RecordHash != request.RecordHash || !evidence.ArchivedAt.Equal(request.ArchivedAt) {
			return errors.New("archive changed since root transaction inventory")
		}
		raw, err := s.readRetentionFile(s.archivePath(request.Address))
		if err != nil {
			return err
		}
		grace, err := s.readRetentionFile(s.archiveGracePath(request.Address))
		if err != nil {
			return err
		}
		receipt, err := json.Marshal(request)
		if err != nil {
			return err
		}
		return s.commitJournalLockedChecked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
			{Path: relativeStorePath(s.root, s.archiveReceiptPath(request.OperationID)), After: &receipt},
			{Path: relativeStorePath(s.root, s.archivePath(request.Address)), Expected: &raw},
			{Path: relativeStorePath(s.root, s.archiveGracePath(request.Address)), Expected: &grace},
		}}, held.CheckContext)
	})
}

// ForgetArchiveReceipt is called only after durable root finalization. It
// removes exact receipt bytes and never touches the reusable archive address.
func (s *ThreadStore) ForgetArchiveReceipt(held *storagegc.Locked, request ArchiveDetachRequest) error {
	if !s.layout.Local {
		if err := request.validate(); err != nil {
			return err
		}
		if err := s.withRetentionWrite(held, func() error { return nil }); err != nil {
			return err
		}
		backend, err := s.retentionArchiveBackend(held, request)
		if err != nil {
			return err
		}
		if backend != s {
			return backend.ForgetArchiveReceipt(held, request)
		}
	}
	if s.layout.Local && (s.slot == nil || request.SlotEnvironment != s.slot.EnvironmentRoot) {
		return errors.New("archive request backing store mismatch")
	}
	if err := request.validate(); err != nil {
		return err
	}
	return s.withRetentionWrite(held, func() error {
		if s.layout.Local {
			if err := s.requireLocalRetentionCurrent(); err != nil {
				return err
			}
		}
		raw, exists, err := s.readArchiveReceiptLocked(request)
		if err != nil || !exists {
			return err
		}
		return s.commitJournalLockedChecked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{{Path: relativeStorePath(s.root, s.archiveReceiptPath(request.OperationID)), Expected: &raw}}}, held.CheckContext)
	})
}

// DetachStoreArchive adapts a registered namespace without reacquiring the root
// lock. Namespace construction/registration must precede a collection journal.
func DetachStoreArchive(namespace CouchNamespace, c *storagegc.Coordinator, held *storagegc.Locked, request ArchiveDetachRequest) error {
	if namespace.Dir() == "" || c == nil {
		return fmt.Errorf("archive adapter needs namespace and coordinator")
	}
	store := NewThreadStore(namespace)
	store.coordinator = c
	return store.DetachArchive(held, request)
}
func ForgetStoreArchiveReceipt(namespace CouchNamespace, c *storagegc.Coordinator, held *storagegc.Locked, request ArchiveDetachRequest) error {
	if namespace.Dir() == "" || c == nil {
		return fmt.Errorf("archive adapter needs namespace and coordinator")
	}
	store := NewThreadStore(namespace)
	store.coordinator = c
	return store.ForgetArchiveReceipt(held, request)
}
