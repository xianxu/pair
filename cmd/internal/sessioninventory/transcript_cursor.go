package sessioninventory

import (
	"bytes"
	"context"
	"crypto/sha256"
)

// TranscriptCursor retains bounded framing and file evidence for one already
// validated transcript. It carries no scanner state, native events or history.
// Its authority comes only from the supplied validation and guarded runtime IO.
type TranscriptCursor struct {
	transcript   FileEntry
	rawOffset    int64
	observedSize int64
	frame        JSONLFrameState
	boundary     []byte
	boundaryHash [32]byte
}

// TranscriptBatch contains only the records completed during one frozen pass.
// Backlog includes growth observed after that pass's frozen read endpoint.
type TranscriptBatch struct {
	Entry     FileEntry
	Records   []FramedJSONLRecord
	BytesRead int64
	Backlog   bool
}

// NewTranscriptCursor fences one existing root validation without discovery.
// It copies only compact framing; validated records and events are not retained.
func NewTranscriptCursor(rt Runtime, validated TargetValidation) (TranscriptCursor, error) {
	if rt == nil || !titleValidationRoot(validated, validated.State.Agent, validated.State.NativeID) || ValidateScannerState(validated.State) != nil || len(validated.Observations) != 1 {
		return TranscriptCursor{}, ErrArtifactChanged
	}
	entry := validated.Observations[0].Entry
	result, ok := validated.Results[targetArtifactKey(entry.Artifact)]
	if !ok || result.Disputed || entry.Artifact.Kind != ArtifactTranscript || result.RawObservedOffset < 0 || result.RawObservedOffset != entry.Size || result.FrameState.ParserCompleteOffset < 0 || result.FrameState.ParserCompleteOffset+int64(len(result.FrameState.IncompleteTail)) != result.RawObservedOffset || int64(len(result.FrameState.IncompleteTail)) > titlePartialLimit || !equalFingerprint(result.Fingerprint, fingerprintFromEntry(entry)) {
		return TranscriptCursor{}, ErrArtifactChanged
	}
	c := TranscriptCursor{transcript: entry, rawOffset: result.RawObservedOffset, observedSize: entry.Size, frame: JSONLFrameState{ParserCompleteOffset: result.FrameState.ParserCompleteOffset, IncompleteTail: append([]byte(nil), result.FrameState.IncompleteTail...)}}
	before, err := c.checkTranscript(rt, nil)
	if err != nil || !equalFingerprint(fingerprintFromEntry(entry), fingerprintFromEntry(before)) {
		return TranscriptCursor{}, ErrArtifactChanged
	}
	boundary, err := c.readBoundary(rt)
	if err != nil {
		return TranscriptCursor{}, err
	}
	after, err := c.checkTranscript(rt, &before)
	if err != nil || !equalFingerprint(fingerprintFromEntry(before), fingerprintFromEntry(after)) {
		return TranscriptCursor{}, ErrArtifactChanged
	}
	c.boundary = boundary
	c.boundaryHash = sha256.Sum256(boundary)
	return c, nil
}

// Next reads at most maxBytes payload bytes from a frozen entry snapshot; zero
// means the full snapshot. A failed or canceled pass returns the original cursor
// and no batch. The caller commits the candidate only after its own scanner and
// owner checks. Calls on one cursor lineage are serial: discard a candidate
// before retrying its parent so geometric partial buffers remain reusable.
func (c TranscriptCursor) Next(ctx context.Context, rt Runtime, maxBytes int64) (TranscriptCursor, TranscriptBatch, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return c, TranscriptBatch{}, err
	}
	if rt == nil || c.transcript.StableFileID == "" {
		return c, TranscriptBatch{}, ErrArtifactChanged
	}
	if maxBytes < 0 {
		return c, TranscriptBatch{}, ErrReadLimit
	}
	next := c
	next.boundary = append([]byte(nil), c.boundary...)
	guarded := titleContextRuntime{Runtime: rt, ctx: ctx}
	var records []FramedJSONLRecord
	entry, n, backlog, err := next.advance(ctx, guarded, maxBytes, func(before *FileEntry) (FileEntry, error) { return next.checkTranscript(guarded, before) }, func(_ context.Context, _ FileEntry, newRecords []FramedJSONLRecord) error {
		records = append(records, newRecords...)
		return nil
	})
	if ctx.Err() != nil {
		return c, TranscriptBatch{}, ctx.Err()
	}
	if err != nil {
		return c, TranscriptBatch{}, err
	}
	return next, TranscriptBatch{Entry: entry, Records: records, BytesRead: n, Backlog: backlog}, nil
}

func (p *TranscriptCursor) checkTranscript(rt Runtime, before *FileEntry) (FileEntry, error) {
	now, err := rt.StatFile(p.transcript.Artifact)
	if err != nil || !sameTitleIdentity(p.transcript, now) || now.Size < p.rawOffset || now.Size < p.observedSize {
		return FileEntry{}, ErrArtifactChanged
	}
	if before != nil && now.Size == before.Size && !equalFingerprint(fingerprintFromEntry(*before), fingerprintFromEntry(now)) {
		return FileEntry{}, ErrArtifactChanged
	}
	p.observedSize = now.Size
	return now, nil
}

