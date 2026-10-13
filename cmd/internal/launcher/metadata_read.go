package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"golang.org/x/sys/unix"
)

const maxAgentReadyBytes int64 = 8192

// readBoundedRegularMetadata refuses special files before opening, and opens
// nonblocking without following a final symlink so replacement by a FIFO cannot
// strand an admission observer. Metadata lives on the local runtime filesystem.
func readBoundedRegularMetadata(ctx context.Context, path string, maxBytes int64) ([]byte, error) {
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
		return nil, unsafeTerminalError(fmt.Sprintf("metadata %q is not a regular file", path))
	}
	if before.Size() > maxBytes {
		return nil, unsafeTerminalError(fmt.Sprintf("metadata %q exceeds size bound", path))
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
		return nil, unsafeTerminalError(fmt.Sprintf("metadata %q exceeds size bound", path))
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
