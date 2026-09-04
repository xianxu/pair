package couchcore

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

var ErrThreadNotFound = errors.New("thread not found")

type ThreadExistsError struct{ Address ThreadAddress }

func (e *ThreadExistsError) Error() string {
	return fmt.Sprintf("thread already exists: %+v", e.Address)
}

type ThreadRevisionError struct {
	Address ThreadAddress
	Want    uint64
	Got     uint64
}

// ThreadSnapshot is a read-only view. It used to carry the manifest and each
// record's exact bytes as compare-and-swap evidence for
// CommitThreadReplacements, which pair#170 M4 deleted -- so copying them was
// per-call work for no consumer.
type ThreadSnapshot struct {
	Generation uint64
	Records    []ThreadRecord
	// Unreadable are manifest-listed addresses whose record could not be read
	// or decoded. They are carried rather than raised, because one unreadable
	// record must not remove every other row: a store with 13 threads and one
	// corrupt file has 13 threads, one of which needs attention (#181).
	//
	// "Unreadable" is deliberately not "invalid". A decode failure can mean the
	// record is corrupt, or that this binary is older than the store that wrote
	// it -- and the second case would otherwise classify every thread as debris.
	Unreadable []ThreadAddress
}

func (e *ThreadRevisionError) Error() string {
	return fmt.Sprintf("thread revision conflict for %+v: expected %d, found %d", e.Address, e.Want, e.Got)
}

type threadManifest struct {
	SchemaVersion int             `json:"schema_version"`
	Generation    uint64          `json:"generation"`
	Threads       []ThreadAddress `json:"threads"`
	// DeprecatedLegacyCutover and DeprecatedLegacyMigrationVersion are
	// TOMBSTONES, not fields. The one-time import of the old tree-keyed
	// registry went with pair#170 M4, but these keys are in the operator's
	// live manifest, and this envelope is decoded with DisallowUnknownFields:
	// removing them outright makes the manifest undecodable, which takes the
	// WHOLE STORE down rather than one record.
	//
	// Unlike the record tombstones, these PERSIST: a record is rebuilt from a
	// domain type that has no such field, so it sheds them, while the manifest
	// is decoded and re-marshalled through this struct and carries them
	// forward verbatim. That asymmetry is deliberate rather than tolerated --
	// clearing `legacy_cutover` would tell a rolled-back pre-M4 binary that the
	// registry cutover had never run, and it would run it again. Measured, and
	// guarded by TestPreM4ManifestStillLoads plus
	// TestManifestTombstonesSurviveAWrite.
	DeprecatedLegacyCutover          json.RawMessage `json:"legacy_cutover,omitempty"`
	DeprecatedLegacyMigrationVersion json.RawMessage `json:"legacy_migration_version,omitempty"`
}

// pair:m5-concept integration
type ThreadStore struct {
	namespace CouchNamespace
	root      string
	hooks     threadStoreHooks
}

func NewThreadStore(namespace CouchNamespace) *ThreadStore {
	return &ThreadStore{namespace: namespace, root: filepath.Join(namespace.Dir(), "threadstore")}
}

type threadStoreHooks struct {
	AfterJournal func() error
	AfterTarget  func(int) error
}

func newThreadStoreWithHooks(namespace CouchNamespace, hooks threadStoreHooks) *ThreadStore {
	store := NewThreadStore(namespace)
	store.hooks = hooks
	return store
}

func (s *ThreadStore) manifestPath() string { return filepath.Join(s.root, "manifest.json") }

func (s *ThreadStore) recordPath(address ThreadAddress) string {
	return filepath.Join(s.root, "records", address.RepoScope, string(address.Tag)+".json")
}

func (s *ThreadStore) pathLaunchPreferencePath(repoIdentity, physicalPath string) string {
	digest := sha256.Sum256([]byte(repoIdentity + "\x00" + physicalPath))
	return filepath.Join(s.root, "path-preferences", fmt.Sprintf("%x.json", digest[:]))
}

