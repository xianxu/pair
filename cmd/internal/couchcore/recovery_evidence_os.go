package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/scrollbackcmd"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

// OSRecoveryEvidenceReader uses existing native roots, VT replay and retention
// leases. It neither publishes a candidate cache nor edits agent defaults.
type OSRecoveryEvidenceReader struct {
	DataDir, HomeDir string
	Git              GitRunner
	parkScrollback   func(context.Context, string, string, string) (string, bool)
	prepareTerminal  func(context.Context, string, string, string, string) (launcher.RecoveryTerminalPlan, error)
	retireTerminal   func(context.Context, launcher.RecoveryTerminalPlan) error
}

const recoveryCaptureMetadataLimit = 8192

// VerifySelection reopens only the selected native artifact. It is an exact
// pre-effect check, independent of the earlier inventory and screen observation.
func (r OSRecoveryEvidenceReader) VerifySelection(ctx context.Context, record ThreadRecord, snapshot threadrecord.RecoveryContext, key, nativeID string) error {
	if ctx == nil {
		return errors.New("native recovery verification requires context")
	}
	ctx, cancel := context.WithTimeout(ctx, sessioninventory.RecoveryScanTimeout)
	defer cancel()
	if err := threadrecord.ValidateRecoveryContext(snapshot, toPersistedThreadAddress(record.Address)); err != nil {
		return err
	}
	if snapshot.SourceKind() != sessioninventory.RecoverySourceNative || snapshot.Source.Revision > record.Revision || nativeID == "" {
		return errors.New("native recovery source changed")
	}
	for _, candidate := range snapshot.Candidates {
		if candidate.Key != key {
			continue
		}
		if !candidate.Eligible || candidate.NativeID != nativeID || candidate.Agent != RecordAgent(record) {
			return errors.New("native recovery selection changed")
		}
		runtime, err := sessioninventory.NewNativeTranscriptRuntime(r.HomeDir, r.DataDir, sessioninventory.Agent(candidate.Agent), "", os.Getenv)
		if err != nil {
			return err
		}
		physical, err := runtime.ResolveRecoveryCWD(ctx, record.WorkingPath)
		if err != nil {
			return err
		}
		if physical != candidate.CWD {
			return errors.New("native recovery physical cwd changed")
		}
		return sessioninventory.RevalidateRecoveryCandidate(ctx, runtime, candidate.InventoryEntry(), sessioninventory.Agent(candidate.Agent), nativeID, physical, snapshot.Source.Cutoff)
	}
	return errors.New("native recovery selection is absent from snapshot")
}

func recoveryCaptureSource(record ThreadRecord, paths artifactpath.Paths, agent string) (threadrecord.RecoveryCapture, error) {
	if intent := retainedRecoveryIntent(record); intent != nil {
		if intent.Kind() == threadrecord.RecoveryInferred && intent.Context.Source == nil {
			return intent.Context.Capture, nil
		}
		// Tagged orientation may name the outgoing tag after Fresh selected a new
		// address. That screen remains readable by peek, never inference authority
		// for this tag. Missing orientation similarly selects this tag's own screen.
		if o := intent.Orientation; o != nil && o.Capture != nil && o.Capture.Token != "" {
			expected, err := paths.ParkedScrollbackArtifacts(o.Capture.Token)
			if err == nil && expected.Raw == o.Capture.Raw && expected.Events == o.Capture.Events {
				return *o.Capture, nil
			}
		}
		current, err := paths.ScrollbackArtifacts(agent)
		return threadrecord.RecoveryCapture{Raw: current.Raw, Events: current.Events}, err
	}
	if record.RecoveryContext != nil && record.RecoveryContext.Source == nil {
		return record.RecoveryContext.Capture, nil
	}
	if record.VerifiedPark != nil && record.PreviousSession != nil && record.PreviousSession.Capture != nil {
		p, err := paths.ParkedScrollbackArtifacts(record.PreviousSession.Capture.Token)
		return threadrecord.RecoveryCapture{Raw: p.Raw, Events: p.Events, Token: record.PreviousSession.Capture.Token}, err
	}
	p, err := paths.ScrollbackArtifacts(agent)
	return threadrecord.RecoveryCapture{Raw: p.Raw, Events: p.Events}, err
}

