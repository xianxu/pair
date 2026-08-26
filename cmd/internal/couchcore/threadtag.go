package couchcore

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

const threadTagAttempts = 8

// AllocateThreadTag draws a 64-bit opaque suffix and returns only after the
// corresponding composite record has been durably claimed without replacement.
func (s *ThreadStore) AllocateThreadTag(repoScope, workingPath string, createdAt time.Time, entropy io.Reader, artifacts ThreadArtifactCollisionChecker) (ThreadRecord, error) {
	if entropy == nil {
		return ThreadRecord{}, errors.New("allocate thread tag: nil entropy reader")
	}
	if artifacts == nil {
		return ThreadRecord{}, errors.New("allocate thread tag: nil artifact collision checker")
	}
	for attempt := 0; attempt < threadTagAttempts; attempt++ {
		var random [8]byte
		if _, err := io.ReadFull(entropy, random[:]); err != nil {
			return ThreadRecord{}, fmt.Errorf("allocate thread tag: %w", err)
		}
		record := ThreadRecord{
			SchemaVersion: ThreadSchemaVersion,
			Address: ThreadAddress{
				RepoScope: repoScope,
				Tag:       ThreadTag("couch-" + hex.EncodeToString(random[:])),
			},
			StartingPath: workingPath,
			WorkingPath:  workingPath,
			CreatedAt:    createdAt,
			Revision:     1,
			Reservation:  true,
		}
		collision, err := artifacts.Collides(record.Address)
		if err != nil {
			return ThreadRecord{}, fmt.Errorf("allocate thread tag: check scoped artifacts: %w", err)
		}
		if collision {
			continue
		}
		created, err := s.CreateThread(record)
		if err == nil {
			return created, nil
		}
		var exists *ThreadExistsError
		if !errors.As(err, &exists) {
			return ThreadRecord{}, err
		}
	}
	return ThreadRecord{}, fmt.Errorf("allocate thread tag: exhausted %d collision attempts", threadTagAttempts)
}
