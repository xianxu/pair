package couchcore

import (
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
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
	entries := make([]launcher.ThreadIndexEntry, 0, len(records))
	byAddress := make(map[launcher.ThreadIndexAddress]ThreadRecord, len(records))
	for _, record := range records {
		address := launcher.ThreadIndexAddress{RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)}
		entries = append(entries, launcher.ThreadIndexEntry{
			Address: address, WorkingPath: record.WorkingPath, CreatedAt: record.CreatedAt,
			Name: record.Name, Description: record.Description, PublishedSummary: record.PublishedSummary,
		})
		byAddress[address] = cloneThreadRecord(record)
	}
	matches, err := launcher.ResolveThreadIndexReference(entries, repoScope, ref)
	result := make([]ThreadRecord, 0, len(matches))
	for _, match := range matches {
		if record, ok := byAddress[match.Address]; ok {
			result = append(result, cloneThreadRecord(record))
		}
	}
	if err == nil {
		return result, nil
	}
	if errors.Is(err, launcher.ErrThreadIndexReferenceNotFound) {
		return nil, fmt.Errorf("%w: %q", ErrThreadReferenceNotFound, ref)
	}
	var ambiguous *launcher.AmbiguousThreadIndexReferenceError
	if errors.As(err, &ambiguous) {
		candidates := make([]ThreadAddress, len(ambiguous.Candidates))
		for i, address := range ambiguous.Candidates {
			candidates[i] = ThreadAddress{RepoScope: address.RepoScope, Tag: ThreadTag(address.Tag)}
		}
		return result, &AmbiguousThreadReferenceError{Reference: ref, Candidates: candidates}
	}
	return nil, err
}