func validateRecoveryCaptureSnapshot(ctx context.Context, c threadrecord.RecoveryCapture) error {
	if ctx == nil {
		return errors.New("recovery capture requires context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, member := range []struct {
		path     string
		identity artifactpath.CaptureFileIdentity
	}{{c.Raw, c.RawIdentity}, {c.Events, c.EventsIdentity}} {
		got, err := storagegc.CaptureIdentity(member.path)
		if err != nil {
			return err
		}
		if got != member.identity {
			return errors.New("recovery screen changed; refresh resume candidates")
		}
	}
	return ctx.Err()
}

func recoveryLease(ctx context.Context, paths artifactpath.Paths, capture threadrecord.RecoveryCapture, agent string) (*storagegc.ProcessLease, error) {
	return storagegc.AcquireSelectedProcessTarget(ctx, func(key string) string {
		switch key {
		case "PAIR_DATA_DIR":
			return paths.ScopeDir()
		case "PAIR_TAG":
			return paths.Tag()
		case "PAIR_AGENT":
			return agent
		}
		return ""
	}, "recovery-reader", capture.Raw)
}

// Read is the direct observation convenience entrypoint. Interactive callers
// supply their injected clock and any selected source through ReadRequest.
func (r OSRecoveryEvidenceReader) Read(ctx context.Context, record ThreadRecord) (RecoveryObservation, error) {
	return r.ReadRequest(ctx, record, RecoveryObservationRequest{ObservedAt: time.Now().UTC()})
}

func (r OSRecoveryEvidenceReader) ReadRequest(ctx context.Context, record ThreadRecord, q RecoveryObservationRequest) (RecoveryObservation, error) {
	if ctx == nil {
		return RecoveryObservation{}, errors.New("recovery read requires context")
	}
	ctx, cancel := context.WithTimeout(ctx, sessioninventory.RecoveryScanTimeout)
	defer cancel()
	if q.ObservedAt.IsZero() || q.BytesAlreadyRead < 0 || q.BytesAlreadyRead > sessioninventory.RecoveryTotalReadLimit {
		return RecoveryObservation{}, errors.New("invalid recovery observation request")
	}
	if record.LatestLaunchProfile == nil {
		return RecoveryObservation{}, errors.New("recovery has no saved agent configuration")
	}
	var retainedSource *threadrecord.NativeRecoverySource
	if intent := retainedRecoveryIntent(record); intent != nil && intent.Kind() == threadrecord.RecoveryInferred {
		retainedSource = intent.Context.Source
	} else if record.RecoveryContext != nil {
		retainedSource = record.RecoveryContext.Source
	}
	if q.Source != nil {
		if err := threadrecord.ValidateNativeRecoverySourceAt(q.Source, q.ObservedAt, q.Source.Revision); err != nil {
			return RecoveryObservation{}, err
		}
	}
	out, screenErr := r.readScreen(ctx, record, q.BytesAlreadyRead)
	if out.BytesRead < q.BytesAlreadyRead {
		out.BytesRead = q.BytesAlreadyRead
	}
	out.Agent = record.LatestLaunchProfile.Agent
	out.Branch, out.Issue = r.recoveryBranchIssue(ctx, record)
	if screenErr != nil {
		out.Screen.Availability = ScreenUnavailable
		out.Screen.Reason = screenErr.Error()
		out.Reasons = append(out.Reasons, "Previous screen unavailable: "+screenErr.Error())
	}
	if retainedSource != nil || out.producingLaunch == nil || out.Capture.LastActivity.IsZero() || q.Source != nil || screenErr != nil || !q.ObservedAt.Before(out.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)) {
		if q.Source != nil {
			copy := *q.Source
			out.Source = &copy
		} else {
			hint := out.producingLaunch
			if retainedSource != nil {
				hint = retainedSource.LaunchHint
			}
			// A new explicit listing gets a new observation; only a selected key
			// or pending launch supplies a fixed source. Failed attempts must not
			// pin refresh to a retired revision or borrow the replacement's clock.
			out.Source = &threadrecord.NativeRecoverySource{Kind: sessioninventory.RecoverySourceNative, Cutoff: q.ObservedAt.UTC(), Revision: record.Revision, LaunchHint: hint}
		}
		if screenErr == nil && !q.ObservedAt.Before(out.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)) {
			out.Screen = ScreenObservation{Availability: ScreenUnavailable, Reason: "previous screen expired", LastActivity: out.Screen.LastActivity, ActivitySource: out.Screen.ActivitySource}
			out.Reasons = append(out.Reasons, "Previous screen expired")
		}
	}
	if q.SkipCandidates {
		if out.Source == nil {
			if err := validateRecoveryCaptureSnapshot(ctx, out.Capture); err != nil {
				out.Incomplete = true
				out.Reasons = append(out.Reasons, err.Error())
			}
		}
		return out, nil
	}
	runtime, err := sessioninventory.NewNativeTranscriptRuntime(r.HomeDir, r.DataDir, sessioninventory.Agent(out.Agent), "", os.Getenv)
	if err != nil {
		out.Incomplete = true
		out.Reasons = append(out.Reasons, "native transcript home unavailable; candidate scan incomplete")
		return out, nil
	}
	request := sessioninventory.RecoveryCollectRequest{Agent: sessioninventory.Agent(out.Agent), TargetCWD: record.WorkingPath, ProducingLaunch: out.producingLaunch, CapturedAt: &out.Capture.LastActivity, RenderedTail: out.renderedTail, TTYActivity: &out.Capture.LastActivity, BytesAlreadyRead: out.BytesRead}
	if out.Source != nil {
		request.SourceKind = sessioninventory.RecoverySourceNative
		request.ObservationCutoff = &out.Source.Cutoff
		request.ProducingLaunch = out.Source.LaunchHint
		request.CapturedAt, request.RenderedTail, request.TTYActivity = nil, nil, nil
	}
	evidence := sessioninventory.CollectRecoveryEvidence(ctx, runtime, request)
	out.Ranked, out.Incomplete, out.BytesRead = evidence.Ranked, evidence.Incomplete, evidence.BytesRead
	out.Reasons = append(out.Reasons, evidence.Reasons...)
	if out.Source == nil {
		if err := validateRecoveryCaptureSnapshot(ctx, out.Capture); err != nil {
			out.Ranked = nil
			out.Incomplete = true
			out.Reasons = append(out.Reasons, err.Error())
		}
	}
	return out, nil
}

