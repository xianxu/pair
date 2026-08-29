package sessioninventory

import (
	"encoding/json"
	"errors"
)

// PublishTargetValidations is the one durable advancement boundary shared by
// watchers and interactive owner queries.
func PublishTargetValidations(store CatalogStore, path string, validations []TargetValidation) error {
	merge := func(current Catalog) (Catalog, error) {
		return CatalogWithTargetValidations(current, validations)
	}
	_, err := store.Update(path, merge)
	if errors.Is(err, ErrCatalogCorrupt) {
		_, err = store.Repair(path, merge)
	}
	return err
}

// CatalogWithTargetValidations is the pure publication rule shared by the OS
// store and the stateful fake.
func CatalogWithTargetValidations(current Catalog, validations []TargetValidation) (Catalog, error) {
	entries := make(map[string]CatalogEntry)
	for _, validation := range validations {
		stateRaw, err := json.Marshal(validation.State)
		if err != nil {
			return Catalog{}, err
		}
		for _, observation := range validation.Observations {
			entry := observation.Entry
			fingerprint := fingerprintFromEntry(entry)
			rawOffset, parserOffset := entry.Size, entry.Size
			key := targetArtifactKey(entry.Artifact)
			if result, ok := validation.Results[key]; ok {
				fingerprint = result.Fingerprint
				rawOffset = result.RawObservedOffset
				parserOffset = result.FrameState.ParserCompleteOffset
			}
			entries[catalogEntryKey(observation.Agent, entry.Artifact)] = CatalogEntry{
				Agent: observation.Agent, Artifact: entry.Artifact, Fingerprint: fingerprint,
				Authorization: AuthorizationAuthorized, Facts: []Fact{validation.Fact},
				ScannerSchema: observation.ScannerSchema, ProviderContract: observation.ProviderContract,
				RawObservedOffset: rawOffset, ParserCompleteOffset: parserOffset,
				ScannerState: append(json.RawMessage(nil), stateRaw...),
			}
		}
	}
	byKey := make(map[string]CatalogEntry, len(current.Entries)+len(entries))
	for _, entry := range current.Entries {
		byKey[catalogEntryKey(entry.Agent, entry.Artifact)] = entry
	}
	for key, entry := range entries {
		if existing, ok := byKey[key]; ok {
			byKey[key] = MergeCatalogPublication(existing, entry)
		} else {
			byKey[key] = entry
		}
	}
	current.Entries = current.Entries[:0]
	for _, entry := range byKey {
		current.Entries = append(current.Entries, entry)
	}
	return current, nil
}
