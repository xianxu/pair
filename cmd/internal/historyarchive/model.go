package historyarchive

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Version            = 1
	MaxMetadataBytes   = 4 << 20
	MaxFiles           = 10000
	MaxBundleBytes     = 1 << 30
	TargetBundleBytes  = 256 << 20
	MaxCiphertextBytes = 2 << 30
	// MaxSnapshotRawBytes bounds a database snapshot's raw size. Snapshots
	// compress well, so the ciphertext bound is their real limit (pair#616).
	MaxSnapshotRawBytes = 8 << 30
)

// ArchiveRequest explicitly selects a source and destination; it grants no deletion authority.
// pair:m5-concept pure
type ArchiveRequest struct {
	Version    int            `json:"version"`
	Kind       string         `json:"kind"`
	Source     SourceIdentity `json:"source"`
	Files      []string       `json:"files"`
	Recipients []string       `json:"recipients"`
	Sink       string         `json:"sink"`
	Catalog    string         `json:"catalog"`
}

// pair:m5-concept pure
type SourceIdentity struct {
	Host       string `json:"host"`
	Root       string `json:"root"`
	Repository string `json:"repository,omitempty"`
	Worktree   string `json:"worktree,omitempty"`
}

// pair:m5-concept pure
type FileEntry struct {
	Path   string    `json:"path"`
	Size   int64     `json:"size"`
	Mtime  time.Time `json:"mtime"`
	SHA256 string    `json:"sha256"`
}

// pair:m5-concept pure
type Manifest struct {
	Version     int            `json:"version"`
	Kind        string         `json:"kind"`
	Source      SourceIdentity `json:"source"`
	From        time.Time      `json:"from"`
	To          time.Time      `json:"to"`
	Compression string         `json:"compression"`
	Recipients  []string       `json:"recipients"`
	Files       []FileEntry    `json:"files"`
}

// pair:m5-concept pure
type BundleLabel struct {
	Version     int                        `json:"version"`
	ID          string                     `json:"id"`
	Kind        string                     `json:"kind"`
	From        time.Time                  `json:"from"`
	To          time.Time                  `json:"to"`
	FileCount   int                        `json:"file_count"`
	RawBytes    int64                      `json:"raw_bytes"`
	Bytes       int64                      `json:"bytes"`
	Compression string                     `json:"compression"`
	Recipients  []string                   `json:"recipients"`
	SHA256      string                     `json:"sha256"`
	Extensions  map[string]json.RawMessage `json:"-"`
}

// pair:m5-concept pure
type CatalogRecord struct {
	Label      BundleLabel `json:"label"`
	Sink       string      `json:"sink"`
	BundleKey  string      `json:"bundle_key"`
	LabelKey   string      `json:"label_key"`
	VerifiedAt time.Time   `json:"verified_at"`
	Manifest   Manifest    `json:"manifest"`
}

