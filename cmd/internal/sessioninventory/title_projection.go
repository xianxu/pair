package sessioninventory

import (
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
// full validation is never silently invoked while admitted. Growth racing
// admission (#653), or same-identity Agy database growth (#656), drops admission
// and re-admits on the next Poll. Every other failure stops it for good.
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
	TranscriptCursor
	other  []FileEntry
	since  []FileEntry
	result TitleProjectionResult
	err    error
}
type titleOwnerRuntime struct {
	Runtime
	cache *OwnerLaunchCache
}

// errTitleReadmit marks growth that invalidates admission without proving a
// terminal mutation. Poll re-admits through the full validator next time.
var errTitleReadmit = errors.New("title projection must re-admit")

// grewFrom reports the same file identity at a larger size. Database growth
// still requires full revalidation; it does not prove append-only contents.
func grewFrom(old, now FileEntry) bool {
	return sameTitleIdentity(old, now) && now.Size > old.Size
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

// classifyAdmission makes a baseline-only admission change transient
// (#653): an ErrArtifactChanged while every artifact in the baseline is
// unchanged or only grown, and one grew, re-admits on the next Poll. A genuine
// validation failure under growth therefore surfaces as terminal once the
// transcript pauses between polls.
func (p *TitleProjection) classifyAdmission(err error) error {
	if err != nil && errors.Is(err, ErrArtifactChanged) && p.onlyGrew(p.since) {
		return errTitleReadmit
	}
	return err
}

// onlyGrew reports that the observed artifact metadata changed only by growth:
// each is unchanged or the same file grown, and at least one grew.
func (p *TitleProjection) onlyGrew(observed []FileEntry) bool {
	grew := false
	for _, old := range observed {
		now, err := p.runtime.StatFile(old.Artifact)
		switch {
		case err != nil:
			return false
		case equalFingerprint(fingerprintFromEntry(old), fingerprintFromEntry(now)):
		case grewFrom(old, now):
			grew = true
		default:
			return false
		}
	}
	return grew
}

// baseline stats the artifacts admission is about to validate, before its
// first read: the reference against which classifyAdmission tells growth from
// mutation. A proof names its artifacts; a proofless binding's are found the
// way ValidateNamedContent finds them, at the cost of one walk per admission.
func (p *TitleProjection) baseline(inventory IncrementalInventory, binding sessionledger.Record) []FileEntry {
	var entries []FileEntry
	if binding.AuthorizationProof != nil {
		for _, artifact := range binding.AuthorizationProof.Artifacts {
			if entry, err := p.runtime.StatFile(Artifact{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath}); err == nil {
				entries = append(entries, entry)
			}
		}
		return entries
	}
	selected := inventory.Select(TargetRequest{Mode: TargetExplicitResume, Agent: p.agent, NativeID: binding.RootNativeID}, inventory.Observe(p.agent))
	for _, o := range selected.Eligible {
		entries = append(entries, o.Entry)
	}
	return entries
}

// reset drops the admission but keeps the last displayed facts, so a busy
// slot's count stays visible until the next Poll re-admits it.
func (p *TitleProjection) reset(last TitleProjectionResult) (TitleProjectionResult, error) {
	p.admitted, p.nativeID, p.state = false, "", ScannerState{}
	p.transcript, p.other, p.since = FileEntry{}, nil, nil
	p.rawOffset, p.observedSize = 0, 0
	p.frame, p.boundary, p.boundaryHash = JSONLFrameState{}, nil, [32]byte{}
	p.result = last
	return p.result, nil
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
		last := p.result
		if err := p.classifyAdmission(p.admit(ctx, *current.Binding)); errors.Is(err, errTitleReadmit) {
			return p.reset(last)
		} else if err != nil {
			return p.stop(err)
		}
		if _, _, err := p.owner(ctx); err != nil {
			return p.stop(err)
		}
		return p.result, nil
	}
	last := p.result
	err = p.tail(ctx)
	if err != nil && !errors.Is(err, errTitleReadmit) {
		return p.stop(err)
	}
	if _, _, err := p.owner(ctx); err != nil {
		return p.stop(err)
	}
	if errors.Is(err, errTitleReadmit) {
		return p.reset(last)
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
	p.since = p.baseline(inventory, binding)
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
	current, err := p.checkArtifacts(nil)
	if err != nil {
		return err
	}
	if !sameTitleIdentity(p.transcript, current) || current.Size != p.rawOffset || !equalFingerprint(fingerprintFromEntry(p.transcript), fingerprintFromEntry(current)) {
		return ErrArtifactChanged
	}
	window, err := p.readBoundary(p.runtime)
	if err != nil {
		return err
	}
	after, err := p.checkArtifacts(nil)
	if err != nil {
		return err
	}
	if !equalFingerprint(fingerprintFromEntry(current), fingerprintFromEntry(after)) {
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
func (p *TitleProjection) checkArtifacts(before *FileEntry) (FileEntry, error) {
	var observed []FileEntry
	for i, old := range p.other {
		now, err := p.runtime.StatFile(old.Artifact)
		if err != nil {
			return FileEntry{}, ErrArtifactChanged
		}
		if equalFingerprint(fingerprintFromEntry(old), fingerprintFromEntry(now)) {
			continue
		}
		if p.agent != AgentAgy || old.Artifact.Kind != ArtifactDatabase || !grewFrom(old, now) {
			return FileEntry{}, ErrArtifactChanged
		}
		if observed == nil {
			observed = append([]FileEntry(nil), p.other...)
		}
		observed[i] = now
	}
	now, err := p.runtime.StatFile(p.transcript.Artifact)
	if err != nil || !sameTitleIdentity(p.transcript, now) || now.Size < p.rawOffset || now.Size < p.observedSize {
		return FileEntry{}, ErrArtifactChanged
	}
	// Preserve the read's paired fingerprint check before deciding to retry.
	if before != nil && now.Size == before.Size && !equalFingerprint(fingerprintFromEntry(*before), fingerprintFromEntry(now)) {
		return FileEntry{}, ErrArtifactChanged
	}
	p.observedSize = now.Size
	if observed != nil {
		// A growing database must not hide a mutation of the transcript or
		// another artifact. Never commit its new fingerprint as authority.
		if now.Size == p.transcript.Size && !equalFingerprint(fingerprintFromEntry(p.transcript), fingerprintFromEntry(now)) {
			return FileEntry{}, ErrArtifactChanged
		}
		if p.admitted {
			window, err := p.readBoundary(p.runtime)
			if err != nil {
				return FileEntry{}, err
			}
			if sha256.Sum256(window) != p.boundaryHash {
				return FileEntry{}, ErrArtifactChanged
			}
			// Fence the recovery read itself. Compare with this pass's snapshots,
			// including the grown database, rather than its older admission.
			for _, entry := range append(observed, now) {
				after, err := p.runtime.StatFile(entry.Artifact)
				canGrow := entry.Artifact.Kind == ArtifactDatabase || entry.Artifact.Kind == ArtifactTranscript
				if err != nil || (!equalFingerprint(fingerprintFromEntry(entry), fingerprintFromEntry(after)) && !(canGrow && grewFrom(entry, after))) {
					return FileEntry{}, ErrArtifactChanged
				}
			}
		}
		return FileEntry{}, errTitleReadmit
	}
	return now, nil
}
func (p *TitleProjection) tail(ctx context.Context) error {
	_, _, _, err := p.TranscriptCursor.advance(ctx, p.runtime, 0, p.checkArtifacts, p.apply)
	if err == nil && p.transcript.ModTime != nil {
		p.result.Activity = *p.transcript.ModTime
	}
	return err
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
