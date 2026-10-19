package historyarchive

import (
	"context"
	"errors"
	"testing"
)

func TestSnapshotFailureUsesEvidenceNotDiagnosticText(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status SnapshotStatus
	}{
		{errors.New("database is locked"), SnapshotRejected},
		{ErrSQLiteDeadline, SnapshotDeferred},
	} {
		got := snapshotFailure(tc.err, "database is locked")
		if got.Status != tc.status {
			t.Fatalf("%v: %v, want %v", tc.err, got.Status, tc.status)
		}
	}
	if snapshotFailure(context.DeadlineExceeded, "").Status != SnapshotDeferred {
		t.Fatal("deadline evidence lost")
	}
}

func TestSnapshotReadSecondsSizeScaledAndClamped(t *testing.T) {
	for _, tc := range []struct {
		bytes   int64
		seconds int
	}{
		{0, 5}, {-1, 5}, {475000000, 5}, {475000001, 6}, {1993663216, 21}, {2850000000, 30}, {1<<63 - 1, 30},
	} {
		if got := SnapshotReadSeconds(tc.bytes); got != tc.seconds {
			t.Fatalf("%d bytes: %ds, want %ds", tc.bytes, got, tc.seconds)
		}
	}
	last := 5
	for size := int64(0); size < MaxSnapshotRawBytes; size += 1234567 {
		got := SnapshotReadSeconds(size)
		if got < last || got < 5 || got > 30 {
			t.Fatalf("nonmonotone/unbounded %d -> %d", size, got)
		}
		last = got
	}
}
