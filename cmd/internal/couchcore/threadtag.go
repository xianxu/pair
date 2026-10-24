package couchcore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

const threadTagAttempts = 8

// AllocateThreadTag reserves a monotonic conversation identity and returns only after the
// corresponding composite record has been durably claimed without replacement.
func (s *ThreadStore) AllocateThreadTag(repoScope, workingPath string, createdAt time.Time, nextTag func() (string, error), artifacts ThreadArtifactClaimer) (ThreadRecord, error) {
	return s.allocateThreadTag(context.Background(), repoScope, workingPath, createdAt, nextTag, artifacts, false)
}

// allocateThreadTag retains the shared address claim loop; startup alone requires
// target history to remain empty under the reservation publication lock.
func (s *ThreadStore) allocateThreadTag(ctx context.Context, repoScope, workingPath string, createdAt time.Time, nextTag func() (string, error), artifacts ThreadArtifactClaimer, emptyStartup bool) (ThreadRecord, error) {
	if nextTag == nil {
		return ThreadRecord{}, errors.New("allocate thread tag: nil identity allocator")
	}
	if artifacts == nil {
		return ThreadRecord{}, errors.New("allocate thread tag: nil artifact collision checker")
	}
	backend, err := s.storeForPath(workingPath, repoScope, "")
	if err != nil {
		return ThreadRecord{}, err
	}
	if backend != s {
		if emptyStartup {
			return ThreadRecord{}, ErrStartResolutionChanged
		}
		return backend.AllocateThreadTag(repoScope, workingPath, createdAt, nextTag, artifacts)
	}
	for attempt := 0; attempt < threadTagAttempts; attempt++ {
		tag, err := nextTag()
		if err != nil {
			return ThreadRecord{}, fmt.Errorf("allocate thread tag: %w", err)
		}
		record := ThreadRecord{
			SchemaVersion: ThreadSchemaVersion,
			Address: ThreadAddress{
				RepoScope: repoScope,
				Tag:       ThreadTag(tag),
			},
			StartingPath: workingPath,
			WorkingPath:  workingPath,
			CreatedAt:    createdAt,
			Revision:     1,
			Reservation:  true,
		}
		claim, err := artifacts.Claim(record.Address)
		if errors.Is(err, launcher.ErrThreadAddressClaimed) {
			continue
		}
		if err != nil {
			return ThreadRecord{}, fmt.Errorf("allocate thread tag: claim scoped artifacts: %w", err)
		}
		var created ThreadRecord
		if emptyStartup {
			err = s.withEmptyStartupTarget(ctx, s, repoScope, func() error {
				if err := s.createThreadLocked(record); err != nil {
					return err
				}
				created = cloneThreadRecord(record)
				return nil
			})
		} else {
			created, err = s.CreateThread(record)
		}
		if err == nil {
			return created, nil
		}
		if releaseErr := claim.Release(); releaseErr != nil {
			return ThreadRecord{}, errors.Join(err, fmt.Errorf("release scoped artifact claim: %w", releaseErr))
		}
		var exists *ThreadExistsError
		if s.layout.Local || !errors.As(err, &exists) {
			return ThreadRecord{}, err
		}
	}
	return ThreadRecord{}, fmt.Errorf("allocate thread tag: exhausted %d collision attempts", threadTagAttempts)
}
