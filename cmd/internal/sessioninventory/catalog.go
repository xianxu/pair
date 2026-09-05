package sessioninventory

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"time"
)

const CatalogVersion = 1

type AuthorizationState string

const (
	AuthorizationUnknown    AuthorizationState = ""
	AuthorizationCandidate  AuthorizationState = "candidate"
	AuthorizationAuthorized AuthorizationState = "authorized"
	AuthorizationDisputed   AuthorizationState = "disputed"
)

// ArtifactFingerprint is the content-free continuity tuple for one observed
// native artifact generation.
// pair:156-concept pure new final
type ArtifactFingerprint struct {
	StableFileID    StableFileID    `json:"stable_file_id"`
	GenerationToken GenerationToken `json:"generation_token,omitempty"`
	MutationToken   MutationToken   `json:"mutation_token"`
	Size            int64           `json:"size"`
	BirthTime       *time.Time      `json:"birth_time,omitempty"`
	ModTime         *time.Time      `json:"mod_time,omitempty"`
}

// GenerationID names WHICH INCARNATION of an artifact path this fingerprint
// describes -- the question a lifecycle record's ArtifactGeneration exists to
// answer, so that a record from a replaced transcript never dedupes against one
// from its predecessor.
//
// GenerationToken is the ideal answer and is almost never available:
// filemeta_linux.go never populates it, filemeta_other.go never populates it,
// and filemeta_darwin.go populates it only from st_gen, which APFS reports as 0
// for unprivileged callers. The field is `omitempty` precisely because it is
// optional -- so a consumer that requires it rejects every artifact on every
// platform pair supports.
//
// The fallback is the identity the platform DOES provide: dev:ino, plus birth
// time when the filesystem records one. A file replaced at a new inode differs;
// a file recreated at a reused inode differs by birth time. The token still
// wins when present, because it additionally distinguishes inode reuse that
// birth time cannot.
func (f ArtifactFingerprint) GenerationID() string {
	if f.GenerationToken != "" {
		return string(f.GenerationToken)
	}
	if f.StableFileID == "" {
		return ""
	}
	id := "file:" + string(f.StableFileID)
	if f.BirthTime != nil {
		id += ":btime:" + strconv.FormatInt(f.BirthTime.UnixNano(), 10)
	}
	return id
}

// CatalogEntry owns the last scanner classification and accepted offsets for
// one native artifact path.
// pair:156-concept pure new final CatalogEntry / Catalog
type CatalogEntry struct {
	Agent                Agent               `json:"agent"`
	Artifact             Artifact            `json:"artifact"`
	Fingerprint          ArtifactFingerprint `json:"fingerprint"`
	Authorization        AuthorizationState  `json:"authorization"`
	Facts                []Fact              `json:"facts,omitempty"`
	ScannerSchema        string              `json:"scanner_schema"`
	ProviderContract     ProviderContract    `json:"provider_contract,omitempty"`
	RawObservedOffset    int64               `json:"raw_observed_offset"`
	ParserCompleteOffset int64               `json:"parser_complete_offset"`
	ScannerState         json.RawMessage     `json:"scanner_state,omitempty"`
}

type Catalog struct {
	Version    int            `json:"version"`
	Generation uint64         `json:"generation"`
	Entries    []CatalogEntry `json:"entries"`
}

func ValidateCatalog(catalog Catalog) error {
	if catalog.Version != CatalogVersion {
		return errors.New("unsupported session inventory catalog version")
	}
	seen := map[string]bool{}
	for _, entry := range catalog.Entries {
		key := catalogEntryKey(entry.Agent, entry.Artifact)
		if !validAgent(entry.Agent) || !validArtifact(entry.Artifact) || seen[key] {
			return errors.New("invalid or duplicate session inventory catalog entry")
		}
		seen[key] = true
		if entry.Fingerprint.StableFileID == "" || entry.Fingerprint.MutationToken == "" || entry.Fingerprint.Size < 0 || entry.ScannerSchema == "" {
			return errors.New("incomplete session inventory artifact fingerprint")
		}
		switch entry.Authorization {
		case AuthorizationCandidate, AuthorizationAuthorized, AuthorizationDisputed:
		default:
			return errors.New("invalid session inventory authorization state")
		}
		if entry.RawObservedOffset < 0 || entry.ParserCompleteOffset < 0 || entry.ParserCompleteOffset > entry.RawObservedOffset || entry.RawObservedOffset > entry.Fingerprint.Size {
			return errors.New("invalid session inventory parser offsets")
		}
		if len(entry.ScannerState) != 0 && !json.Valid(entry.ScannerState) {
			return errors.New("invalid session inventory scanner state")
		}
		if entry.ProviderContract != "" {
			contract, ok := ProviderContractFor(entry.Agent, entry.Artifact.StorageRoot, entry.ScannerSchema)
			if !ok || contract != entry.ProviderContract {
				return errors.New("invalid session inventory provider contract")
			}
		}
	}
	return nil
}

