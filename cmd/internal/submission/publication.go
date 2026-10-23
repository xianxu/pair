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

// ObservePublication recovers a publisher's interrupted intent. Only an absent
// bundle directory proves absence; a partial or unreadable bundle is unknown.
// Capture owners still compare the returned evidence with their original claim.
func (s *Store) ObservePublication(id string) (Submission, PublicationObservation, error) {
	unknown := PublicationObservation{State: PublicationUnknown}
	if !ValidID(id) {
		return Submission{}, unknown, fmt.Errorf("invalid submission ID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return Submission{}, unknown, err
	}
	defer r.Close()
	if _, err = r.Lstat(id); os.IsNotExist(err) {
		return Submission{ID: id}, PublicationObservation{State: PublicationAbsent}, nil
	} else if err != nil {
		return Submission{}, unknown, err
	}
	sub, err := readSubmission(r, id)
	if err != nil {
		return Submission{}, unknown, err
	}
	if err = s.confirmPublication(r, id); err != nil {
		return sub, unknown, err
	}
	return sub, PublicationObservation{State: PublicationConfirmed, Path: s.ManifestPath(id)}, nil
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
	files, err := EvidenceFiles(expected)
	if err != nil {
		return unknown, err
	}
	if len(files) != len(crops) {
		return unknown, fmt.Errorf("publication crop set differs")
	}
	for _, file := range files {
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
