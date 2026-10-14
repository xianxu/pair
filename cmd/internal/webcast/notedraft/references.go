package notedraft

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func referenceDir(r *os.Root, create bool) error {
	st, err := r.Lstat(".refs")
	if os.IsNotExist(err) && create {
		if err = r.Mkdir(".refs", 0700); err != nil {
			return err
		}
		return syncCaptureDir(r, ".")
	}
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("invalid capture reference directory")
	}
	return nil
}
func (s *diskStore) putRef(id string, m []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validRecord(id); err != nil {
		return err
	}
	if err := validMetadata(m); err != nil {
		return err
	}
	r, err := s.open()
	if err != nil {
		return err
	}
	defer r.Close()
	if err = referenceDir(r, true); err != nil {
		return err
	}
	path := ".refs/" + id + ".json"
	_, err = regularInfo(r, path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	entries, err := boundedEntries(r, ".refs")
	if err != nil {
		return err
	}
	if !exists && len(entries) >= 9999 {
		return fmt.Errorf("capture reference store full")
	}
	tmp := ".refs/" + tempCaptureName()
	defer r.Remove(tmp)
	if err = writeCaptureFile(r, tmp, m); err != nil {
		return err
	}
	if err = r.Rename(tmp, path); err != nil {
		return err
	}
	return syncCaptureDir(r, ".refs")
}
func (s *diskStore) readRef(id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validRecord(id); err != nil {
		return nil, err
	}
	r, err := s.open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err = referenceDir(r, false); err != nil {
		return nil, err
	}
	m, err := readBounded(r, ".refs/"+id+".json", maxMetadataBytes)
	if err == nil {
		err = validMetadata(m)
	}
	return m, err
}
func (s *diskStore) listRefs() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err = referenceDir(r, false); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	entries, err := boundedEntries(r, ".refs")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if validRecord(id) != nil {
			continue
		}
		if _, err = regularInfo(r, ".refs/"+entry.Name()); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}
func (s *diskStore) removeRef(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validRecord(id); err != nil {
		return err
	}
	r, err := s.open()
	if err != nil {
		return err
	}
	defer r.Close()
	if err = referenceDir(r, false); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	path := ".refs/" + id + ".json"
	if _, err = regularInfo(r, path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err = r.Remove(path); err != nil {
		return err
	}
	return syncCaptureDir(r, ".refs")
}

// persistLocked is called with Service.mu held. Reference metadata is outside
// source capacity; the published bundle remains owned by core.Store.
func (s *Service) persistLocked(r *captureRecord) error {
	if r.Ephemeral {
		return nil
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if r.BundlePath != "" {
		return s.disk.putRef(r.ID, data)
	}
	return s.disk.updateSourceMetadata(r.ID, data)
}

// recoverReferencesLocked runs after source metadata has been loaded, before
// starting workers. A confirmed reference wins an interrupted source retirement.
func (s *Service) recoverReferencesLocked() error {
	ids, err := s.disk.listRefs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		data, e := s.disk.readRef(id)
		if e != nil {
			continue
		}
		var ref captureRecord
		if e = json.Unmarshal(data, &ref); e != nil || validCaptureRecord(&ref, id) != nil || ref.BundlePath == "" {
			continue
		}
		if ref.RecordingPath != s.opts.RecordingPath {
			continue
		}
		if ref.State.Phase != Bundled && ref.State.Phase != Transferring && ref.State.Phase != Retired {
			continue
		}
		if ref.State.Phase == Retired {
			if e = s.disk.remove(id); e != nil {
				return e
			}
			if e = s.disk.removeRef(id); e != nil {
				return e
			}
			delete(s.records, id)
			continue
		}
		s.records[id] = &ref
		if _, e = readBundleAsset(&ref); e == nil {
			if e = s.disk.remove(id); e != nil {
				return e
			}
		}
	}
	return nil
}

// Metadata transitions must remain possible when retained pixels are missing or
// corrupt, so the published note can explicitly report unavailable evidence.
func (s *diskStore) updateSourceMetadata(id string, m []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validMetadata(m); err != nil {
		return err
	}
	root, err := s.open()
	if err != nil {
		return err
	}
	defer root.Close()
	if err = recordDir(root, id); err != nil {
		return err
	}
	if _, err = readBounded(root, id+"/metadata.json", maxMetadataBytes); err != nil {
		return err
	}
	// Admission reserved the complete metadata allowance for this source.
	// Ownership transitions consume that reservation, not new capture capacity;
	// unrelated unreadable records must not prevent saving or retiring it.

	return replaceMetadata(root, id, m)
}

func cleanupReferenceStaging(r *os.Root) error {
	if err := referenceDir(r, false); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	entries, err := boundedEntries(r, ".refs")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !stagingID.MatchString(entry.Name()) {
			continue
		}
		path := ".refs/" + entry.Name()
		if _, err = regularInfo(r, path); err != nil {
			return err
		}
		if err = r.Remove(path); err != nil {
			return err
		}
	}
	return syncCaptureDir(r, ".refs")
}
