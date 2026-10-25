package launcher

import (
	"context"
	"io"

	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// AcquireTerminalLease keeps the existing session-lease API while sharing the
// stable-inode kernel lease with namespace probe admission (#866).
func AcquireTerminalLease(ctx context.Context, path string) (io.Closer, error) {
	return zellijquery.AcquireFileLease(ctx, path)
}