func decodeStrict(b []byte, v any) error {
	if len(b) > MaxMetadataBytes {
		return errors.New("metadata exceeds 4 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	return nil
}
func metadataBound(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) > MaxMetadataBytes {
		return errors.New("metadata exceeds 4 MiB")
	}
	return nil
}
func validPath(p string) bool {
	return p != "" && p != "." && !strings.ContainsAny(p, "\\\x00") && !strings.HasPrefix(p, "/") && path.Clean(p) == p && p != ".." && !strings.HasPrefix(p, "../")
}
func validHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func validateRecipients(rs []string) error {
	if len(rs) == 0 {
		return errors.New("recipients required")
	}
	seen := map[string]bool{}
	for _, r := range rs {
		r = strings.ToUpper(r)
		if !validHex(r, 40) && !validHex(r, 64) {
			return errors.New("recipient must be a full 40 or 64 hex fingerprint")
		}
		if seen[r] {
			return errors.New("duplicate recipient")
		}
		seen[r] = true
	}
	return nil
}
func validateSource(s SourceIdentity) error {
	if strings.TrimSpace(s.Host) == "" || !filepath.IsAbs(s.Root) || strings.ContainsRune(s.Root, 0) {
		return errors.New("source requires host and absolute root")
	}
	return nil
}
func validTime(t time.Time) bool {
	_, off := t.Zone()
	return !t.IsZero() && off == 0 && t.Year() >= 1 && t.Year() <= 9999
}
func validateKind(k string) error {
	i, err := Defaults()
	if err != nil {
		return err
	}
	_, err = i.Resolve(k)
	return err
}
func parseRequest(b []byte) (ArchiveRequest, error) {
	var r ArchiveRequest
	if err := decodeStrict(b, &r); err != nil {
		return r, err
	}
	for j := range r.Recipients {
		r.Recipients[j] = strings.ToUpper(r.Recipients[j])
	}
	return r, nil
}
func DecodeRequest(b []byte) (ArchiveRequest, error) {
	r, err := parseRequest(b)
	if err != nil {
		return r, err
	}
	return r, ValidateRequest(r)
}

// DecodeRequestWithRoots resolves root tokens from caller-supplied environment values.
func DecodeRequestWithRoots(b []byte, home string, env map[string]string) (ArchiveRequest, error) {
	r, err := parseRequest(b)
	if err != nil {
		return r, err
	}
	inventory, err := Defaults()
	if err != nil {
		return r, err
	}
	r.Source.Root, err = inventory.ResolveRoot(r.Kind, r.Source.Root, home, env)
	if err != nil {
		return r, err
	}
	return r, ValidateRequest(r)
}
func ValidateRequest(r ArchiveRequest) error { return validateRequest(r, false) }

// rawBundleLimit is the raw-size bound for a kind's files and bundles: the
// general bound, or MaxSnapshotRawBytes for snapshot kinds, which are bounded
// by their ciphertext instead. Kind-based, so catalog reads agree with writes.
func rawBundleLimit(kind string) int64 {
	if inv, err := defaultInventory(); err == nil {
		if rule, err := inv.Resolve(kind); err == nil && rule.SnapshotRequired {
			return MaxSnapshotRawBytes
		}
	}
	return MaxBundleBytes
}

// defaultInventory decodes the embedded inventory once; validation runs per
// file and per catalog record.
var defaultInventory = sync.OnceValues(Defaults)

// validateRequest admits a snapshot-required kind only when collected: the
// continuous pass took a coherent private snapshot itself (pair#592). Manual
// requests never select a live database.
func validateRequest(r ArchiveRequest, collected bool) error {
	if r.Version != Version {
		return errors.New("unsupported request version")
	}
	i, err := Defaults()
	if err != nil {
		return err
	}
	rule, err := i.Resolve(r.Kind)
	if err != nil {
		return err
	}
	if rule.SnapshotRequired && !collected {
		return errors.New("snapshot source requires the snapshot collector; manual selection is not supported")
	}
	if err := validateSource(r.Source); err != nil {
		return err
	}
	if err := validateRecipients(r.Recipients); err != nil {
		return err
	}
	if !filepath.IsAbs(r.Sink) || !filepath.IsAbs(r.Catalog) || strings.ContainsRune(r.Sink+r.Catalog, 0) {
		return errors.New("absolute sink and catalog paths required")
	}
	if len(r.Files) == 0 || len(r.Files) > MaxFiles {
		return errors.New("select 1 to 10000 files")
	}
	seen := map[string]bool{}
	for _, p := range r.Files {
		if !validPath(p) || seen[p] {
			return fmt.Errorf("invalid or duplicate file path %q", p)
		}
		seen[p] = true
	}
	return metadataBound(r)
}
func ValidateManifest(m Manifest) error {
	if m.Version != Version {
		return errors.New("unsupported manifest version")
	}
	if err := validateKind(m.Kind); err != nil {
		return err
	}
	if err := validateSource(m.Source); err != nil {
		return err
	}
	if !validTime(m.From) || !validTime(m.To) || m.To.Before(m.From) {
		return errors.New("invalid UTC manifest range")
	}
	if _, err := compressionLevel(m.Compression); err != nil {
		return errors.New("unsupported compression")
	}
	if err := validateRecipients(m.Recipients); err != nil {
		return err
	}
	if len(m.Files) == 0 || len(m.Files) > MaxFiles {
		return errors.New("invalid manifest file count")
	}
	seen := map[string]bool{}
	var total int64
	limit := rawBundleLimit(m.Kind)
	var earliest, latest time.Time
	for _, f := range m.Files {
		if !validPath(f.Path) || seen[f.Path] {
			return errors.New("invalid or duplicate manifest path")
		}
		seen[f.Path] = true
		if f.Size < 0 || f.Size > limit-total {
			return errors.New("raw bundle exceeds limit")
		}
		total += f.Size
		if !validHex(f.SHA256, 64) || !validTime(f.Mtime) {
			return errors.New("invalid file hash or UTC mtime")
		}
		if earliest.IsZero() || f.Mtime.Before(earliest) {
			earliest = f.Mtime
		}
		if latest.IsZero() || f.Mtime.After(latest) {
			latest = f.Mtime
		}
	}
	if !m.From.Equal(earliest) || !m.To.Equal(latest) {
		return errors.New("range does not match file mtimes")
	}
	return metadataBound(m)
}
func ValidateLabel(l BundleLabel) error {
	if l.Version != Version || !validHex(l.ID, 64) || !validHex(l.SHA256, 64) {
		return errors.New("invalid label version or hash")
	}
	if err := validateKind(l.Kind); err != nil {
		return err
	}
	if !validTime(l.From) || !validTime(l.To) || l.To.Before(l.From) || !validCompression(l.Compression) || l.FileCount < 1 || l.FileCount > MaxFiles || l.RawBytes < 0 || l.RawBytes > rawBundleLimit(l.Kind) || l.Bytes <= 0 || l.Bytes > MaxCiphertextBytes {
		return errors.New("invalid label range or bounds")
	}
	if err := validateRecipients(l.Recipients); err != nil {
		return err
	}
	return metadataBound(l)
}
func ValidateRecord(r CatalogRecord) error {
	if err := ValidateLabel(r.Label); err != nil {
		return err
	}
	id, err := r.Manifest.ID()
	if err != nil {
		return err
	}
	b, l, err := r.Manifest.Keys()
	if err != nil {
		return err
	}
	var raw int64
	for _, f := range r.Manifest.Files {
		raw += f.Size
	}
	if r.Label.ID != id || r.Label.Kind != r.Manifest.Kind || !r.Label.From.Equal(r.Manifest.From) || !r.Label.To.Equal(r.Manifest.To) || r.Label.Compression != r.Manifest.Compression || r.Label.FileCount != len(r.Manifest.Files) || r.Label.RawBytes != raw || !equalRecipients(r.Label.Recipients, r.Manifest.Recipients) || r.BundleKey != b || r.LabelKey != l || strings.TrimSpace(r.Sink) == "" || !validTime(r.VerifiedAt) {
		return errors.New("catalog record does not match manifest or verification")
	}
	return metadataBound(r)
}

// Compression describes the stream, not the current writer policy. Future
// canonical positive levels remain readable by the same zstd decompressor.
func compressionLevel(descriptor string) (int, error) {
	value := strings.TrimPrefix(descriptor, "zstd -")
	level, err := strconv.Atoi(value)
	if err != nil || level < 1 || "zstd -"+strconv.Itoa(level) != descriptor {
		return 0, errors.New("unsupported compression descriptor")
	}
	return level, nil
}
func validCompression(descriptor string) bool {
	_, err := compressionLevel(descriptor)
	return err == nil
}
