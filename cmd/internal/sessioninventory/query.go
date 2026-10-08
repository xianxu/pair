package sessioninventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

var ErrRootTranscript = errors.New("established root must have exactly one transcript artifact")

// SessionQuery preserves the owner's binding state. Root is populated only for
// an established binding whose scanner-authorized root is present.
type SessionQuery struct {
	Status      BindingStatus
	Root        *Node
	Diagnostics []Diagnostic
}

// SessionCatalogAdvancer is the persistent owner shared by every production
// interactive query. Runtime remains the scanner IO seam; this optional
// capability lets catalog-loss tests keep exercising proof fallback directly.
type SessionCatalogAdvancer interface {
	LoadSessionInventoryCatalog() (Catalog, error)
	PublishSessionInventoryValidations([]TargetValidation) error
}

// SessionForOwner is the pure owner lookup shared by native-session consumers.
// Ambiguous, provisional, and unbound owners never receive a root fallback.
func SessionForOwner(inventory Inventory, scopeKey, tag string, agent Agent) SessionQuery {
	query := SessionQuery{Status: BindingUnbound, Diagnostics: append([]Diagnostic(nil), inventory.Diagnostics...)}
	for _, binding := range inventory.Bindings {
		if binding.ScopeKey != scopeKey || binding.Tag != tag || binding.Agent != agent {
			continue
		}
		query.Status = binding.Status
		if binding.Status != BindingEstablished || binding.RootNodeID == nil {
			return query
		}
		for _, forest := range inventory.Forests {
			if forest.Agent != agent {
				continue
			}
			for _, root := range forest.Roots {
				if root.StableID == *binding.RootNodeID {
					value := cloneNode(root)
					query.Root = &value
					return query
				}
			}
		}
		return query
	}
	return query
}

// QuerySession supplies optional parsed content for one exact owner. It reads
// proof-named artifacts, or the named root of a v3 filename confirmation.
// Resume admission uses QueryResumeTarget instead.
func QuerySession(runtime Runtime, scopeKey, tag string, agent Agent) (SessionQuery, error) {
	return QuerySessionContext(context.Background(), runtime, scopeKey, tag, agent)
}

func QuerySessionContext(ctx context.Context, runtime Runtime, scopeKey, tag string, agent Agent) (SessionQuery, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return SessionQuery{}, err
	}
	query := SessionQuery{Status: BindingUnbound}
	current, _, ok, diagnostics, err := readOwnerLaunch(ctx, runtime, scopeKey, tag, agent)
	query.Diagnostics = diagnostics
	if err != nil {
		return SessionQuery{}, err
	}
	if !ok {
		return query, nil
	}
	query.Status = BindingProvisional
	if current.Conflict {
		query.Status = BindingAmbiguous
		return query, nil
	}
	if current.Binding == nil {
		return query, nil
	}
	if current.Binding.AuthorizationProof == nil && current.Binding.Version < 3 {
		nativeID := current.Binding.RootNativeID
		query.Diagnostics = append(query.Diagnostics, diagnosticWithSource(DiagnosticBindingStale, agent, &nativeID, "ledger proof", "legacy binding proof migration is pending"))
		return query, nil
	}
	catalog := Catalog{Version: CatalogVersion}
	advancer, persists := runtime.(SessionCatalogAdvancer)
	if persists {
		if saved, readErr := advancer.LoadSessionInventoryCatalog(); readErr == nil {
			catalog = saved
		}
		if err := ctx.Err(); err != nil {
			return SessionQuery{}, err
		}
	}
	incremental := NewIncrementalInventory(runtime, catalog)
	var validation TargetValidation
	if current.Binding.AuthorizationProof != nil {
		validation, diagnostics, err = incremental.ValidateBindingProof(agent, *current.Binding.AuthorizationProof)
	} else {
		validation, diagnostics, err = incremental.ValidateNamedContent(agent, current.Binding.RootNativeID)
	}
	if contextErr := ctx.Err(); contextErr != nil {
		return SessionQuery{}, contextErr
	}
	query.Diagnostics = append(query.Diagnostics, diagnostics...)
	if err != nil {
		// Status stays provisional for the caller, but the reason is recorded:
		// without it a failed proof reads as "no turn yet" (pair#328).
		nativeID := current.Binding.RootNativeID
		query.Diagnostics = append(query.Diagnostics, diagnosticWithSource(DiagnosticBindingStale, agent, &nativeID, "ledger proof", "binding proof no longer validates: "+err.Error()))
		return query, nil
	}
	if persists && !catalogCoversValidation(catalog, validation) {
		if publishErr := advancer.PublishSessionInventoryValidations([]TargetValidation{validation}); publishErr != nil {
			query.Diagnostics = append(query.Diagnostics, diagnosticWithSource(DiagnosticStorageUnreadable, agent, &current.Binding.RootNativeID, "session inventory catalog", "validated query advancement could not be persisted"))
		}
		if err := ctx.Err(); err != nil {
			return SessionQuery{}, err
		}
	}
	inventory := BuildForest([]Fact{validation.Fact})
	for _, forest := range inventory.Forests {
		if forest.Agent == agent && len(forest.Roots) == 1 {
			root := cloneNode(forest.Roots[0])
			query.Root = &root
			query.Status = BindingEstablished
			return query, nil
		}
	}
	return query, nil
}