func (s *ThreadStore) withLock(fn func() error) (err error) {
	if s == nil || s.namespace.Dir() == "" {
		return errors.New("thread store has no namespace")
	}
	lock, err := acquireThreadStoreLock(s.root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if err := s.recoverStoreJournalLocked(); err != nil {
		return err
	}
	return fn()
}

func (s *ThreadStore) RecoverStoreJournal() (err error) {
	if s == nil || s.namespace.Dir() == "" {
		return errors.New("thread store has no namespace")
	}
	lock, err := acquireThreadStoreLock(s.root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	return s.recoverStoreJournalLocked()
}

func (s *ThreadStore) CreateThread(record ThreadRecord) (ThreadRecord, error) {
	record = cloneThreadRecord(record)
	if err := validateThreadAddress(record.Address); err != nil {
		return ThreadRecord{}, err
	}
	var created ThreadRecord
	err := s.withLock(func() error {
		manifest, manifestRaw, manifestExists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		if _, exists, err := readOptionalFile(s.recordPath(record.Address)); err != nil {
			return err
		} else if exists || manifestContains(manifest, record.Address) {
			return &ThreadExistsError{Address: record.Address}
		}
		if err := ValidateThreadRecord(record); err != nil {
			return err
		}
		recordRaw, err := json.MarshalIndent(toPersistedThreadRecord(record), "", "  ")
		if err != nil {
			return err
		}
		recordRaw = append(recordRaw, '\n')
		nextManifest := manifest
		nextManifest.SchemaVersion = 1
		nextManifest.Generation++
		nextManifest.Threads = append(nextManifest.Threads, record.Address)
		sortThreadAddresses(nextManifest.Threads)
		nextManifestRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		nextManifestRaw = append(nextManifestRaw, '\n')
		var expectedManifest *[]byte
		if manifestExists {
			copy := append([]byte{}, manifestRaw...)
			expectedManifest = &copy
		}
		afterRecord := append([]byte{}, recordRaw...)
		afterManifest := append([]byte{}, nextManifestRaw...)
		journal := storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
			{Path: relativeStorePath(s.root, s.recordPath(record.Address)), After: &afterRecord},
			{Path: relativeStorePath(s.root, s.manifestPath()), Expected: expectedManifest, After: &afterManifest},
		}}
		if err := s.commitJournalLocked(journal); err != nil {
			return err
		}
		created = cloneThreadRecord(record)
		return nil
	})
	return created, err
}

func (s *ThreadStore) GetThread(address ThreadAddress) (ThreadRecord, error) {
	if err := validateThreadAddress(address); err != nil {
		return ThreadRecord{}, err
	}
	var result ThreadRecord
	err := s.withLock(func() error {
		record, err := s.readThreadLocked(address)
		if err != nil {
			return err
		}
		result = cloneThreadRecord(record)
		return nil
	})
	return result, err
}

func (s *ThreadStore) GetPathLaunchPreference(repoIdentity, physicalPath string) (PathLaunchPreference, bool, error) {
	if repoIdentity == "" {
		return PathLaunchPreference{}, false, errors.New("path launch preference has no repository identity")
	}
	if !filepath.IsAbs(physicalPath) {
		return PathLaunchPreference{}, false, errors.New("path launch preference path must be absolute")
	}
	var result PathLaunchPreference
	var found bool
	err := s.withLock(func() error {
		raw, exists, err := readOptionalFile(s.pathLaunchPreferencePath(repoIdentity, physicalPath))
		if err != nil || !exists {
			return err
		}
		if err := strictThreadStoreJSON(raw, &result); err != nil {
			return err
		}
		if err := validatePathLaunchPreference(result); err != nil {
			return err
		}
		if result.RepoIdentity != repoIdentity || result.PhysicalPath != physicalPath {
			return errors.New("path launch preference path/address mismatch")
		}
		result = clonePathLaunchPreference(result)
		found = true
		return nil
	})
	return result, found, err
}

func (s *ThreadStore) UpdateExistingThread(address ThreadAddress, expectedRevision uint64, mutate func(*ThreadRecord) error) (ThreadRecord, error) {
	if err := validateThreadAddress(address); err != nil {
		return ThreadRecord{}, err
	}
	if mutate == nil {
		return ThreadRecord{}, errors.New("thread update has nil mutation")
	}
	var result ThreadRecord
	err := s.withLock(func() error {
		currentRaw, err := os.ReadFile(s.recordPath(address))
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
		}
		if err != nil {
			return err
		}
		current, err := s.decodeThreadRaw(address, currentRaw)
		if err != nil {
			return err
		}
		if current.Revision != expectedRevision {
			return &ThreadRevisionError{Address: address, Want: expectedRevision, Got: current.Revision}
		}
		next := cloneThreadRecord(current)
		if err := mutate(&next); err != nil {
			return err
		}
		if next.Address != current.Address || next.SchemaVersion != current.SchemaVersion || next.StartingPath != current.StartingPath || next.CreatedAt != current.CreatedAt {
			return errors.New("thread update changed immutable identity or origin fields")
		}
		next.Revision++
		if err := ValidateThreadRecord(next); err != nil {
			return err
		}
		raw, err := json.MarshalIndent(toPersistedThreadRecord(next), "", "  ")
		if err != nil {
			return err
		}
		expected := append([]byte(nil), currentRaw...)
		after := append(raw, '\n')
		journal := storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{{
			Path: relativeStorePath(s.root, s.recordPath(address)), Expected: &expected, After: &after,
		}}}
		if err := s.commitJournalLocked(journal); err != nil {
			return err
		}
		result = cloneThreadRecord(next)
		return nil
	})
	return result, err
}

