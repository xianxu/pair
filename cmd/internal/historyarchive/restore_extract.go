package historyarchive

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func restoreEntryCapacity(budget *RestoreExtractionBudget) error {
	if budget == nil || budget.Entries < 0 || budget.Entries >= MaxRestoreEntries || budget.Files < 0 || budget.Files > MaxRestoreFiles {
		return errors.New("restore extraction entry budget exhausted or invalid")
	}
	return nil
}

// CreateRestoreDir exclusively creates one private directory, accounting for an
// actual new entry before any further work. Existing names are never adopted.
func CreateRestoreDir(parent *os.File, name string, budget *RestoreExtractionBudget) (*os.File, error) {
	if parent == nil || !validPath(name) || strings.Contains(name, "/") {
		return nil, errors.New("unsafe restore directory name")
	}
	if err := restoreEntryCapacity(budget); err != nil {
		return nil, err
	}
	if err := unix.Mkdirat(int(parent.Fd()), name, 0700); err != nil {
		return nil, err
	}
	budget.Entries++
	child, err := childDir(parent, name, false)
	if err == nil {
		err = parent.Sync()
	}
	if err != nil {
		if child != nil {
			child.Close()
		}
		return nil, err
	}
	return child, nil
}

// CreateRestoreFile safely walks relative components, reusing only nofollow
// directories and counting each created directory/file. Files are always new.
// Payload file counts belong to ExtractBundle, because this helper also creates
// ciphertext and manifest entries for the coordinator.
func CreateRestoreFile(parent *os.File, relative string, budget *RestoreExtractionBudget) (*os.File, error) {
	if parent == nil || !validPath(relative) {
		return nil, errors.New("unsafe restore file path")
	}
	if err := restoreEntryCapacity(budget); err != nil {
		return nil, err
	}
	dir, err := childDir(parent, ".", false)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(relative, "/")
	for _, part := range parts[:len(parts)-1] {
		next, e := childDir(dir, part, false)
		if errors.Is(e, unix.ENOENT) {
			next, e = CreateRestoreDir(dir, part, budget)
		}
		dir.Close()
		if e != nil {
			return nil, e
		}
		dir = next
	}
	defer dir.Close()
	if err = restoreEntryCapacity(budget); err != nil {
		return nil, err
	}
	f, err := openRegularAt(dir, parts[len(parts)-1], unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return nil, err
	}
	budget.Entries++
	if err = dir.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// restorePlainReader bounds every decompressed byte, including tar records that
// archive/tar consumes internally (PAX headers, padding, and end markers).
type restorePlainReader struct {
	ctx       context.Context
	reader    io.Reader
	remaining int64
}

func (r *restorePlainReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	if int64(len(p)) > r.remaining+1 {
		p = p[:int(r.remaining+1)]
	}
	n, err := r.reader.Read(p)
	if int64(n) > r.remaining {
		return n, errors.New("decompressed restore stream exceeds bound")
	}
	r.remaining -= int64(n)
	if cancelErr := r.ctx.Err(); cancelErr != nil {
		return n, cancelErr
	}
	return n, err
}

func regularRestoreHeader(h *tar.Header) bool {
	return h.Typeflag == tar.TypeReg || h.Typeflag == tar.TypeRegA
}

// ExtractBundle verifies the entire manifest-declared tar, including excluded
// members, and writes only literal selected paths into private owned staging.
// The caller must also require successful decoder completion before publication.
func ExtractBundle(ctx context.Context, plain io.Reader, label BundleLabel, output *os.File, targetParent string, paths []string, budget *RestoreExtractionBudget) (manifest Manifest, selected int, err error) {
	if err = ctx.Err(); err != nil {
		return manifest, 0, err
	}
	if plain == nil || output == nil || !filepath.IsAbs(targetParent) || filepath.Clean(targetParent) != targetParent || budget == nil || budget.Entries < 0 || budget.Entries > MaxRestoreEntries || budget.Files < 0 || budget.Files > MaxRestoreFiles {
		return manifest, 0, errors.New("invalid restore extraction arguments")
	}
	if err = ValidateLabel(label); err != nil {
		return manifest, 0, err
	}
	wanted := make(map[string]bool, len(paths))
	for _, path := range paths {
		if !validPath(path) {
			return manifest, 0, errors.New("unsafe requested restore path")
		}
		wanted[path] = true
	}
	bounded := &restorePlainReader{ctx: ctx, reader: plain, remaining: label.RawBytes + MaxRestoreTarOverhead}
	tr := tar.NewReader(bounded)
	header, err := tr.Next()
	if err != nil {
		return manifest, 0, err
	}
	if header.Name != "MANIFEST.json" || !regularRestoreHeader(header) || header.Size <= 0 || header.Size > MaxMetadataBytes {
		return manifest, 0, errors.New("restore tar must start with a bounded regular MANIFEST.json")
	}
	raw, err := io.ReadAll(io.LimitReader(tr, MaxMetadataBytes+1))
	if err != nil {
		return manifest, 0, err
	}
	if int64(len(raw)) != header.Size {
		return manifest, 0, errors.New("restore manifest size mismatch")
	}
	if err = decodeJSONNoDuplicateKeys(raw, &manifest, true); err != nil {
		return manifest, 0, err
	}
	if err = ValidateManifest(manifest); err != nil {
		return manifest, 0, err
	}
	id, err := manifest.ID()
	if err != nil {
		return manifest, 0, err
	}
	var rawBytes int64
	for _, entry := range manifest.Files {
		rawBytes += entry.Size
	}
	if label.ID != id || label.Version != manifest.Version || label.Kind != manifest.Kind || !label.From.Equal(manifest.From) || !label.To.Equal(manifest.To) || label.FileCount != len(manifest.Files) || label.RawBytes != rawBytes || label.Compression != manifest.Compression || !equalRecipients(label.Recipients, manifest.Recipients) {
		return manifest, 0, errors.New("restore label does not match manifest summary")
	}
	// The manifest is the first available source-root authority for remote-only
	// discovery. Refuse overlap before creating even the staged manifest.
	if overlaps(targetParent, manifest.Source.Root) {
		return manifest, 0, errors.New("restore target overlaps archived source root")
	}
	f, err := CreateRestoreFile(output, "MANIFEST.json", budget)
	if err != nil {
		return manifest, 0, err
	}
	_, err = f.Write(raw)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return manifest, 0, err
	}
	declared := make(map[string]FileEntry, len(manifest.Files))
	seen := make(map[string]bool, len(manifest.Files))
	for _, entry := range manifest.Files {
		declared["files/"+entry.Path] = entry
	}
	for {
		header, err = tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return manifest, selected, err
		}
		entry, ok := declared[header.Name]
		if !ok || seen[header.Name] || !regularRestoreHeader(header) || header.Size != entry.Size {
			return manifest, selected, fmt.Errorf("invalid, duplicate or undeclared tar member %q", header.Name)
		}
		seen[header.Name] = true
		hash := sha256.New()
		keep := len(paths) == 0 || wanted[entry.Path]
		var n int64
		if keep {
			if budget.Files >= MaxRestoreFiles {
				return manifest, selected, errors.New("restore payload file budget exhausted")
			}
			file, e := CreateRestoreFile(output, header.Name, budget)
			if e != nil {
				return manifest, selected, e
			}
			budget.Files++
			n, err = io.Copy(io.MultiWriter(file, hash), tr)
			if err == nil {
				err = file.Sync()
			}
			err = errors.Join(err, file.Close())
		} else {
			n, err = io.Copy(hash, tr)
		}
		if err != nil {
			return manifest, selected, err
		}
		if n != entry.Size || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), entry.SHA256) {
			return manifest, selected, fmt.Errorf("restored payload checksum/size mismatch: %q", entry.Path)
		}
		if keep {
			selected++
		}
	}
	if len(seen) != len(declared) {
		return manifest, selected, errors.New("restore tar is missing manifest-declared members")
	}
	// tar.Reader stops at the first pair of zero blocks. Drain the bounded
	// decompressed stream to reject appended archives/data and limit padding.
	buf := make([]byte, 32<<10)
	for {
		n, e := bounded.Read(buf)
		for _, b := range buf[:n] {
			if b != 0 {
				return manifest, selected, errors.New("restore tar contains trailing nonpadding data")
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return manifest, selected, e
		}
	}
	if err = ctx.Err(); err != nil {
		return manifest, selected, err
	}
	if err = output.Sync(); err != nil {
		return manifest, selected, err
	}
	return manifest, selected, nil
}
