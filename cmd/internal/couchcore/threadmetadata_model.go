package couchcore

// ThreadMetadataPatch distinguishes an omitted field from an explicit empty
// value. Empty name/description/summary clears only that field.
type ThreadMetadataPatch struct {
	Name             *string
	Description      *string
	PublishedSummary *string
}

// ApplyThreadMetadata is the pure metadata transition. Revision ownership
// remains in ThreadStore.UpdateExistingThread, which applies this transition
// only after its expected-revision comparison succeeds.
func ApplyThreadMetadata(record ThreadRecord, patch ThreadMetadataPatch) ThreadRecord {
	next := cloneThreadRecord(record)
	if patch.Name != nil {
		next.Name = *patch.Name
	}
	if patch.Description != nil {
		next.Description = *patch.Description
	}
	if patch.PublishedSummary != nil {
		next.PublishedSummary = *patch.PublishedSummary
	}
	return next
}

func threadHasMetadata(record ThreadRecord) bool {
	return record.Name != "" || record.Description != "" || record.PublishedSummary != ""
}
