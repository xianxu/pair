package couchcore

import (
	"encoding/json"
	"fmt"
)

const legacyMigrationVersion = 1

// MigrateLegacyRecord enriches one M1 cutover record from the old tree-keyed
// naming table. It is deliberately pure: the ThreadStore owns revision and
// journal publication after every candidate has been validated together.
// pair:m5-concept pure
func MigrateLegacyRecord(record ThreadRecord, legacy NameEntry) (ThreadRecord, bool, error) {
	if err := ValidateThreadRecord(record); err != nil {
		return ThreadRecord{}, false, err
	}
	next := cloneThreadRecord(record)
	hasLegacyIncarnation := false
	for _, incarnation := range next.Incarnations {
		if incarnation.LegacyActorID != "" {
			hasLegacyIncarnation = true
			break
		}
	}
	if !hasLegacyIncarnation || legacy.Tree == "" {
		return next, false, nil
	}
	if string(legacy.Tree) != next.StartingPath {
		return ThreadRecord{}, false, fmt.Errorf("legacy metadata tree %q does not match thread path %q", legacy.Tree, next.StartingPath)
	}
	changed := false
	if next.Name == "" && legacy.Name != "" {
		next.Name = legacy.Name
		changed = true
	}
	if next.Description == "" && legacy.Description != "" {
		next.Description = legacy.Description
		changed = true
	}
	if err := ValidateThreadRecord(next); err != nil {
		return ThreadRecord{}, false, err
	}
	return next, changed, nil
}

// MigrateLegacyRecords atomically enriches every M1 cutover record and marks
// the manifest only after all addressed records have decoded and validated.
// The legacy registry is an input owned by Store and is never a journal target.
// pair:m5-concept integration
func (s *ThreadStore) MigrateLegacyRecords(names NamingTable) error {
	return s.withLock(func() error {
		manifest, manifestRaw, manifestExists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		if manifest.LegacyMigrationVersion >= legacyMigrationVersion {
			return nil
		}
		entries := make([]storeJournalEntry, 0, len(manifest.Threads)+1)
		for _, address := range manifest.Threads {
			raw, exists, err := readOptionalFile(s.recordPath(address))
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("legacy migration manifest record %+v is missing", address)
			}
			record, err := s.decodeThreadRaw(address, raw)
			if err != nil {
				return err
			}
			next, changed, err := MigrateLegacyRecord(record, names.Entry(Worktree(record.StartingPath)))
			if err != nil {
				return err
			}
			if !changed {
				continue
			}
			next.Revision++
			if err := ValidateThreadRecord(next); err != nil {
				return err
			}
			nextRaw, err := json.MarshalIndent(toPersistedThreadRecord(next), "", "  ")
			if err != nil {
				return err
			}
			expected := append([]byte(nil), raw...)
			after := append(nextRaw, '\n')
			entries = append(entries, storeJournalEntry{
				Path: relativeStorePath(s.root, s.recordPath(address)), Expected: &expected, After: &after,
			})
		}
		nextManifest := manifest
		nextManifest.SchemaVersion = 1
		nextManifest.Generation++
		nextManifest.LegacyMigrationVersion = legacyMigrationVersion
		nextManifestRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		var expectedManifest *[]byte
		if manifestExists {
			copy := append([]byte(nil), manifestRaw...)
			expectedManifest = &copy
		}
		afterManifest := append(nextManifestRaw, '\n')
		entries = append(entries, storeJournalEntry{
			Path: relativeStorePath(s.root, s.manifestPath()), Expected: expectedManifest, After: &afterManifest,
		})
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: entries})
	})
}