func (inventory IncrementalInventory) ValidateBindingProof(agent Agent, proof sessionledger.AuthorizationProof) (TargetValidation, []Diagnostic, error) {
	if err := sessionledger.ValidateAuthorizationProof(proof, proof.RootNativeID); err != nil {
		return TargetValidation{}, nil, err
	}
	state, err := DecodeScannerState(proof.ScannerState)
	if err != nil || state.Agent != agent || state.NativeID != proof.RootNativeID || state.Role != RoleRoot || state.ScannerSchema != proof.ScannerSchema {
		return TargetValidation{}, nil, errors.New("binding proof scanner state disagrees with owner")
	}
	snapshot := inventory.Observe(agent)
	diagnostics := snapshot.Diagnostics
	artifacts := make([]Artifact, 0, len(proof.Artifacts))
	for _, artifact := range proof.Artifacts {
		artifacts = append(artifacts, Artifact{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath})
	}
	selected := inventory.Select(TargetRequest{Mode: TargetEstablished, Agent: agent, NativeID: proof.RootNativeID, AuthorizedArtifacts: artifacts}, snapshot)
	if selected.Unavailable || len(selected.Eligible) != len(proof.Artifacts) {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	factArtifacts := make([]Artifact, 0, len(selected.Eligible))
	prior := TargetValidation{State: state, Observations: make([]ArtifactObservation, len(selected.Eligible)), Results: map[string]IncrementalResult{}}
	for i, observation := range selected.Eligible {
		prior.Observations[i] = cloneObservation(observation)
		factArtifacts = append(factArtifacts, observation.Entry.Artifact)
		artifact, ok := proofArtifactByKey(proof, targetArtifactKey(observation.Entry.Artifact))
		if !ok {
			return TargetValidation{}, diagnostics, ErrArtifactChanged
		}
		// Proofs deliberately exclude timestamps. Preserve the current timestamps
		// while reconstructing the proof-owned prior tuple so timestamp metadata
		// cannot force a body replay or weaken generation continuity.
		prior.Observations[i].Entry.StableFileID = StableFileID(artifact.StableFileID)
		prior.Observations[i].Entry.GenerationToken = GenerationToken(artifact.GenerationToken)
		prior.Observations[i].Entry.MutationToken = MutationToken(artifact.MutationToken)
		prior.Observations[i].Entry.Size = artifact.Size
		if observation.Entry.Artifact.Kind == ArtifactTranscript {
			prior.Results[targetArtifactKey(observation.Entry.Artifact)] = IncrementalResult{
				Fingerprint:       ArtifactFingerprint{StableFileID: StableFileID(artifact.StableFileID), GenerationToken: GenerationToken(artifact.GenerationToken), MutationToken: MutationToken(artifact.MutationToken), Size: artifact.Size, BirthTime: cloneStdTime(observation.Entry.BirthTime), ModTime: cloneStdTime(observation.Entry.ModTime)},
				RawObservedOffset: artifact.Size, FrameState: JSONLFrameState{ParserCompleteOffset: artifact.ParserCompleteOffset},
			}
		}
	}
	prior.Fact, err = ScannerStateFact(state, factArtifacts)
	if err != nil {
		return TargetValidation{}, diagnostics, err
	}
	if catalogPrior, ok := inventory.catalogPriorForProof(agent, proof, selected.Eligible); ok {
		prior = catalogPrior
	}
	schemaChanged := false
	for _, observation := range selected.Eligible {
		if observation.ScannerSchema != prior.State.ScannerSchema {
			schemaChanged = true
		}
	}
	unchanged := !schemaChanged && observationsMatchValidation(selected.Eligible, prior)
	if unchanged {
		return prior, diagnostics, nil
	}
	advanced, found, err := AdvanceTargetValidation(inventory.runtime, prior, selected.Eligible)
	diagnostics = append(diagnostics, found...)
	if err == nil || (!schemaChanged && !proofAllowsFullRevalidation(proof, selected.Eligible)) {
		return advanced, diagnostics, err
	}
	// Some filesystems expose stable file identity but no true generation
	// token. A proof-authorized transcript may still grow normally after the
	// binding is committed, or have only its metadata touched -- a resuming
	// agent can bump ctime without writing a byte (pair#328). In either case the
	// file is not smaller, so validate the one proof-named target from byte
	// zero rather than revoking the established root or broadening into a
	// corpus scan. The re-read, not the metadata, decides: a same-size rewrite
	// into another conversation still fails the identity check below.
	validated, fallbackDiagnostics := ValidateTargetWork(inventory.runtime, agent, selected.Eligible)
	diagnostics = append(diagnostics, fallbackDiagnostics...)
	if len(validated) != 1 {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	candidate := validated[0]
	if candidate.State.Agent != agent || candidate.State.NativeID != proof.RootNativeID || candidate.State.Role != RoleRoot || candidate.State.Disputed {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	return candidate, diagnostics, nil
}

func proofAllowsFullRevalidation(proof sessionledger.AuthorizationProof, current []ArtifactObservation) bool {
	if len(current) != len(proof.Artifacts) || len(current) == 0 {
		return false
	}
	missingGeneration := false
	for _, observation := range current {
		artifact, ok := proofArtifactByKey(proof, targetArtifactKey(observation.Entry.Artifact))
		if !ok || string(observation.Entry.StableFileID) != artifact.StableFileID || observation.Entry.Size < artifact.Size {
			return false
		}
		if artifact.GenerationToken == "" {
			missingGeneration = true
			if observation.Entry.GenerationToken != "" {
				return false
			}
		} else if string(observation.Entry.GenerationToken) != artifact.GenerationToken {
			return false
		}
	}
	return missingGeneration
}

func proofArtifactByKey(proof sessionledger.AuthorizationProof, key string) (sessionledger.ArtifactProof, bool) {
	for _, artifact := range proof.Artifacts {
		if artifact.StorageRoot+"\x00"+artifact.RelativePath == key {
			return artifact, true
		}
	}
	return sessionledger.ArtifactProof{}, false
}

func (inventory IncrementalInventory) catalogPriorForProof(agent Agent, proof sessionledger.AuthorizationProof, current []ArtifactObservation) (TargetValidation, bool) {
	entries := make(map[string]CatalogEntry, len(inventory.catalog.Entries))
	for _, entry := range inventory.catalog.Entries {
		if entry.Agent == agent {
			entries[targetArtifactKey(entry.Artifact)] = entry
		}
	}
	prior := TargetValidation{Observations: make([]ArtifactObservation, len(current)), Results: map[string]IncrementalResult{}}
	artifacts := make([]Artifact, 0, len(current))
	for i, observation := range current {
		key := targetArtifactKey(observation.Entry.Artifact)
		entry, ok := entries[key]
		proofArtifact, proofOK := proofArtifactByKey(proof, key)
		expectedSchema, _, recognized := artifactScannerShape(agent, observation.Entry.Artifact)
		if !ok || !proofOK || !recognized || entry.Authorization != AuthorizationAuthorized || entry.ScannerSchema != expectedSchema ||
			entry.Fingerprint.StableFileID != StableFileID(proofArtifact.StableFileID) ||
			entry.Fingerprint.GenerationToken != GenerationToken(proofArtifact.GenerationToken) ||
			entry.Fingerprint.Size < proofArtifact.Size || entry.ParserCompleteOffset < proofArtifact.ParserCompleteOffset {
			return TargetValidation{}, false
		}
		state, err := DecodeScannerState(entry.ScannerState)
		if err != nil || state.Agent != agent || state.NativeID != proof.RootNativeID || state.Role != RoleRoot || state.ScannerSchema != expectedSchema {
			return TargetValidation{}, false
		}
		if i == 0 {
			prior.State = state
		} else if string(entry.ScannerState) != string(entries[targetArtifactKey(current[0].Entry.Artifact)].ScannerState) {
			return TargetValidation{}, false
		}
		prior.Observations[i] = observation
		prior.Observations[i].Entry.StableFileID = entry.Fingerprint.StableFileID
		prior.Observations[i].Entry.GenerationToken = entry.Fingerprint.GenerationToken
		prior.Observations[i].Entry.MutationToken = entry.Fingerprint.MutationToken
		prior.Observations[i].Entry.Size = entry.Fingerprint.Size
		prior.Observations[i].Entry.BirthTime = cloneStdTime(entry.Fingerprint.BirthTime)
		prior.Observations[i].Entry.ModTime = cloneStdTime(entry.Fingerprint.ModTime)
		prior.Observations[i].ScannerSchema = entry.ScannerSchema
		prior.Observations[i].ProviderContract = entry.ProviderContract
		artifacts = append(artifacts, observation.Entry.Artifact)
		if observation.Entry.Artifact.Kind == ArtifactTranscript {
			prior.Results[key] = IncrementalResult{Fingerprint: entry.Fingerprint, RawObservedOffset: entry.RawObservedOffset, FrameState: JSONLFrameState{ParserCompleteOffset: entry.ParserCompleteOffset}}
		}
	}
	var err error
	prior.Fact, err = ScannerStateFact(prior.State, artifacts)
	return prior, err == nil
}

func observationsMatchValidation(current []ArtifactObservation, prior TargetValidation) bool {
	previous := make(map[string]ArtifactFingerprint, len(prior.Observations))
	for _, observation := range prior.Observations {
		previous[targetArtifactKey(observation.Entry.Artifact)] = fingerprintFromEntry(observation.Entry)
	}
	for _, observation := range current {
		fingerprint, ok := previous[targetArtifactKey(observation.Entry.Artifact)]
		if !ok || !equalFingerprint(fingerprint, fingerprintFromEntry(observation.Entry)) {
			return false
		}
	}
	return len(current) == len(previous)
}

func catalogCoversValidation(catalog Catalog, validation TargetValidation) bool {
	entries := make(map[string]CatalogEntry, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		entries[catalogEntryKey(entry.Agent, entry.Artifact)] = entry
	}
	for _, observation := range validation.Observations {
		entry, ok := entries[catalogEntryKey(observation.Agent, observation.Entry.Artifact)]
		if !ok || entry.Authorization != AuthorizationAuthorized {
			return false
		}
		fingerprint := fingerprintFromEntry(observation.Entry)
		rawOffset, parserOffset := observation.Entry.Size, observation.Entry.Size
		if result, ok := validation.Results[targetArtifactKey(observation.Entry.Artifact)]; ok {
			fingerprint = result.Fingerprint
			rawOffset = result.RawObservedOffset
			parserOffset = result.FrameState.ParserCompleteOffset
		}
		if !equalFingerprint(entry.Fingerprint, fingerprint) || entry.RawObservedOffset != rawOffset || entry.ParserCompleteOffset != parserOffset || string(entry.ScannerState) != string(mustScannerStateJSON(validation.State)) {
			return false
		}
	}
	return len(validation.Observations) != 0
}

func mustScannerStateJSON(state ScannerState) []byte {
	raw, _ := json.Marshal(state)
	return raw
}

// RootTranscript returns the one scanner-authorized transcript for a root.
func RootTranscript(root Node) (Artifact, error) {
	var transcript Artifact
	count := 0
	for _, artifact := range root.Artifacts {
		if artifact.Kind == ArtifactTranscript {
			transcript = artifact
			count++
		}
	}
	if count != 1 {
		return Artifact{}, fmt.Errorf("%w: got %d", ErrRootTranscript, count)
	}
	return transcript, nil
}

// readOwnerLaunch reads only Pair-owned identity records, never native bodies.
func readOwnerLaunch(ctx context.Context, runtime Runtime, scopeKey, tag string, agent Agent) (sessionledger.Current, []sessionledger.Record, bool, []Diagnostic, error) {
	var diagnostics []Diagnostic
	pairRoot := runtime.PairDataRoot()
	files, listErr := runtime.ListFiles(pairRoot)
	if err := ctx.Err(); err != nil {
		return sessionledger.Current{}, nil, false, diagnostics, err
	}
	var issues *ListingIssuesError
	if listErr != nil && !errors.As(listErr, &issues) {
		return sessionledger.Current{}, nil, false, diagnostics, listErr
	}
	if issues != nil {
		for _, artifact := range issues.Artifacts {
			diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticArtifactPathInvalid, "", nil, artifact, "non-regular Pair storage entry rejected"))
		}
	}
	var ledger Artifact
	for _, file := range files {
		candidateTag, ok := artifactpath.TagFromHistorySidecar(file.Artifact.RelativePath)
		if ok && candidateTag == tag && artifactpath.IsLedgerHistorySidecar(file.Artifact.RelativePath) {
			ledger = file.Artifact
			break
		}
	}
	if ledger.RelativePath == "" {
		return sessionledger.Current{}, nil, false, diagnostics, nil
	}
	// Both the ledger and its launch boundary snapshots grow without a writer
	// size cap. Keep chunked reads without imposing a reader-only cutoff.
	raw, err := readJSONLArtifact(runtime, ledger, unlimitedRecordSize)
	if err != nil {
		return sessionledger.Current{}, nil, false, diagnostics, err
	}
	if err := ctx.Err(); err != nil {
		return sessionledger.Current{}, nil, false, diagnostics, err
	}
	parsed := sessionledger.ParseLedger(raw)
	for _, ordinal := range parsed.MalformedOrdinals {
		diagnostics = append(diagnostics, diagnosticWithSource(DiagnosticPairRecordMalformed, agent, nil, fmt.Sprintf("ledger:%s:%d", tag, ordinal), "Pair ledger row is malformed"))
	}
	current, ok := sessionledger.CurrentLaunch(parsed.Records, sessionledger.Owner{ScopeKey: scopeKey, Tag: tag, Agent: string(agent)})
	if !ok {
		return sessionledger.Current{}, nil, false, diagnostics, nil
	}
	return current, parsed.Records, true, diagnostics, nil
}

// ResumeTarget is durable requested/observed identity, independent of optional
// native parsing and disposable catalog state. Requested-resume probation is
// usable; an unmaterialized fresh chosen ID instead requires a fresh launch.
type ResumeTarget struct {
	Status            BindingStatus
	FreshRequired     bool
	NativeID          string
	LaunchOrdinal     uint64
	RequestedNativeID string
	// FellBackFrom is the ordinal of an unturned fresh launch this target
	// looked past to the conversation it replaced (pair#214); zero otherwise.
	FellBackFrom uint64
	Diagnostics  []Diagnostic
}

func QueryResumeTarget(runtime Runtime, scopeKey, tag string, agent Agent) (ResumeTarget, error) {
	return QueryResumeTargetContext(context.Background(), runtime, scopeKey, tag, agent)
}
func QueryResumeTargetContext(ctx context.Context, runtime Runtime, scopeKey, tag string, agent Agent) (ResumeTarget, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ResumeTarget{}, err
	}
	current, records, ok, diagnostics, err := readOwnerLaunch(ctx, runtime, scopeKey, tag, agent)
	result := ResumeTarget{Status: BindingUnbound, Diagnostics: diagnostics}
	if err != nil || !ok {
		return result, err
	}
	result = ResumeTargetForRuntimeLaunch(runtime, current)
	if result.FreshRequired {
		// A chosen-id launch whose file a complete listing proves absent: its
		// agent never took a turn, so it left no conversation to resume. The
		// conversation it replaced is the thread's real state (pair#214 D1),
		// so resume falls back to that generation. Restart decisions read
		// ResumeTargetForRuntimeLaunch per launch and are unaffected.
		owner := sessionledger.Owner{ScopeKey: scopeKey, Tag: tag, Agent: string(agent)}
		if earlier, found := sessionledger.PreviousEstablished(records, owner, current.Launch.Ordinal); found {
			fallback := ResumeTargetForLaunch(earlier)
			fallback.FellBackFrom = current.Launch.Ordinal
			fallback.Diagnostics = result.Diagnostics
			result = fallback
		}
	}
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	return result, nil
}

