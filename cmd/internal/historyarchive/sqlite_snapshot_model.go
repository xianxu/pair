package historyarchive

import (
	"context"
	"errors"
	"strings"
)

// SnapshotStatus describes acquisition, never encrypted publication.
// pair:m5-concept pure
type SnapshotStatus string

const (
	SnapshotReady    SnapshotStatus = "ready"
	SnapshotDeferred SnapshotStatus = "deferred"
	SnapshotRejected SnapshotStatus = "rejected"
)

// SnapshotResult is a coherent staging candidate or a bounded refusal.
// pair:m5-concept pure
type SnapshotResult struct {
	Status SnapshotStatus
	Detail string
}

// ErrSQLiteDeadline is evidence that the process budget expired.
var ErrSQLiteDeadline = errors.New("SQLite reader deadline exceeded")

// SnapshotReadSeconds gives twice the estimated copy time at 190 decimal MB/s,
// rounded up and clamped to 5..30 seconds. Clamp before arithmetic can overflow.
// pair:m5-concept pure
func SnapshotReadSeconds(rawBytes int64) int {
	const bytesPerSecond = 190_000_000 / 2
	if rawBytes <= 5*bytesPerSecond {
		return 5
	}
	if rawBytes >= 30*bytesPerSecond {
		return 30
	}
	return int((rawBytes + bytesPerSecond - 1) / bytesPerSecond)
}

func snapshotFailure(err error, diagnostic string) SnapshotResult {
	status := SnapshotRejected
	if errors.Is(err, ErrSQLiteDeadline) || errors.Is(err, context.DeadlineExceeded) {
		status = SnapshotDeferred
	}
	detail := err.Error()
	if diagnostic = strings.TrimSpace(diagnostic); diagnostic != "" {
		detail += ": " + diagnostic
	}
	if len(detail) > 8192 {
		detail = detail[:8192] + " [truncated]"
	}
	return SnapshotResult{Status: status, Detail: detail}
}
