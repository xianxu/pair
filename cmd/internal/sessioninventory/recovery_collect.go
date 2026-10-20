package sessioninventory

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"time"
)

const (
	RecoveryMetadataLimit  = 10000
	RecoveryCandidateLimit = 64
	RecoveryTotalReadLimit = int64(256 << 20)
	RecoveryScanTimeout    = 30 * time.Second
)

// RecoveryRuntime is an optional bounded/cancellable extension of Runtime.
// Implementations visit at most limit metadata entries (including directories),
// refuse special/symlink artifacts, and honor context between bounded IO calls.
// Runtime's original eager ListFiles/blocking ReadAt cannot satisfy this contract.
type RecoveryRuntime interface {
	VisitRecoveryFiles(context.Context, StorageRoot, int, func(FileEntry) error) (int, error)
	StatRecoveryFile(context.Context, Artifact) (FileEntry, error)
	ReadRecoveryAt(context.Context, FileEntry, int64, int64) ([]byte, bool, error)
	ResolveRecoveryCWD(context.Context, string) (string, error)
}

// RecoverySourceKind distinguishes screen comparison from native observation.
type RecoverySourceKind string

const (
	RecoverySourceCapture RecoverySourceKind = "capture-backed"
	RecoverySourceNative  RecoverySourceKind = "native-only"
)

type RecoveryCollectRequest struct {
	// Empty SourceKind preserves the capture-backed request contract.
	SourceKind        RecoverySourceKind
	ObservationCutoff *time.Time
	// Actual tty/source IO already consumed by the caller, charged to the shared budget.
	BytesAlreadyRead int64
	Agent            Agent
	TargetCWD        string
	ProducingLaunch  *time.Time
	CapturedAt       *time.Time
	RenderedTail     []string
	TTYActivity      *time.Time
}
type RecoveryEvidence struct {
	Candidates      []RecoveryCandidate
	Ranked          []RankedRecoveryCandidate
	Diagnostics     []Diagnostic
	Incomplete      bool
	Reasons         []string
	BytesRead       int64
	MetadataEntries int
	Excluded        int
}

func (e *RecoveryEvidence) incomplete(reason string) {
	e.Incomplete = true
	for _, v := range e.Reasons {
		if v == reason {
			return
		}
	}
	if len(e.Reasons) < 32 {
		e.Reasons = append(e.Reasons, reason)
	}
}