// ResumeTargetForLaunch is the pure shared identity projection for launchers,
// restarts and owner queries. Parser/cache state does not enter this decision.
func ResumeTargetForLaunch(current sessionledger.Current) ResumeTarget {
	result := ResumeTarget{Status: BindingProvisional, LaunchOrdinal: current.Launch.Ordinal, RequestedNativeID: current.Launch.RequestedNativeID}
	if current.Conflict {
		result.Status = BindingAmbiguous
		return result
	}
	if current.Binding != nil {
		result.Status = BindingEstablished
		result.NativeID = current.Binding.RootNativeID
	} else if current.Launch.RequestOrigin != sessionledger.RequestOriginChosen {
		result.NativeID = current.Launch.RequestedNativeID
	}
	return result
}

// ResumeTargetForRuntimeLaunch admits an unconfirmed Pair-chosen UUID only
// after its root filename exists. Confirmed and requested-resume identities
// remain usable without a native file. No transcript contents are inspected.
func ResumeTargetForRuntimeLaunch(runtime Runtime, current sessionledger.Current) ResumeTarget {
	result := ResumeTargetForLaunch(current)
	if result.Status == BindingProvisional && current.Launch.RequestOrigin == sessionledger.RequestOriginChosen {
		observations, diagnostics := ObserveAgentMetadata(runtime, Agent(current.Launch.Agent))
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		selected := SelectTargetWork(TargetRequest{Mode: TargetExplicitResume, Agent: Agent(current.Launch.Agent), NativeID: current.Launch.RequestedNativeID}, observations)
		if !selected.Unavailable {
			result.NativeID = current.Launch.RequestedNativeID
			return result
		}
		// A matching filename proves presence even in a partial listing. Only
		// complete enumeration (or an absent root) can prove absence.
		for _, diagnostic := range diagnostics {
			if diagnostic.Code != DiagnosticStorageAbsent {
				return result
			}
		}
		result.FreshRequired = true
	}
	return result
}

