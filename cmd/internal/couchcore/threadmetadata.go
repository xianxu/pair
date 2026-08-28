package couchcore

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrThreadReferenceNotFound = errors.New("thread reference not found")

// ApplyThreadMetadata performs the composite-address revision CAS and changes
// only the fields named by patch.
func (s *ThreadStore) ApplyThreadMetadata(address ThreadAddress, expectedRevision uint64, patch ThreadMetadataPatch) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, expectedRevision, func(record *ThreadRecord) error {
		*record = ApplyThreadMetadata(*record, patch)
		return nil
	})
}

func (c *Couch) ResolveThreadReference(repoScope, ref string) ([]ThreadRecord, error) {
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return nil, err
	}
	return ResolveThreadReference(snapshot.Records, repoScope, ref)
}

func (c *Couch) ApplyThreadMetadata(address ThreadAddress, patch ThreadMetadataPatch) (ThreadRecord, error) {
	current, err := c.Threads.GetThread(address)
	if err != nil {
		return ThreadRecord{}, err
	}
	return c.Threads.ApplyThreadMetadata(address, current.Revision, patch)
}

type AmbiguousThreadReferenceError struct {
	Reference  string
	Candidates []ThreadAddress
}

func (e *AmbiguousThreadReferenceError) Error() string {
	return fmt.Sprintf("thread reference %q matches %d threads", e.Reference, len(e.Candidates))
}

// ResolveThreadReference resolves within repoScope when it is non-empty.
// Exact tag equality is authoritative. Otherwise name and canonical working
// path match case-insensitively by substring. Ambiguity is returned with every
// candidate and never collapsed to an arbitrary winner.
func ResolveThreadReference(records []ThreadRecord, repoScope, ref string) ([]ThreadRecord, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("%w: empty reference", ErrThreadReferenceNotFound)
	}
	eligible := make([]ThreadRecord, 0, len(records))
	for _, record := range records {
		if repoScope == "" || record.Address.RepoScope == repoScope {
			eligible = append(eligible, record)
		}
	}
	exact := make([]ThreadRecord, 0, 1)
	for _, record := range eligible {
		if record.Address.Tag == ThreadTag(ref) {
			exact = append(exact, record)
		}
	}
	if len(exact) > 0 {
		return finishThreadReference(ref, exact)
	}
	needle := strings.ToLower(ref)
	matches := make([]ThreadRecord, 0, len(eligible))
	for _, record := range eligible {
		if strings.Contains(strings.ToLower(record.Name), needle) ||
			strings.Contains(strings.ToLower(record.WorkingPath), needle) {
			matches = append(matches, record)
		}
	}
	return finishThreadReference(ref, matches)
}

func finishThreadReference(ref string, matches []ThreadRecord) ([]ThreadRecord, error) {
	result := make([]ThreadRecord, len(matches))
	for i := range matches {
		result[i] = cloneThreadRecord(matches[i])
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Address.RepoScope != result[j].Address.RepoScope {
			return result[i].Address.RepoScope < result[j].Address.RepoScope
		}
		return result[i].Address.Tag < result[j].Address.Tag
	})
	switch len(result) {
	case 0:
		return nil, fmt.Errorf("%w: %q", ErrThreadReferenceNotFound, ref)
	case 1:
		return result, nil
	default:
		candidates := make([]ThreadAddress, len(result))
		for i := range result {
			candidates[i] = result[i].Address
		}
		return result, &AmbiguousThreadReferenceError{Reference: ref, Candidates: candidates}
	}
}
