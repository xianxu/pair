package sessioninventory

import "context"

// TitleLaunch is the exact current typed launch, independent of resume fallback.
type TitleLaunch struct {
	Ordinal uint64
	Agent   Agent
}

// CurrentTitleLaunch reads only Pair-owned launch authority. An unavailable,
// conflicting, untyped, or different-agent latest launch cannot drive a title.
func CurrentTitleLaunch(ctx context.Context, runtime Runtime, scope, tag string, expectedAgent Agent) (TitleLaunch, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return TitleLaunch{}, false, err
	}
	if runtime == nil || scope == "" || tag == "" || !validAgent(expectedAgent) {
		return TitleLaunch{}, false, nil
	}
	current, ok, _, err := readCurrentLaunch(ctx, titleContextRuntime{Runtime: runtime, ctx: ctx}, scope, tag, "")
	if err != nil || !ok || current.Conflict || Agent(current.Launch.Agent) != expectedAgent {
		return TitleLaunch{}, false, err
	}
	return TitleLaunch{Ordinal: current.Launch.Ordinal, Agent: Agent(current.Launch.Agent)}, true, nil
}

// titleContextRuntime carries cancellation through existing scanner IO without
// replacing the underlying runtime's launch cache or work accounting. Reads
// remain synchronous; SQLite implementations can join a cancellable subprocess.
type titleContextRuntime struct {
	Runtime
	ctx context.Context
}

func (r titleContextRuntime) OwnerLaunchCache() *OwnerLaunchCache {
	return ownerLaunchCacheOf(r.Runtime)
}
func (r titleContextRuntime) DecodeWork(a Artifact, n int64) { observeDecodeWork(r.Runtime, a, n) }
func (r titleContextRuntime) ListFiles(root StorageRoot) ([]FileEntry, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	out, err := r.Runtime.ListFiles(root)
	if r.ctx.Err() != nil {
		return nil, r.ctx.Err()
	}
	return out, err
}
func (r titleContextRuntime) StatFile(a Artifact) (FileEntry, error) {
	if err := r.ctx.Err(); err != nil {
		return FileEntry{}, err
	}
	out, err := r.Runtime.StatFile(a)
	if r.ctx.Err() != nil {
		return FileEntry{}, r.ctx.Err()
	}
	return out, err
}
func (r titleContextRuntime) ReadAt(a Artifact, offset, limit int64) ([]byte, bool, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, false, err
	}
	out, eof, err := r.Runtime.ReadAt(a, offset, limit)
	if r.ctx.Err() != nil {
		return nil, false, r.ctx.Err()
	}
	return out, eof, err
}
func (r titleContextRuntime) ReadFile(a Artifact, limit int64) ([]byte, error) {
	if limit < unlimitedRecordSize {
		return nil, ErrReadLimit
	}
	entry, err := r.StatFile(a)
	if err != nil {
		return nil, err
	}
	if limit >= 0 && entry.Size > limit {
		return nil, ErrReadLimit
	}
	var out []byte
	for offset := int64(0); offset < entry.Size; {
		raw, _, err := r.ReadAt(a, offset, min(readChunkSize, entry.Size-offset))
		if err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			return nil, ErrArtifactChanged
		}
		out = append(out, raw...)
		offset += int64(len(raw))
	}
	return out, nil
}
func (r titleContextRuntime) QuerySQLite(a Artifact, query string, limit int64) (SQLiteResult, error) {
	if err := r.ctx.Err(); err != nil {
		return SQLiteResult{}, err
	}
	var out SQLiteResult
	var err error
	if io, ok := r.Runtime.(interface {
		QuerySQLiteContext(context.Context, Artifact, string, int64) (SQLiteResult, error)
	}); ok {
		out, err = io.QuerySQLiteContext(r.ctx, a, query, limit)
	} else {
		out, err = r.Runtime.QuerySQLite(a, query, limit)
	}
	if r.ctx.Err() != nil {
		return SQLiteResult{}, r.ctx.Err()
	}
	return out, err
}