func (s *ThreadStore) BeginPark(address ThreadAddress, expectedRevision uint64, identity ParkIdentity) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		if next.Park != nil {
			return fmt.Errorf("thread %+v already has active park %q", address, next.Park.Identity.Nonce)
		}
		for _, historical := range next.ParkHistory {
			if historical.Identity.Nonce == identity.Nonce {
				return fmt.Errorf("park nonce %q is already historical", identity.Nonce)
			}
		}
		exactMatches := 0
		eligibleMatches := 0
		var target *ThreadIncarnation
		for i := range next.Incarnations {
			incarnation := &next.Incarnations[i]
			if incarnation.PID != identity.PID || incarnation.Identity != identity.ProcessIdentity {
				continue
			}
			exactMatches++
			if incarnation.State == IncarnationLive || incarnation.State == IncarnationUnknown {
				eligibleMatches++
				target = incarnation
			}
		}
		if identity.Address != address || exactMatches != 1 || eligibleMatches != 1 {
			return fmt.Errorf("park identity must name exactly one live or unknown incarnation")
		}
		if next.LatestLaunchProfile == nil {
			if target.LaunchProfile == nil {
				return errors.New("park target has no successful launch profile")
			}
			profile := cloneLaunchProfile(*target.LaunchProfile)
			next.LatestLaunchProfile = &profile
		}
		transaction, _, err := AdvanceParkTransaction(nil, ParkEvent{
			Kind: ParkBegin, Identity: identity,
			BaseRevision: expectedRevision, RecordRevision: expectedRevision + 1,
		})
		if err != nil {
			return err
		}
		next.Park = &transaction
		return nil
	})
}

func (s *ThreadStore) AdvancePark(address ThreadAddress, expectedRevision uint64, event ParkEvent) (ThreadRecord, error) {
	switch event.Kind {
	case ParkRequestCommitted, ParkFailureObserved:
	default:
		return ThreadRecord{}, fmt.Errorf("park event %q requires its dedicated store operation", event.Kind)
	}
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		if next.Park == nil {
			return errors.New("thread has no active park transaction")
		}
		if event.Identity != next.Park.Identity {
			return errors.New("park event identity does not match active transaction")
		}
		event.BaseRevision = 0
		event.RecordRevision = expectedRevision + 1
		advanced, decision, err := AdvanceParkTransaction(next.Park, event)
		if err != nil {
			return err
		}
		if decision.Finalize || decision.HistoricalNoOp {
			return errors.New("phase advance produced a terminal park decision")
		}
		next.Park = &advanced
		return nil
	})
}

func (s *ThreadStore) AppendParkAttempt(address ThreadAddress, expectedRevision uint64, identity ParkIdentity) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		if next.Park == nil || next.Park.Identity != identity {
			return errors.New("park attempt identity does not match active transaction")
		}
		advanced, decision, err := AdvanceParkTransaction(next.Park, ParkEvent{
			Kind: ParkAttemptAppended, Identity: identity, RecordRevision: expectedRevision + 1,
		})
		if err != nil {
			return err
		}
		if decision.Finalize || decision.HistoricalNoOp {
			return errors.New("attempt append produced a terminal park decision")
		}
		next.Park = &advanced
		return nil
	})
}

func (s *ThreadStore) FinalizePark(address ThreadAddress, expectedRevision uint64, identity ParkIdentity, attempt uint64, parkedAt time.Time) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		if next.Park == nil || next.Park.Identity != identity || next.Park.Tombstoned || next.Park.Closed {
			return errors.New("park success does not match an active non-tombstoned transaction")
		}
		match := -1
		for i, incarnation := range next.Incarnations {
			if incarnation.PID != identity.PID || incarnation.Identity != identity.ProcessIdentity {
				continue
			}
			if match != -1 || incarnation.State != IncarnationLive && incarnation.State != IncarnationUnknown {
				return errors.New("park success does not match exactly one live or unknown incarnation")
			}
			match = i
		}
		if match == -1 {
			return errors.New("park success incarnation is absent or replaced")
		}
		closed, decision, err := AdvanceParkTransaction(next.Park, ParkEvent{
			Kind: ParkCompletionSucceeded, Identity: identity, Attempt: attempt,
			RecordRevision: expectedRevision + 1,
		})
		if err != nil {
			return err
		}
		if !decision.Finalize || decision.HistoricalNoOp || !closed.Closed || closed.Tombstoned {
			return errors.New("park success did not produce a finalization decision")
		}
		next.Incarnations = append(next.Incarnations[:match], next.Incarnations[match+1:]...)
		next.Park = nil
		next.ParkHistory = append(next.ParkHistory, closed)
		next.VerifiedPark = &VerifiedPark{Identity: identity, Attempt: attempt, ParkedAt: parkedAt}
		next.LastActiveAt = MonotonicLastActiveAt(next.LastActiveAt, parkedAt)
		return nil
	})
}

