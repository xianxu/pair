package sessionledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var ErrStaleLaunch = errors.New("ledger launch generation is no longer current")

type AppendOutcome uint8

const (
	AppendNotAuthoritative AppendOutcome = iota
	AppendIndeterminate
	AppendCommitted
)

func (o AppendOutcome) String() string {
	switch o {
	case AppendNotAuthoritative:
		return "not-authoritative"
	case AppendIndeterminate:
		return "indeterminate"
	case AppendCommitted:
		return "committed"
	default:
		return "unknown"
	}
}

// AppendOutcomeError reports failures after a complete ledger row may have
// become recovery authority. Callers must not interpret these as ordinary
// failed appends: Indeterminate means durability could not be established;
// Committed means the row is durable but lock cleanup failed.
type AppendOutcomeError struct {
	Outcome AppendOutcome
	Err     error
}

func (e *AppendOutcomeError) Error() string {
	return fmt.Sprintf("ledger append %s: %v", e.Outcome, e.Err)
}

func (e *AppendOutcomeError) Unwrap() error { return e.Err }

// AppendOutcomeOf converts nil, typed post-write failures, and ordinary
// pre-authority failures into one exhaustive caller-facing result.
func AppendOutcomeOf(err error) AppendOutcome {
	if err == nil {
		return AppendCommitted
	}
	var outcomeErr *AppendOutcomeError
	if errors.As(err, &outcomeErr) {
		return outcomeErr.Outcome
	}
	return AppendNotAuthoritative
}

type Unlocker interface{ Close() error }

type AppendFile interface {
	io.Writer
	Close() error
	Sync() error
}

type Runtime interface {
	MkdirAll(string, os.FileMode) error
	Lock(string) (Unlocker, error)
	ReadFile(string) ([]byte, error)
	OpenAppend(string, os.FileMode) (AppendFile, error)
	SyncDirectory(string) error
}

// LedgerStore is the sole cross-process writer for typed Pair ledger records.
// pair:155-concept integration new M2
type LedgerStore struct{ Runtime Runtime }

// Append encodes before locking, derives the physical ordinal while locked,
// writes one complete JSONL record, fsyncs, and then releases the lock.
func (s LedgerStore) Append(path string, record Record) (_ Record, err error) {
	encoded, err := EncodeRecord(record)
	if err != nil {
		return Record{}, fmt.Errorf("encode ledger record: %w", err)
	}
	ordinal, err := s.appendEncoded(path, encoded)
	if err != nil {
		if ordinal == 0 {
			return Record{}, err
		}
		record.Ordinal = ordinal
		return record, err
	}
	record.Ordinal = ordinal
	return record, nil
}

// AppendLegacy serializes an already-encoded pre-v1 row through the same sole
// lock/fsync writer during migration. New code must use Append.
func (s LedgerStore) AppendLegacy(path string, encoded []byte) (uint64, error) {
	if !json.Valid(encoded) || bytes.ContainsRune(encoded, '\n') {
		return 0, errors.New("invalid legacy ledger row")
	}
	return s.appendEncoded(path, append([]byte(nil), encoded...))
}

// AppendBindingIfCurrent joins a binding to a launch and verifies under the
// same append lock that no newer launch for the owner has superseded it.
func (s LedgerStore) AppendBindingIfCurrent(path string, owner Owner, launchOrdinal uint64, rootNativeID string) (Record, error) {
	record := Record{
		Version: 1, Kind: RecordBinding, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent,
		LaunchOrdinal: launchOrdinal, RootNativeID: rootNativeID,
	}
	encoded, err := EncodeRecord(record)
	if err != nil {
		return Record{}, fmt.Errorf("encode ledger binding: %w", err)
	}
	ordinal, err := s.appendEncodedChecked(path, encoded, func(raw []byte) error {
		current, ok := CurrentLaunch(ParseLedger(raw).Records, owner)
		if !ok || current.Launch.Ordinal != launchOrdinal {
			return ErrStaleLaunch
		}
		return nil
	})
	if err != nil {
		if ordinal == 0 {
			return Record{}, err
		}
		record.Ordinal = ordinal
		return record, err
	}
	record.Ordinal = ordinal
	return record, nil
}

