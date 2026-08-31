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

// QuerySession reads one exact owner ledger and validates only its proof-named
// artifacts. Proofless legacy bindings remain unavailable for automatic use.
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
	pairRoot := runtime.PairDataRoot()
	files, listErr := runtime.ListFiles(pairRoot)
	if err := ctx.Err(); err != nil {
		return SessionQuery{}, err
	}
	var issues *ListingIssuesError
	if listErr != nil && !errors.As(listErr, &issues) {
		return SessionQuery{}, listErr
	}
	if issues != nil {
		for _, artifact := range issues.Artifacts {
			query.Diagnostics = append(query.Diagnostics, artifactDiagnostic(DiagnosticArtifactPathInvalid, "", nil, artifact, "non-regular Pair storage entry rejected"))
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
		return query, nil
	}
	raw, err := runtime.ReadFile(ledger, 8<<20)
	if err != nil {
		return SessionQuery{}, err
	}
	if err := ctx.Err(); err != nil {
		return SessionQuery{}, err
	}
	parsed := sessionledger.ParseLedger(raw)
	for _, ordinal := range parsed.MalformedOrdinals {
		query.Diagnostics = append(query.Diagnostics, diagnosticWithSource(DiagnosticPairRecordMalformed, agent, nil, fmt.Sprintf("ledger:%s:%d", tag, ordinal), "Pair ledger row is malformed"))
	}
	current, ok := sessionledger.CurrentLaunch(parsed.Records, sessionledger.Owner{ScopeKey: scopeKey, Tag: tag, Agent: string(agent)})
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
	if current.Binding.AuthorizationProof == nil {
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
	validation, diagnostics, err := NewIncrementalInventory(runtime, catalog).ValidateBindingProof(agent, *current.Binding.AuthorizationProof)
	if contextErr := ctx.Err(); contextErr != nil {
		return SessionQuery{}, contextErr
	}
	query.Diagnostics = append(query.Diagnostics, diagnostics...)
	if err != nil {
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
	unchanged := observationsMatchValidation(selected.Eligible, prior)
	if unchanged {
		return prior, diagnostics, nil
	}
	advanced, found, err := AdvanceTargetValidation(inventory.runtime, prior, selected.Eligible)
	diagnostics = append(diagnostics, found...)
	if err == nil || !proofAllowsFullGrowthRevalidation(proof, selected.Eligible) {
		return advanced, diagnostics, err
	}
	// Some filesystems expose stable file identity but no true generation
	// token. A proof-authorized transcript may still grow normally after the
	// binding is committed. In that exact monotonic-growth case, validate the
	// one proof-named target from byte zero rather than revoking the established
	// root or broadening into a corpus scan.
	validated, fallbackDiagnostics := ValidateTargetWork(inventory.runtime, agent, selected.Eligible)
	diagnostics = append(diagnostics, fallbackDiagnostics...)
	if len(validated) != 1 {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	candidate := validated[0]
	if candidate.State.Agent != agent || candidate.State.NativeID != proof.RootNativeID || candidate.State.Role != RoleRoot || candidate.State.ScannerSchema != proof.ScannerSchema || candidate.State.Disputed {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	return candidate, diagnostics, nil
}

func proofAllowsFullGrowthRevalidation(proof sessionledger.AuthorizationProof, current []ArtifactObservation) bool {
	if len(current) != len(proof.Artifacts) || len(current) == 0 {
		return false
	}
	grew, missingGeneration := false, false
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
		grew = grew || observation.Entry.Size > artifact.Size
	}
	return grew && missingGeneration
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
		if err != nil || state.Agent != agent || state.NativeID != proof.RootNativeID || state.Role != RoleRoot || state.ScannerSchema != proof.ScannerSchema {
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
