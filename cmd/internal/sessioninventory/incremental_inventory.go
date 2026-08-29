package sessioninventory

import (
	"errors"
	"fmt"
)

var ErrArtifactChanged = errors.New("session inventory artifact changed during observation")

// IncrementalResult contains bytes framed only after one artifact generation
// reaches a stable observed EOF.
// pair:156-concept pure new final ScannerState / IncrementalResult
type IncrementalResult struct {
	Fingerprint       ArtifactFingerprint
	RawObservedOffset int64
	FrameState        JSONLFrameState
	Records           []FramedJSONLRecord
	Disputed          bool
}

type TargetValidation struct {
	State        ScannerState
	Fact         Fact
	Observations []ArtifactObservation
	Results      map[string]IncrementalResult
	Events       []NativeEventFact
}

// IncrementalInventory is the production authority seam for metadata
// discovery, catalog reconciliation, and bounded target selection.
// pair:156-concept integration new final
type IncrementalInventory struct {
	runtime Runtime
	catalog Catalog
}

type IncrementalSnapshot struct {
	Observations []ArtifactObservation
	Delta        CatalogDelta
	Diagnostics  []Diagnostic
}

func NewIncrementalInventory(runtime Runtime, catalog Catalog) IncrementalInventory {
	if catalog.Version == 0 {
		catalog.Version = CatalogVersion
	}
	return IncrementalInventory{runtime: runtime, catalog: CloneCatalog(catalog)}
}

func (inventory IncrementalInventory) Observe(agent Agent) IncrementalSnapshot {
	observations, diagnostics := ObserveAgentMetadata(inventory.runtime, agent)
	return IncrementalSnapshot{
		Observations: observations,
		Delta:        ReconcileCatalog(inventory.catalog, observations),
		Diagnostics:  diagnostics,
	}
}

func (inventory IncrementalInventory) Select(request TargetRequest, snapshot IncrementalSnapshot) TargetResult {
	return SelectTargetWork(request, snapshot.Observations)
}

func (inventory IncrementalInventory) CatalogSessionCandidateExists(snapshot IncrementalSnapshot, agent Agent, nativeID string) bool {
	return CatalogSessionCandidateExists(inventory.catalog, snapshot.Observations, agent, nativeID)
}

// ObserveAgentMetadata discovers candidate shapes without reading artifact
// bodies or granting scanner authority.
func ObserveAgentMetadata(runtime Runtime, agent Agent) ([]ArtifactObservation, []Diagnostic) {
	var observations []ArtifactObservation
	var diagnostics []Diagnostic
	for _, root := range runtime.NativeRoots(agent) {
		files, found, ok := scannerFiles(runtime, agent, root)
		diagnostics = append(diagnostics, found...)
		if !ok {
			continue
		}
		for _, entry := range files {
			schema, kind, recognized := artifactScannerShape(agent, entry.Artifact)
			if !recognized {
				continue
			}
			entry.Artifact.Kind = kind
			contract, _ := ProviderContractFor(agent, entry.Artifact.StorageRoot, schema)
			observations = append(observations, ArtifactObservation{Agent: agent, Entry: entry, ScannerSchema: schema, ProviderContract: contract})
		}
	}
	return sortedTargetObservations(observations, agent), diagnostics
}

func artifactScannerShape(agent Agent, artifact Artifact) (string, ArtifactKind, bool) {
	switch agent {
	case AgentClaude:
		_, _, _, ok := claudePathFact(artifact.RelativePath)
		return "claude-v1", ArtifactTranscript, ok && artifact.StorageRoot == "claude-projects"
	case AgentCodex:
		_, ok := codexPathID(artifact.RelativePath)
		return "codex-v1", ArtifactTranscript, ok && artifact.StorageRoot == "codex-sessions"
	case AgentMuse:
		_, _, _, ok := musePathFact(artifact.RelativePath)
		return "muse-v1", ArtifactTranscript, ok && artifact.StorageRoot == "muse-sessions"
	case AgentAgy:
		if artifact.StorageRoot == "agy-conversations" {
			_, ok := agyDatabasePathID(artifact.RelativePath)
			return "agy-v1", ArtifactDatabase, ok
		}
		if artifact.StorageRoot == "agy-brain" {
			_, ok := agyTranscriptPathID(artifact.RelativePath)
			return "agy-transcript-v1", ArtifactTranscript, ok
		}
	}
	return "", "", false
}

