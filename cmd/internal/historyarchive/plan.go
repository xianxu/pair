package historyarchive

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

func canonicalRecipients(rs []string) []string {
	out := append([]string(nil), rs...)
	for j := range out {
		out[j] = strings.ToUpper(out[j])
	}
	sort.Strings(out)
	return out
}
func equalRecipients(a, b []string) bool {
	a = canonicalRecipients(a)
	b = canonicalRecipients(b)
	if len(a) != len(b) {
		return false
	}
	for j := range a {
		if a[j] != b[j] {
			return false
		}
	}
	return true
}
func (m Manifest) ID() (string, error) {
	if err := ValidateManifest(m); err != nil {
		return "", err
	}
	m.Files = append([]FileEntry(nil), m.Files...)
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	for i := range m.Files {
		m.Files[i].Mtime = m.Files[i].Mtime.UTC()
		m.Files[i].SHA256 = strings.ToLower(m.Files[i].SHA256)
	}
	m.From = m.From.UTC()
	m.To = m.To.UTC()
	m.Recipients = canonicalRecipients(m.Recipients)
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
func (m Manifest) Keys() (string, string, error) {
	id, err := m.ID()
	if err != nil {
		return "", "", err
	}
	return BundleKeys(m.Kind, m.From, m.To, id)
}

// BundleKeys derives object names from the public identity and time range.
func BundleKeys(kind string, from, to time.Time, id string) (string, string, error) {
	if err := validateKind(kind); err != nil {
		return "", "", err
	}
	if !validTime(from) || !validTime(to) || to.Before(from) || !validHex(id, 64) {
		return "", "", errors.New("invalid bundle key identity or UTC range")
	}
	base := fmt.Sprintf("%s/%s/%s-%s-%s-%s", kind, from.UTC().Format("2006/01"), kind, from.UTC().Format("20060102T150405.000000000Z"), to.UTC().Format("20060102T150405.000000000Z"), id)
	return base + ".tar.zst.gpg", base + ".json", nil
}
func PlanBundles(r ArchiveRequest, files []FileEntry) ([]Manifest, error) {
	return planBundles(r, files, false)
}
func planBundles(r ArchiveRequest, files []FileEntry, collected bool) ([]Manifest, error) {
	if err := validateRequest(r, collected); err != nil {
		return nil, err
	}
	if len(files) != len(r.Files) {
		return nil, errors.New("described files do not match request")
	}
	selected := map[string]bool{}
	for _, p := range r.Files {
		selected[p] = true
	}
	fs := append([]FileEntry(nil), files...)
	for j := range fs {
		f := &fs[j]
		if !selected[f.Path] {
			return nil, errors.New("unexpected or duplicate described file")
		}
		delete(selected, f.Path)
		if f.Size < 0 || f.Size > rawBundleLimit(r.Kind) || !validHex(f.SHA256, 64) || f.Mtime.IsZero() {
			return nil, errors.New("invalid described file")
		}
		f.Mtime = f.Mtime.UTC()
		f.SHA256 = strings.ToLower(f.SHA256)
	}
	sort.Slice(fs, func(i, j int) bool {
		a, b := fs[i].Mtime.Format("2006-01"), fs[j].Mtime.Format("2006-01")
		if a != b {
			return a < b
		}
		return fs[i].Path < fs[j].Path
	})
	compression := "zstd -19"
	if r.Kind == "codex-database" {
		compression = "zstd -9"
	}
	var out []Manifest
	var current Manifest
	var raw int64
	metadataBytes := 0
	base := Manifest{Version: Version, Kind: r.Kind, Source: r.Source, Compression: compression, Recipients: canonicalRecipients(r.Recipients)}
	// Reserve catalog/label overhead too, so an admitted bundle can be cataloged.
	// Keys and timestamps use a conservative upper bound; entries are counted exactly.
	prototype := CatalogRecord{Manifest: base, Sink: r.Sink, BundleKey: strings.Repeat("k", 256+2*len(r.Kind)), LabelKey: strings.Repeat("k", 256+2*len(r.Kind)), Label: BundleLabel{Version: Version, ID: strings.Repeat("a", 64), Kind: r.Kind, FileCount: MaxFiles, RawBytes: rawBundleLimit(r.Kind), Bytes: MaxCiphertextBytes, Compression: base.Compression, Recipients: base.Recipients, SHA256: strings.Repeat("a", 64)}}
	encodedBase, err := json.Marshal(prototype)
	if err != nil {
		return nil, err
	}
	overhead := len(encodedBase) + 128
	flush := func() error {
		if len(current.Files) == 0 {
			return nil
		}
		if err := ValidateManifest(current); err != nil {
			return err
		}
		out = append(out, current)
		current = Manifest{}
		raw = 0
		metadataBytes = 0
		return nil
	}
	for _, f := range fs {
		entry, err := json.Marshal(f)
		if err != nil {
			return nil, err
		}
		entryBytes := len(entry) + 1
		if entryBytes > MaxMetadataBytes-overhead {
			return nil, errors.New("single file metadata exceeds limit")
		}
		if len(current.Files) > 0 && (f.Mtime.Format("2006-01") != current.From.Format("2006-01") || f.Size > TargetBundleBytes-raw || entryBytes > MaxMetadataBytes-overhead-metadataBytes) {
			if err := flush(); err != nil {
				return nil, err
			}
		}
		if len(current.Files) == 0 {
			current = Manifest{Version: Version, Kind: r.Kind, Source: r.Source, From: f.Mtime, To: f.Mtime, Compression: compression, Recipients: canonicalRecipients(r.Recipients)}
		}
		current.Files = append(current.Files, f)
		raw += f.Size
		metadataBytes += entryBytes
		if f.Mtime.Before(current.From) {
			current.From = f.Mtime
		}
		if f.Mtime.After(current.To) {
			current.To = f.Mtime
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return out, nil
}
