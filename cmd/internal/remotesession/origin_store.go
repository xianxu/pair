package remotesession

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

// OriginStore retains conservative edit evidence across editor/supervisor
// lifetimes. Absence is never a claim of clean provenance for legacy documents.
type OriginStore struct {
	mu      sync.Mutex
	path    string
	record  originRecord
	blocked bool
	publish func(string, []byte) error
}

func readRemotePrivate(path string, limit int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return nil, errors.New("unsafe remote state file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, errors.New("remote state changed during open")
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > limit {
		return nil, errors.New("remote state exceeds size limit")
	}
	return raw, nil
}
func OpenOriginStore(path string) (*OriginStore, error) {
	s := &OriginStore{path: path, record: originRecord{documents: make(map[string]map[string]json.RawMessage)}, publish: func(path string, raw []byte) error { return durablefile.WriteAtomic(path, raw, ".remote-origin-*") }}
	raw, err := readRemotePrivate(path, maxOriginsBytes)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	s.record, err = decodeOrigins(raw)
	if err != nil {
		return nil, err
	}
	return s, nil
}
func (s *OriginStore) Marked(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked || !validDocumentID(id) {
		return false, errors.New("remote origin evidence unavailable")
	}
	_, marked := s.record.documents[id]
	return marked, nil
}
func (s *OriginStore) MarkRemote(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked || !validDocumentID(id) {
		return errors.New("remote origin store blocked or invalid document")
	}
	if _, marked := s.record.documents[id]; marked {
		return nil
	}
	if len(s.record.documents) >= maxOriginMarkers {
		return errors.New("remote origin capacity reached; retire settled forgotten registrations locally")
	}
	next := s.copyRecord()
	next.documents[id] = map[string]json.RawMessage{"remote_edited": json.RawMessage("true")}
	return s.commit(next)
}

// retire is called only by the local registration lifecycle after forgotten
// IDs have no unsettled operations. No remote action exposes retirement.
func (s *OriginStore) retire(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked {
		return errors.New("remote origin store blocked")
	}
	next := s.copyRecord()
	for _, id := range ids {
		if !validDocumentID(id) {
			return errors.New("invalid retired document ID")
		}
		delete(next.documents, id)
	}
	return s.commit(next)
}
func (s *OriginStore) copyRecord() originRecord {
	next := originRecord{root: s.record.root, documents: make(map[string]map[string]json.RawMessage, len(s.record.documents))}
	for id, fields := range s.record.documents {
		next.documents[id] = fields
	}
	return next
}
func (s *OriginStore) commit(next originRecord) error {
	raw, err := encodeOrigins(next)
	if err != nil {
		return err
	}
	if err := s.publish(s.path, raw); err != nil {
		s.blocked = true
		return err
	}
	s.record = next
	return nil
}
