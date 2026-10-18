package couchcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"golang.org/x/sys/unix"
)

// AttentionStoreLimit bounds the saved list: AttentionCap items with bounded
// text fit well inside it.
const AttentionStoreLimit = 256 << 10

const attentionStoreVersion = 1

// AttentionStore keeps the attention list across a Couch restart (pair#673):
// one versioned file under the Couch namespace, replaced atomically.
type AttentionStore struct{ Dir string }

type attentionFile struct {
	Version int             `json:"version"`
	Items   []AttentionItem `json:"items"`
}

// Load returns the saved items; no file is an empty list.
func (s AttentionStore) Load() ([]AttentionItem, error) {
	path := filepath.Join(s.Dir, "list.json")
	if err := provisionSafePath(path); err != nil {
		return nil, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("unsafe attention list")
	}
	if info.Size() > AttentionStoreLimit {
		return nil, errors.New("attention list exceeds byte cap")
	}
	raw, err := io.ReadAll(io.LimitReader(file, AttentionStoreLimit+1))
	if err != nil {
		return nil, err
	}
	var saved attentionFile
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&saved); err != nil {
		return nil, fmt.Errorf("attention list: %w", err)
	}
	if saved.Version != attentionStoreVersion {
		return nil, fmt.Errorf("attention list version %d, want %d", saved.Version, attentionStoreVersion)
	}
	return saved.Items, nil
}

// Save replaces the saved list.
func (s AttentionStore) Save(items []AttentionItem) error {
	raw, err := json.Marshal(attentionFile{Version: attentionStoreVersion, Items: items})
	if err != nil {
		return err
	}
	if len(raw) > AttentionStoreLimit {
		return errors.New("attention list exceeds byte cap")
	}
	if err := provisionMkdirAll(s.Dir); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, "list.json")
	if err := provisionSafePath(path); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("unsafe attention list")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return durablefile.WriteAtomicStaged(path, raw, filepath.Join(s.Dir, ".list-stage"))
}
