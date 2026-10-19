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
