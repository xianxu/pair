package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"reflect"
)

// ReplaceThreadExpected retires old and publishes next in one store journal.
// Readable sources require the exact observed revision. A retained explicit
// Fresh intent permits bookkeeping replacement even when unproved remnants
// remain: the exact intent must appear on both the source and new start claim.
// Ordinary replacement retains the existing archivable-record guard. Neither
// mode signals processes or grants permission to reuse a native conversation.
func (s *ThreadStore) ReplaceThreadExpected(old ThreadAddress, expectedRevision uint64, next ThreadRecord) error {
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
			if record.RecoveryIntent == nil || next.RecoveryIntent == nil || record.RecoveryIntent.Kind() != threadrecord.RecoveryFresh {
				return errors.New("fresh replacement lacks retained source selection")
			}
			expected := threadrecord.CloneRecoveryIntent(record.RecoveryIntent)
			expected.Address, expected.SourceRevision = toPersistedThreadAddress(next.Address), next.Revision
			if !reflect.DeepEqual(expected, next.RecoveryIntent) || len(next.Incarnations) != 1 || next.Incarnations[0].Start == nil || !reflect.DeepEqual(expected, next.Incarnations[0].Start.RecoveryIntent) {
				return errors.New("fresh replacement selection changed")
			}
		} else if err := archivableRecord(record); err != nil {
			return err
		}
		if _, exists, err := s.readOptionalPayload(s.recordPath(next.Address)); err != nil {
			return err
		} else if exists || manifestContains(manifest, next.Address) {
			return &ThreadExistsError{Address: next.Address}
		}
		entries, err := s.archiveJournalEntries(old, raw)
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