func (r OSRecoveryEvidenceReader) readScreen(ctx context.Context, record ThreadRecord, consumed int64) (RecoveryObservation, error) {
	out := RecoveryObservation{BytesRead: consumed}
	if ctx == nil {
		return out, errors.New("recovery read requires context")
	}
	ctx, cancel := context.WithTimeout(ctx, sessioninventory.RecoveryScanTimeout)
	defer cancel()
	if record.LatestLaunchProfile == nil {
		return out, errors.New("recovery has no saved agent configuration")
	}
	out.Agent = record.LatestLaunchProfile.Agent
	paths, err := r.recoveryPaths(record)
	if err != nil {
		return out, err
	}
	capture, err := recoveryCaptureSource(record, paths, out.Agent)
	if err != nil {
		return out, err
	}
	out.Screen = recoveryRawFileActivity(ctx, paths, capture, out.Agent)
	lease, err := recoveryLease(ctx, paths, capture, out.Agent)
	if err != nil {
		return out, err
	}
	if lease == nil {
		return out, errors.New("recovery capture lease unavailable")
	}
	defer lease.Close()
	original := capture
	archivedMetadata, err := recoveryCaptureMetadata(ctx, paths, &capture, out.Agent, &out.BytesRead)
	if err != nil {
		return out, err
	}
	capture.RawIdentity, err = storagegc.CaptureIdentity(capture.Raw)
	if err != nil {
		return out, err
	}
	capture.EventsIdentity, err = storagegc.CaptureIdentity(capture.Events)
	if err != nil {
		return out, err
	}
	if original.RawIdentity.Inode != 0 && (original.RawIdentity != capture.RawIdentity || original.EventsIdentity != capture.EventsIdentity) {
		return out, errors.New("retained recovery screen changed; refresh evidence")
	}
	if capture.RawIdentity.Size == 0 {
		return out, errors.New("previous screen is empty")
	}
	if capture.LastActivity.IsZero() && archivedMetadata == nil {
		capture.LastActivity = capture.RawIdentity.ModifiedAt
	}
	if capture.CapturedAt.IsZero() {
		capture.CapturedAt = capture.RawIdentity.ModifiedAt
	}
	out.Capture = capture
	out.Screen.Availability = ScreenAvailable
	if capture.RawIdentity.Size > scrollbackcmd.MaxReplayRawBytes || capture.EventsIdentity.Size > scrollbackcmd.MaxReplayEventsBytes {
		return out, errors.New("previous screen exceeds bounded replay size limit")
	}
	remaining := sessioninventory.RecoveryTotalReadLimit - out.BytesRead
	if consumed < 0 || consumed > sessioninventory.RecoveryTotalReadLimit || capture.RawIdentity.Size < 0 || capture.EventsIdentity.Size < 0 || capture.EventsIdentity.Size > remaining {
		return out, errors.New("previous screen exceeds recovery read budget")
	}
	// Charge each attempted read at its own boundary. A rejected witness never
	// spends raw replay bytes that cannot be read.
	out.BytesRead += capture.EventsIdentity.Size
	events, err := osfs.ReadRegularMetadata(ctx, capture.Events, artifactpath.MaxCaptureWitnessEventsBytes)
	if err != nil {
		return out, err
	}
	witness, witnessErr := artifactpath.DecodeCaptureWitness(events)
	if witnessErr != nil && !errors.Is(witnessErr, artifactpath.ErrCaptureWitnessAbsent) {
		return out, witnessErr
	}
	if witnessErr == nil {
		if archivedMetadata != nil {
			if archivedMetadata.Witness == nil || *archivedMetadata.Witness != witness {
				return out, errors.New("retained recovery witness does not match archive metadata")
			}
		} else if witness.RawFile != (artifactpath.CaptureFileID{Device: capture.RawIdentity.Device, Inode: capture.RawIdentity.Inode}) || witness.EventsFile != (artifactpath.CaptureFileID{Device: capture.EventsIdentity.Device, Inode: capture.EventsIdentity.Inode}) {
			return out, errors.New("current recovery witness names another file generation")
		}
		if witness.ScopeKey != record.Address.RepoScope || witness.Tag != string(record.Address.Tag) || witness.Agent != out.Agent {
			return out, errors.New("recovery capture belongs to another producer")
		}
		capture.LaunchOrdinal, capture.Nonce = witness.LaunchOrdinal, witness.Nonce
		out.Generation = witness.Generation
	}
	if witnessErr != nil && (capture.LaunchOrdinal != 0 || capture.Nonce != "" || (archivedMetadata != nil && archivedMetadata.Witness != nil)) {
		return out, errors.New("recovery producer authority disappeared")
	}
	if out.Generation == "" {
		out.Generation = recoveryLegacyGeneration(capture)
	}
	out.Capture = capture
	remaining = sessioninventory.RecoveryTotalReadLimit - out.BytesRead
	if capture.RawIdentity.Size > remaining || capture.EventsIdentity.Size > remaining-capture.RawIdentity.Size {
		return out, errors.New("previous screen exceeds recovery read budget")
	}
	out.BytesRead += capture.RawIdentity.Size + capture.EventsIdentity.Size
	lines, err := scrollbackcmd.RenderBoundedLinesContext(ctx, capture.Raw, capture.Events, 100)
	if err != nil {
		return out, err
	}
	if err := validateRecoveryCaptureSnapshot(ctx, capture); err != nil {
		return out, err
	}
	out.Branch, out.Issue = r.recoveryBranchIssue(ctx, record)
	// Legacy capture provenance cannot borrow the latest empty/recovery launch.
	// Its screen can still orient an explicitly chosen fresh conversation.
	if capture.LastActivity.IsZero() {
		out.Reasons = []string{"original capture activity time is unavailable; archive publication time is only an upper bound, capture-backed inference unavailable"}
		return out, nil
	}
	if witnessErr != nil || witness.OpenedAt.IsZero() {
		out.Reasons = []string{"capture producing time is unknown; native choices have no launch association"}
		return out, nil
	}
	out.producingLaunch = &witness.OpenedAt
	out.renderedTail = lines
	return out, validateRecoveryCaptureSnapshot(ctx, capture)
}

