package sessioninventory

import (
	"context"
	"errors"
	"fmt"
)

type nativeSelectorRuntime struct {
	Runtime
	io            RecoveryRuntime
	ctx           context.Context
	bytes         int64
	entries       int
	failure       error
	exactClaudeID string
	snapshots     map[string]FileEntry
}

func newNativeSelectorRuntime(ctx context.Context, rt Runtime) (*nativeSelectorRuntime, error) {
	io, ok := rt.(RecoveryRuntime)
	if !ok {
		return nil, errors.New("bounded native selector IO unavailable")
	}
	return &nativeSelectorRuntime{Runtime: rt, io: io, ctx: ctx, snapshots: map[string]FileEntry{}}, nil
}
func (r *nativeSelectorRuntime) fail(err error) error {
	if err != nil {
		r.failure = err
	}
	return err
}
func (r *nativeSelectorRuntime) ListFiles(root StorageRoot) ([]FileEntry, error) {
	var files []FileEntry
	remaining := RecoveryMetadataLimit - r.entries
	if remaining <= 0 {
		return nil, r.fail(ErrReadLimit)
	}
	visit := func(e FileEntry) error {
		files = append(files, e)
		r.snapshots[targetArtifactKey(e.Artifact)] = e
		return r.ctx.Err()
	}
	var count int
	var err error
	if exact, ok := r.Runtime.(nativeSelectorFileVisitor); ok && root.Agent == AgentClaude && r.exactClaudeID != "" {
		count, err = exact.VisitNativeSelectorFiles(r.ctx, root, r.exactClaudeID, remaining, visit)
	} else {
		count, err = r.io.VisitRecoveryFiles(r.ctx, root, remaining, visit)
	}
	r.entries += count
	if count < 0 || count > remaining {
		return nil, r.fail(ErrReadLimit)
	}
	return files, r.fail(err)
}
func (r *nativeSelectorRuntime) StatFile(a Artifact) (FileEntry, error) {
	e, err := r.io.StatRecoveryFile(r.ctx, a)
	if err == nil {
		r.snapshots[targetArtifactKey(a)] = e
	}
	return e, r.fail(err)
}
func (r *nativeSelectorRuntime) ReadAt(a Artifact, offset, limit int64) ([]byte, bool, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, false, r.fail(err)
	}
	if limit <= 0 || limit > RecoveryTotalReadLimit-r.bytes {
		return nil, false, r.fail(ErrReadLimit)
	}
	e, ok := r.snapshots[targetArtifactKey(a)]
	if !ok {
		var err error
		e, err = r.StatFile(a)
		if err != nil {
			return nil, false, err
		}
	}
	if limit > readChunkSize {
		limit = readChunkSize
	}
	raw, eof, err := r.io.ReadRecoveryAt(r.ctx, e, offset, limit)
	r.bytes += int64(len(raw))
	return raw, eof, r.fail(err)
}
func (r *nativeSelectorRuntime) ReadFile(a Artifact, limit int64) ([]byte, error) {
	e, err := r.StatFile(a)
	if err != nil {
		return nil, err
	}
	if limit >= 0 && e.Size > limit {
		return nil, r.fail(ErrReadLimit)
	}
	var out []byte
	for offset := int64(0); offset < e.Size; {
		raw, _, err := r.ReadAt(a, offset, readChunkSize)
		if err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			return nil, r.fail(ErrArtifactChanged)
		}
		out = append(out, raw...)
		offset += int64(len(raw))
	}
	return out, nil
}
func (r *nativeSelectorRuntime) QuerySQLite(a Artifact, query string, limit int64) (SQLiteResult, error) {
	if err := r.ctx.Err(); err != nil {
		return SQLiteResult{}, r.fail(err)
	}
	if limit < 0 || limit > 1<<20 {
		limit = 1 << 20
	}
	var out SQLiteResult
	var err error
	if io, ok := r.Runtime.(interface {
		QueryNativeSelectorSQLite(context.Context, Artifact, string, int64) (SQLiteResult, error)
	}); ok {
		out, err = io.QueryNativeSelectorSQLite(r.ctx, a, query, limit)
	} else {
		out, err = r.Runtime.QuerySQLite(a, query, limit)
	}
	if err == nil {
		err = r.ctx.Err()
	}
	if err != nil {
		return out, r.fail(fmt.Errorf("native selector SQLite: %w", err))
	}
	return out, nil
}

// Optional exact-ID discovery; runtimes without it retain bounded full discovery.
type nativeSelectorFileVisitor interface {
	VisitNativeSelectorFiles(context.Context, StorageRoot, string, int, func(FileEntry) error) (int, error)
}