// ValidateTargetWork performs only the already-selected targeted reads and
// returns scanner authority after stable EOF validation.
func ValidateTargetWork(runtime Runtime, agent Agent, eligible []ArtifactObservation) ([]TargetValidation, []Diagnostic) {
	if agent == AgentAgy {
		return validateAgyTargetWork(runtime, eligible)
	}
	var validations []TargetValidation
	var diagnostics []Diagnostic
	for _, observation := range eligible {
		root, ok := nativeRootByName(runtime, agent, observation.Entry.Artifact.StorageRoot)
		if !ok {
			continue
		}
		observed, err := ObserveStableArtifact(runtime, root, observation.Entry, JSONLFrameState{}, jsonRecordLimit)
		if err != nil || len(observed.FrameState.IncompleteTail) != 0 {
			diagnostics = append(diagnostics, artifactDiagnostic(DiagnosticNodeMalformed, agent, nil, observation.Entry.Artifact, "target artifact did not reach a valid stable EOF"))
			continue
		}
		var state ScannerState
		var found []Diagnostic
		switch agent {
		case AgentClaude:
			state, found, err = ValidateClaudeDelta(observation.Entry, nil, observed.Records)
		case AgentCodex:
			state, found, err = ValidateCodexDelta(observation.Entry, nil, observed.Records)
		case AgentMuse:
			state, found, err = ValidateMuseDelta(observation.Entry, nil, observed.Records)
		}
		diagnostics = append(diagnostics, found...)
		if err != nil || state.Disputed || !state.FirstRecordValidated {
			continue
		}
		fact, err := ScannerStateFact(state, []Artifact{observation.Entry.Artifact})
		if err != nil {
			continue
		}
		events, eventDiagnostics := NativeEventsFromRecords(agent, StableID("node", string(agent), fact.NativeID), observed.Records)
		diagnostics = append(diagnostics, eventDiagnostics...)
		validations = append(validations, TargetValidation{State: state, Fact: fact, Observations: []ArtifactObservation{observation}, Results: map[string]IncrementalResult{targetArtifactKey(observation.Entry.Artifact): observed}, Events: events})
	}
	return validations, diagnostics
}

