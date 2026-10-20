package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// The supervisor's namespace lease excludes cross-process writers. Shared
// per-path gates also serialize store instances without holding across effects.
var drainStoreGates sync.Map
var drainStoreGatesMu sync.Mutex

type drainStoreGate struct {
	gate chan struct{}
	refs int
}
type drainStore struct {
	namespace    string
	beforeRename func() error
	afterRename  func() error
}

func newDrainStore(namespace string) *drainStore {
	path, err := filepath.Abs(namespace)
	if err == nil {
		namespace = path
	}
	return &drainStore{namespace: namespace}
}
func (s *drainStore) lock(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	drainStoreGatesMu.Lock()
	value, _ := drainStoreGates.LoadOrStore(s.namespace, &drainStoreGate{gate: make(chan struct{}, 1)})
	g := value.(*drainStoreGate)
	g.refs++
	drainStoreGatesMu.Unlock()
	releaseRef := func() {
		drainStoreGatesMu.Lock()
		defer drainStoreGatesMu.Unlock()
		g.refs--
		if g.refs == 0 {
			drainStoreGates.Delete(s.namespace)
		}
	}
	select {
	case g.gate <- struct{}{}:
		return func() { <-g.gate; releaseRef() }, nil
	case <-ctx.Done():
		releaseRef()
		return nil, ctx.Err()
	}
}
func validateDrainRecord(data []byte) error {
	obj, err := drainJSONObject(data)
	if err != nil {
		return err
	}
	if err := drainRequired(obj, "schema_version"); err != nil {
		return err
	}
	var version int
	if err := json.Unmarshal(obj["schema_version"], &version); err != nil {
		return err
	}
	if version != 1 {
		return fmt.Errorf("unsupported drain record schema_version %d", version)
	}
	return nil
}
func (s *drainStore) Read(ctx context.Context) (json.RawMessage, error) {
	unlock, err := s.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	path := filepath.Join(s.namespace, "drain.json")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDrainRecordBytes {
		return nil, fmt.Errorf("unsafe/oversized drain record")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxDrainRecordBytes+1))
	if err != nil {
		return nil, err
	}
	if err := validateDrainRecord(data); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
func (s *drainStore) Write(ctx context.Context, data json.RawMessage) (err error) {
	if err := validateDrainRecord(data); err != nil {
		return err
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := os.MkdirAll(s.namespace, 0700); err != nil {
		return err
	}
	path := filepath.Join(s.namespace, "drain.json")
	if info, e := os.Lstat(path); e == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("unsafe drain record")
	} else if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	f, err := os.CreateTemp(s.namespace, ".drain-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = f.Close(); _ = os.Remove(tmp) }()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if s.beforeRename != nil {
		if err := s.beforeRename(); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if s.afterRename != nil {
		if err := s.afterRename(); err != nil {
			return err
		}
	}
	// A failure after rename is uncertain publication, never grounds to delete
	// the newly published record (it may be the sole owner-intent witness).
	dir, err := os.Open(s.namespace)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
