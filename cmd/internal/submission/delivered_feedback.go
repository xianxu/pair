package submission

import (
	"fmt"
	"os"
)

// DeliveredFeedback contains exact confirmed versions for one recording. The
// caller owns both maps; no returned state participates in Store authority.
type DeliveredFeedback struct {
	RecordingID string
	Sketches    map[PreviewItem]struct{}
	Notes       map[PreviewNote]struct{}
}

// DeliveredFeedbackVersions scans existing durable manifests across targets.
// Prepared/uncertain deliveries never suppress evidence. Every metadata entry
// is validated; any error returns no partial suppression set.
func (s *Store) DeliveredFeedbackVersions(recordingID string) (DeliveredFeedback, error) {
	if recordingID == "" {
		return DeliveredFeedback{}, fmt.Errorf("recording identity required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	root, err := s.open()
	if err != nil {
		return DeliveredFeedback{}, err
	}
	defer root.Close()
	entries, err := submissionEntries(root)
	if err != nil {
		return DeliveredFeedback{}, err
	}
	out := DeliveredFeedback{Sketches: map[PreviewItem]struct{}{}, Notes: map[PreviewNote]struct{}{}}
	for _, entry := range entries {
		if !identifier.MatchString(entry.Name()) {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return DeliveredFeedback{}, fmt.Errorf("symlink in submission store")
		}
		if !entry.IsDir() {
			continue
		} // Metadata and staging residue are not bundles.
		sub, err := readMetadata(root, entry.Name(), true)
		if err != nil {
			return DeliveredFeedback{}, err
		}
		if sub.Source.RecordingID != recordingID || sub.Delivery != Dispatched && sub.Delivery != Resolved {
			continue
		}
		// A disposition renamed just before an interrupted fsync is visible but not
		// yet confirmed. Repeating directory confirmation makes it safe to suppress.
		if err = s.confirmPublication(root, sub.ID); err != nil {
			return DeliveredFeedback{}, err
		}
		for _, note := range sub.Notes {
			out.Notes[PreviewNote{ID: note.ID, Revision: note.Revision}] = struct{}{}
		}
		for _, sketch := range sub.Sketches {
			out.Sketches[PreviewItem{RecordingID: sketch.RecordingID, SketchID: sketch.ID, Sequence: sketch.NextSequence, Digest: sketch.Digest}] = struct{}{}
		}
	}
	out.RecordingID = recordingID
	return out, nil
}