// AdvanceTargetValidation consumes only bytes after the prior parser-complete
// offset for one already-selected JSONL target.
func AdvanceTargetValidation(runtime Runtime, prior TargetValidation, current []ArtifactObservation) (TargetValidation, []Diagnostic, error) {
	if prior.State.Agent == AgentAgy {
		return advanceAgyTargetValidation(runtime, prior, current)
	}
	if len(current) != 1 || len(prior.Observations) != 1 {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	observation := current[0]
	key := targetArtifactKey(observation.Entry.Artifact)
	previous, ok := prior.Results[key]
	if !ok {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	currentFingerprint := fingerprintFromEntry(observation.Entry)
	if !equalFingerprint(previous.Fingerprint, currentFingerprint) {
		if previous.Fingerprint.StableFileID != currentFingerprint.StableFileID || previous.Fingerprint.GenerationToken == "" || previous.Fingerprint.GenerationToken != currentFingerprint.GenerationToken || currentFingerprint.Size <= previous.RawObservedOffset {
			return TargetValidation{}, nil, ErrArtifactChanged
		}
		contract, ok := ProviderContractFor(observation.Agent, observation.Entry.Artifact.StorageRoot, observation.ScannerSchema)
		if !ok || contract != observation.ProviderContract {
			return TargetValidation{}, nil, ErrArtifactChanged
		}
	}
	root, ok := nativeRootByName(runtime, prior.State.Agent, observation.Entry.Artifact.StorageRoot)
	if !ok {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	observed, err := ObserveStableArtifact(runtime, root, observation.Entry, previous.FrameState, jsonRecordLimit)
	if err != nil || len(observed.FrameState.IncompleteTail) != 0 {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	state := cloneScannerState(prior.State)
	var diagnostics []Diagnostic
	switch state.Agent {
	case AgentClaude:
		state, diagnostics, err = ValidateClaudeDelta(observation.Entry, &state, observed.Records)
	case AgentCodex:
		state, diagnostics, err = ValidateCodexDelta(observation.Entry, &state, observed.Records)
	case AgentMuse:
		state, diagnostics, err = ValidateMuseDelta(observation.Entry, &state, observed.Records)
	}
	if err != nil || state.Disputed {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	fact, err := ScannerStateFact(state, []Artifact{observation.Entry.Artifact})
	if err != nil {
		return TargetValidation{}, diagnostics, err
	}
	newEvents, eventDiagnostics := NativeEventsFromRecords(state.Agent, StableID("node", string(state.Agent), state.NativeID), observed.Records)
	diagnostics = append(diagnostics, eventDiagnostics...)
	return TargetValidation{State: state, Fact: fact, Observations: []ArtifactObservation{observation}, Results: map[string]IncrementalResult{key: observed}, Events: append(append([]NativeEventFact(nil), prior.Events...), newEvents...)}, diagnostics, nil
}

func advanceAgyTargetValidation(runtime Runtime, prior TargetValidation, current []ArtifactObservation) (TargetValidation, []Diagnostic, error) {
	if len(current) != 2 {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	var database, transcript *ArtifactObservation
	for i := range current {
		if current[i].Entry.Artifact.Kind == ArtifactDatabase {
			database = &current[i]
		} else if current[i].Entry.Artifact.Kind == ArtifactTranscript {
			transcript = &current[i]
		}
	}
	if database == nil || transcript == nil || observationNativeID(AgentAgy, database.Entry.Artifact) != prior.State.NativeID || observationNativeID(AgentAgy, transcript.Entry.Artifact) != prior.State.NativeID {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	transcriptKey := targetArtifactKey(transcript.Entry.Artifact)
	previous, ok := prior.Results[transcriptKey]
	if !ok {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	currentTranscriptFingerprint := fingerprintFromEntry(transcript.Entry)
	if !equalFingerprint(previous.Fingerprint, currentTranscriptFingerprint) {
		if previous.Fingerprint.StableFileID != currentTranscriptFingerprint.StableFileID || previous.Fingerprint.GenerationToken == "" || previous.Fingerprint.GenerationToken != currentTranscriptFingerprint.GenerationToken || currentTranscriptFingerprint.Size <= previous.RawObservedOffset {
			return TargetValidation{}, nil, ErrArtifactChanged
		}
		contract, ok := ProviderContractFor(AgentAgy, transcript.Entry.Artifact.StorageRoot, transcript.ScannerSchema)
		if !ok || contract != transcript.ProviderContract {
			return TargetValidation{}, nil, ErrArtifactChanged
		}
	}
	root, ok := nativeRootByName(runtime, AgentAgy, transcript.Entry.Artifact.StorageRoot)
	if !ok {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	observed, err := ObserveStableArtifact(runtime, root, transcript.Entry, previous.FrameState, jsonRecordLimit)
	if err != nil || len(observed.FrameState.IncompleteTail) != 0 {
		return TargetValidation{}, nil, ErrArtifactChanged
	}
	state := cloneScannerState(prior.State)
	var diagnostics []Diagnostic
	priorDatabase := prior.Observations[0]
	for _, observation := range prior.Observations {
		if observation.Entry.Artifact.Kind == ArtifactDatabase {
			priorDatabase = observation
		}
	}
	if equalFingerprint(fingerprintFromEntry(priorDatabase.Entry), fingerprintFromEntry(database.Entry)) {
		applyAgyTranscriptRecords(&state, transcript.Entry.Artifact, observed.Records, &diagnostics)
	} else {
		state, diagnostics, err = ValidateAgyDelta(runtime, database.Entry, transcript.Entry, &state, observed.Records)
		if err != nil {
			return TargetValidation{}, diagnostics, err
		}
	}
	if state.Disputed {
		return TargetValidation{}, diagnostics, ErrArtifactChanged
	}
	artifacts := []Artifact{database.Entry.Artifact, transcript.Entry.Artifact}
	fact, err := ScannerStateFact(state, artifacts)
	if err != nil {
		return TargetValidation{}, diagnostics, err
	}
	newEvents, found := NativeEventsFromRecords(AgentAgy, StableID("node", string(AgentAgy), state.NativeID), observed.Records)
	diagnostics = append(diagnostics, found...)
	return TargetValidation{State: state, Fact: fact, Observations: current, Results: map[string]IncrementalResult{transcriptKey: observed}, Events: append(append([]NativeEventFact(nil), prior.Events...), newEvents...)}, diagnostics, nil
}

func validateAgyTargetWork(runtime Runtime, eligible []ArtifactObservation) ([]TargetValidation, []Diagnostic) {
	byID := map[string][]ArtifactObservation{}
	for _, observation := range eligible {
		byID[observationNativeID(AgentAgy, observation.Entry.Artifact)] = append(byID[observationNativeID(AgentAgy, observation.Entry.Artifact)], observation)
	}
	var validations []TargetValidation
	var diagnostics []Diagnostic
	for id, joined := range byID {
		var database, transcript *ArtifactObservation
		for i := range joined {
			if joined[i].Entry.Artifact.Kind == ArtifactDatabase {
				database = &joined[i]
			} else if joined[i].Entry.Artifact.Kind == ArtifactTranscript {
				transcript = &joined[i]
			}
		}
		if id == "" || database == nil || transcript == nil {
			continue
		}
		root, ok := nativeRootByName(runtime, AgentAgy, transcript.Entry.Artifact.StorageRoot)
		if !ok {
			continue
		}
		observed, err := ObserveStableArtifact(runtime, root, transcript.Entry, JSONLFrameState{}, jsonRecordLimit)
		if err != nil || len(observed.FrameState.IncompleteTail) != 0 {
			continue
		}
		state, found, err := ValidateAgyDelta(runtime, database.Entry, transcript.Entry, nil, observed.Records)
		diagnostics = append(diagnostics, found...)
		if err != nil || state.Disputed || !state.FirstRecordValidated {
			continue
		}
		artifacts := []Artifact{database.Entry.Artifact, transcript.Entry.Artifact}
		fact, err := ScannerStateFact(state, artifacts)
		if err != nil {
			continue
		}
		events, eventDiagnostics := NativeEventsFromRecords(AgentAgy, StableID("node", string(AgentAgy), fact.NativeID), observed.Records)
		diagnostics = append(diagnostics, eventDiagnostics...)
		validations = append(validations, TargetValidation{State: state, Fact: fact, Observations: joined, Results: map[string]IncrementalResult{targetArtifactKey(transcript.Entry.Artifact): observed}, Events: events})
	}
	return validations, diagnostics
}

func nativeRootByName(runtime Runtime, agent Agent, name string) (StorageRoot, bool) {
	for _, root := range runtime.NativeRoots(agent) {
		if root.Name == name {
			return root, true
		}
	}
	return StorageRoot{}, false
}

// ObserveStableArtifact reads from the supplied parser state through a stable
// EOF, following append growth but rejecting replacement, truncation, and
// same-size mutation before returning any records as usable evidence.
func ObserveStableArtifact(runtime Runtime, root StorageRoot, initial FileEntry, frame JSONLFrameState, recordLimit int64) (IncrementalResult, error) {
	result := IncrementalResult{FrameState: JSONLFrameState{ParserCompleteOffset: frame.ParserCompleteOffset, IncompleteTail: append([]byte(nil), frame.IncompleteTail...)}}
	if runtime == nil || initial.Artifact.StorageRoot != root.Name || frame.ParserCompleteOffset < 0 || int64(len(frame.IncompleteTail)) > initial.Size-frame.ParserCompleteOffset {
		result.Disputed = true
		return result, ErrArtifactChanged
	}
	current := initial
	readOffset := frame.ParserCompleteOffset + int64(len(frame.IncompleteTail))
	var records []FramedJSONLRecord
	for {
		for readOffset < current.Size {
			limit := min(readChunkSize, current.Size-readOffset)
			raw, _, err := runtime.ReadAt(initial.Artifact, readOffset, limit)
			if err != nil {
				return result, fmt.Errorf("read incremental session artifact: %w", err)
			}
			if len(raw) == 0 {
				result.Disputed = true
				return result, ErrArtifactChanged
			}
			readOffset += int64(len(raw))
			framed, next, err := FrameJSONLSuffix(result.FrameState, raw, recordLimit)
			if err != nil {
				return result, err
			}
			result.FrameState = next
			records = append(records, framed...)
		}

		observed, err := resampleArtifact(runtime, root, initial.Artifact)
		if err != nil {
			if errors.Is(err, ErrArtifactChanged) {
				result.Disputed = true
			}
			return result, err
		}
		if observed.StableFileID != initial.StableFileID || observed.GenerationToken != initial.GenerationToken || observed.Size < current.Size {
			result.Disputed = true
			return result, ErrArtifactChanged
		}
		if observed.Size == current.Size {
			if observed.MutationToken != current.MutationToken || !equalOptionalTime(observed.BirthTime, current.BirthTime) || !equalOptionalTime(observed.ModTime, current.ModTime) {
				result.Disputed = true
				return result, ErrArtifactChanged
			}
			result.Fingerprint = fingerprintFromEntry(observed)
			result.RawObservedOffset = observed.Size
			result.Records = records
			return result, nil
		}
		if initial.GenerationToken == "" {
			result.Disputed = true
			return result, ErrArtifactChanged
		}
		current = observed
	}
}

func resampleArtifact(runtime Runtime, root StorageRoot, artifact Artifact) (FileEntry, error) {
	files, err := runtime.ListFiles(root)
	if err != nil {
		var listingIssues *ListingIssuesError
		if !errors.As(err, &listingIssues) {
			return FileEntry{}, fmt.Errorf("resample session artifact metadata: %w", err)
		}
	}
	for _, entry := range files {
		if entry.Artifact.StorageRoot == artifact.StorageRoot && entry.Artifact.RelativePath == artifact.RelativePath {
			entry.Artifact.Kind = artifact.Kind
			return entry, nil
		}
	}
	return FileEntry{}, ErrArtifactChanged
}
