package sessioninventory

import (
	"errors"
	"fmt"
)

var ErrArtifactChanged = errors.New("session inventory artifact changed during observation")

// IncrementalResult contains bytes framed only after one artifact generation
// reaches a stable observed EOF.
// pair:156-concept pure new final ScannerState / IncrementalResult
type IncrementalResult struct {
	Fingerprint       ArtifactFingerprint
	RawObservedOffset int64
	FrameState        JSONLFrameState
	Records           []FramedJSONLRecord
	Disputed          bool
}

// ObserveStableArtifact reads from the supplied parser state through a stable
// EOF, following append growth but rejecting replacement, truncation, and
// same-size mutation before returning any records as usable evidence.
func ObserveStableArtifact(runtime Runtime, root StorageRoot, initial FileEntry, frame JSONLFrameState, recordLimit int64) (IncrementalResult, error) {
	result := IncrementalResult{FrameState: JSONLFrameState{ParserCompleteOffset: frame.ParserCompleteOffset, IncompleteTail: append([]byte(nil), frame.IncompleteTail...)}}
	if runtime == nil || initial.Artifact.StorageRoot != root.Name || frame.ParserCompleteOffset < 0 || int64(len(frame.IncompleteTail)) > initial.Size-frame.ParserCompleteOffset {
		result.Disputed = true
		return result, ErrArtifactChanged
	}
	current := initial
	readOffset := frame.ParserCompleteOffset + int64(len(frame.IncompleteTail))
	var records []FramedJSONLRecord
	for {
		for readOffset < current.Size {
			limit := min(readChunkSize, current.Size-readOffset)
			raw, _, err := runtime.ReadAt(initial.Artifact, readOffset, limit)
			if err != nil {
				return result, fmt.Errorf("read incremental session artifact: %w", err)
			}
			if len(raw) == 0 {
				result.Disputed = true
				return result, ErrArtifactChanged
			}
			readOffset += int64(len(raw))
			framed, next, err := FrameJSONLSuffix(result.FrameState, raw, recordLimit)
			if err != nil {
				return result, err
			}
			result.FrameState = next
			records = append(records, framed...)
		}

		observed, err := resampleArtifact(runtime, root, initial.Artifact)
		if err != nil {
			if errors.Is(err, ErrArtifactChanged) {
				result.Disputed = true
			}
			return result, err
		}
		if observed.StableFileID != initial.StableFileID || observed.GenerationToken != initial.GenerationToken || observed.Size < current.Size {
			result.Disputed = true
			return result, ErrArtifactChanged
		}
		if observed.Size == current.Size {
			if observed.MutationToken != current.MutationToken || !equalOptionalTime(observed.BirthTime, current.BirthTime) || !equalOptionalTime(observed.ModTime, current.ModTime) {
				result.Disputed = true
				return result, ErrArtifactChanged
			}
			result.Fingerprint = fingerprintFromEntry(observed)
			result.RawObservedOffset = observed.Size
			result.Records = records
			return result, nil
		}
		if initial.GenerationToken == "" {
			result.Disputed = true
			return result, ErrArtifactChanged
		}
		current = observed
	}
}

func resampleArtifact(runtime Runtime, root StorageRoot, artifact Artifact) (FileEntry, error) {
	files, err := runtime.ListFiles(root)
	if err != nil {
		var listingIssues *ListingIssuesError
		if !errors.As(err, &listingIssues) {
			return FileEntry{}, fmt.Errorf("resample session artifact metadata: %w", err)
		}
	}
	for _, entry := range files {
		if entry.Artifact.StorageRoot == artifact.StorageRoot && entry.Artifact.RelativePath == artifact.RelativePath {
			entry.Artifact.Kind = artifact.Kind
			return entry, nil
		}
	}
	return FileEntry{}, ErrArtifactChanged
}
