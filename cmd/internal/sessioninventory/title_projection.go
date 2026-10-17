package sessioninventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

const titlePartialLimit = int64(32 << 20)
const titleBoundaryBytes = int64(4096)

// TitleProjectionResult contains only display facts for one wrapper invocation.
type TitleProjectionResult struct {
	Awaiting bool
	Activity time.Time
	Usage    TokenUsage
	HasUsage bool
}

// TitleProjection admits existing inventory authority once, then follows only
// its pinned append-only transcript. No transcript/event history is retained;
// full validation is never silently invoked after admission.
type TitleProjection struct {
	runtime      Runtime
	ownerRuntime titleOwnerRuntime
	scope, tag   string
	agent        Agent
	ordinal      uint64
	admitted     bool
	ownerSeen    bool
	nativeID     string
	state        ScannerState
	transcript   FileEntry
	other        []FileEntry
	rawOffset    int64
	observedSize int64
	frame        JSONLFrameState
	boundary     []byte
	boundaryHash [32]byte
	result       TitleProjectionResult
	err          error
}
type titleOwnerRuntime struct {
	Runtime
	cache *OwnerLaunchCache
}

func (r titleOwnerRuntime) OwnerLaunchCache() *OwnerLaunchCache { return r.cache }
func (r titleOwnerRuntime) DecodeWork(a Artifact, n int64) {
	observeDecodeWork(r.Runtime, a, n)
}
func NewTitleProjection(rt Runtime, scope, tag string, agent Agent, ordinal uint64) *TitleProjection {
	cache := ownerLaunchCacheOf(rt)
	if cache == nil {
		cache = NewOwnerLaunchCache()
	}
	return &TitleProjection{runtime: rt, ownerRuntime: titleOwnerRuntime{rt, cache}, scope: scope, tag: tag, agent: agent, ordinal: ordinal}
}
func (p *TitleProjection) stop(err error) (TitleProjectionResult, error) {
	p.err = err
	p.result = TitleProjectionResult{}
	p.boundary = nil
	p.frame.IncompleteTail = nil
	return p.result, err
}
func (p *TitleProjection) owner(ctx context.Context) (sessionledger.Current, bool, error) {
	current, ok, _, err := readCurrentLaunch(ctx, p.ownerRuntime, p.scope, p.tag, "")
	if err != nil {
		return current, false, err
	}
	if !ok {
		if p.admitted || p.ownerSeen {
			return current, false, ErrArtifactChanged
		}
		return current, false, nil
	}
	if current.Launch.Ordinal != p.ordinal || Agent(current.Launch.Agent) != p.agent || current.Conflict {
		return current, false, ErrArtifactChanged
	}
	p.ownerSeen = true
	if p.admitted && (current.Binding == nil || current.Binding.RootNativeID != p.nativeID) {
		return current, false, ErrArtifactChanged
	}
	return current, ok, nil
}
func (p *TitleProjection) Poll() (TitleProjectionResult, error) {
	return p.PollContext(context.Background())
}

