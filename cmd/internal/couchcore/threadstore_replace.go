package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"reflect"
)

// ReplaceThreadExpected retires old and publishes next in one store journal.
// Readable sources require the exact observed revision, the archivable-record
// guard, and the authority that says why old may go (pair#695): a teardown
// proof at that revision, whose fold already cleared a park or start claim
// its dead owners left. The one exception is FreshLeavesRemnants, a confirmed
// Fresh whose source could not be proven down: it replaces over remnants and
// their bookkeeping by the operator's choice. A Fresh intent must carry over
// unchanged to the new record and its start claim. Replacement signals no
// process and grants no permission to reuse a native conversation.
func (s *ThreadStore) ReplaceThreadExpected(old ThreadAddress, expectedRevision uint64, next ThreadRecord, authority RetireAuthority) error {
	if s.layout.Local {
		return errors.New("a slot store replaces its current record through replaceSlotCurrent")
	}
	if err := validateThreadAddress(old); err != nil {
		return err
	}
	next = cloneThreadRecord(next)
	if err := validateThreadAddress(next.Address); err != nil {
		return err
	}
	if err := ValidateThreadRecord(next); err != nil {
		return err
	}
	if err := s.validateLocalOrigin(next); err != nil {
		return err
	}
	oldBackend, err := s.storeForAddress(old)
	if err != nil {
		return err
	}
	nextBackend, err := s.storeForPath(next.StartingPath, next.Address.RepoScope, "")
	if err != nil {
		return err
	}
	// Both halves must land in this one journal. A record routed anywhere else
	// -- a slot store above all -- would split the replace across two
	// journals, which is the crash window this method exists to close.
	if oldBackend != s || nextBackend != s {
		return fmt.Errorf("replace of %s by %s spans two thread stores; refusing", old.Tag, next.Address.Tag)
	}
	return s.withLock(func() error {
		manifest, manifestRaw, manifestExists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		raw, exists, err := s.readOptionalPayload(s.recordPath(old))
		if err != nil {
			return err
		}
		if !exists || !manifestContains(manifest, old) {
			return fmt.Errorf("%w: %+v", ErrThreadNotFound, old)
		}
		record, err := s.decodeThreadRaw(old, raw)
		if err != nil {
			return err
		}
		if record.Revision != expectedRevision {
			return &ThreadRevisionError{Address: old, Want: expectedRevision, Got: record.Revision}
		}
		if record.RecoveryIntent != nil && record.RecoveryIntent.Kind() == threadrecord.RecoveryFresh {
			if next.RecoveryIntent == nil {
				return errors.New("fresh replacement lacks retained source selection")
			}
			expected := threadrecord.CloneRecoveryIntent(record.RecoveryIntent)
			expected.Address, expected.SourceRevision = toPersistedThreadAddress(next.Address), next.Revision
			if !reflect.DeepEqual(expected, next.RecoveryIntent) || len(next.Incarnations) != 1 || next.Incarnations[0].Start == nil || !reflect.DeepEqual(expected, next.Incarnations[0].Start.RecoveryIntent) {
				return errors.New("fresh replacement selection changed")
			}
		}
		if _, remnants := authority.(FreshLeavesRemnants); !remnants {
			if err := archivableRecord(record); err != nil {
				return err
			}
		}
		if err := authorizeRetire(authority, old, &record, raw, &next); err != nil {
			return err
		}
		if _, exists, err := s.readOptionalPayload(s.recordPath(next.Address)); err != nil {
			return err
		} else if exists || manifestContains(manifest, next.Address) {
			return &ThreadExistsError{Address: next.Address}
		}
		kept, err := retiredBytes(authority, &record, raw)
		if err != nil {
			return err
		}
		entries, err := s.archiveJournalEntries(old, raw, kept)
		if err != nil {
			return err
		}
		created, err := s.createJournalEntries(next)
		if err != nil {
			return err
		}
		entries = append(entries, created...)
		nextManifest := manifest
		nextManifest.Generation++
		nextManifest.Threads = append(removeThreadAddress(nextManifest.Threads, old), next.Address)
		sortThreadAddresses(nextManifest.Threads)
		nextManifestRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		nextManifestRaw = append(nextManifestRaw, '\n')
		var expectedManifest *[]byte
		if manifestExists {
			copy := append([]byte{}, manifestRaw...)
			expectedManifest = &copy
		}
		entries = append(entries, storeJournalEntry{Path: relativeStorePath(s.root, s.manifestPath()), Expected: expectedManifest, After: &nextManifestRaw})
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: entries})
	})
}
