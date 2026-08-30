package pairlifecycle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type RecordKind string

const (
	RecordRequest    RecordKind = "request"
	RecordCompletion RecordKind = "completion"
)

type PublicationOutcome uint8

const (
	NotCommitted PublicationOutcome = iota
	Indeterminate
	Conflict
	Committed
)

func (o PublicationOutcome) String() string {
	switch o {
	case NotCommitted:
		return "not-committed"
	case Indeterminate:
		return "indeterminate"
	case Conflict:
		return "conflict"
	case Committed:
		return "committed"
	default:
		return "unknown"
	}
}

type PublicationError struct {
	Outcome PublicationOutcome
	Err     error
}

func (e *PublicationError) Error() string {
	return fmt.Sprintf("lifecycle publication %s: %v", e.Outcome, e.Err)
}
func (e *PublicationError) Unwrap() error { return e.Err }

func PublicationOutcomeOf(err error) PublicationOutcome {
	if err == nil {
		return Committed
	}
	var publicationError *PublicationError
	if errors.As(err, &publicationError) {
		return publicationError.Outcome
	}
	return NotCommitted
}

type Unlocker interface{ Close() error }

type StoreFile interface {
	io.Writer
	Sync() error
	Close() error
	Name() string
}

type StoreRuntime interface {
	MkdirAll(string, os.FileMode) error
	Lock(string) (Unlocker, error)
	ReadFile(string) ([]byte, error)
	CreateTemp(string, string) (StoreFile, error)
	Remove(string) error
	Rename(string, string) error
	SyncDirectory(string) error
}

// ArtifactPaths is implemented by artifactpath.LifecyclePaths. Keeping this
// narrow structural seam avoids making the lifecycle protocol depend on the
// repository-wide path-constructor package while still requiring callers to
// supply its validated paths.
type ArtifactPaths interface {
	Dir() string
	Lock() string
	Request(uint64) (string, error)
	Completion(uint64) (string, error)
	CompletionKey(uint64) (string, error)
}

// Store publishes immutable attempt records while holding the transaction's
// stable advisory lock.
type Store struct{ Runtime StoreRuntime }

func (s Store) PublishRequest(paths ArtifactPaths, request QuitRequest) error {
	if err := ValidateQuitRequest(request); err != nil {
		return fmt.Errorf("validate quit request: %w", err)
	}
	if err := validateCompletionKey(paths, request.Attempt, request.CompletionKey); err != nil {
		return err
	}
	return s.publish(paths, RecordRequest, request.Attempt, request)
}

func (s Store) PublishCompletion(paths ArtifactPaths, completion QuitCompletion) error {
	if err := ValidateQuitCompletion(completion); err != nil {
		return fmt.Errorf("validate quit completion: %w", err)
	}
	if err := validateCompletionKey(paths, completion.Attempt, completion.CompletionKey); err != nil {
		return err
	}
	return s.publish(paths, RecordCompletion, completion.Attempt, completion)
}

func (s Store) publish(paths ArtifactPaths, kind RecordKind, attempt uint64, record any) (err error) {
	if s.Runtime == nil || paths.Dir() == "" {
		return errors.New("lifecycle store runtime or paths are empty")
	}
	final, err := recordPath(paths, kind, attempt)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode lifecycle %s: %w", kind, err)
	}
	raw = append(raw, '\n')

	if err := s.Runtime.MkdirAll(paths.Dir(), 0o700); err != nil {
		return fmt.Errorf("create lifecycle directory: %w", err)
	}
	lock, err := s.Runtime.Lock(paths.Lock())
	if err != nil {
		return fmt.Errorf("lock lifecycle transaction: %w", err)
	}
	outcome := NotCommitted
	defer func() {
		if unlockErr := lock.Close(); unlockErr != nil {
			err = joinPublicationError(outcome, err, fmt.Errorf("unlock lifecycle transaction: %w", unlockErr))
		}
	}()

	if found, existing, inspectErr := s.inspectFinal(paths, final, kind, attempt); inspectErr != nil {
		outcome = Conflict
		return publicationError(outcome, inspectErr)
	} else if found {
		if !bytes.Equal(existing, raw) {
			outcome = Conflict
			return publicationError(outcome, fmt.Errorf("immutable lifecycle %s differs from existing final", kind))
		}
		outcome = Indeterminate
		if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
			return publicationError(outcome, fmt.Errorf("sync lifecycle directory: %w", err))
		}
		outcome = Committed
		return nil
	}

	temp, err := s.Runtime.CreateTemp(paths.Dir(), ".pair-publish-*")
	if err != nil {
		return fmt.Errorf("create lifecycle temporary file: %w", err)
	}
	tempName := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = s.Runtime.Remove(tempName)
		}
	}()
	if err := writeStoreAll(temp, raw); err != nil {
		return errors.Join(fmt.Errorf("write lifecycle temporary file: %w", err), temp.Close())
	}
	if err := temp.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync lifecycle temporary file: %w", err), temp.Close())
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close lifecycle temporary file: %w", err)
	}

	// Recheck while locked immediately before rename. Compliant writers use the
	// same stable inode, so rename is publication rather than the immutability
	// guard.
	if found, existing, inspectErr := s.inspectFinal(paths, final, kind, attempt); inspectErr != nil {
		outcome = Conflict
		return publicationError(outcome, inspectErr)
	} else if found {
		if !bytes.Equal(existing, raw) {
			outcome = Conflict
			return publicationError(outcome, fmt.Errorf("immutable lifecycle %s appeared with different content", kind))
		}
		outcome = Indeterminate
		if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
			return publicationError(outcome, fmt.Errorf("sync lifecycle directory: %w", err))
		}
		outcome = Committed
		return nil
	}
	if err := s.Runtime.Rename(tempName, final); err != nil {
		return fmt.Errorf("rename lifecycle %s: %w", kind, err)
	}
	removeTemp = false
	outcome = Indeterminate
	if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
		return publicationError(outcome, fmt.Errorf("sync lifecycle directory: %w", err))
	}
	outcome = Committed
	return nil
}