// CollectRecoveryEvidence never resolves a missing epoch from a newer launch,
// publishes metadata, or grants automatic selection authority.
func CollectRecoveryEvidence(ctx context.Context, runtime Runtime, q RecoveryCollectRequest) RecoveryEvidence {
	var out RecoveryEvidence
	if q.BytesAlreadyRead < 0 || q.BytesAlreadyRead > RecoveryTotalReadLimit {
		out.incomplete("invalid consumed tty read budget")
		return out
	}
	out.BytesRead = q.BytesAlreadyRead
	if ctx == nil {
		out.incomplete("recovery scan requires context")
		return out
	}
	ctx, cancel := context.WithTimeout(ctx, RecoveryScanTimeout)
	defer cancel()
	io, ok := runtime.(RecoveryRuntime)
	if !ok {
		out.incomplete("runtime lacks bounded recovery IO")
		return out
	}
	cutoff := q.CapturedAt
	switch q.SourceKind {
	case "", RecoverySourceCapture:
		if q.ObservationCutoff != nil {
			out.incomplete("capture source includes native observation cutoff")
			return out
		}
		if q.ProducingLaunch == nil || q.ProducingLaunch.IsZero() {
			out.incomplete("capture producing launch is unknown")
		}
		if q.CapturedAt == nil || q.CapturedAt.IsZero() {
			out.incomplete("capture timestamp is unknown")
		}
	case RecoverySourceNative:
		if q.ObservationCutoff == nil || q.ObservationCutoff.IsZero() {
			out.incomplete("native observation cutoff is unknown")
			return out
		}
		if q.CapturedAt != nil || len(q.RenderedTail) != 0 || q.TTYActivity != nil {
			out.incomplete("native source includes capture evidence")
			return out
		}
		if q.ProducingLaunch != nil && q.ProducingLaunch.IsZero() {
			out.incomplete("native producing launch hint is zero")
			return out
		}
		cutoff = q.ObservationCutoff
	default:
		out.incomplete("unknown recovery source kind")
		return out
	}
	if q.ProducingLaunch != nil && cutoff != nil && q.ProducingLaunch.After(*cutoff) {
		if q.SourceKind == RecoverySourceNative {
			out.incomplete("native observation predates producing launch hint")
		} else {
			out.incomplete("capture predates producing launch")
		}
		return out
	}
	cwd, err := io.ResolveRecoveryCWD(ctx, q.TargetCWD)
	if err != nil || cwd == "" {
		out.incomplete("physical target cwd unavailable")
		return out
	}
	var pool []FileEntry
	for _, root := range runtime.NativeRoots(q.Agent) {
		if err := ctx.Err(); err != nil {
			out.incomplete(err.Error())
			return out
		}
		remaining := RecoveryMetadataLimit - out.MetadataEntries
		if remaining <= 0 {
			out.incomplete("metadata limit reached")
			break
		}
		count, err := io.VisitRecoveryFiles(ctx, root, remaining, func(e FileEntry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if e.Artifact.StorageRoot != root.Name {
				return ErrPathEscape
			}
			if _, ok := observeEntry(q.Agent, e); ok {
				pool = append(pool, e)
			}
			if len(pool) > RecoveryMetadataLimit {
				return ErrReadLimit
			}
			return nil
		})
		out.MetadataEntries += count
		if count < 0 || count > remaining {
			out.incomplete("invalid metadata count")
			return out
		}
		if err != nil {
			out.incomplete("native metadata listing incomplete")
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return out
			}
		}
	}
	sort.Slice(pool, func(i, j int) bool { return targetArtifactKey(pool[i].Artifact) < targetArtifactKey(pool[j].Artifact) })
	seen := map[string]bool{}
	seenArtifacts := map[string]bool{}
	for _, e := range pool {
		artifactKey := targetArtifactKey(e.Artifact)
		if seenArtifacts[artifactKey] {
			out.incomplete("duplicate native artifact metadata")
			continue
		}
		seenArtifacts[artifactKey] = true
		if err := ctx.Err(); err != nil {
			out.incomplete(err.Error())
			break
		}
		if e.ModTime == nil {
			out.incomplete("native modification time unknown")
		} else if q.ProducingLaunch != nil && e.ModTime.Before(*q.ProducingLaunch) {
			out.Excluded++
			continue
		}
		parser := newRecoveryParser(q.Agent, e)
		parser.cutoff = cutoff
		err := readRecoverySnapshot(ctx, io, e, &out, func(records []FramedJSONLRecord) error {
			for _, r := range records {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := parser.add(r); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			out.incomplete("native transcript incomplete: " + err.Error())
			continue
		}
		var summary []byte
		if q.Agent == AgentGrok {
			a := e.Artifact
			a.RelativePath = path.Join(path.Dir(a.RelativePath), "summary.json")
			se, err := io.StatRecoveryFile(ctx, a)
			if err != nil {
				out.incomplete("Grok summary unavailable")
			} else if se.Size > RecoveryRecordLimit {
				out.incomplete("Grok summary exceeds limit")
			} else {
				err = readRecoveryBytes(ctx, io, se, &out, func(chunk []byte) error { summary = append(summary, chunk...); return nil })
				if err != nil {
					out.incomplete("Grok summary incomplete")
					continue
				}
			}
		}
		c, diags, err := parser.finish(summary)
		out.Diagnostics = append(out.Diagnostics, diags...)
		if err != nil {
			out.incomplete("native parser rejected evidence: " + err.Error())
			continue
		}
		if c.Incomplete {
			out.incomplete("native parser evidence incomplete")
		}
		if !c.RootEligible || c.Interactive.State != InteractiveYes {
			out.Excluded++
			continue
		}
		if cutoff != nil && !c.CreatedAt.IsZero() && c.CreatedAt.After(*cutoff) {
			out.Excluded++
			continue
		}
		physical, err := io.ResolveRecoveryCWD(ctx, c.CWD)
		if err != nil {
			out.incomplete("native physical cwd unavailable")
			continue
		}
		if physical != cwd {
			out.Excluded++
			continue
		}
		c.CWD = physical
		key := string(c.Agent) + ":" + c.NativeID
		if !seen[key] && len(seen) >= RecoveryCandidateLimit {
			out.incomplete("eligible root limit reached")
			break
		}
		// Duplicate artifacts are retained only until ranking coalesces the root.
		// Do not allow duplicates to bypass the bounded message-memory budget.
		if seen[key] {
			out.incomplete("duplicate native root artifacts")
			continue
		}
		seen[key] = true
		out.Candidates = append(out.Candidates, c)
	}
	if err := ctx.Err(); err != nil {
		out.incomplete(err.Error())
	}
	if q.SourceKind == RecoverySourceNative {
		out.Ranked = rankRecoveryCandidates(out.Candidates, nil, cwd, nil, true)
	} else {
		out.Ranked = RankRecoveryCandidates(out.Candidates, q.RenderedTail, cwd, q.TTYActivity)
	}
	return out
}
func sameRecoveryEntry(a, b FileEntry) bool {
	return a.Artifact.StorageRoot == b.Artifact.StorageRoot && a.Artifact.RelativePath == b.Artifact.RelativePath && a.StableFileID == b.StableFileID && a.GenerationToken == b.GenerationToken && a.MutationToken == b.MutationToken && a.Size == b.Size && equalOptionalTime(a.ModTime, b.ModTime)
}
func readRecoveryBytes(ctx context.Context, r RecoveryRuntime, e FileEntry, out *RecoveryEvidence, visit func([]byte) error) error {
	if e.Size < 0 || e.Size > RecoveryTotalReadLimit-out.BytesRead {
		return ErrReadLimit
	}
	before, err := r.StatRecoveryFile(ctx, e.Artifact)
	if err != nil {
		return err
	}
	if !sameRecoveryEntry(e, before) {
		return errors.New("native file changed before read")
	}
	offset := int64(0)
	for offset < e.Size {
		if err := ctx.Err(); err != nil {
			return err
		}
		n := readChunkSize
		if e.Size-offset < n {
			n = e.Size - offset
		}
		chunk, eof, err := r.ReadRecoveryAt(ctx, e, offset, n)
		if int64(len(chunk)) > n {
			return errors.New("runtime exceeded chunk bound")
		}
		out.BytesRead += int64(len(chunk))
		if err != nil {
			return err
		}
		if len(chunk) == 0 {
			return errors.New("native file truncated")
		}
		if err = visit(chunk); err != nil {
			return err
		}
		offset += int64(len(chunk))
		if eof && offset < e.Size {
			return errors.New("native file truncated")
		}
	}
	after, err := r.StatRecoveryFile(ctx, e.Artifact)
	if err != nil {
		return err
	}
	if !sameRecoveryEntry(before, after) {
		return errors.New("native file changed during read")
	}
	return ctx.Err()
}
func readRecoverySnapshot(ctx context.Context, r RecoveryRuntime, e FileEntry, out *RecoveryEvidence, visit func([]FramedJSONLRecord) error) error {
	var frame JSONLFrameState
	err := readRecoveryBytes(ctx, r, e, out, func(chunk []byte) error {
		records, next, err := FrameJSONLSuffix(frame, chunk, RecoveryRecordLimit)
		if err != nil {
			return err
		}
		frame = next
		return visit(records)
	})
	if err != nil {
		return err
	}
	if len(frame.IncompleteTail) > 0 {
		return fmt.Errorf("%w", errTruncatedRecord)
	}
	return nil
}

// RevalidateRecoveryCandidate checks only the explicitly named transcript. It
// cannot substitute a newly ranked root for an expired or changed selection.
func RevalidateRecoveryCandidate(ctx context.Context, runtime Runtime, expected FileEntry, agent Agent, nativeID, targetCWD string, capturedAt time.Time) error {
	if ctx == nil || capturedAt.IsZero() || nativeID == "" {
		return errors.New("exact recovery revalidation requires context, native ID and capture time")
	}
	ctx, cancel := context.WithTimeout(ctx, RecoveryScanTimeout)
	defer cancel()
	io, ok := runtime.(RecoveryRuntime)
	if !ok {
		return errors.New("runtime lacks bounded recovery IO")
	}
	if expected.StableFileID == "" || expected.MutationToken == "" || expected.ModTime == nil {
		return errors.New("incomplete expected native fingerprint")
	}
	if _, ok := observeEntry(agent, expected); !ok {
		return errors.New("unrecognized expected native artifact")
	}
	parser := newRecoveryParser(agent, expected)
	parser.cutoff = &capturedAt
	var budget RecoveryEvidence
	if err := readRecoverySnapshot(ctx, io, expected, &budget, func(records []FramedJSONLRecord) error {
		for _, r := range records {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := parser.add(r); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	c, _, err := parser.finish(nil)
	if err != nil {
		return err
	}
	if c.NativeID != nativeID || !c.RootEligible || c.Interactive.State != InteractiveYes || c.CreatedAt.After(capturedAt) {
		return errors.New("native recovery identity or interactive provenance changed")
	}
	target, err := io.ResolveRecoveryCWD(ctx, targetCWD)
	if err != nil {
		return err
	}
	actual, err := io.ResolveRecoveryCWD(ctx, c.CWD)
	if err != nil {
		return err
	}
	if target == "" || actual != target {
		return errors.New("native recovery cwd changed")
	}
	return ctx.Err()
}