// CommitStartClaim is the durable transition that admission used to perform
// around its capacity decision (pair#170 M4 deleted the decision, not the
// transition): clear the pristine reservation, append the first creating
// incarnation carrying the repository identity, and apply the start claim --
// all in ONE revision-checked write.
//
// It is one commit rather than three because a crash between them leaves a
// reserved record with a live incarnation, which is the state
// ProjectActionableThreads hides: an invisible thread. Naming the transition is
// what makes that atomicity checkable.
//
// It does NOT decide whether the start is allowed. Its callers have already
// decided -- a new thread by allocating the tag, a resume by proving a verified
// park or a surviving detached session -- and the revision CAS is what makes
// the decision still true at the moment of the write.
func (s *ThreadStore) CommitStartClaim(address ThreadAddress, expectedRevision uint64, repoIdentity string, startedAt time.Time, event StartEvent) (ThreadRecord, error) {
	if repoIdentity == "" {
		return ThreadRecord{}, errors.New("start claim has no repository identity")
	}
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		if len(next.Incarnations) != 0 {
			return fmt.Errorf("thread %+v already has %d incarnation(s)", address, len(next.Incarnations))
		}
		if next.Park != nil {
			return fmt.Errorf("thread %+v has an open park transaction", address)
		}
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{
			State: IncarnationCreating, StartedAt: startedAt, RepoIdentity: repoIdentity,
		}}
		advanced, err := AdvanceStartTransaction(*next, event)
		if err != nil {
			return err
		}
		*next = advanced
		return nil
	})
}

// RetireIncarnation removes the one live incarnation whose exact process
// identity matches, leaving the record with no incarnation and NO verified park.
//
// It is FinalizePark's removal half without the park transaction, and that is
// the whole difference between detach and park: park tears the zellij session
// down and records a verified park as the resume authority, while detach leaves
// the session alive and lets its survival BE the authority. Writing a verified
// park here would claim a teardown that never happened.
//
// Exact {PID, Identity} is the authorization -- the same rule observeExactProcess
// and MarkIncarnationUnknown use -- so a recycled PID cannot retire a thread
// that is genuinely live. It refuses an `unknown` incarnation deliberately:
// unknown is precisely the state the fail-closed projector exists to keep out of
// the switcher, and retiring one would let an unproven thread present as cleanly
// detached.
func (s *ThreadStore) RetireIncarnation(address ThreadAddress, expectedRevision uint64, identity ProcessIdentity) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		if next.Park != nil {
			return errors.New("cannot retire an incarnation while a park transaction is open")
		}
		if len(next.Incarnations) != 1 {
			return fmt.Errorf("retire needs exactly one incarnation, found %d", len(next.Incarnations))
		}
		incarnation := next.Incarnations[0]
		if incarnation.State != IncarnationLive {
			return fmt.Errorf("retire needs a live incarnation, found %q", incarnation.State)
		}
		if incarnation.Start != nil {
			return errors.New("cannot retire an incarnation with an open start transaction")
		}
		if incarnation.PID != identity.PID || incarnation.Identity != identity.Identity {
			return errors.New("retire does not match the recorded incarnation process identity")
		}
		next.Incarnations = nil
		return nil
	})
}

func (s *ThreadStore) AbandonPark(address ThreadAddress, expectedRevision uint64, identity ParkIdentity) (ThreadRecord, error) {
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		if next.Park == nil || next.Park.Identity != identity {
			return errors.New("park abandon identity does not match active transaction")
		}
		closed, decision, err := AdvanceParkTransaction(next.Park, ParkEvent{
			Kind: ParkAbandoned, Identity: identity, RecordRevision: expectedRevision + 1,
		})
		if err != nil {
			return err
		}
		if decision.Finalize || decision.HistoricalNoOp || !closed.Closed || !closed.Tombstoned {
			return errors.New("park abandon did not produce a tombstone")
		}
		next.Park = nil
		next.ParkHistory = append(next.ParkHistory, closed)
		return nil
	})
}

