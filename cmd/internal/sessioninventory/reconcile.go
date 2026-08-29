package sessioninventory

import "time"

type CatalogWorkKind string

const (
	CatalogWorkNew        CatalogWorkKind = "new"
	CatalogWorkAppend     CatalogWorkKind = "append"
	CatalogWorkRevalidate CatalogWorkKind = "revalidate"
	CatalogWorkDelete     CatalogWorkKind = "delete"
)

// ArtifactObservation is one metadata-only filesystem observation.
// CatalogDelta is the deterministic work/reuse decision against a catalog
// generation.
// pair:156-concept pure new final ArtifactObservation / CatalogDelta
type ArtifactObservation struct {
	Agent            Agent
	Entry            FileEntry
	ScannerSchema    string
	ProviderContract ProviderContract
}

type CatalogWork struct {
	Kind        CatalogWorkKind
	Observation *ArtifactObservation
	Prior       *CatalogEntry
}

type CatalogDelta struct {
	BaseGeneration uint64
	Reused         []CatalogEntry
	Work           []CatalogWork
}

// ReconcileCatalog is pure: it compares metadata and returns bounded work
// without reading content or mutating either input.
// pair:156-concept pure new final
func ReconcileCatalog(catalog Catalog, observations []ArtifactObservation) CatalogDelta {
	priorByKey := make(map[string]CatalogEntry, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		priorByKey[catalogEntryKey(entry.Agent, entry.Artifact)] = cloneCatalogEntry(entry)
	}
	observationByKey := make(map[string]ArtifactObservation, len(observations))
	for _, observation := range observations {
		observationByKey[catalogEntryKey(observation.Agent, observation.Entry.Artifact)] = cloneObservation(observation)
	}

	delta := CatalogDelta{BaseGeneration: catalog.Generation}
	for _, entry := range sortedCatalogEntries(catalog.Entries) {
		key := catalogEntryKey(entry.Agent, entry.Artifact)
		observation, ok := observationByKey[key]
		if !ok {
			prior := cloneCatalogEntry(entry)
			delta.Work = append(delta.Work, CatalogWork{Kind: CatalogWorkDelete, Prior: &prior})
			continue
		}
		delete(observationByKey, key)
		kind, unchanged := catalogObservationKind(entry, observation)
		if unchanged {
			delta.Reused = append(delta.Reused, cloneCatalogEntry(entry))
			continue
		}
		prior := cloneCatalogEntry(entry)
		observed := cloneObservation(observation)
		delta.Work = append(delta.Work, CatalogWork{Kind: kind, Observation: &observed, Prior: &prior})
	}

	keys := make([]string, 0, len(observationByKey))
	for key := range observationByKey {
		keys = append(keys, key)
	}
	sortStrings(keys)
	for _, key := range keys {
		observation := cloneObservation(observationByKey[key])
		delta.Work = append(delta.Work, CatalogWork{Kind: CatalogWorkNew, Observation: &observation})
	}
	return delta
}

func catalogObservationKind(prior CatalogEntry, observation ArtifactObservation) (CatalogWorkKind, bool) {
	current := fingerprintFromEntry(observation.Entry)
	if prior.ScannerSchema != observation.ScannerSchema || prior.ProviderContract != observation.ProviderContract {
		return CatalogWorkRevalidate, false
	}
	if equalFingerprint(prior.Fingerprint, current) {
		return "", true
	}
	if prior.Fingerprint.StableFileID != current.StableFileID || prior.Fingerprint.GenerationToken == "" || current.GenerationToken == "" || prior.Fingerprint.GenerationToken != current.GenerationToken {
		return CatalogWorkRevalidate, false
	}
	if current.Size > prior.Fingerprint.Size && trustedAppendContract(observation) {
		return CatalogWorkAppend, false
	}
	return CatalogWorkRevalidate, false
}

func trustedAppendContract(observation ArtifactObservation) bool {
	contract, ok := ProviderContractFor(observation.Agent, observation.Entry.Artifact.StorageRoot, observation.ScannerSchema)
	return ok && contract == observation.ProviderContract
}

func fingerprintFromEntry(entry FileEntry) ArtifactFingerprint {
	return ArtifactFingerprint{
		StableFileID: entry.StableFileID, GenerationToken: entry.GenerationToken, MutationToken: entry.MutationToken,
		Size: entry.Size, BirthTime: cloneStdTime(entry.BirthTime), ModTime: cloneStdTime(entry.ModTime),
	}
}

func equalFingerprint(left, right ArtifactFingerprint) bool {
	return left.StableFileID == right.StableFileID && left.GenerationToken == right.GenerationToken && left.MutationToken == right.MutationToken && left.Size == right.Size && equalOptionalTime(left.BirthTime, right.BirthTime) && equalOptionalTime(left.ModTime, right.ModTime)
}

func equalOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Equal(*right)
}

func cloneObservation(observation ArtifactObservation) ArtifactObservation {
	cloned := observation
	cloned.Entry.BirthTime = cloneStdTime(observation.Entry.BirthTime)
	cloned.Entry.ModTime = cloneStdTime(observation.Entry.ModTime)
	return cloned
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
