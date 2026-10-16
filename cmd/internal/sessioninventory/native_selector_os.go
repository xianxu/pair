package sessioninventory

import (
	"bytes"
	"context"
	"encoding/csv"
	"os/exec"
)

// QueryNativeSelectorSQLite supplies bounded, cancellable SQLite reads for
// explicit native inventory validation.
func (r OSRuntime) QueryNativeSelectorSQLite(ctx context.Context, a Artifact, query string, limit int64) (SQLiteResult, error) {
	if err := ctx.Err(); err != nil {
		return SQLiteResult{}, err
	}
	path, err := r.resolveArtifact(a)
	if err != nil {
		return SQLiteResult{}, err
	}
	if limit <= 0 || limit > 1<<20 {
		return SQLiteResult{}, ErrReadLimit
	}
	stdout, stderr := newBoundedBuffer(limit), newBoundedBuffer(8192)
	command := exec.CommandContext(ctx, "sqlite3", "-readonly", "-csv", "-header", path, query)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return SQLiteResult{}, err
	}
	if stdout.exceeded || stderr.exceeded {
		return SQLiteResult{}, ErrReadLimit
	}
	records, err := csv.NewReader(bytes.NewReader(stdout.Bytes())).ReadAll()
	if err != nil {
		return SQLiteResult{}, err
	}
	if len(records) == 0 {
		return SQLiteResult{}, nil
	}
	return SQLiteResult{Columns: records[0], Rows: records[1:]}, nil
}