// recoveryRawFileActivity observes only descriptive metadata for an owned raw
// pathname. Capture witnesses, events and archive metadata remain separate
// authority checks, so their refusal cannot erase this readable fact.
func recoveryRawFileActivity(ctx context.Context, paths artifactpath.Paths, capture threadrecord.RecoveryCapture, agent string) ScreenObservation {
	if ctx.Err() != nil {
		return ScreenObservation{}
	}
	current, err := paths.ScrollbackArtifacts(agent)
	if err != nil {
		return ScreenObservation{}
	}
	if capture.Raw != current.Raw {
		owner, err := storagegc.SelectedOwner(paths.ScopeDir(), "", paths.Tag())
		if err != nil {
			return ScreenObservation{}
		}
		member, err := artifactpath.MatchArtifact(capture.Raw, []artifactpath.StorageOwner{owner}, nil)
		if err != nil {
			return ScreenObservation{}
		}
		archived, err := artifactpath.ParseParkedCapture(member, time.UTC)
		if err != nil || archived.Raw != capture.Raw {
			return ScreenObservation{}
		}
	}
	identity, err := storagegc.CaptureIdentity(capture.Raw)
	if err != nil {
		return ScreenObservation{}
	}
	return ScreenObservation{Availability: ScreenUnavailable, LastActivity: identity.ModifiedAt, ActivitySource: "capture-file activity"}
}