func (s LedgerStore) appendEncoded(path string, encoded []byte) (_ uint64, err error) {
	return s.appendEncodedChecked(path, encoded, nil)
}

func (s LedgerStore) appendEncodedChecked(path string, encoded []byte, check func([]byte) error) (_ uint64, err error) {
	if path == "" || s.Runtime == nil {
		return 0, errors.New("ledger path or runtime is empty")
	}
	dir := filepath.Dir(path)
	if err := s.Runtime.MkdirAll(dir, 0o700); err != nil {
		return 0, fmt.Errorf("create ledger directory: %w", err)
	}
	lock, err := s.Runtime.Lock(path + ".lock")
	if err != nil {
		return 0, fmt.Errorf("lock ledger: %w", err)
	}
	outcome := AppendNotAuthoritative
	defer func() {
		if unlockErr := lock.Close(); unlockErr != nil {
			err = joinAppendError(outcome, err, fmt.Errorf("unlock ledger: %w", unlockErr))
		}
	}()

	raw, err := s.Runtime.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		raw, err = nil, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read ledger: %w", err)
	}
	if check != nil {
		if err := check(raw); err != nil {
			return 0, err
		}
	}
	ordinal := physicalLineCount(raw) + 1
	payload := make([]byte, 0, len(encoded)+2)
	if len(raw) != 0 && raw[len(raw)-1] != '\n' {
		payload = append(payload, '\n')
	}
	payload = append(payload, encoded...)
	payload = append(payload, '\n')

	file, err := s.Runtime.OpenAppend(path, 0o600)
	if err != nil {
		return 0, fmt.Errorf("open ledger append: %w", err)
	}
	written, writeErr := writeAll(file, payload)
	if writeErr != nil {
		closeErr := file.Close()
		if written == len(payload) {
			outcome = AppendIndeterminate
			return ordinal, appendError(outcome, errors.Join(fmt.Errorf("append ledger: %w", writeErr), closeErr))
		}
		return 0, errors.Join(fmt.Errorf("append ledger: %w", writeErr), closeErr)
	}
	outcome = AppendIndeterminate
	if err := file.Sync(); err != nil {
		closeErr := file.Close()
		return ordinal, appendError(outcome, errors.Join(fmt.Errorf("sync ledger: %w", err), closeErr))
	}
	if err := file.Close(); err != nil {
		return ordinal, appendError(outcome, fmt.Errorf("close ledger: %w", err))
	}
	if err := s.Runtime.SyncDirectory(dir); err != nil {
		return ordinal, appendError(outcome, fmt.Errorf("sync ledger directory: %w", err))
	}
	outcome = AppendCommitted
	return ordinal, nil
}

func appendError(outcome AppendOutcome, err error) error {
	return &AppendOutcomeError{Outcome: outcome, Err: err}
}

func joinAppendError(outcome AppendOutcome, err, cleanupErr error) error {
	if outcome == AppendNotAuthoritative {
		return errors.Join(err, cleanupErr)
	}
	var outcomeErr *AppendOutcomeError
	if errors.As(err, &outcomeErr) {
		err = outcomeErr.Err
	}
	return appendError(outcome, errors.Join(err, cleanupErr))
}

func physicalLineCount(raw []byte) uint64 {
	if len(raw) == 0 {
		return 0
	}
	count := uint64(bytes.Count(raw, []byte{'\n'}))
	if raw[len(raw)-1] != '\n' {
		count++
	}
	return count
}

func writeAll(writer io.Writer, raw []byte) (int, error) {
	written := 0
	for len(raw) != 0 {
		n, err := writer.Write(raw)
		written += n
		if err != nil {
			return written, err
		}
		if n <= 0 {
			return written, io.ErrShortWrite
		}
		raw = raw[n:]
	}
	return written, nil
}