func (s *ThreadStore) Snapshot() (ThreadSnapshot, error) {
	var snapshot ThreadSnapshot
	err := s.withLock(func() error {
		manifest, _, _, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		snapshot = ThreadSnapshot{Generation: manifest.Generation}
		for _, address := range manifest.Threads {
			// A record that cannot be read or decoded is REPORTED, not raised.
			// Raising made one corrupt file fail the whole inventory: `couch
			// --list` exited 1 and the switcher showed nothing, which is the
			// opposite of what a total projection promises. It also made
			// ReasonInvalid unreachable in production -- a documented state
			// with a label, an Enter notice and an archive exit that no store
			// could ever produce.
			raw, err := os.ReadFile(s.recordPath(address))
			if err != nil {
				snapshot.Unreadable = append(snapshot.Unreadable, address)
				continue
			}
			record, err := s.decodeThreadRaw(address, raw)
			if err != nil {
				snapshot.Unreadable = append(snapshot.Unreadable, address)
				continue
			}
			snapshot.Records = append(snapshot.Records, cloneThreadRecord(record))
		}
		return nil
	})
	return snapshot, err
}

func (s *ThreadStore) DeletePristineThread(address ThreadAddress) error {
	return s.deleteThreadIf(address, func(record ThreadRecord) error {
		if !record.Reservation || len(record.Incarnations) != 0 {
			return fmt.Errorf("thread %+v is no longer a pristine reservation", address)
		}
		return nil
	})
}

func (s *ThreadStore) AdvanceStart(address ThreadAddress, expectedRevision uint64, event StartEvent) (ThreadRecord, error) {
	if event.Kind == StartRegistered || event.Kind == StartRecoveredUnknown {
		return s.advanceSuccessfulStart(address, expectedRevision, event)
	}
	return s.UpdateExistingThread(address, expectedRevision, func(next *ThreadRecord) error {
		advanced, err := AdvanceStartTransaction(*next, event)
		if err != nil {
			return err
		}
		*next = advanced
		return nil
	})
}

func (s *ThreadStore) advanceSuccessfulStart(address ThreadAddress, expectedRevision uint64, event StartEvent) (ThreadRecord, error) {
	if err := validateThreadAddress(address); err != nil {
		return ThreadRecord{}, err
	}
	var result ThreadRecord
	err := s.withLock(func() error {
		manifest, manifestRaw, manifestExists, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		threadRaw, err := os.ReadFile(s.recordPath(address))
		if err != nil {
			return err
		}
		current, err := s.decodeThreadRaw(address, threadRaw)
		if err != nil {
			return err
		}
		if current.Revision != expectedRevision {
			return &ThreadRevisionError{Address: address, Want: expectedRevision, Got: current.Revision}
		}
		incarnation, err := exactStartIncarnation(&current, event.Nonce)
		if err != nil {
			return err
		}
		var profile *LaunchProfile
		if incarnation.Start.LaunchProfile != nil {
			copy := cloneLaunchProfile(*incarnation.Start.LaunchProfile)
			profile = &copy
		}
		next, err := AdvanceStartTransaction(current, event)
		if err != nil {
			return err
		}
		next.Revision++
		if err := ValidateThreadRecord(next); err != nil {
			return err
		}
		nextThreadRaw, err := json.MarshalIndent(toPersistedThreadRecord(next), "", "  ")
		if err != nil {
			return err
		}
		nextThreadRaw = append(nextThreadRaw, '\n')
		if profile == nil {
			if err := writeAtomicBytes(s.recordPath(address), nextThreadRaw); err != nil {
				return err
			}
			result = cloneThreadRecord(next)
			return nil
		}
		if incarnation.RepoIdentity == "" {
			return errors.New("successful start has no repository identity")
		}
		repoIdentity := incarnation.RepoIdentity
		physicalPath := current.StartingPath
		preferencePath := s.pathLaunchPreferencePath(repoIdentity, physicalPath)
		preferenceRaw, preferenceExists, err := readOptionalFile(preferencePath)
		if err != nil {
			return err
		}
		var currentPreference *PathLaunchPreference
		if preferenceExists {
			var decoded PathLaunchPreference
			if err := strictThreadStoreJSON(preferenceRaw, &decoded); err != nil {
				return err
			}
			if err := validatePathLaunchPreference(decoded); err != nil {
				return err
			}
			if decoded.RepoIdentity != repoIdentity || decoded.PhysicalPath != physicalPath {
				return errors.New("path launch preference path/address mismatch")
			}
			currentPreference = &decoded
		}
		nextPreference, err := RecordSuccessfulLaunch(currentPreference, repoIdentity, physicalPath, *profile)
		if err != nil {
			return err
		}
		nextPreferenceRaw, err := json.MarshalIndent(nextPreference, "", "  ")
		if err != nil {
			return err
		}
		nextPreferenceRaw = append(nextPreferenceRaw, '\n')

		nextManifest := manifest
		nextManifest.Generation++
		nextManifestRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		nextManifestRaw = append(nextManifestRaw, '\n')
		expectedThread := append([]byte(nil), threadRaw...)
		afterThread := append([]byte(nil), nextThreadRaw...)
		var expectedPreference *[]byte
		if preferenceExists {
			copy := append([]byte(nil), preferenceRaw...)
			expectedPreference = &copy
		}
		afterPreference := append([]byte(nil), nextPreferenceRaw...)
		var expectedManifest *[]byte
		if manifestExists {
			copy := append([]byte(nil), manifestRaw...)
			expectedManifest = &copy
		}
		afterManifest := append([]byte(nil), nextManifestRaw...)
		journal := storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
			{Path: relativeStorePath(s.root, s.recordPath(address)), Expected: &expectedThread, After: &afterThread},
			{Path: relativeStorePath(s.root, preferencePath), Expected: expectedPreference, After: &afterPreference},
			{Path: relativeStorePath(s.root, s.manifestPath()), Expected: expectedManifest, After: &afterManifest},
		}}
		if err := s.commitJournalLocked(journal); err != nil {
			return err
		}
		result = cloneThreadRecord(next)
		return nil
	})
	return result, err
}

