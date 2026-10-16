package historyarchive

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"sort"
)

// SelectedFiles retains prepass identities for one explicit selection. Do not
// use it concurrently or change Root between Describe and WriteTar.
type SelectedFiles struct {
	Root string
	// Limit bounds one file's size; zero is the general bound.
	Limit         int64
	described     map[string]os.FileInfo
	describedRoot string
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(b []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(b)
}
func fileDigest(ctx context.Context, r io.Reader, w io.Writer, limit int64) (int64, string, error) {
	h := sha256.New()
	dest := io.Writer(h)
	if w != nil {
		dest = io.MultiWriter(w, h)
	}
	n, e := io.CopyBuffer(dest, io.LimitReader(contextReader{ctx, r}, limit+1), make([]byte, 64*1024))
	if e == nil && n > limit {
		e = errors.New("archive file exceeds byte limit")
	}
	return n, hex.EncodeToString(h.Sum(nil)), e
}
func sameSource(before, after os.FileInfo) bool {
	return os.SameFile(before, after) && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}
func (s *SelectedFiles) Describe(ctx context.Context, paths []string) ([]FileEntry, error) {
	s.described = nil
	if len(paths) == 0 || len(paths) > MaxFiles {
		return nil, errors.New("invalid selected file count")
	}
	root, e := openSafeDir(s.Root, false)
	if e != nil {
		return nil, e
	}
	defer root.Close()
	infos := make(map[string]os.FileInfo, len(paths))
	out := make([]FileEntry, 0, len(paths))
	for _, p := range paths {
		if e = ctx.Err(); e != nil {
			return nil, e
		}
		if _, ok := infos[p]; ok {
			return nil, errors.New("duplicate selected path")
		}
		f, e := openRegularAt(root, p, unix.O_RDONLY)
		if e != nil {
			return nil, fmt.Errorf("source %q: %w", p, e)
		}
		before, e := f.Stat()
		if e != nil {
			f.Close()
			return nil, e
		}
		limit := s.Limit
		if limit == 0 {
			limit = MaxBundleBytes
		}
		if before.Size() > limit {
			f.Close()
			return nil, errors.New("selected file exceeds byte limit")
		}
		n, digest, e := fileDigest(ctx, f, nil, limit)
		after, se := f.Stat()
		f.Close()
		if e != nil {
			return nil, e
		}
		if se != nil {
			return nil, se
		}
		if !sameSource(before, after) || n != before.Size() {
			return nil, errors.New("source changed during description")
		}
		infos[p] = before
		out = append(out, FileEntry{Path: p, Size: n, Mtime: before.ModTime().UTC(), SHA256: digest})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	s.described = infos
	s.describedRoot = s.Root
	return out, nil
}
func (s *SelectedFiles) WriteTar(ctx context.Context, w io.Writer, m Manifest) error {
	if e := ValidateManifest(m); e != nil {
		return e
	}
	if s.Root != s.describedRoot || s.described == nil {
		return errors.New("source must be described before writing")
	}
	raw, e := json.Marshal(m)
	if e != nil {
		return e
	}
	if len(raw) > MaxMetadataBytes {
		return errors.New("manifest exceeds metadata limit")
	}
	root, e := openSafeDir(s.Root, false)
	if e != nil {
		return e
	}
	defer root.Close()
	tw := tar.NewWriter(w)
	if e = ctx.Err(); e != nil {
		return e
	}
	if e = tw.WriteHeader(&tar.Header{Name: "MANIFEST.json", Mode: 0600, Size: int64(len(raw)), Typeflag: tar.TypeReg}); e != nil {
		return e
	}
	if _, e = tw.Write(raw); e != nil {
		return e
	}
	for _, entry := range m.Files {
		before, ok := s.described[entry.Path]
		if !ok {
			return errors.New("manifest contains undescribed source")
		}
		f, e := openRegularAt(root, entry.Path, unix.O_RDONLY)
		if e != nil {
			return e
		}
		info, e := f.Stat()
		if e != nil {
			f.Close()
			return e
		}
		if !sameSource(before, info) || info.Size() != entry.Size || !info.ModTime().Equal(entry.Mtime) {
			f.Close()
			return errors.New("source changed since description")
		}
		e = tw.WriteHeader(&tar.Header{Name: "files/" + entry.Path, Mode: 0600, Size: entry.Size, ModTime: entry.Mtime, Typeflag: tar.TypeReg})
		if e != nil {
			f.Close()
			return e
		}
		n, digest, e := fileDigest(ctx, f, tw, entry.Size)
		after, se := f.Stat()
		f.Close()
		if e != nil {
			return e
		}
		if se != nil {
			return se
		}
		if !sameSource(before, after) || n != entry.Size || digest != entry.SHA256 {
			return errors.New("source changed while writing archive")
		}
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	return tw.Close()
}
