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