// Preserve copies a settled source before park; the executor has confirmed it
// is not busy, and capture identities are revalidated around the copy.
// Existing immutable archives are reused; current capture bytes are copied by
// Pair's capture publisher and checked on both sides before gaining authority.
func (r OSRecoveryEvidenceReader) Preserve(ctx context.Context, record ThreadRecord, in RecoveryObservation) (RecoveryObservation, error) {
	if ctx == nil {
		return in, errors.New("recovery preservation requires context")
	}
	ctx, cancel := context.WithTimeout(ctx, sessioninventory.RecoveryScanTimeout)
	defer cancel()
	if err := validateRecoveryCaptureSnapshot(ctx, in.Capture); err != nil {
		return in, err
	}
	paths, err := r.recoveryPaths(record)
	if err != nil {
		return in, err
	}
	lease, err := recoveryLease(ctx, paths, in.Capture, in.Agent)
	if err != nil {
		return in, err
	}
	if lease == nil {
		return in, errors.New("recovery capture lease unavailable")
	}
	defer lease.Close()
	metadataBefore, err := recoveryCaptureMetadata(ctx, paths, &in.Capture, in.Agent, nil)
	if err != nil {
		return in, err
	}
	if metadataBefore != nil {
		if metadataBefore.Witness != nil && metadataBefore.Witness.Generation != in.Generation {
			return in, errors.New("retained recovery producer changed")
		}
		return in, validateRecoveryCaptureSnapshot(ctx, in.Capture)
	}
	sourceHashes, err := recoveryCaptureHashes(ctx, in.Capture)
	if err != nil {
		return in, err
	}
	park := r.parkScrollback
	if park == nil {
		park = func(ctx context.Context, dataDir, tag, agent string) (string, bool) {
			return (launcher.OSRuntime{DataDir: dataDir}).ParkScrollbackContext(ctx, tag, agent, false)
		}
	}
	base, ok := park(ctx, paths.ScopeDir(), string(record.Address.Tag), in.Agent)
	if !ok {
		return in, errors.New("could not preserve recovery screen")
	}
	owner := lease.Owner
	member, err := artifactpath.MatchArtifact(base, []artifactpath.StorageOwner{owner}, nil)
	if err != nil {
		return in, fmt.Errorf("preserved recovery capture: %w", err)
	}
	archived, err := artifactpath.ParseParkedCapture(member, time.UTC)
	if err != nil {
		return in, err
	}
	raw, err := osfs.ReadRegularMetadata(ctx, archived.Metadata, recoveryCaptureMetadataLimit)
	if err != nil {
		return in, err
	}
	metadata, err := artifactpath.DecodeCaptureMetadata(raw, archived)
	if err != nil {
		return in, err
	}
	if metadata.EventsIdentity == nil {
		return in, errors.New("preserved recovery screen has no events")
	}
	if err := validateRecoveryCaptureSnapshot(ctx, in.Capture); err != nil {
		return in, err
	}
	if metadata.Witness != nil && metadata.Witness.Generation != in.Generation {
		return in, errors.New("preserved recovery producer changed")
	}
	if in.Capture.LaunchOrdinal != 0 && metadata.Witness == nil {
		return in, errors.New("preserved recovery witness disappeared")
	}
	in.Capture.Raw, in.Capture.Events = filepath.Clean(archived.Raw), filepath.Clean(archived.Events)
	in.Capture.RawIdentity, in.Capture.EventsIdentity = metadata.RawIdentity, *metadata.EventsIdentity
	in.Capture.CapturedAt = metadata.CapturedAt
	in.Capture.Token = archived.Token
	archiveHashes, err := recoveryCaptureHashes(ctx, in.Capture)
	if err != nil {
		return in, err
	}
	if sourceHashes != archiveHashes {
		return in, errors.New("preserved recovery bytes differ from original screen")
	}
	if metadata.Witness == nil {
		in.Generation = recoveryLegacyGeneration(in.Capture)
	}
	return in, validateRecoveryCaptureSnapshot(ctx, in.Capture)
}