// MarkIncarnationUnknown retains capacity for one exact live process after its
// in-memory handle has been quiesced but a higher-level ownership handoff
// failed. PID alone is never sufficient because it may already have been
// reused. Revision conflicts retry against the new record so unrelated
// metadata updates survive.
func (s *ThreadStore) MarkIncarnationUnknown(address ThreadAddress, expected ProcessIdentity) (ThreadRecord, error) {
	for {
		current, err := s.GetThread(address)
		if err != nil {
			return ThreadRecord{}, err
		}
		found := false
		for _, incarnation := range current.Incarnations {
			if incarnation.PID == expected.PID && incarnation.Identity == expected.Identity && incarnation.State == IncarnationLive {
				found = true
				break
			}
		}
		if !found {
			return ThreadRecord{}, fmt.Errorf("live incarnation %d/%q not found in thread %+v", expected.PID, expected.Identity, address)
		}
		updated, err := s.UpdateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
			for i := range next.Incarnations {
				incarnation := &next.Incarnations[i]
				if incarnation.PID == expected.PID && incarnation.Identity == expected.Identity && incarnation.State == IncarnationLive {
					incarnation.State = IncarnationUnknown
					return nil
				}
			}
			return fmt.Errorf("live incarnation %d/%q disappeared from thread %+v", expected.PID, expected.Identity, address)
		})
		var conflict *ThreadRevisionError
		if errors.As(err, &conflict) {
			continue
		}
		return updated, err
	}
}

// DeleteStart removes only the exact nonce/revision after reconciliation has
// proven its pre-exec helper absent. Any concurrent metadata or state change
// leaves the transaction occupied.
func (s *ThreadStore) DeleteStart(address ThreadAddress, expectedRevision uint64, nonce string) error {
	current, err := s.GetThread(address)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			return nil
		}
		return err
	}
	if current.VerifiedPark != nil {
		_, err := s.UpdateExistingThread(address, expectedRevision, func(record *ThreadRecord) error {
			if record.Reservation || record.VerifiedPark == nil || len(record.Incarnations) != 1 {
				return fmt.Errorf("thread %+v is no longer parked start %q at revision %d", address, nonce, expectedRevision)
			}
			incarnation := record.Incarnations[0]
			if incarnation.State != IncarnationCreating || incarnation.Start == nil || incarnation.Start.Nonce != nonce {
				return fmt.Errorf("thread %+v is no longer parked start %q at revision %d", address, nonce, expectedRevision)
			}
			record.Incarnations = nil
			return nil
		})
		return err
	}
	if current.LatestLaunchProfile != nil {
		// A record that has ever started successfully is durable history and is
		// never deleted -- roll the start claim back instead.
		//
		// The verified park used to be the only rollback authority (see
		// starttransaction.go's "Until this transition the verified park remains
		// the rollback authority"), which was fine while every resumable thread
		// had one. A DETACHED thread has none: its authority is the surviving
		// zellij session. Without this branch, any post-claim failure on a
		// detached resume deletes the record -- and with it the agent and argv
		// needed to reattach -- while the session it names keeps running.
		//
		// threadHasMetadata already protects a NAMED record; this protects the
		// unnamed one, whose LatestLaunchProfile nothing else guards.
		_, err := s.UpdateExistingThread(address, expectedRevision, func(record *ThreadRecord) error {
			if record.Reservation || len(record.Incarnations) != 1 {
				return fmt.Errorf("thread %+v is no longer start %q at revision %d", address, nonce, expectedRevision)
			}
			incarnation := record.Incarnations[0]
			if incarnation.State != IncarnationCreating || incarnation.Start == nil || incarnation.Start.Nonce != nonce {
				return fmt.Errorf("thread %+v is no longer start %q at revision %d", address, nonce, expectedRevision)
			}
			record.Incarnations = nil
			return nil
		})
		return err
	}
	return s.deleteThreadIf(address, func(record ThreadRecord) error {
		if record.Revision != expectedRevision || record.Reservation || threadHasMetadata(record) || len(record.Incarnations) != 1 {
			return fmt.Errorf("thread %+v is no longer start %q at revision %d", address, nonce, expectedRevision)
		}
		incarnation := record.Incarnations[0]
		if incarnation.State != IncarnationCreating || incarnation.Start == nil || incarnation.Start.Nonce != nonce {
			return fmt.Errorf("thread %+v is no longer start %q at revision %d", address, nonce, expectedRevision)
		}
		return nil
	})
}

