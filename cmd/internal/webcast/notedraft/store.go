package notedraft

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
)

const maxMetadataBytes = 64 << 10

var captureID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var stagingID = regexp.MustCompile(`^\.stage-[0-9a-f]{32}$`)

// Stores in one process share a writer for each canonical root. Independent
// processes must not write the same private capture root.
var captureWriters sync.Map

type diskStore struct {
	root string
	mu   *sync.Mutex
}

func newDiskStore(root string) (*diskStore, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, fmt.Errorf("capture root must be absolute and clean")
	}
	if st, e := os.Lstat(root); e == nil && st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("capture root is a symlink")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(canonical, 0700); err != nil {
		return nil, err
	}
	lock, _ := captureWriters.LoadOrStore(canonical, &sync.Mutex{})
	s := &diskStore{root: canonical, mu: lock.(*sync.Mutex)}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err = cleanupStaging(r); err != nil {
		return nil, err
	}
	if err = cleanupReferenceStaging(r); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *diskStore) open() (*os.Root, error) {
	for p := s.root; ; p = filepath.Dir(p) {
		st, err := os.Lstat(p)
		if err != nil {
			return nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink in capture root")
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return os.OpenRoot(s.root)
}
func validRecord(id string) error {
	if !captureID.MatchString(id) {
		return fmt.Errorf("invalid capture ID")
	}
	return nil
}
func validMetadata(m []byte) error {
	if len(m) > maxMetadataBytes || !json.Valid(m) {
		return fmt.Errorf("invalid capture metadata (maximum 64 KiB JSON)")
	}
	return nil
}
func recordDir(r *os.Root, id string) error {
	if err := validRecord(id); err != nil {
		return err
	}
	st, err := r.Lstat(id)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("non-directory capture record")
	}
	return nil
}
func regularInfo(r *os.Root, path string) (os.FileInfo, error) {
	st, err := r.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("nonregular capture file")
	}
	return st, nil
}
func readBounded(r *os.Root, path string, limit int64) ([]byte, error) {
	st, err := regularInfo(r, path)
	if err != nil {
		return nil, err
	}
	if st.Size() > limit {
		return nil, fmt.Errorf("capture file exceeds limit")
	}
	f, err := r.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(st, opened) {
		return nil, fmt.Errorf("capture file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = fmt.Errorf("capture file exceeds limit")
	}
	return data, err
}
func readRecord(r *os.Root, id string) ([]byte, []byte, error) {
	if err := recordDir(r, id); err != nil {
		return nil, nil, err
	}
	m, err := readBounded(r, id+"/metadata.json", maxMetadataBytes)
	if err != nil {
		return nil, nil, err
	}
	if err = validMetadata(m); err != nil {
		return nil, nil, err
	}
	p, err := readBounded(r, id+"/capture.png", MaxPNGBytes)
	if os.IsNotExist(err) {
		err = nil
	}
	if err == nil && len(p) > 0 {
		err = core.ValidatePNG(p)
	}
	return m, p, err
}
func (s *diskStore) read(id string) ([]byte, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	return readRecord(r, id)
}
func boundedEntries(r *os.Root, path string) ([]os.DirEntry, error) {
	f, err := r.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(10001)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > 10000 {
		return nil, fmt.Errorf("too many capture store entries")
	}
	return entries, nil
}

// Called only by the owning service at startup, before workers can publish.
// Cleanup is bounded and never descends into an arbitrary directory.
func cleanupStaging(r *os.Root) error {
	entries, err := boundedEntries(r, ".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		staged := stagingID.MatchString(name)
		if !entry.IsDir() || (!staged && !captureID.MatchString(name)) {
			continue
		}
		children, err := boundedEntries(r, name)
		if err != nil {
			return err
		}
		for _, child := range children {
			owned := stagingID.MatchString(child.Name()) || (staged && (child.Name() == "metadata.json" || child.Name() == "capture.png"))
			if !owned {
				continue
			}
			path := name + "/" + child.Name()
			if _, err = regularInfo(r, path); err != nil {
				return err
			}
			if err = r.Remove(path); err != nil {
				return err
			}
		}
		if staged {
			if err = r.Remove(name); err != nil {
				return err
			}
		} else if err = syncCaptureDir(r, name); err != nil {
			return err
		}
	}
	return syncCaptureDir(r, ".")
}

func listRecords(r *os.Root) ([]string, error) {
	entries, err := boundedEntries(r, ".")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if captureID.MatchString(e.Name()) && e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}
func (s *diskStore) list() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return listRecords(r)
}
func storeUsage(r *os.Root) (int, int64, error) {
	ids, err := listRecords(r)
	if err != nil {
		return 0, 0, err
	}
	var total int64
	for _, id := range ids {
		if err = recordDir(r, id); err != nil {
			return 0, 0, err
		}
		for _, name := range []string{"metadata.json", "capture.png"} {
			st, e := regularInfo(r, id+"/"+name)
			if os.IsNotExist(e) && name == "capture.png" {
				continue
			}
			if e != nil {
				return 0, 0, e
			}
			total += st.Size()
		}
	}
	return len(ids), total, nil
}
func (s *diskStore) usage() (int, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return 0, 0, err
	}
	defer r.Close()
	return storeUsage(r)
}
func syncCaptureDir(r *os.Root, path string) error {
	f, err := r.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func writeCaptureFile(r *os.Root, path string, b []byte) error {
	f, err := r.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err != nil {
		return err
	}
	return ce
}
func tempCaptureName() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return ".stage-" + hex.EncodeToString(b[:])
}
func replaceMetadata(r *os.Root, id string, m []byte) error {
	path := id + "/" + tempCaptureName()
	defer r.Remove(path)
	if err := writeCaptureFile(r, path, m); err != nil {
		return err
	}
	if err := r.Rename(path, id+"/metadata.json"); err != nil {
		return err
	}
	return syncCaptureDir(r, id)
}
func (s *diskStore) put(id string, m, p []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putLocked(id, m, p)
}
func (s *diskStore) putLocked(id string, m, p []byte) error {
	if err := validRecord(id); err != nil {
		return err
	}
	if err := validMetadata(m); err != nil {
		return err
	}
	if len(p) > 0 {
		if err := core.ValidatePNG(p); err != nil {
			return err
		}
	}
	r, err := s.open()
	if err != nil {
		return err
	}
	defer r.Close()
	n, total, err := storeUsage(r)
	if err != nil {
		return err
	}
	_, err = r.Lstat(id)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if exists {
		old, oldPNG, e := readRecord(r, id)
		if e != nil {
			return e
		}
		if len(oldPNG) > 0 && !bytes.Equal(oldPNG, p) {
			return fmt.Errorf("capture PNG is immutable")
		}
		if reason := Capacity(n, total-int64(len(old)+len(oldPNG)), 0, int64(len(m)+len(p))); reason != "" {
			return fmt.Errorf("%s", reason)
		}
		if len(oldPNG) == 0 && len(p) > 0 {
			tmp := id + "/" + tempCaptureName()
			defer r.Remove(tmp)
			if e = writeCaptureFile(r, tmp, p); e != nil {
				return e
			}
			if e = r.Rename(tmp, id+"/capture.png"); e != nil {
				return e
			}
			if e = syncCaptureDir(r, id); e != nil {
				return e
			}
		}
		if err = replaceMetadata(r, id, m); err != nil {
			return err
		}
		return syncCaptureDir(r, ".")
	}
	if reason := Capacity(n, total, 1, int64(len(m)+len(p))); reason != "" {
		return fmt.Errorf("%s", reason)
	}
	stage := tempCaptureName()
	if err = r.Mkdir(stage, 0700); err != nil {
		return err
	}
	defer func() {
		_ = r.Remove(stage + "/metadata.json")
		_ = r.Remove(stage + "/capture.png")
		_ = r.Remove(stage)
	}()
	if err = writeCaptureFile(r, stage+"/metadata.json", m); err != nil {
		return err
	}
	if len(p) > 0 {
		if err = writeCaptureFile(r, stage+"/capture.png", p); err != nil {
			return err
		}
	}
	if err = syncCaptureDir(r, stage); err != nil {
		return err
	}
	if err = r.Rename(stage, id); err != nil {
		return err
	}
	return syncCaptureDir(r, ".")
}
func (s *diskStore) update(id string, m []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validMetadata(m); err != nil {
		return err
	}
	r, err := s.open()
	if err != nil {
		return err
	}
	defer r.Close()
	old, _, err := readRecord(r, id)
	if err != nil {
		return err
	}
	n, total, err := storeUsage(r)
	if err != nil {
		return err
	}
	if reason := Capacity(n, total-int64(len(old)), 0, int64(len(m))); reason != "" {
		return fmt.Errorf("%s", reason)
	}
	if err = replaceMetadata(r, id, m); err != nil {
		return err
	}
	return syncCaptureDir(r, ".")
}
func (s *diskStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.removeLocked(id)
}
func (s *diskStore) removeLocked(id string) error {
	if err := validRecord(id); err != nil {
		return err
	}
	r, err := s.open()
	if err != nil {
		return err
	}
	defer r.Close()
	if err = recordDir(r, id); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	// Never recursively delete arbitrary contents: only this store's two files.
	entries, err := boundedEntries(r, id)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != "metadata.json" && entry.Name() != "capture.png" {
			return fmt.Errorf("foreign entry in capture record")
		}
	}
	for _, name := range []string{"metadata.json", "capture.png"} {
		if _, err = regularInfo(r, id+"/"+name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	// Retire the complete record atomically before unlinking its contents.
	// A crash during unlink leaves a staging directory handled on reopen.
	stage := tempCaptureName()
	if err = r.Rename(id, stage); err != nil {
		return err
	}
	if err = syncCaptureDir(r, "."); err != nil {
		return err
	}
	for _, name := range []string{"metadata.json", "capture.png"} {
		if err = r.Remove(stage + "/" + name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err = r.Remove(stage); err != nil {
		return err
	}
	return syncCaptureDir(r, ".")
}

// reserve holds the shared writer from budget observation through publication.
// Every source reserves its metadata maximum; pending captures also reserve PNG.
func (s *diskStore) reserve(id string, m []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return err
	}
	defer r.Close()
	ids, err := listRecords(r)
	if err != nil {
		return err
	}
	_, total, err := storeUsage(r)
	if err != nil {
		return err
	}
	for _, other := range ids {
		info, e := regularInfo(r, other+"/metadata.json")
		if e != nil {
			return errors.New(StoreFull)
		}
		total += max(int64(0), int64(maxMetadataBytes)-info.Size())
		data, e := readMetadata(r, other)
		var record captureRecord
		unknown := e != nil || json.Unmarshal(data, &record) != nil || validCaptureRecord(&record, other) != nil
		if unknown || record.State.Phase == Preparing {
			total += MaxPNGBytes
		}
	}

	if reason := Capacity(len(ids), total, 1, MaxPNGBytes+maxMetadataBytes); reason != "" {
		return errors.New(reason)
	}
	return s.putLocked(id, m, nil)
}
func readMetadata(r *os.Root, id string) ([]byte, error) {
	if err := recordDir(r, id); err != nil {
		return nil, err
	}
	data, err := readBounded(r, id+"/metadata.json", maxMetadataBytes)
	if err == nil {
		err = validMetadata(data)
	}
	return data, err
}
func (s *diskStore) metadata(id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return readMetadata(r, id)
}

// sweepSources only removes proven unsaved expiry or an unclaimed source whose
// recording directory is absent. Uncertain writer/publication intents survive.
func (s *diskStore) sweepSources(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	root, err := s.open()
	if err != nil {
		return err
	}
	defer root.Close()
	ids, err := listRecords(root)
	if err != nil {
		return err
	}
	for _, id := range ids {
		data, e := readMetadata(root, id)
		if e != nil {
			continue
		}
		var r captureRecord
		if json.Unmarshal(data, &r) != nil || validCaptureRecord(&r, id) != nil {
			continue
		}
		expired := (r.State.Phase == Preparing || r.State.Phase == Ready) && !now.Before(r.Expires)
		gone := r.State.Phase == Attached && recordingAbsent(r.RecordingPath)
		if expired || gone || r.State.Phase == Retired {
			if e = s.removeLocked(id); e != nil {
				return e
			}
		}
	}
	if err = referenceDir(root, false); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	entries, err := boundedEntries(root, ".refs")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if validRecord(id) != nil {
			continue
		}
		path := ".refs/" + entry.Name()
		data, err := readBounded(root, path, maxMetadataBytes)
		if err != nil {
			continue
		}
		var r captureRecord
		if json.Unmarshal(data, &r) != nil || validCaptureRecord(&r, id) != nil {
			continue
		}
		if r.State.Phase == Retired || (r.State.Phase == Bundled && recordingAbsent(r.RecordingPath)) {
			if err = s.removeLocked(id); err != nil {
				return err
			}
			if err = root.Remove(path); err != nil {
				return err
			}
		}
	}
	if err = syncCaptureDir(root, ".refs"); err != nil {
		return err
	}
	return nil
}