func recoveryLegacyGeneration(c threadrecord.RecoveryCapture) string {
	raw, _ := json.Marshal(c)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func recoveryCaptureHashes(ctx context.Context, c threadrecord.RecoveryCapture) ([2][32]byte, error) {
	var hashes [2][32]byte
	if err := validateRecoveryCaptureSnapshot(ctx, c); err != nil {
		return hashes, err
	}
	for i, p := range []string{c.Raw, c.Events} {
		limit := int64(64 << 20)
		if i == 1 {
			limit = artifactpath.MaxCaptureWitnessEventsBytes
		}
		raw, err := osfs.ReadRegularMetadata(ctx, p, limit)
		if err != nil {
			return hashes, err
		}
		hashes[i] = sha256.Sum256(raw)
	}
	return hashes, validateRecoveryCaptureSnapshot(ctx, c)
}

// Current captures have no archive metadata. Retained captures must belong to
// this exact owner and match their publisher's immutable identities and clock.
func recoveryCaptureMetadata(ctx context.Context, paths artifactpath.Paths, c *threadrecord.RecoveryCapture, agent string, charged *int64) (*artifactpath.CaptureMetadata, error) {
	current, err := paths.ScrollbackArtifacts(agent)
	if err != nil {
		return nil, err
	}
	if current.Raw == c.Raw {
		if current.Events != c.Events || c.Token != "" {
			return nil, errors.New("current recovery members mismatch")
		}
		return nil, nil
	}
	owner, err := storagegc.SelectedOwner(paths.ScopeDir(), "", paths.Tag())
	if err != nil {
		return nil, err
	}
	member, err := artifactpath.MatchArtifact(c.Raw, []artifactpath.StorageOwner{owner}, nil)
	if err != nil {
		return nil, err
	}
	archived, err := artifactpath.ParseParkedCapture(member, time.UTC)
	if err != nil || archived.Events != c.Events || (c.Token != "" && c.Token != archived.Token) {
		return nil, errors.New("recovery source is not exact owner archive")
	}
	if charged != nil {
		// Authorization above can refuse without IO. Reserve only when the
		// bounded metadata read itself is about to be attempted.
		if *charged > sessioninventory.RecoveryTotalReadLimit-recoveryCaptureMetadataLimit {
			return nil, errors.New("previous screen metadata exceeds recovery read budget")
		}
		*charged += recoveryCaptureMetadataLimit
	}
	raw, err := osfs.ReadRegularMetadata(ctx, archived.Metadata, recoveryCaptureMetadataLimit)
	if err != nil {
		return nil, err
	}
	metadata, err := artifactpath.DecodeCaptureMetadata(raw, archived)
	if err != nil {
		return nil, err
	}
	if metadata.EventsIdentity == nil {
		return nil, errors.New("retained recovery has no events identity")
	}
	if c.RawIdentity.Inode != 0 && (c.RawIdentity != metadata.RawIdentity || c.EventsIdentity != *metadata.EventsIdentity) {
		return nil, errors.New("retained recovery metadata changed")
	}
	if !c.CapturedAt.IsZero() && !c.CapturedAt.Equal(metadata.CapturedAt) {
		return nil, errors.New("retained recovery publication clock changed")
	}
	c.RawIdentity, c.EventsIdentity = metadata.RawIdentity, *metadata.EventsIdentity
	c.CapturedAt = metadata.CapturedAt
	c.Token = archived.Token
	if err := validateRecoveryCaptureSnapshot(ctx, *c); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (r OSRecoveryEvidenceReader) recoveryBranchIssue(ctx context.Context, record ThreadRecord) (string, string) {
	if intent := retainedRecoveryIntent(record); intent != nil {
		if intent.Kind() == threadrecord.RecoveryInferred {
			return intent.Context.Branch, intent.Context.Issue
		}
		if intent.Orientation != nil {
			return intent.Orientation.Branch, intent.Orientation.Issue
		}
	}
	if c := record.RecoveryContext; c != nil {
		return c.Branch, c.Issue
	}
	git := r.Git
	if git == nil {
		git = ExecGit{}
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	branch, err := git.RunContext(ctx, record.WorkingPath, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || len(branch) > 1024 || !utf8.ValidString(branch) || strings.IndexFunc(branch, unicode.IsControl) >= 0 {
		return "", ""
	}
	issue := ""
	prefix := branch
	if i := strings.IndexByte(prefix, '-'); i >= 0 {
		prefix = prefix[:i]
	}
	if n, err := strconv.ParseUint(prefix, 10, 64); err == nil && n != 0 {
		issue = "#" + strconv.FormatUint(n, 10)
	}
	return branch, issue
}

func (r OSRecoveryEvidenceReader) recoveryPaths(record ThreadRecord) (artifactpath.Paths, error) {
	root, err := filepath.EvalSymlinks(r.DataDir)
	if err != nil {
		return artifactpath.Paths{}, err
	}
	return artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
}

// PrepareTerminal captures exact process authority without retiring anything.
// The caller preserves the screen before invoking the returned second gate.
func (r OSRecoveryEvidenceReader) PrepareTerminal(ctx context.Context, record ThreadRecord) (func(context.Context) error, error) {
	root, name, err := r.recoveryTerminalAddress(ctx, record)
	if err != nil {
		return nil, err
	}
	prepare := r.prepareTerminal
	if prepare == nil {
		prepare = launcher.PrepareRecoveryTerminal
	}
	retire := r.retireTerminal
	if retire == nil {
		retire = launcher.RetireRecoveryTerminal
	}
	plan, err := prepare(ctx, root, record.Address.RepoScope, string(record.Address.Tag), name)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error { return retire(ctx, plan) }, nil
}
