package launcher

import (
	"context"
	"errors"
	"math"
	"os"

	"github.com/xianxu/pair/cmd/internal/osfs"
)

func readNativeOwnerMetadata(ctx context.Context, path string, limit int64) ([]byte, error) {
	if limit == -1 {
		if ctx == nil {
			return nil, errors.New("native ledger read requires context")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Ledger writers have no file-size cap (#508). Bound this read by the
		// finite observed snapshot instead; regular-file/no-symlink checks and
		// generation fences still belong to the shared metadata reader.
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Size() < 0 || info.Size() >= math.MaxInt64-1 {
			return nil, errors.New("invalid native ledger snapshot size")
		}
		limit = info.Size() + 1
	}
	return osfs.ReadRegularMetadata(ctx, path, limit)
}
