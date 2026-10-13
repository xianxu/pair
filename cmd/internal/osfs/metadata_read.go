package osfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"golang.org/x/sys/unix"
)

// InvalidMetadataError identifies a static type or size violation. IO errors
// and concurrent file changes remain distinct, inconclusive observations.
type InvalidMetadataError struct{ Path, Reason string }

func (e *InvalidMetadataError) Error() string { return fmt.Sprintf("metadata %q %s", e.Path, e.Reason) }

// ReadRegularMetadata refuses special files before opening, and opens
// nonblocking without following a final symlink so replacement by a FIFO cannot
// strand an admission observer. Metadata lives on the local runtime filesystem.
func ReadRegularMetadata(ctx context.Context, path string, maxBytes int64) ([]byte, error) {
	if ctx == nil {
		return nil, errors.New("metadata read requires context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxBytes <= 0 || maxBytes == math.MaxInt64 {
		return nil, errors.New("invalid metadata size bound")
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, &InvalidMetadataError{Path: path, Reason: "is not a regular file"}
	}
	if before.Size() > maxBytes {
		return nil, &InvalidMetadataError{Path: path, Reason: "exceeds size bound"}
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !sameMetadataFile(before, opened) {
		return nil, fmt.Errorf("metadata %q changed during open", path)
	}
	raw, err := io.ReadAll(io.LimitReader(contextMetadataReader{ctx: ctx, reader: f}, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, &InvalidMetadataError{Path: path, Reason: "exceeds size bound"}
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	current, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !sameMetadataFile(opened, after) || !sameMetadataFile(after, current) {
		return nil, fmt.Errorf("metadata %q changed during read", path)
	}
	return raw, nil
}

func sameMetadataFile(a, b os.FileInfo) bool {
	return a.Mode().IsRegular() && b.Mode().IsRegular() && os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

type contextMetadataReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextMetadataReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
