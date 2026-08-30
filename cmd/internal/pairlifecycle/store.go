package pairlifecycle

import (
	"bytes"
	"context"
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

type LockedAttempt struct {
	store   Store
	paths   ArtifactPaths
	request QuitRequest
}

func (a *LockedAttempt) Request() QuitRequest { return a.request }

// ConsumeAttempt holds one transaction lock across request authority,
// completion dedupe, effective cleanup, and immutable result publication.
func (s Store) ConsumeAttempt(ctx context.Context, paths ArtifactPaths, attempt uint64, cleanup func(context.Context, *LockedAttempt, QuitRequest) CleanupResult) (_ QuitCompletion, err error) {
	if s.Runtime == nil || paths.Dir() == "" || cleanup == nil {
		return QuitCompletion{}, errors.New("consume attempt input is empty")
	}
	if err := s.Runtime.MkdirAll(paths.Dir(), 0o700); err != nil {
		return QuitCompletion{}, err
	}
	lock, err := s.Runtime.Lock(paths.Lock())
	if err != nil {
		return QuitCompletion{}, err
	}
	outcome := NotCommitted
	defer func() {
		if unlockErr := lock.Close(); unlockErr != nil {
			err = joinPublicationError(outcome, err, unlockErr)
		}
	}()
	if err := ctx.Err(); err != nil {
		return QuitCompletion{}, err
	}
	requestPath, err := paths.Request(attempt)
	if err != nil {
		return QuitCompletion{}, err
	}
	raw, err := s.Runtime.ReadFile(requestPath)
	if err != nil {
		return QuitCompletion{}, fmt.Errorf("read committed quit request: %w", err)
	}
	request, err := decodeQuitRequest(raw, paths, attempt)
	if err != nil {
		return QuitCompletion{}, fmt.Errorf("validate committed quit request: %w", err)
	}
	outcome = Indeterminate
	if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
		return QuitCompletion{}, publicationError(outcome, err)
	}

	completionPath, err := paths.Completion(attempt)
	if err != nil {
		return QuitCompletion{}, err
	}
	if completionRaw, readErr := s.Runtime.ReadFile(completionPath); readErr == nil {
		completion, decodeErr := decodeQuitCompletion(completionRaw, paths, attempt)
		if decodeErr != nil {
			outcome = Conflict
			return QuitCompletion{}, publicationError(outcome, decodeErr)
		}
		if matchErr := MatchQuitCompletion(request, completion); matchErr != nil {
			outcome = Conflict
			return QuitCompletion{}, publicationError(outcome, matchErr)
		}
		if syncErr := s.Runtime.SyncDirectory(paths.Dir()); syncErr != nil {
			return QuitCompletion{}, publicationError(outcome, syncErr)
		}
		outcome = Committed
		return completion, nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return QuitCompletion{}, readErr
	}

	locked := &LockedAttempt{store: s, paths: paths, request: request}
	result := cleanup(ctx, locked, request)
	completion, err := locked.PublishCompletion(result)
	outcome = PublicationOutcomeOf(err)
	return completion, err
}

func (a *LockedAttempt) PublishCompletion(result CleanupResult) (QuitCompletion, error) {
	if result.CompletedAt.IsZero() {
		return QuitCompletion{}, errors.New("cleanup result completed_at is required")
	}
	completion := QuitCompletion{
		SchemaVersion: a.request.SchemaVersion, Identity: a.request.Identity, Attempt: a.request.Attempt,
		Session: a.request.Session, Mode: a.request.Mode, CompletionKey: a.request.CompletionKey,
		Outcome: result.Outcome, CompletedAt: result.CompletedAt,
	}
	if result.Outcome == CompletionFailure {
		if len(result.Failures) == 0 {
			return QuitCompletion{}, errors.New("failed cleanup result has no failure")
		}
		completion.FailureCode = result.Failures[0].Code
	}
	if err := ValidateQuitCompletion(completion); err != nil {
		return QuitCompletion{}, err
	}
	raw, err := json.Marshal(completion)
	if err != nil {
		return QuitCompletion{}, err
	}
	raw = append(raw, '\n')
	if err := a.store.publishCompletionLocked(a.paths, completion, raw); err != nil {
		return completion, err
	}
	return completion, nil
}

func (s Store) publishCompletionLocked(paths ArtifactPaths, completion QuitCompletion, raw []byte) error {
	final, err := paths.Completion(completion.Attempt)
	if err != nil {
		return err
	}
	if found, existing, inspectErr := s.inspectFinal(paths, final, RecordCompletion, completion.Attempt); inspectErr != nil {
		return publicationError(Conflict, inspectErr)
	} else if found {
		if !bytes.Equal(existing, raw) {
			return publicationError(Conflict, errors.New("immutable completion differs"))
		}
		if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
			return publicationError(Indeterminate, err)
		}
		return nil
	}
	temp, err := s.Runtime.CreateTemp(paths.Dir(), ".pair-result-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	keep := true
	defer func() {
		if keep {
			_ = s.Runtime.Remove(name)
		}
	}()
	if err := writeStoreAll(temp, raw); err != nil {
		return errors.Join(err, temp.Close())
	}
	if err := temp.Sync(); err != nil {
		return errors.Join(err, temp.Close())
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if found, existing, inspectErr := s.inspectFinal(paths, final, RecordCompletion, completion.Attempt); inspectErr != nil {
		return publicationError(Conflict, inspectErr)
	} else if found {
		if !bytes.Equal(existing, raw) {
			return publicationError(Conflict, errors.New("immutable completion appeared with different content"))
		}
		if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
			return publicationError(Indeterminate, err)
		}
		return nil
	}
	if err := s.Runtime.Rename(name, final); err != nil {
		return err
	}
	keep = false
	if err := s.Runtime.SyncDirectory(paths.Dir()); err != nil {
		return publicationError(Indeterminate, err)
	}
	return nil
}

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

func decodeQuitRequest(raw []byte, paths ArtifactPaths, attempt uint64) (QuitRequest, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var request QuitRequest
	if err := decoder.Decode(&request); err != nil {
		return QuitRequest{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return QuitRequest{}, err
	}
	if err := ValidateQuitRequest(request); err != nil {
		return QuitRequest{}, err
	}
	if request.Attempt != attempt {
		return QuitRequest{}, fmt.Errorf("request attempt %d does not match %d", request.Attempt, attempt)
	}
	if err := validateCompletionKey(paths, attempt, request.CompletionKey); err != nil {
		return QuitRequest{}, err
	}
	return request, nil
}

func decodeQuitCompletion(raw []byte, paths ArtifactPaths, attempt uint64) (QuitCompletion, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var completion QuitCompletion
	if err := decoder.Decode(&completion); err != nil {
		return QuitCompletion{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return QuitCompletion{}, err
	}
	if err := ValidateQuitCompletion(completion); err != nil {
		return QuitCompletion{}, err
	}
	if completion.Attempt != attempt {
		return QuitCompletion{}, fmt.Errorf("completion attempt %d does not match %d", completion.Attempt, attempt)
	}
	if err := validateCompletionKey(paths, attempt, completion.CompletionKey); err != nil {
		return QuitCompletion{}, err
	}
	return completion, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
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