// Reconcile turns a visible prepared final into authority only after strict
// validation and reader-assisted directory sync under the stable lock.
func (s Store) Reconcile(paths ArtifactPaths, kind RecordKind, attempt uint64) (err error) {
	if s.Runtime == nil || paths.Dir() == "" {
		return errors.New("lifecycle store runtime or paths are empty")
	}
	final, err := recordPath(paths, kind, attempt)
	if err != nil {
		return err
	}
	if err := s.Runtime.MkdirAll(paths.Dir(), 0o700); err != nil {
		return fmt.Errorf("create lifecycle directory: %w", err)
	}
	lock, err := s.Runtime.Lock(paths.Lock())
	if err != nil {
		return publicationError(Indeterminate, fmt.Errorf("lock lifecycle reconciliation: %w", err))
	}
	outcome := Indeterminate
	defer func() {
		if unlockErr := lock.Close(); unlockErr != nil {
			err = joinPublicationError(outcome, err, fmt.Errorf("unlock lifecycle reconciliation: %w", unlockErr))
		}
	}()
	found, _, inspectErr := s.inspectFinal(paths, final, kind, attempt)
	if inspectErr != nil {
		outcome = Conflict
		return publicationError(outcome, inspectErr)
	}
	if !found {
		outcome = NotCommitted
		return fmt.Errorf("lifecycle %s final is absent", kind)
	}
	if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
		return publicationError(outcome, fmt.Errorf("sync lifecycle reconciliation directory: %w", err))
	}
	outcome = Committed
	return nil
}

func (s Store) inspectFinal(paths ArtifactPaths, path string, kind RecordKind, attempt uint64) (bool, []byte, error) {
	raw, err := s.Runtime.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("read lifecycle %s final: %w", kind, err)
	}
	if err := decodeAndValidateRecord(raw, paths, kind, attempt); err != nil {
		return true, raw, fmt.Errorf("invalid lifecycle %s final; refusing overwrite: %w", kind, err)
	}
	return true, raw, nil
}

func decodeAndValidateRecord(raw []byte, paths ArtifactPaths, kind RecordKind, attempt uint64) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var recordAttempt uint64
	var completionKey string
	switch kind {
	case RecordRequest:
		var request QuitRequest
		if err := decoder.Decode(&request); err != nil {
			return err
		}
		if err := ValidateQuitRequest(request); err != nil {
			return err
		}
		recordAttempt = request.Attempt
		completionKey = request.CompletionKey
	case RecordCompletion:
		var completion QuitCompletion
		if err := decoder.Decode(&completion); err != nil {
			return err
		}
		if err := ValidateQuitCompletion(completion); err != nil {
			return err
		}
		recordAttempt = completion.Attempt
		completionKey = completion.CompletionKey
	default:
		return fmt.Errorf("unknown lifecycle record kind %q", kind)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing lifecycle data: %w", err)
	}
	if recordAttempt != attempt {
		return fmt.Errorf("record attempt %d does not match path attempt %d", recordAttempt, attempt)
	}
	if err := validateCompletionKey(paths, attempt, completionKey); err != nil {
		return err
	}
	return nil
}

func validateCompletionKey(paths ArtifactPaths, attempt uint64, got string) error {
	want, err := paths.CompletionKey(attempt)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("completion_key %q does not match lifecycle path key %q", got, want)
	}
	return nil
}

func recordPath(paths ArtifactPaths, kind RecordKind, attempt uint64) (string, error) {
	switch kind {
	case RecordRequest:
		return paths.Request(attempt)
	case RecordCompletion:
		return paths.Completion(attempt)
	default:
		return "", fmt.Errorf("unknown lifecycle record kind %q", kind)
	}
}

func publicationError(outcome PublicationOutcome, err error) error {
	return &PublicationError{Outcome: outcome, Err: err}
}

func joinPublicationError(outcome PublicationOutcome, err, cleanupErr error) error {
	if err == nil {
		return publicationError(outcome, cleanupErr)
	}
	var current *PublicationError
	if errors.As(err, &current) {
		return publicationError(current.Outcome, errors.Join(current.Err, cleanupErr))
	}
	if outcome == NotCommitted {
		return errors.Join(err, cleanupErr)
	}
	return publicationError(outcome, errors.Join(err, cleanupErr))
}

func writeStoreAll(writer io.Writer, raw []byte) error {
	for len(raw) > 0 {
		n, err := writer.Write(raw)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		raw = raw[n:]
	}
	return nil
}
