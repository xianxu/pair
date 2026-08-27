package couchcore

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

const standaloneUpsertAttempts = 8

// RegisterStandalonePair is the cmd/pair-go composition adapter. It resolves
// the same canonical Couch namespace as the supervisor, identifies the Pair
// process with a PID-reuse-safe token, and delegates persistence to ThreadStore.
// pair:m5-concept integration
func RegisterStandalonePair(registration launcher.StandaloneThreadRegistration) error {
	storeDir := registration.CouchStoreDir
	if storeDir == "" {
		storeDir = filepath.Join(registration.GlobalDataDir, "couch")
	}
	namespace, err := ResolveCouchNamespace(storeDir, registration.WorkingPath)
	if err != nil {
		return err
	}
	process, err := (OSProcOps{}).Current()
	if err != nil {
		return fmt.Errorf("identify standalone Pair process: %w", err)
	}
	_, err = NewThreadStore(namespace).UpsertStandalonePair(registration, process)
	return err
}

// UpsertStandalonePair makes an ordinary Pair create visible through Couch's
// authoritative per-thread records. Create and update both use the store lock,
// atomic record writes, and revision CAS; concurrent first writers converge by
// retrying the existing record rather than maintaining a shadow registry.
// pair:m5-concept integration
func (s *ThreadStore) UpsertStandalonePair(registration launcher.StandaloneThreadRegistration, process ProcessIdentity) (ThreadRecord, error) {
	address := ThreadAddress{RepoScope: registration.RepoScope, Tag: ThreadTag(registration.Tag)}
	profile := LaunchProfile{Agent: registration.Agent, Argv: cloneArgv(registration.Argv)}
	if err := validateThreadAddress(address); err != nil {
		return ThreadRecord{}, err
	}
	if !filepath.IsAbs(registration.WorkingPath) || registration.CreatedAt.IsZero() {
		return ThreadRecord{}, errors.New("standalone Pair registration requires an absolute path and creation time")
	}
	if process.PID <= 0 || process.Identity == "" || profile.Agent == "" || registration.Argv == nil {
		return ThreadRecord{}, errors.New("standalone Pair registration has incomplete process or launch profile")
	}
	incarnation := ThreadIncarnation{
		PID: process.PID, Identity: process.Identity, State: IncarnationLive,
		StartedAt: registration.CreatedAt, LaunchProfile: &profile,
	}
	created, err := s.CreateThread(ThreadRecord{
		SchemaVersion: ThreadSchemaVersion,
		Address:       address,
		StartingPath:  registration.WorkingPath,
		WorkingPath:   registration.WorkingPath,
		CreatedAt:     registration.CreatedAt,
		Revision:      1,
		Incarnations:  []ThreadIncarnation{incarnation},
	})
	if err == nil {
		return created, nil
	}
	var exists *ThreadExistsError
	if !errors.As(err, &exists) {
		return ThreadRecord{}, err
	}

	for attempt := 0; attempt < standaloneUpsertAttempts; attempt++ {
		current, err := s.GetThread(address)
		if err != nil {
			return ThreadRecord{}, err
		}
		for _, existing := range current.Incarnations {
			if existing.PID == process.PID && existing.Identity == process.Identity &&
				existing.State == IncarnationLive && existing.Start == nil &&
				existing.LaunchProfile != nil && reflect.DeepEqual(*existing.LaunchProfile, profile) {
				return current, nil
			}
		}
		updated, err := s.UpdateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
			next.Reservation = false
			for i := range next.Incarnations {
				if next.Incarnations[i].PID == process.PID && next.Incarnations[i].Identity == process.Identity {
					next.Incarnations[i] = incarnation
					return nil
				}
			}
			next.Incarnations = append(next.Incarnations, incarnation)
			return nil
		})
		if err == nil {
			return updated, nil
		}
		var conflict *ThreadRevisionError
		if !errors.As(err, &conflict) {
			return ThreadRecord{}, err
		}
	}
	return ThreadRecord{}, fmt.Errorf("standalone Pair thread upsert did not converge after %d attempts", standaloneUpsertAttempts)
}