func CloneCatalog(catalog Catalog) Catalog {
	cloned := Catalog{Version: catalog.Version, Generation: catalog.Generation, Entries: make([]CatalogEntry, len(catalog.Entries))}
	for i, entry := range catalog.Entries {
		cloned.Entries[i] = cloneCatalogEntry(entry)
	}
	return cloned
}

func cloneCatalogEntry(entry CatalogEntry) CatalogEntry {
	cloned := entry
	cloned.Fingerprint.BirthTime = cloneStdTime(entry.Fingerprint.BirthTime)
	cloned.Fingerprint.ModTime = cloneStdTime(entry.Fingerprint.ModTime)
	cloned.ScannerState = append(json.RawMessage(nil), entry.ScannerState...)
	cloned.Facts = make([]Fact, len(entry.Facts))
	for i, fact := range entry.Facts {
		cloned.Facts[i] = cloneCatalogFact(fact)
	}
	return cloned
}

func cloneCatalogFact(fact Fact) Fact {
	cloned := fact
	cloned.ParentID = cloneString(fact.ParentID)
	cloned.Time = cloneTime(fact.Time)
	cloned.Artifacts = append([]Artifact(nil), fact.Artifacts...)
	cloned.EdgeProvenance = append([]EdgeProvenance(nil), fact.EdgeProvenance...)
	return cloned
}

// MergeCatalogPublication combines two independently validated publications
// for the same artifact without allowing a late stale writer to lower the
// accepted cursor or erase a fail-closed dispute.
func MergeCatalogPublication(current, incoming CatalogEntry) CatalogEntry {
	if catalogEntryKey(current.Agent, current.Artifact) != catalogEntryKey(incoming.Agent, incoming.Artifact) {
		return cloneCatalogEntry(current)
	}
	if current.Authorization == AuthorizationDisputed {
		return cloneCatalogEntry(current)
	}
	if incoming.Authorization == AuthorizationDisputed {
		return cloneCatalogEntry(incoming)
	}
	if current.Fingerprint.StableFileID != incoming.Fingerprint.StableFileID ||
		(current.Fingerprint.GenerationToken == "") != (incoming.Fingerprint.GenerationToken == "") ||
		(current.Fingerprint.GenerationToken != "" && current.Fingerprint.GenerationToken != incoming.Fingerprint.GenerationToken) ||
		current.ScannerSchema != incoming.ScannerSchema ||
		current.ProviderContract != incoming.ProviderContract {
		return cloneCatalogEntry(current)
	}
	if incoming.RawObservedOffset < current.RawObservedOffset || incoming.ParserCompleteOffset < current.ParserCompleteOffset ||
		(incoming.RawObservedOffset == current.RawObservedOffset && incoming.ParserCompleteOffset == current.ParserCompleteOffset) {
		return cloneCatalogEntry(current)
	}
	return cloneCatalogEntry(incoming)
}

func cloneStdTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func sortedCatalogEntries(entries []CatalogEntry) []CatalogEntry {
	result := make([]CatalogEntry, len(entries))
	for i, entry := range entries {
		result[i] = cloneCatalogEntry(entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return catalogEntryKey(result[i].Agent, result[i].Artifact) < catalogEntryKey(result[j].Agent, result[j].Artifact)
	})
	return result
}

func catalogEntryKey(agent Agent, artifact Artifact) string {
	return string(agent) + "\x00" + artifact.StorageRoot + "\x00" + artifact.RelativePath
}
