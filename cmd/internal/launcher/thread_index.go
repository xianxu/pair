package launcher

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

var (
	ErrThreadIndexReferenceNotFound = errors.New("thread reference not found")
	// ErrThreadIndexAbsent is the only read failure that permits standalone
	// Pair to use its legacy tag behavior. Once a Couch manifest exists it is
	// authoritative: malformed or incomplete state must fail closed.
	ErrThreadIndexAbsent = errors.New("thread index absent")
)

type ThreadIndexAddress struct {
	RepoScope string `json:"repo_scope"`
	Tag       string `json:"tag"`
}

// ThreadIndexEntry is Pair's portable, read-only projection of the durable
// thread record. Couch owns lifecycle details; standalone Pair needs only the
// identity and human-facing fields used for resolution and picker display.
type ThreadIndexEntry struct {
	Address          ThreadIndexAddress
	WorkingPath      string
	CreatedAt        time.Time
	Name             string
	Description      string
	PublishedSummary string
}

type ThreadIndex struct{ Entries []ThreadIndexEntry }

type AmbiguousThreadIndexReferenceError struct {
	Reference  string
	Candidates []ThreadIndexAddress
}

func (e *AmbiguousThreadIndexReferenceError) Error() string {
	return fmt.Sprintf("thread reference %q matches %d threads", e.Reference, len(e.Candidates))
}

type threadIndexManifest struct {
	SchemaVersion int                  `json:"schema_version"`
	Generation    uint64               `json:"generation"`
	Threads       []ThreadIndexAddress `json:"threads"`
	LegacyCutover bool                 `json:"legacy_cutover,omitempty"`
}

var threadIndexRecordValidators = threadrecord.Validators{
	RepoScope: ValidateRepoScopeKey,
	Tag:       ValidatePairTag,
}

// LoadThreadIndex reads one atomic manifest snapshot and its addressed records.
// It performs no recovery writes: an incomplete/corrupt store fails closed and
// the next Couch owner remains responsible for journal recovery.
func LoadThreadIndex(couchStoreDir string, readFile func(string) (string, error)) (ThreadIndex, error) {
	if readFile == nil {
		return ThreadIndex{}, errors.New("thread index has no reader")
	}
	root := filepath.Join(couchStoreDir, "threadstore")
	raw, err := readFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return ThreadIndex{}, err
	}
	var manifest threadIndexManifest
	if err := strictjson.Decode([]byte(raw), &manifest); err != nil {
		return ThreadIndex{}, fmt.Errorf("decode thread manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.Threads == nil {
		return ThreadIndex{}, fmt.Errorf("unsupported thread manifest schema %d", manifest.SchemaVersion)
	}
	index := ThreadIndex{Entries: make([]ThreadIndexEntry, 0, len(manifest.Threads))}
	seen := map[ThreadIndexAddress]bool{}
	for _, address := range manifest.Threads {
		if err := validateThreadIndexAddress(address); err != nil {
			return ThreadIndex{}, err
		}
		if seen[address] {
			return ThreadIndex{}, fmt.Errorf("duplicate thread address %+v", address)
		}
		seen[address] = true
		path := filepath.Join(root, "records", address.RepoScope, address.Tag+".json")
		recordRaw, err := readFile(path)
		if err != nil {
			return ThreadIndex{}, fmt.Errorf("read thread record %+v: %w", address, err)
		}
		record, err := threadrecord.DecodePersisted([]byte(recordRaw), threadrecord.Address{
			RepoScope: address.RepoScope, Tag: address.Tag,
		}, threadIndexRecordValidators)
		if err != nil {
			return ThreadIndex{}, fmt.Errorf("decode thread record %+v: %w", address, err)
		}
		index.Entries = append(index.Entries, ThreadIndexEntry{
			Address: address, WorkingPath: record.WorkingPath, CreatedAt: record.CreatedAt,
			Name: record.Name, Description: record.Description, PublishedSummary: record.PublishedSummary,
		})
	}
	return index, nil
}

