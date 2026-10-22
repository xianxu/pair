package remotesession

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"
	"sync"
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
	s := &OriginStore{path: path, record: originRecord{documents: make(map[string]map[string]json.RawMessage)}, publish: func(path string, raw []byte) error { return writeRemoteAtomic(path, raw, maxOriginsBytes) }}
	raw, err := readRemotePrivate(path, maxOriginsBytes)
	if errors.Is(err, os.ErrNotExist) {
		if err := reclaimRemoteStage(path, maxOriginsBytes); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	s.record, err = decodeOrigins(raw)
	if err != nil {
		return nil, err
	}
	if err := reclaimRemoteStage(path, maxOriginsBytes); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *OriginStore) Marked(id string) (bool, error) {
	marked, _, err := s.Inspect(id)
	return marked, err
}

// Inspect reads the cached edit evidence and forget fence in one lookup.
func (s *OriginStore) Inspect(id string) (marked, pending bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked || !validDocumentID(id) {
		return false, false, errors.New("remote origin evidence unavailable")
	}
	fields := s.record.documents[id]
	_, marked = fields["remote_edited"]
	_, pending = fields["forget_pending"]
	return marked, pending, nil
}
func (s *OriginStore) MarkRemote(id string) error {
	return s.markFlag(id, "remote_edited")
}

// MarkSubmitted records conservative, non-expiring submit evidence before the
// handoff. This release never clears it: retained receipts cannot prove that an
// old uncertain delivery settled after their retention window expires.
func (s *OriginStore) MarkSubmitted(id string) error {
	return s.markFlag(id, "submit_seen")
}

func (s *OriginStore) Submitted(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked || !validDocumentID(id) {
		return false, errors.New("remote origin evidence unavailable")
	}
	_, seen := s.record.documents[id]["submit_seen"]
	return seen, nil
}

func (s *OriginStore) markFlag(id, flag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked || !validDocumentID(id) {
		return errors.New("remote origin store blocked or invalid document")
	}
	if _, pending := s.record.documents[id]["forget_pending"]; pending {
		return errors.New("remote origin document has pending forget")
	}
	if _, marked := s.record.documents[id][flag]; marked {
		return nil
	}
	if _, present := s.record.documents[id]; !present && len(s.record.documents) >= maxOriginMarkers {
		return errors.New("remote origin capacity reached; retire settled forgotten registrations locally")
	}
	next := s.copyRecord()
	fields := make(map[string]json.RawMessage, len(next.documents[id])+1)
	for key, value := range next.documents[id] {
		fields[key] = value
	}
	fields[flag] = json.RawMessage("true")
	next.documents[id] = fields
	return s.commit(next)
}

// BeginForget durably records the single owner's intent before registry removal.
// The caller must fence new work for this exact registration ID before calling.
// An intent-only record does not assert remote editing.
func (s *OriginStore) BeginForget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked || !validDocumentID(id) {
		return errors.New("remote origin store blocked or invalid document")
	}
	if _, pending := s.record.documents[id]["forget_pending"]; pending {
		return nil
	}
	if _, present := s.record.documents[id]; !present && len(s.record.documents) >= maxOriginMarkers {
		return errors.New("remote origin capacity reached; retire settled forgotten registrations locally")
	}
	next := s.copyRecord()
	fields := make(map[string]json.RawMessage, len(next.documents[id])+1)
	for key, value := range next.documents[id] {
		fields[key] = value
	}
	fields["forget_pending"] = json.RawMessage("true")
	next.documents[id] = fields
	return s.commit(next)
}

// PendingForgets returns exact persisted IDs for recovery by the sole owner.
// Mere absence from the registry never creates authority to retire an ID.
func (s *OriginStore) PendingForgets() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked {
		return nil, errors.New("remote origin evidence unavailable")
	}
	ids := make([]string, 0)
	for id, fields := range s.record.documents {
		if _, pending := fields["forget_pending"]; pending {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// RetireForgotten is called only after the sole Couch owner proves that this
// exact registration ID is absent and its work and receipts have settled. It
// requires a persisted intent; no remote action exposes retirement.
func (s *OriginStore) RetireForgotten(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.blocked || !validDocumentID(id) {
		return errors.New("remote origin store blocked or invalid document")
	}
	if _, seen := s.record.documents[id]["submit_seen"]; seen {
		return errors.New("submitted document origin cannot be retired")
	}
	if _, pending := s.record.documents[id]["forget_pending"]; !pending {
		return errors.New("remote origin retirement requires persisted forget intent")
	}
	next := s.copyRecord()
	delete(next.documents, id)
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
