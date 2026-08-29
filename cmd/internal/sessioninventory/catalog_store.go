package sessioninventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
)

type CatalogCommitOutcome = commitoutcome.Outcome

const (
	CatalogNotAuthoritative = commitoutcome.NotAuthoritative
	CatalogIndeterminate    = commitoutcome.Indeterminate
	CatalogCommitted        = commitoutcome.Committed
)

func CatalogOutcomeOf(err error) CatalogCommitOutcome { return commitoutcome.Of(err) }

type CatalogUnlocker interface{ Close() error }

type CatalogFile interface {
	io.Writer
	Sync() error
	Close() error
	Name() string
}

type CatalogStoreRuntime interface {
	MkdirAll(string, os.FileMode) error
	Lock(string) (CatalogUnlocker, error)
	ReadFile(string) ([]byte, error)
	CreateTemp(string, string) (CatalogFile, error)
	Remove(string) error
	Rename(string, string) error
	SyncDirectory(string) error
}

// CatalogStore is the sole locked publisher of the durable session inventory
// classification catalog.
type CatalogStore struct{ Runtime CatalogStoreRuntime }

func (s CatalogStore) Read(path string) (Catalog, error) {
	if path == "" || s.Runtime == nil {
		return Catalog{}, errors.New("catalog path or runtime is empty")
	}
	raw, err := s.Runtime.ReadFile(path)
	if err != nil {
		return Catalog{}, fmt.Errorf("read session inventory catalog: %w", err)
	}
	return decodeCatalog(raw)
}

func (s CatalogStore) Update(path string, mutate func(Catalog) (Catalog, error)) (_ Catalog, err error) {
	if path == "" || s.Runtime == nil || mutate == nil {
		return Catalog{}, errors.New("catalog update input is empty")
	}
	dir := filepath.Dir(path)
	if err := s.Runtime.MkdirAll(dir, 0o700); err != nil {
		return Catalog{}, fmt.Errorf("create session inventory catalog directory: %w", err)
	}
	lock, err := s.Runtime.Lock(path + ".lock")
	if err != nil {
		return Catalog{}, fmt.Errorf("lock session inventory catalog: %w", err)
	}
	outcome := CatalogNotAuthoritative
	defer func() {
		if unlockErr := lock.Close(); unlockErr != nil {
			err = commitoutcome.Join(outcome, err, fmt.Errorf("unlock session inventory catalog: %w", unlockErr))
		}
	}()

	current, err := s.readCurrent(path)
	if err != nil {
		return Catalog{}, err
	}
	next, err := mutate(CloneCatalog(current))
	if err != nil {
		return Catalog{}, err
	}
	next.Version = CatalogVersion
	next.Generation = current.Generation + 1
	next.Entries = sortedCatalogEntries(next.Entries)
	if err := ValidateCatalog(next); err != nil {
		return Catalog{}, fmt.Errorf("validate session inventory catalog: %w", err)
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return Catalog{}, fmt.Errorf("encode session inventory catalog: %w", err)
	}
	raw = append(raw, '\n')

	temp, err := s.Runtime.CreateTemp(dir, ".inventory-publish-*")
	if err != nil {
		return Catalog{}, fmt.Errorf("create session inventory catalog temporary file: %w", err)
	}
	tempName := temp.Name()
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = s.Runtime.Remove(tempName)
		}
	}()
	if err := writeCatalogAll(temp, raw); err != nil {
		return Catalog{}, errors.Join(fmt.Errorf("write session inventory catalog: %w", err), temp.Close())
	}
	if err := temp.Sync(); err != nil {
		return Catalog{}, errors.Join(fmt.Errorf("sync session inventory catalog: %w", err), temp.Close())
	}
	if err := temp.Close(); err != nil {
		return Catalog{}, fmt.Errorf("close session inventory catalog: %w", err)
	}
	if err := s.Runtime.Rename(tempName, path); err != nil {
		return Catalog{}, fmt.Errorf("publish session inventory catalog: %w", err)
	}
	keepTemp = false
	outcome = CatalogIndeterminate
	if err := s.Runtime.SyncDirectory(dir); err != nil {
		return next, commitoutcome.Wrap(outcome, fmt.Errorf("sync session inventory catalog directory: %w", err))
	}
	outcome = CatalogCommitted
	return next, nil
}

func (s CatalogStore) readCurrent(path string) (Catalog, error) {
	raw, err := s.Runtime.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Catalog{Version: CatalogVersion}, nil
	}
	if err != nil {
		return Catalog{}, fmt.Errorf("read session inventory catalog: %w", err)
	}
	return decodeCatalog(raw)
}

func decodeCatalog(raw []byte) (Catalog, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var catalog Catalog
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("decode session inventory catalog: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return Catalog{}, fmt.Errorf("decode session inventory catalog trailing data: %w", err)
	}
	if err := ValidateCatalog(catalog); err != nil {
		return Catalog{}, fmt.Errorf("validate session inventory catalog: %w", err)
	}
	return catalog, nil
}

func writeCatalogAll(writer io.Writer, raw []byte) error {
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