func validateThreadIndexAddress(address ThreadIndexAddress) error {
	if err := ValidateRepoScopeKey(address.RepoScope); err != nil {
		return fmt.Errorf("invalid thread repo scope %q: %w", address.RepoScope, err)
	}
	if err := ValidatePairTag(address.Tag); err != nil {
		return fmt.Errorf("invalid thread tag %q: %w", address.Tag, err)
	}
	return nil
}

// ResolveThreadIndexReference is the one portable name/tag/path matcher used
// by standalone Pair and Couch's richer ThreadRecord adapter.
func ResolveThreadIndexReference(entries []ThreadIndexEntry, repoScope, ref string) ([]ThreadIndexEntry, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("%w: empty reference", ErrThreadIndexReferenceNotFound)
	}
	eligible := make([]ThreadIndexEntry, 0, len(entries))
	for _, entry := range entries {
		if repoScope == "" || entry.Address.RepoScope == repoScope {
			eligible = append(eligible, entry)
		}
	}
	exact := make([]ThreadIndexEntry, 0, 1)
	for _, entry := range eligible {
		if entry.Address.Tag == ref {
			exact = append(exact, entry)
		}
	}
	if len(exact) > 0 {
		return finishThreadIndexReference(ref, exact)
	}
	needle := strings.ToLower(ref)
	matches := make([]ThreadIndexEntry, 0, len(eligible))
	for _, entry := range eligible {
		if strings.Contains(strings.ToLower(entry.Name), needle) ||
			strings.Contains(strings.ToLower(entry.WorkingPath), needle) {
			matches = append(matches, entry)
		}
	}
	return finishThreadIndexReference(ref, matches)
}

func finishThreadIndexReference(ref string, matches []ThreadIndexEntry) ([]ThreadIndexEntry, error) {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Address.RepoScope != matches[j].Address.RepoScope {
			return matches[i].Address.RepoScope < matches[j].Address.RepoScope
		}
		return matches[i].Address.Tag < matches[j].Address.Tag
	})
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: %q", ErrThreadIndexReferenceNotFound, ref)
	case 1:
		return matches, nil
	default:
		candidates := make([]ThreadIndexAddress, len(matches))
		for i := range matches {
			candidates[i] = matches[i].Address
		}
		return matches, &AmbiguousThreadIndexReferenceError{Reference: ref, Candidates: candidates}
	}
}

// MergeThreadHistory overlays human names and adds durable parked threads that
// have no recently touched sidecar. Existing history order is preserved.
func MergeThreadHistory(history []HistoricalTag, index ThreadIndex, scope RepoScope) []HistoricalTag {
	out := append([]HistoricalTag(nil), history...)
	byTag := make(map[string]int, len(out))
	for i := range out {
		byTag[out[i].Tag] = i
	}
	for _, entry := range index.Entries {
		if entry.Address.RepoScope != scope.Key {
			continue
		}
		if i, ok := byTag[entry.Address.Tag]; ok {
			out[i].Name = entry.Name
			if out[i].RepoName == "" {
				out[i].RepoName = scope.DisplayName
			}
			continue
		}
		byTag[entry.Address.Tag] = len(out)
		out = append(out, HistoricalTag{
			Tag: entry.Address.Tag, Name: entry.Name, MTime: entry.CreatedAt, RepoName: scope.DisplayName,
		})
	}
	return out
}

func DecorateThreadSessions(sessions []Session, index ThreadIndex, scope RepoScope) []Session {
	out := append([]Session(nil), sessions...)
	names := map[string]string{}
	for _, entry := range index.Entries {
		if entry.Address.RepoScope == scope.Key {
			names[entry.Address.Tag] = entry.Name
		}
	}
	for i := range out {
		out[i].ThreadName = names[sessionTag(out[i])]
	}
	return out
}
