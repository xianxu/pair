package submission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Stores for the same directory share one writer, including reopened stores.
var writers sync.Map

type Store struct {
	root     string
	mu       *sync.Mutex
	capacity int64
	fail     func(string) error
}

func NewStore(root string) (*Store, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, fmt.Errorf("submission root must be absolute and clean")
	}
	// Resolve the caller's existing ancestors once (macOS /var and /tmp are
	// standard symlinks), then keep all later operations under the real root.
	if st, err := os.Lstat(root); err == nil && st.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("submission root itself is a symlink")
	}
	ancestor := root
	var suffix []string
	for {
		_, err := os.Lstat(ancestor)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		suffix = append(suffix, filepath.Base(ancestor))
		ancestor = filepath.Dir(ancestor)
	}
	canonical, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return nil, err
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		canonical = filepath.Join(canonical, suffix[i])
	}
	root = canonical
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(root, 0700); err != nil {
		return nil, err
	}
	lock, _ := writers.LoadOrStore(root, &sync.Mutex{})
	return &Store{root: root, mu: lock.(*sync.Mutex), capacity: MaxStoreBytes}, nil
}
func noSymlinkParents(path string) error {
	for p := path; ; p = filepath.Dir(p) {
		st, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in submission path: %s", p)
		}
		if p == filepath.Dir(p) {
			return nil
		}
	}
}
func (s *Store) open() (*os.Root, error) {
	if err := noSymlinkParents(s.root); err != nil {
		return nil, err
	}
	return os.OpenRoot(s.root)
}
func (s *Store) ManifestPath(id string) string {
	if !identifier.MatchString(id) {
		return ""
	}
	return filepath.Join(s.root, id, "submission.json")
}
func (s *Store) checkpoint(op string) error {
	if s.fail != nil {
		return s.fail(op)
	}
	return nil
}
func (s *Store) write(r *os.Root, path string, data []byte) error {
	if err := s.checkpoint("write"); err != nil {
		return err
	}
	f, err := r.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = s.checkpoint("sync")
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func (s *Store) syncDir(r *os.Root, path string) error {
	if err := s.checkpoint("sync-dir:" + path); err != nil {
		return err
	}
	f, err := r.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// confirmPublication makes both the manifest name within the bundle and the
// bundle name within the store durable. Visibility after rename alone is not
// an acknowledgment: retries must run these syncs even if no bytes change.
func (s *Store) confirmPublication(r *os.Root, id string) error {
	if err := s.syncDir(r, id); err != nil {
		return err
	}
	return s.syncDir(r, ".")
}

func (s *Store) usage(r *os.Root) (int64, error) {
	var total int64
	count := 0
	err := fs.WalkDir(r.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		count++
		if count > 10000 {
			return fmt.Errorf("submission store has too many entries; clean up %s", s.root)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in submission store")
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("nonregular submission store entry")
			}
			if info.Size() > s.capacity-total {
				return fmt.Errorf("submission store full; clean up %s", s.root)
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}
func bundleFileInfo(r *os.Root, path string) (os.FileInfo, error) {
	// Lstat each component; os.Root additionally prevents traversal outside root.
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := range parts {
		st, err := r.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil {
			return nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink in bundle")
		}
		if i == len(parts)-1 {
			if !st.Mode().IsRegular() {
				return nil, fmt.Errorf("nonregular bundle file")
			}
			return st, nil
		}
	}
	return nil, fmt.Errorf("empty bundle path")
}
func readBytes(r *os.Root, path string, limit int64) ([]byte, error) {
	info, err := bundleFileInfo(r, path)
	if err != nil {
		return nil, err
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("bundle file exceeds limit")
	}
	f, err := r.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(b)) > limit {
		err = fmt.Errorf("bundle file exceeds limit")
	}
	return b, err
}
func readMetadata(r *os.Root, id string, skipResolvedCrops bool) (Submission, error) {
	var out Submission
	if !identifier.MatchString(id) {
		return out, fmt.Errorf("invalid submission ID")
	}
	b, err := readBytes(r, filepath.Join(id, "submission.json"), MaxBundleBytes)
	if err != nil {
		return out, err
	}
	out, err = DecodePreserving(b)
	if err != nil {
		return out, err
	}
	if out.ID != id {
		return out, fmt.Errorf("manifest identity mismatch")
	}
	if err = Validate(out); err != nil {
		return out, err
	}
	if skipResolvedCrops && out.Delivery == Resolved {
		return out, nil
	}
	total := int64(len(b))
	files, err := EvidenceFiles(out)
	if err != nil {
		return out, err
	}
	for _, file := range files {
		info, err := bundleFileInfo(r, filepath.Join(id, file))
		if err != nil {
			return out, err
		}
		if info.Size() > MaxPNGBytes {
			return out, fmt.Errorf("crop exceeds 8 MiB")
		}
		if info.Size() > MaxBundleBytes-total {
			return out, fmt.Errorf("bundle exceeds 64 MiB")
		}
		total += info.Size()
	}
	return out, nil
}
func readSubmission(r *os.Root, id string) (Submission, error) {
	out, err := readMetadata(r, id, false)
	if err != nil {
		return out, err
	}
	manifestInfo, err := bundleFileInfo(r, filepath.Join(id, "submission.json"))
	if err != nil {
		return out, err
	}
	total := manifestInfo.Size()
	if total > MaxBundleBytes {
		return out, fmt.Errorf("bundle exceeds 64 MiB")
	}
	files, err := EvidenceFiles(out)
	if err != nil {
		return out, err
	}
	for _, file := range files {
		data, err := readBytes(r, filepath.Join(id, file), MaxPNGBytes)
		if err != nil {
			return out, err
		}
		if int64(len(data)) > MaxBundleBytes-total {
			return out, fmt.Errorf("bundle exceeds 64 MiB")
		}
		total += int64(len(data))
		if err = validateEvidencePNG(out, file, data); err != nil {
			return out, err
		}
	}
	return out, nil
}

// ReadMetadata validates the manifest and referenced crop paths, file types,
// and byte sizes. It deliberately does not read or decode PNG contents; Read
// remains the full evidence-validation operation.
func (s *Store) ReadMetadata(id string) (Submission, error) {
	r, err := s.open()
	if err != nil {
		return Submission{}, err
	}
	defer r.Close()
	return readMetadata(r, id, false)
}
func (s *Store) Read(id string) (Submission, error) {
	r, err := s.open()
	if err != nil {
		return Submission{}, err
	}
	defer r.Close()
	return readSubmission(r, id)
}

// Pending returns unfinished bundles for this immutable target. It never infers delivery.
func (s *Store) Pending(target Target) ([]Submission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return s.pending(r, target, false)
}
func (s *Store) pending(r *os.Root, target Target, includeSuperseded bool) ([]Submission, error) {
	entries, err := submissionEntries(r)
	if err != nil {
		return nil, err
	}
	var result []Submission
	for _, entry := range entries {
		// Only submission directories participate in delivery reconciliation.
		// Finder metadata, staging residue and stray files are not bundles.
		if !entry.IsDir() || !identifier.MatchString(entry.Name()) {
			continue
		}
		sub, err := readMetadata(r, entry.Name(), true)
		if err != nil {
			return nil, err
		}
		if sub.Target == target && sub.Delivery != Resolved && (includeSuperseded || !sub.Superseded) {
			result = append(result, sub)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Prepare publishes an entire private bundle. A failure after rename may leave
// a complete visible bundle; retries confirm publication durability before
// returning success. Earlier failures leave no visible bundle.
func (s *Store) Prepare(sub Submission, crops map[string][]byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := Validate(sub); err != nil {
		return "", err
	}
	if sub.Delivery != Prepared || sub.Superseded {
		return "", fmt.Errorf("new submission must be prepared")
	}
	data, err := json.MarshalIndent(sub, "", "  ")
	if err != nil {
		return "", err
	}
	total := int64(len(data))
	expected := map[string]bool{}
	files, err := EvidenceFiles(sub)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		expected[file] = true
	}
	if len(expected) != len(crops) {
		return "", fmt.Errorf("crop files do not match manifest")
	}
	for name, b := range crops {
		if !expected[name] {
			return "", fmt.Errorf("unexpected crop file")
		}
		if err = validateEvidencePNG(sub, name, b); err != nil {
			return "", err
		}
		total += int64(len(b))
	}
	if total > MaxBundleBytes {
		return "", fmt.Errorf("bundle exceeds 64 MiB")
	}
	r, err := s.open()
	if err != nil {
		return "", err
	}
	defer r.Close()
	if _, err = r.Lstat(sub.ID); err == nil {
		old, err := readSubmission(r, sub.ID)
		if err != nil {
			return "", err
		}
		old.Delivery = Prepared
		old.Superseded = false
		oldData, marshalErr := json.MarshalIndent(old, "", "  ")
		if marshalErr != nil {
			return "", marshalErr
		}
		if !bytes.Equal(oldData, data) {
			return "", fmt.Errorf("submission ID already has different immutable data")
		}
		for name, b := range crops {
			existing, err := readBytes(r, filepath.Join(sub.ID, name), MaxPNGBytes)
			if err != nil {
				return "", err
			}
			if !bytes.Equal(existing, b) {
				return "", fmt.Errorf("submission crop content changed")
			}
		}
		if err = s.confirmPublication(r, sub.ID); err != nil {
			return "", err
		}
		return s.ManifestPath(sub.ID), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	pending, err := s.pending(r, sub.Target, true)
	if err != nil {
		return "", err
	}
	used, err := s.usage(r)
	if err != nil {
		return "", err
	}
	if total > s.capacity-used {
		return "", fmt.Errorf("submission store full; clean up %s", s.root)
	}
	stage, err := os.MkdirTemp(s.root, ".staging-")
	if err != nil {
		return "", err
	}
	name := filepath.Base(stage)
	defer r.RemoveAll(name)
	for file, b := range crops {
		if err = s.write(r, filepath.Join(name, file), b); err != nil {
			return "", err
		}
	}
	if err = s.write(r, filepath.Join(name, "submission.json"), data); err != nil {
		return "", err
	}
	if err = s.syncDir(r, name); err != nil {
		return "", err
	}
	// Release older authority durably before publishing the successor. A failed
	// publication may leave superseded evidence, never fabricated delivery.
	for _, old := range pending {
		if old.Delivery != Prepared && old.Delivery != Delivering {
			continue
		}
		if old.Superseded {
			// A previous attempt may have renamed its marker before fsync failed.
			err = s.confirmPublication(r, old.ID)
		} else {
			old.Delivery, old.Superseded, err = ApplyDisposition(old.Delivery, old.Superseded, Supersede)
			if err == nil {
				err = s.writeManifest(r, old)
			}
		}
		if err != nil {
			return "", err
		}
	}
	if err = s.checkpoint("rename"); err != nil {
		return "", err
	}
	if err = r.Rename(name, sub.ID); err != nil {
		return "", err
	}
	if err = s.confirmPublication(r, sub.ID); err != nil {
		return "", err
	}
	return s.ManifestPath(sub.ID), nil
}
func (s *Store) Transition(id string, target Target, event Event) (Submission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.open()
	if err != nil {
		return Submission{}, err
	}
	defer r.Close()
	sub, err := readMetadata(r, id, false)
	if err != nil {
		return sub, err
	}
	if sub.Target != target {
		return sub, fmt.Errorf("submission target changed")
	}
	next, superseded, err := ApplyDisposition(sub.Delivery, sub.Superseded, event)
	if err != nil {
		return sub, err
	}
	if next == sub.Delivery && superseded == sub.Superseded {
		return sub, s.confirmPublication(r, id)
	}
	sub.Delivery, sub.Superseded = next, superseded
	return sub, s.writeManifest(r, sub)
}

// writeManifest uses the same atomic durability path for disposition and delivery.
func (s *Store) writeManifest(r *os.Root, sub Submission) error {
	data, err := json.MarshalIndent(sub, "", "  ")
	if err != nil {
		return err
	}
	used, err := s.usage(r)
	if err != nil {
		return err
	}
	if int64(len(data)) > s.capacity-used {
		return fmt.Errorf("submission store full; clean up %s", s.root)
	}
	stage, err := os.MkdirTemp(s.root, ".staging-")
	if err != nil {
		return err
	}
	name := filepath.Base(stage)
	defer r.RemoveAll(name)
	temp := filepath.Join(name, "submission.json")
	if err = s.write(r, temp, data); err != nil {
		return err
	}
	if err = s.checkpoint("rename"); err != nil {
		return err
	}
	if err = r.Rename(temp, filepath.Join(sub.ID, "submission.json")); err != nil {
		return err
	}
	if err = s.confirmPublication(r, sub.ID); err != nil {
		return err
	}
	return nil
}

// DeliveredSketchVersions preserves the original sketch-only API without a
// second scan or a second acknowledgment authority.
func (s *Store) DeliveredSketchVersions(recordingID string) (map[PreviewItem]struct{}, error) {
	feedback, err := s.DeliveredFeedbackVersions(recordingID)
	if err != nil {
		return nil, err
	}
	return feedback.Sketches, nil
}

func submissionEntries(root *os.Root) ([]os.DirEntry, error) {
	f, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(10001)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(entries) > 10000 {
		return nil, fmt.Errorf("too many submission entries")
	}
	return entries, nil
}
