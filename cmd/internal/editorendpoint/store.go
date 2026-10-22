package editorendpoint

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

// EndpointStore is borrowed by the sole Couch owner, never opened as a CLI
// writer. Path and Stage are supplied by the namespace artifact resolver.
type EndpointStore struct {
	Namespace, Path, Stage string
	WriteFile              func(path string, raw []byte, stage string) error
}

func (s EndpointStore) validate() error {
	if !canonicalPath(s.Namespace) || filepath.Dir(s.Path) != s.Namespace || filepath.Dir(s.Stage) != s.Namespace || s.Path == s.Stage || !canonicalPath(s.Path) || !canonicalPath(s.Stage) {
		return errors.New("invalid endpoint store paths")
	}
	info, err := os.Lstat(s.Namespace)
	if err != nil {
		return err
	}
	physical, err := filepath.EvalSymlinks(s.Namespace)
	if err != nil {
		return err
	}
	if !info.IsDir() || physical != s.Namespace || info.Mode().Perm()&0077 != 0 || !owned(info) {
		return fmt.Errorf("endpoint namespace must be a physical private owned directory (path %q, physical %q, mode %v, owned %t)", s.Namespace, physical, info.Mode(), owned(info))
	}
	return nil
}

func owned(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Getuid())
}

// readEndpointFile cannot follow symlinks, block on a FIFO, or exceed the
// authority budget. Retained authority and stage evidence use the same guard.
func readEndpointFile(path string) ([]byte, os.FileInfo, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxStateBytes || info.Mode().Perm()&0077 != 0 || !owned(info) {
		return nil, nil, fmt.Errorf("unsafe endpoint file %q", path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxStateBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if len(raw) > MaxStateBytes {
		return nil, nil, errors.New("endpoint file exceeds bound")
	}
	return raw, info, nil
}

func (s EndpointStore) Load() (EndpointState, error) {
	if err := s.validate(); err != nil {
		return EndpointState{}, err
	}
	raw, _, err := readEndpointFile(s.Path)
	state := EndpointState{Version: 1, Namespace: s.Namespace, Records: []EndpointRecord{}}
	if err == nil {
		state, err = DecodeEndpointState(raw)
	} else if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return EndpointState{}, err
	}
	if state.Namespace != s.Namespace {
		return EndpointState{}, errors.New("endpoint state namespace mismatch")
	}
	// A stage is never authority. Validate committed state first, then reclaim
	// only this owner's exact bounded regular stage. Keep evidence on any doubt.
	_, before, err := readEndpointFile(s.Stage)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return EndpointState{}, err
	}
	after, err := os.Lstat(s.Stage)
	if err != nil || !os.SameFile(before, after) {
		return EndpointState{}, errors.New("endpoint stage identity changed")
	}
	if err = os.Remove(s.Stage); err != nil {
		return EndpointState{}, err
	}
	return state, nil
}

func (s EndpointStore) Publish(state EndpointState) error {
	_, err := s.publish(state)
	return err
}

// publish separates deterministic validation/encoding refusal from attempted
// durable IO. The owner poisons only after a writer was actually invoked.
func (s EndpointStore) publish(state EndpointState) (attempted bool, err error) {
	if err := s.validate(); err != nil {
		return false, err
	}
	if state.Namespace != s.Namespace {
		return false, errors.New("endpoint state namespace mismatch")
	}
	raw, err := EncodeEndpointState(state)
	if err != nil {
		return false, err
	}
	write := s.WriteFile
	if write == nil {
		write = durablefile.WriteAtomicStaged
	}
	return true, write(s.Path, raw, s.Stage)
}