// PollContext bounds one poll. Cancellation discards uncommitted display state
// and leaves a valid projection reusable; authority failures remain terminal.
func (p *TitleProjection) PollContext(ctx context.Context) (TitleProjectionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p.err != nil {
		return TitleProjectionResult{}, p.err
	}
	if err := ctx.Err(); err != nil {
		return TitleProjectionResult{}, err
	}
	if p.runtime == nil || p.scope == "" || p.tag == "" || !validAgent(p.agent) || p.ordinal == 0 {
		return p.stop(errors.New("title projection requires exact invocation identity"))
	}
	next := *p
	next.state = cloneScannerState(p.state)
	next.other = append([]FileEntry(nil), p.other...)
	next.boundary = append([]byte(nil), p.boundary...)
	// Framing only appends beyond the retained partial prefix, and completed
	// records are copied. Sharing that prefix preserves geometric growth while
	// a canceled transaction leaves the original slice length/content intact.
	next.runtime = titleContextRuntime{Runtime: p.runtime, ctx: ctx}
	next.ownerRuntime = titleOwnerRuntime{Runtime: titleContextRuntime{Runtime: p.runtime, ctx: ctx}, cache: p.ownerRuntime.cache}
	result, err := next.poll(ctx)
	if ctx.Err() != nil {
		return TitleProjectionResult{}, ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return TitleProjectionResult{}, err
	}
	next.runtime = p.runtime
	next.ownerRuntime = p.ownerRuntime
	*p = next
	return result, err
}
func (p *TitleProjection) poll(ctx context.Context) (TitleProjectionResult, error) {
	if p.err != nil {
		return TitleProjectionResult{}, p.err
	}
	if p.runtime == nil || p.scope == "" || p.tag == "" || !validAgent(p.agent) || p.ordinal == 0 {
		return p.stop(errors.New("title projection requires exact invocation identity"))
	}
	current, ok, err := p.owner(ctx)
	if err != nil {
		return p.stop(err)
	}
	if !p.admitted {
		if !ok || current.Binding == nil || (current.Binding.AuthorizationProof == nil && current.Binding.Version < 3) {
			p.result = TitleProjectionResult{Awaiting: true}
			return p.result, nil
		}
		if err := p.admit(ctx, *current.Binding); err != nil {
			return p.stop(err)
		}
		if _, _, err := p.owner(ctx); err != nil {
			return p.stop(err)
		}
		return p.result, nil
	}
	if err := p.tail(ctx); err != nil {
		return p.stop(err)
	}
	if _, _, err := p.owner(ctx); err != nil {
		return p.stop(err)
	}
	return p.result, nil
}
func (p *TitleProjection) admit(ctx context.Context, binding sessionledger.Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Preserve existing proof/path/provider authority checks. A proof-only or
	// suffix-only result needs one unchanged full validation to seed display facts.
	inventory := NewIncrementalInventory(p.runtime, Catalog{Version: CatalogVersion})
	var v TargetValidation
	var err error
	if binding.AuthorizationProof != nil {
		v, _, err = inventory.ValidateBindingProof(p.agent, *binding.AuthorizationProof)
	} else {
		v, _, err = inventory.ValidateNamedContent(p.agent, binding.RootNativeID)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	if !titleValidationRoot(v, p.agent, binding.RootNativeID) {
		return ErrArtifactChanged
	}
	complete := false
	for _, r := range v.Results {
		if r.FrameState.ParserCompleteOffset == r.RawObservedOffset && len(r.FrameState.IncompleteTail) == 0 && (r.RawObservedOffset == 0 || (len(r.Records) > 0 && r.Records[0].Offset == 0)) {
			complete = true
		}
	}
	if !complete {
		full, _ := ValidateTargetWork(p.runtime, p.agent, v.Observations)
		if len(full) != 1 || !titleValidationRoot(full[0], p.agent, binding.RootNativeID) || full[0].State.ScannerSchema != v.State.ScannerSchema {
			return ErrArtifactChanged
		}
		for _, old := range v.Observations {
			found := false
			for _, now := range full[0].Observations {
				if targetArtifactKey(old.Entry.Artifact) == targetArtifactKey(now.Entry.Artifact) && sameTitleIdentity(old.Entry, now.Entry) {
					found = true
				}
			}
			if !found {
				return ErrArtifactChanged
			}
		}
		v = full[0]
	}
	p.nativeID = binding.RootNativeID
	p.state = cloneScannerState(v.State)
	var transcript *IncrementalResult
	for _, observation := range v.Observations {
		entry := observation.Entry
		if entry.Artifact.Kind == ArtifactTranscript {
			if transcript != nil {
				return ErrRootTranscript
			}
			result, ok := v.Results[targetArtifactKey(entry.Artifact)]
			if !ok || result.Disputed || len(result.FrameState.IncompleteTail) != 0 {
				return ErrArtifactChanged
			}
			p.transcript = entry
			transcript = &result
		} else {
			p.other = append(p.other, entry)
		}
	}
	if transcript == nil {
		return ErrRootTranscript
	}
	p.rawOffset = transcript.RawObservedOffset
	p.observedSize = p.rawOffset
	p.frame = JSONLFrameState{ParserCompleteOffset: transcript.FrameState.ParserCompleteOffset}
	for _, record := range transcript.Records {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.usage(record.Bytes)
	}
	current, err := p.checkArtifacts()
	if err != nil || !sameTitleIdentity(p.transcript, current) || current.Size != p.rawOffset || !equalFingerprint(fingerprintFromEntry(p.transcript), fingerprintFromEntry(current)) {
		return ErrArtifactChanged
	}
	window, err := p.readBoundary()
	if err != nil {
		return err
	}
	after, err := p.checkArtifacts()
	if err != nil || !equalFingerprint(fingerprintFromEntry(current), fingerprintFromEntry(after)) {
		return ErrArtifactChanged
	}
	p.boundary = window
	p.boundaryHash = sha256.Sum256(window)
	p.transcript = after
	p.result.Awaiting = false
	if after.ModTime != nil {
		p.result.Activity = *after.ModTime
	}
	p.admitted = true
	// Full v.Results.Records and v.Events die here; only compact state was copied.
	return nil
}
func titleValidationRoot(v TargetValidation, agent Agent, id string) bool {
	return v.State.Agent == agent && v.State.NativeID == id && v.State.Role == RoleRoot && !v.State.Disputed && v.State.FirstRecordValidated && v.Fact.NativeID == id && v.Fact.Role == RoleRoot && !v.Fact.Disputed
}
func sameTitleIdentity(a, b FileEntry) bool {
	return a.StableFileID != "" && a.StableFileID == b.StableFileID && a.GenerationToken == b.GenerationToken && equalOptionalTime(a.BirthTime, b.BirthTime)
}
func (p *TitleProjection) readBoundary() ([]byte, error) {
	n := min(titleBoundaryBytes, p.rawOffset)
	if n == 0 {
		return nil, nil
	}
	b, _, err := p.runtime.ReadAt(p.transcript.Artifact, p.rawOffset-n, n)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) != n {
		return nil, ErrArtifactChanged
	}
	return b, nil
}
func (p *TitleProjection) checkArtifacts() (FileEntry, error) {
	for _, old := range p.other {
		now, err := p.runtime.StatFile(old.Artifact)
		if err != nil || !equalFingerprint(fingerprintFromEntry(old), fingerprintFromEntry(now)) {
			return FileEntry{}, ErrArtifactChanged
		}
	}
	now, err := p.runtime.StatFile(p.transcript.Artifact)
	if err != nil || !sameTitleIdentity(p.transcript, now) || now.Size < p.rawOffset || now.Size < p.observedSize {
		return FileEntry{}, ErrArtifactChanged
	}
	p.observedSize = now.Size
	return now, nil
}
func (p *TitleProjection) tail(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := p.checkArtifacts()
	if err != nil {
		return err
	}
	if current.Size == p.rawOffset {
		if !equalFingerprint(fingerprintFromEntry(p.transcript), fingerprintFromEntry(current)) {
			return ErrArtifactChanged
		}
		return nil
	}
	target := current.Size
	for p.rawOffset < target {
		if err := ctx.Err(); err != nil {
			return err
		}
		before, err := p.checkArtifacts()
		if err != nil {
			return err
		}
		boundary, err := p.readBoundary()
		if err != nil || sha256.Sum256(boundary) != p.boundaryHash {
			return ErrArtifactChanged
		}
		readStart, err := p.checkArtifacts()
		if err != nil {
			return err
		}
		if readStart.Size == before.Size && !equalFingerprint(fingerprintFromEntry(before), fingerprintFromEntry(readStart)) {
			return ErrArtifactChanged
		}
		limit := min(readChunkSize, target-p.rawOffset)
		raw, _, err := p.runtime.ReadAt(p.transcript.Artifact, p.rawOffset, limit)
		if err != nil {
			return err
		}
		if int64(len(raw)) != limit {
			return ErrArtifactChanged
		}
		readEnd, err := p.checkArtifacts()
		if err != nil {
			return err
		}
		if readEnd.Size == readStart.Size && !equalFingerprint(fingerprintFromEntry(readStart), fingerprintFromEntry(readEnd)) {
			return ErrArtifactChanged
		}
		records, next, err := frameTitleSuffix(p.frame, raw)
		if err != nil {
			return err
		}
		if err = p.apply(ctx, current, records); err != nil {
			return err
		}
		p.frame = next
		p.rawOffset += int64(len(raw))
		p.boundary = append(p.boundary, raw...)
		if int64(len(p.boundary)) > titleBoundaryBytes {
			p.boundary = append([]byte(nil), p.boundary[len(p.boundary)-int(titleBoundaryBytes):]...)
		}
		p.boundaryHash = sha256.Sum256(p.boundary)
	}
	after, err := p.checkArtifacts()
	if err != nil {
		return err
	}
	// Growth after this bounded snapshot belongs to the next poll.
	p.transcript = after
	p.transcript.Size = p.rawOffset
	if after.ModTime != nil {
		p.result.Activity = *after.ModTime
	}
	return nil
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
func (p *TitleProjection) usage(line []byte) {
	if u, ok := ParseTokenUsage(p.agent, line); ok {
		p.result.Usage, p.result.HasUsage = u, true
	}
}
func (p *TitleProjection) apply(ctx context.Context, entry FileEntry, records []FramedJSONLRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	observeRecordsDecodeWork(p.runtime, p.transcript.Artifact, records)
	var state ScannerState
	var err error
	switch p.agent {
	case AgentCodex:
		state, _, err = ValidateCodexDelta(entry, &p.state, records)
	case AgentClaude:
		state, _, err = ValidateClaudeDelta(entry, &p.state, records)
	case AgentQoder:
		state, _, err = ValidateQoderDelta(entry, &p.state, records)
	case AgentMuse:
		state, _, err = ValidateMuseDelta(entry, &p.state, records)
	case AgentGrok:
		state, _, err = ValidateGrokDelta(entry, &p.state, records)
	case AgentAgy:
		state = cloneScannerState(p.state)
		var diagnostics []Diagnostic
		applyAgyTranscriptRecords(&state, p.transcript.Artifact, records, &diagnostics)
	default:
		return fmt.Errorf("unsupported title agent %q", p.agent)
	}
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if state.Disputed || state.NativeID != p.nativeID || state.Role != RoleRoot {
		return ErrArtifactChanged
	}
	p.state = state
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.usage(record.Bytes)
	}
	return nil
}