func (s *ThreadStore) deleteThreadIf(address ThreadAddress, accept func(ThreadRecord) error) error {
	if err := validateThreadAddress(address); err != nil {
		return err
	}
	return s.withLock(func() error {
		manifest, manifestRaw, _, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		raw, exists, err := readOptionalFile(s.recordPath(address))
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		record, err := s.decodeThreadRaw(address, raw)
		if err != nil {
			return err
		}
		if accept == nil {
			return errors.New("thread delete has nil predicate")
		}
		if err := accept(record); err != nil {
			return err
		}
		nextManifest := manifest
		nextManifest.Generation++
		nextManifest.Threads = removeThreadAddress(nextManifest.Threads, address)
		nextRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		expectedRecord := append([]byte{}, raw...)
		expectedManifest := append([]byte{}, manifestRaw...)
		afterManifest := append(nextRaw, '\n')
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
			{Path: relativeStorePath(s.root, s.recordPath(address)), Expected: &expectedRecord},
			{Path: relativeStorePath(s.root, s.manifestPath()), Expected: &expectedManifest, After: &afterManifest},
		}})
	})
}

func sortThreadAddresses(addresses []ThreadAddress) {
	sort.Slice(addresses, func(i, j int) bool {
		if addresses[i].RepoScope != addresses[j].RepoScope {
			return addresses[i].RepoScope < addresses[j].RepoScope
		}
		return addresses[i].Tag < addresses[j].Tag
	})
}

func (s *ThreadStore) readThreadLocked(address ThreadAddress) (ThreadRecord, error) {
	raw, err := os.ReadFile(s.recordPath(address))
	if errors.Is(err, os.ErrNotExist) {
		return ThreadRecord{}, fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
	}
	if err != nil {
		return ThreadRecord{}, err
	}
	return s.decodeThreadRaw(address, raw)
}

func (s *ThreadStore) decodeThreadRaw(address ThreadAddress, raw []byte) (ThreadRecord, error) {
	record, err := threadrecord.DecodePersisted(raw, toPersistedThreadAddress(address), threadRecordValidators)
	if err != nil {
		return ThreadRecord{}, err
	}
	return fromPersistedThreadRecord(record), nil
}

func (s *ThreadStore) loadManifestLocked() (threadManifest, []byte, bool, error) {
	raw, exists, err := readOptionalFile(s.manifestPath())
	if err != nil {
		return threadManifest{}, nil, false, err
	}
	if !exists {
		return threadManifest{SchemaVersion: 1, Threads: []ThreadAddress{}}, nil, false, nil
	}
	var manifest threadManifest
	if err := strictThreadStoreJSON(raw, &manifest); err != nil {
		return threadManifest{}, nil, true, err
	}
	if manifest.SchemaVersion != 1 || manifest.Threads == nil {
		return threadManifest{}, nil, true, errors.New("invalid thread store manifest")
	}
	seen := map[ThreadAddress]bool{}
	for _, address := range manifest.Threads {
		if err := validateThreadAddress(address); err != nil {
			return threadManifest{}, nil, true, fmt.Errorf("invalid manifest address: %w", err)
		}
		if seen[address] {
			return threadManifest{}, nil, true, fmt.Errorf("duplicate manifest address %+v", address)
		}
		seen[address] = true
	}
	return manifest, raw, true, nil
}

func manifestContains(manifest threadManifest, address ThreadAddress) bool {
	for _, candidate := range manifest.Threads {
		if candidate == address {
			return true
		}
	}
	return false
}

func removeThreadAddress(addresses []ThreadAddress, remove ThreadAddress) []ThreadAddress {
	out := make([]ThreadAddress, 0, len(addresses))
	for _, address := range addresses {
		if address != remove {
			out = append(out, address)
		}
	}
	return out
}

func relativeStorePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		panic(err)
	}
	return rel
}

