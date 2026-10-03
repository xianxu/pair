package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ReplaceThreadExpected retires old and publishes next in ONE store journal --
// the main store's counterpart of the slot store's replaceSlotCurrent, so a :0
// reboot and a :1+ reboot share one crash-safe shape. A crash before the
// journal is durable changes nothing; a crash after it is completed by replay.
// There is never a moment where the path holds both records, or neither.
//
// It is archiveThread and CreateThread composed rather than restated: the same
// journal-entry builders, the same record guard (archivableRecord) and the same
// routing for next (storeForPath, as CreateThread routes it). The one thing it
// adds is that the manifest is edited once, for both halves.
//
// old must be readable at exactly expectedRevision: an unreadable :0 record has
// no path to start in, so reboot archives it alone (DecideReboot) and never
// replaces it.
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
		if err := archivableRecord(record); err != nil {
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
