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

// ResolveThreadReference answers "which thread is this", and it sees the
// UNREADABLE set too.
//
// The rule the projections established -- a record the decoder rejects still
// produces a visible row -- has to hold for every consumer that answers whether
// a thread exists, or the surfaces disagree: `couch --list` printed a row and
// `couch --show <tag>` said "thread reference not found" about the very same
// thread. Archive was fixed once by adding a second resolver that bypasses
// decoding, which is a per-consumer patch where a shared rule belongs.
//
// An unreadable record participates by ADDRESS only. Its tag can be matched;
// its path and name cannot, because reading them is what failed.
func (c *Couch) ResolveThreadReference(repoScope, ref string) ([]ThreadRecord, error) {
	snapshot, err := c.Threads.Snapshot()
	if err != nil {
		return nil, err
	}
	records := snapshot.Records
	for _, address := range snapshot.Unreadable {
		records = append(records, ThreadRecord{Address: address, Reservation: true})
	}
	return ResolveThreadReference(records, repoScope, ref)
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

// ThreadReferenceFields is the complete shared matching surface for one
// thread. It contains values only, so CLI resolution and in-memory menu
// filtering can consume the same rule without store access.
type ThreadReferenceFields struct {
	Address     ThreadAddress
	Name        string
	WorkingPath string
}

// ThreadReferenceMatch orders match strength. Zero is deliberately no match.
type ThreadReferenceMatch uint8

const (
	ThreadReferenceNone ThreadReferenceMatch = iota
	ThreadReferenceFuzzy
	ThreadReferenceExact
)

func (e *AmbiguousThreadReferenceError) Error() string {
	return fmt.Sprintf("thread reference %q matches %d threads", e.Reference, len(e.Candidates))
}

// ClassifyThreadReferenceFields applies the shared per-row rule. Exact opaque
// tag equality is stronger than case-insensitive name/path containment.
func ClassifyThreadReferenceFields(fields ThreadReferenceFields, ref string) (ThreadReferenceMatch, error) {
	normalized, err := normalizeThreadReference(ref)
	if err != nil {
		return ThreadReferenceNone, err
	}
	return classifyNormalizedThreadReferenceFields(fields, normalized), nil
}

// MatchThreadReferenceFields applies exact-over-fuzzy precedence across the
// complete set and returns composite identities in deterministic order.
func MatchThreadReferenceFields(fields []ThreadReferenceFields, ref string) ([]ThreadAddress, error) {
	normalized, err := normalizeThreadReference(ref)
	if err != nil {
		return nil, err
	}
	exact := make([]ThreadAddress, 0, 1)
	fuzzy := make([]ThreadAddress, 0, len(fields))
	for _, candidate := range fields {
		switch classifyNormalizedThreadReferenceFields(candidate, normalized) {
		case ThreadReferenceExact:
			exact = append(exact, candidate.Address)
		case ThreadReferenceFuzzy:
			fuzzy = append(fuzzy, candidate.Address)
		}
	}
	matches := fuzzy
	if len(exact) > 0 {
		matches = exact
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %q", ErrThreadReferenceNotFound, normalized)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].RepoScope != matches[j].RepoScope {
			return matches[i].RepoScope < matches[j].RepoScope
		}
		return matches[i].Tag < matches[j].Tag
	})
	return matches, nil
}

func normalizeThreadReference(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("%w: empty reference", ErrThreadReferenceNotFound)
	}
	if strings.ContainsRune(ref, '\x00') {
		return "", fmt.Errorf("%w: malformed reference", ErrThreadReferenceNotFound)
	}
	return ref, nil
}

func classifyNormalizedThreadReferenceFields(fields ThreadReferenceFields, normalized string) ThreadReferenceMatch {
	if fields.Address.Tag == ThreadTag(normalized) {
		return ThreadReferenceExact
	}
	needle := strings.ToLower(normalized)
	if strings.Contains(strings.ToLower(fields.Name), needle) ||
		strings.Contains(strings.ToLower(fields.WorkingPath), needle) {
		return ThreadReferenceFuzzy
	}
	return ThreadReferenceNone
}

// ResolveThreadReference resolves within repoScope when it is non-empty.
// Exact tag equality is authoritative. Otherwise name and canonical working
// path match case-insensitively by substring. Ambiguity is returned with every
// candidate and never collapsed to an arbitrary winner.
func ResolveThreadReference(records []ThreadRecord, repoScope, ref string) ([]ThreadRecord, error) {
	normalized, err := normalizeThreadReference(ref)
	if err != nil {
		return nil, err
	}
	eligible := make([]ThreadRecord, 0, len(records))
	fields := make([]ThreadReferenceFields, 0, len(records))
	for _, record := range records {
		if repoScope == "" || record.Address.RepoScope == repoScope {
			eligible = append(eligible, record)
			fields = append(fields, ThreadReferenceFields{
				Address:     record.Address,
				Name:        record.Name,
				WorkingPath: record.WorkingPath,
			})
		}
	}
	addresses, err := MatchThreadReferenceFields(fields, normalized)
	if err != nil {
		return nil, err
	}
	wanted := make(map[ThreadAddress]bool, len(addresses))
	for _, address := range addresses {
		wanted[address] = true
	}
	matches := make([]ThreadRecord, 0, len(addresses))
	for _, record := range eligible {
		if wanted[record.Address] {
			matches = append(matches, record)
		}
	}
	return finishThreadReference(normalized, matches)
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