// ValidateNamedContent supplies optional telemetry for filename-handshake
// confirmations. Catalog proof reuse is an optimization, never resume authority.
func (inventory IncrementalInventory) ValidateNamedContent(agent Agent, nativeID string) (TargetValidation, []Diagnostic, error) {
	for _, entry := range inventory.catalog.Entries {
		state, err := DecodeScannerState(entry.ScannerState)
		if err != nil || entry.Agent != agent || entry.Authorization != AuthorizationAuthorized || state.NativeID != nativeID || state.Role != RoleRoot {
			continue
		}
		proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: nativeID, ScannerSchema: state.ScannerSchema, ScannerState: entry.ScannerState}
		for _, candidate := range inventory.catalog.Entries {
			if candidate.Agent != agent || string(candidate.ScannerState) != string(entry.ScannerState) || candidate.Authorization != AuthorizationAuthorized {
				continue
			}
			f := candidate.Fingerprint
			proof.Artifacts = append(proof.Artifacts, sessionledger.ArtifactProof{StorageRoot: candidate.Artifact.StorageRoot, RelativePath: candidate.Artifact.RelativePath, StableFileID: string(f.StableFileID), GenerationToken: string(f.GenerationToken), MutationToken: string(f.MutationToken), Size: f.Size, ParserCompleteOffset: candidate.ParserCompleteOffset})
		}
		if validation, diagnostics, err := inventory.ValidateBindingProof(agent, proof); err == nil {
			return validation, diagnostics, nil
		}
		break
	}
	snapshot := inventory.Observe(agent)
	selected := inventory.Select(TargetRequest{Mode: TargetExplicitResume, Agent: agent, NativeID: nativeID}, snapshot)
	validations, diagnostics := ValidateTargetWork(inventory.runtime, agent, selected.Eligible)
	diagnostics = append(snapshot.Diagnostics, diagnostics...)
	if len(validations) != 1 || validations[0].State.Role != RoleRoot || validations[0].State.NativeID != nativeID {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	return validations[0], diagnostics, nil
}