func (s *ThreadStore) commitJournalLocked(journal storeJournal) error {
	journal, err := assignStoreJournalNonce(journal)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomicBytes(s.journalPath(), append(raw, '\n')); err != nil {
		return err
	}
	if s.hooks.AfterJournal != nil {
		if err := s.hooks.AfterJournal(); err != nil {
			return err
		}
	}
	for i, entry := range journal.Entries {
		if err := s.applyJournalEntry(entry); err != nil {
			return err
		}
		if s.hooks.AfterTarget != nil {
			if err := s.hooks.AfterTarget(i); err != nil {
				return err
			}
		}
	}
	if err := os.Remove(s.journalPath()); err != nil {
		return err
	}
	return syncDirectory(s.root)
}

// archivePath is where a retired record goes: the same layout under a
// different root, so the same reader inspects it and restoring is a file move
// plus a manifest re-add.
func (s *ThreadStore) archivePath(address ThreadAddress) string {
	return filepath.Join(s.root, "archive", address.RepoScope, string(address.Tag)+".json")
}

// ArchiveThread removes a thread from the working set and keeps its record.
//
// The operator's word for this is "delete" -- get it out of my switcher so I
// can start anew -- and archiving is how that is done without being
// irreversible. A record moves, it is not destroyed: the same decoder reads
// `threadstore/archive/<scope>/<tag>.json`, and a mistake is undone by moving
// the file back and re-adding the address to the manifest.
//
// It refuses a thread that is still LIVE or mid-park. Archiving a record while
// couch hosts its child would leave the console owning a thread the store no
// longer lists -- the same shape as the stale incarnations #181 exists to stop
// producing. Everything else goes: parked, detached and every unusable reason,
// because the operator is the one who decides a thread is finished.
func (s *ThreadStore) ArchiveThread(address ThreadAddress) error {
	if err := validateThreadAddress(address); err != nil {
		return err
	}
	return s.withLock(func() error {
		manifest, manifestRaw, _, err := s.loadManifestLocked()
		if err != nil {
			return err
		}
		raw, exists, err := readOptionalFile(s.recordPath(address))
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %+v", ErrThreadNotFound, address)
		}
		// Decode to CHECK, not to gate the move: an undecodable record is
		// exactly what the operator most wants gone, and refusing to archive it
		// would leave a row that can neither be used nor removed. Its bytes are
		// moved as they are.
		record, decodeErr := s.decodeThreadRaw(address, raw)
		if decodeErr == nil {
			if err := archivableRecord(record); err != nil {
				return err
			}
		}
		nextManifest := manifest
		nextManifest.Generation++
		nextManifest.Threads = removeThreadAddress(nextManifest.Threads, address)
		nextRaw, err := json.MarshalIndent(nextManifest, "", "  ")
		if err != nil {
			return err
		}
		// One journal, three effects: the archive copy appears, the record
		// disappears, the manifest stops listing it. A crash between them would
		// otherwise leave a record in no set or in both.
		archived := append([]byte{}, raw...)
		expectedRecord := append([]byte{}, raw...)
		expectedManifest := append([]byte{}, manifestRaw...)
		afterManifest := append(nextRaw, '\n')
		return s.commitJournalLocked(storeJournal{SchemaVersion: 1, Entries: []storeJournalEntry{
			{Path: relativeStorePath(s.root, s.archivePath(address)), After: &archived},
			{Path: relativeStorePath(s.root, s.recordPath(address)), Expected: &expectedRecord},
			{Path: relativeStorePath(s.root, s.manifestPath()), Expected: &expectedManifest, After: &afterManifest},
		}})
	})
}

// ArchivedThreads lists what has been retired, without loading any of it into
// the working set. It is how the operator inspects a decision they can undo.
func (s *ThreadStore) ArchivedThreads() ([]ThreadRecord, error) {
	root := filepath.Join(s.root, "archive")
	var records []ThreadRecord
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		address := ThreadAddress{
			RepoScope: filepath.Base(filepath.Dir(path)),
			Tag:       ThreadTag(strings.TrimSuffix(filepath.Base(path), ".json")),
		}
		record, decodeErr := s.decodeThreadRaw(address, raw)
		if decodeErr != nil {
			// Its address is what could be read, so its address is what is
			// listed. The previous comment said such a record "is still
			// evidence the operator may want" and then dropped it, which is
			// the invisible degradation this issue exists to remove.
			records = append(records, ThreadRecord{Address: address})
			return nil
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Address.RepoScope != records[j].Address.RepoScope {
			return records[i].Address.RepoScope < records[j].Address.RepoScope
		}
		return records[i].Address.Tag < records[j].Address.Tag
	})
	return records, nil
}
