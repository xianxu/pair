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
		return Record{}, err
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

func (s LedgerStore) appendEncoded(path string, encoded []byte) (_ uint64, err error) {
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
	defer func() { err = errors.Join(err, lock.Close()) }()

	raw, err := s.Runtime.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		raw, err = nil, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read ledger: %w", err)
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
	if err := writeAll(file, payload); err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("append ledger: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return 0, fmt.Errorf("sync ledger: %w", err)
	}
	if err := file.Close(); err != nil {
		return 0, fmt.Errorf("close ledger: %w", err)
	}
	if err := s.Runtime.SyncDirectory(dir); err != nil {
		return 0, fmt.Errorf("sync ledger directory: %w", err)
	}
	return ordinal, nil
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

func writeAll(writer io.Writer, raw []byte) error {
	for len(raw) != 0 {
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