func (p *TranscriptCursor) readBoundary(rt Runtime) ([]byte, error) {
	n := min(titleBoundaryBytes, p.rawOffset)
	if n == 0 {
		return nil, nil
	}
	b, _, err := rt.ReadAt(p.transcript.Artifact, p.rawOffset-n, n)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) != n {
		return nil, ErrArtifactChanged
	}
	return b, nil
}

func (p *TranscriptCursor) advance(ctx context.Context, rt Runtime, maxBytes int64, check func(*FileEntry) (FileEntry, error), apply func(context.Context, FileEntry, []FramedJSONLRecord) error) (FileEntry, int64, bool, error) {
	if err := ctx.Err(); err != nil {
		return FileEntry{}, 0, false, err
	}
	current, err := check(nil)
	if err != nil {
		return FileEntry{}, 0, false, err
	}
	if current.Size == p.rawOffset {
		if !equalFingerprint(fingerprintFromEntry(p.transcript), fingerprintFromEntry(current)) {
			return FileEntry{}, 0, false, ErrArtifactChanged
		}
		return current, 0, false, nil
	}
	start := p.rawOffset
	target := current.Size
	if maxBytes > 0 && maxBytes < target-p.rawOffset {
		target = p.rawOffset + maxBytes
	}
	for p.rawOffset < target {
		if err := ctx.Err(); err != nil {
			return FileEntry{}, 0, false, err
		}
		before, err := check(nil)
		if err != nil {
			return FileEntry{}, 0, false, err
		}
		boundary, err := p.readBoundary(rt)
		if err != nil || sha256.Sum256(boundary) != p.boundaryHash {
			return FileEntry{}, 0, false, ErrArtifactChanged
		}
		readStart, err := check(&before)
		if err != nil {
			return FileEntry{}, 0, false, err
		}
		limit := min(readChunkSize, target-p.rawOffset)
		raw, _, err := rt.ReadAt(p.transcript.Artifact, p.rawOffset, limit)
		if err != nil {
			return FileEntry{}, 0, false, err
		}
		if int64(len(raw)) != limit {
			return FileEntry{}, 0, false, ErrArtifactChanged
		}
		_, err = check(&readStart)
		if err != nil {
			return FileEntry{}, 0, false, err
		}
		records, next, err := frameTitleSuffix(p.frame, raw)
		if err != nil {
			return FileEntry{}, 0, false, err
		}
		if err = apply(ctx, current, records); err != nil {
			return FileEntry{}, 0, false, err
		}
		p.frame = next
		p.rawOffset += int64(len(raw))
		p.boundary = append(p.boundary, raw...)
		if int64(len(p.boundary)) > titleBoundaryBytes {
			p.boundary = append([]byte(nil), p.boundary[len(p.boundary)-int(titleBoundaryBytes):]...)
		}
		p.boundaryHash = sha256.Sum256(p.boundary)
	}
	after, err := check(nil)
	if err != nil {
		return FileEntry{}, 0, false, err
	}
	// Growth after this bounded snapshot belongs to the next poll.
	p.transcript = after
	p.transcript.Size = p.rawOffset
	return current, p.rawOffset - start, after.Size > p.rawOffset, nil
}

// Buffer an incomplete record with bounded geometric growth. Reusing the full
// framer only when a newline arrives avoids recopying a growing partial record
// on every 64KiB range, while preserving its framing/CRLF rules.
func frameTitleSuffix(frame JSONLFrameState, raw []byte) ([]FramedJSONLRecord, JSONLFrameState, error) {
	if len(frame.IncompleteTail) == 0 {
		return FrameJSONLSuffix(frame, raw, titlePartialLimit)
	}
	newline := bytes.IndexByte(raw, '\n')
	part := raw
	if newline >= 0 {
		part = raw[:newline+1]
	}
	needed := len(frame.IncompleteTail) + len(part)
	// A terminating newline is not part of the record's size ceiling.
	allowance := int(titlePartialLimit)
	if newline >= 0 {
		allowance++
	}
	if needed > allowance {
		return nil, frame, ErrReadLimit
	}
	if cap(frame.IncompleteTail) < needed {
		next := min(allowance, max(needed, 2*cap(frame.IncompleteTail)))
		grown := make([]byte, len(frame.IncompleteTail), next)
		copy(grown, frame.IncompleteTail)
		frame.IncompleteTail = grown
	}
	frame.IncompleteTail = append(frame.IncompleteTail, part...)
	if newline < 0 {
		return nil, frame, nil
	}
	records, next, err := FrameJSONLSuffix(JSONLFrameState{ParserCompleteOffset: frame.ParserCompleteOffset}, frame.IncompleteTail, titlePartialLimit)
	if err != nil {
		return nil, frame, err
	}
	more, last, err := FrameJSONLSuffix(next, raw[newline+1:], titlePartialLimit)
	return append(records, more...), last, err
}
