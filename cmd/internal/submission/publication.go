package submission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PublicationState string

const (
	PublicationUnknown   PublicationState = "unknown"
	PublicationAbsent    PublicationState = "absent"
	PublicationConfirmed PublicationState = "confirmed"
)

type PublicationObservation struct {
	State PublicationState
	Path  string
}

// ConfirmPrepared never publishes. It compares immutable evidence and repeats
// durability confirmation, because Prepare may fail after the visible rename.
func (s *Store) ConfirmPrepared(expected Submission, crops map[string][]byte) (PublicationObservation, error) {
	unknown := PublicationObservation{State: PublicationUnknown}
	if err := Validate(expected); err != nil {
		return unknown, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	root, err := s.open()
	if err != nil {
		return unknown, err
	}
	defer root.Close()
	if _, err = root.Lstat(expected.ID); os.IsNotExist(err) {
		return PublicationObservation{State: PublicationAbsent}, nil
	} else if err != nil {
		return unknown, err
	}
	actual, err := readSubmission(root, expected.ID)
	if err != nil {
		return unknown, err
	}
	actual.Delivery = Prepared
	actual.Superseded = false
	expected.Delivery = Prepared
	expected.Superseded = false
	have, err := json.Marshal(actual)
	if err != nil {
		return unknown, err
	}
	want, err := json.Marshal(expected)
	if err != nil {
		return unknown, err
	}
	if !bytes.Equal(have, want) {
		return unknown, fmt.Errorf("published submission differs from prepared evidence")
	}
	files := map[string]bool{}
	for _, note := range expected.Notes {
		if note.Crop.File != "" {
			files[note.Crop.File] = true
		}
	}
	if len(files) != len(crops) {
		return unknown, fmt.Errorf("publication crop set differs")
	}
	for file := range files {
		want, ok := crops[file]
		if !ok {
			return unknown, fmt.Errorf("publication crop missing")
		}
		have, err := readBytes(root, filepath.Join(expected.ID, file), MaxPNGBytes)
		if err != nil {
			return unknown, err
		}
		if !bytes.Equal(have, want) {
			return unknown, fmt.Errorf("publication crop content differs")
		}
	}
	if err = s.confirmPublication(root, expected.ID); err != nil {
		return unknown, err
	}
	return PublicationObservation{State: PublicationConfirmed, Path: s.ManifestPath(expected.ID)}, nil
}
