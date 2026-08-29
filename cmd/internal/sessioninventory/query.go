package sessioninventory

import (
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
	query := SessionQuery{Status: BindingUnbound}
	pairRoot := runtime.PairDataRoot()
	files, listErr := runtime.ListFiles(pairRoot)
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
	validation, diagnostics, err := validateBindingProofTarget(runtime, agent, *current.Binding.AuthorizationProof)
	query.Diagnostics = append(query.Diagnostics, diagnostics...)
	if err != nil {
		return query, nil
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

func validateBindingProofTarget(runtime Runtime, agent Agent, proof sessionledger.AuthorizationProof) (TargetValidation, []Diagnostic, error) {
	inventory := NewIncrementalInventory(runtime, Catalog{Version: CatalogVersion})
	return inventory.ValidateBindingProof(agent, proof)
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
	proofByArtifact := make(map[string]sessionledger.ArtifactProof, len(proof.Artifacts))
	for _, artifact := range proof.Artifacts {
		proofByArtifact[artifact.StorageRoot+"\x00"+artifact.RelativePath] = artifact
	}
	factArtifacts := make([]Artifact, 0, len(selected.Eligible))
	prior := TargetValidation{State: state, Observations: selected.Eligible, Results: map[string]IncrementalResult{}}
	unchanged := true
	for i, observation := range selected.Eligible {
		factArtifacts = append(factArtifacts, observation.Entry.Artifact)
		artifact, ok := proofByArtifact[targetArtifactKey(observation.Entry.Artifact)]
		if !ok {
			return TargetValidation{}, diagnostics, ErrArtifactChanged
		}
		current := fingerprintFromEntry(observation.Entry)
		if current.StableFileID != StableFileID(artifact.StableFileID) || current.GenerationToken != GenerationToken(artifact.GenerationToken) || current.MutationToken != MutationToken(artifact.MutationToken) || current.Size != artifact.Size {
			unchanged = false
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
	if unchanged {
		return prior, diagnostics, nil
	}
	advanced, found, err := AdvanceTargetValidation(inventory.runtime, prior, selected.Eligible)
	diagnostics = append(diagnostics, found...)
	return advanced, diagnostics, err
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
