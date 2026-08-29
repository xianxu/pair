package sessionwatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

var ErrResumeUnauthorized = errors.New("resume native session is not scanner-authorized")

type LedgerAppender interface {
	Append(string, sessionledger.Record) (sessionledger.Record, error)
	AppendBindingIfCurrent(string, sessionledger.Owner, uint64, string) (sessionledger.Record, error)
	AppendBindingProofIfCurrent(string, sessionledger.Owner, uint64, sessionledger.AuthorizationProof) (sessionledger.Record, error)
	Reconcile(string, sessionledger.Record) error
}

type PrepareLaunchInput struct {
	Owner              sessionledger.Owner
	LedgerPath         string
	PairLogOffset      uint64
	Inventory          sessioninventory.Inventory
	NativeEvents       []sessioninventory.NativeEventFact
	ResumeNativeID     string
	ArtifactBoundaries []sessionledger.LaunchArtifactBoundary
	ResumeProof        *sessionledger.AuthorizationProof
}

type PreparedLaunch struct {
	Launch  sessionledger.Record
	Binding *sessionledger.Record
}

type ConfigWriter func(ConfigPayload) error

// PrepareLaunch durably delimits a generation before agent input is possible.
// An explicit resume may join immediately, but only through a root authorized
// by the same scanner inventory used by the watcher.
func PrepareLaunch(input PrepareLaunchInput, store LedgerAppender) (PreparedLaunch, error) {
	if input.ArtifactBoundaries != nil {
		return prepareLaunchV2(input, store)
	}
	agent := sessioninventory.Agent(input.Owner.Agent)
	rootNativeIDs := map[string]string{}
	maxPositions := map[string]uint64{}
	for _, forest := range input.Inventory.Forests {
		if forest.Agent != agent {
			continue
		}
		for _, root := range forest.Roots {
			rootNativeIDs[root.StableID] = root.NativeID
			maxPositions[root.StableID] = 0
		}
	}
	if input.ResumeNativeID != "" && !containsNativeID(rootNativeIDs, input.ResumeNativeID) {
		return PreparedLaunch{}, ErrResumeUnauthorized
	}
	for _, event := range input.NativeEvents {
		if event.Agent != "" && event.Agent != agent {
			continue
		}
		if _, authorized := rootNativeIDs[event.RootNodeID]; authorized && event.Position > maxPositions[event.RootNodeID] {
			maxPositions[event.RootNodeID] = event.Position
		}
	}
	watermarks := make([]sessionledger.NativeWatermark, 0, len(rootNativeIDs))
	for rootNodeID, nativeID := range rootNativeIDs {
		watermarks = append(watermarks, sessionledger.NativeWatermark{RootNativeID: nativeID, EventPosition: maxPositions[rootNodeID]})
	}
	slices.SortFunc(watermarks, func(a, b sessionledger.NativeWatermark) int {
		if a.RootNativeID < b.RootNativeID {
			return -1
		}
		if a.RootNativeID > b.RootNativeID {
			return 1
		}
		return 0
	})
	launch, err := store.Append(input.LedgerPath, sessionledger.Record{
		Version: 1, Kind: sessionledger.RecordLaunch,
		ScopeKey: input.Owner.ScopeKey, Tag: input.Owner.Tag, Agent: input.Owner.Agent,
		PairLogOffset: input.PairLogOffset, NativeWatermarks: watermarks,
	})
	prepared := PreparedLaunch{}
	if launch.Ordinal != 0 {
		prepared.Launch = launch
	}
	err = reconcileLedgerAppend(store, input.LedgerPath, launch, err)
	var warning error
	if err != nil {
		if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendCommitted {
			return prepared, err
		}
		warning = err
	}
	prepared.Launch = launch
	if input.ResumeNativeID == "" {
		return prepared, warning
	}
	binding, err := store.AppendBindingIfCurrent(input.LedgerPath, input.Owner, launch.Ordinal, input.ResumeNativeID)
	err = reconcileLedgerAppend(store, input.LedgerPath, binding, err)
	if err != nil {
		if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendCommitted {
			return prepared, errors.Join(warning, err)
		}
		warning = errors.Join(warning, err)
	}
	prepared.Binding = &binding
	return prepared, warning
}

func prepareLaunchV2(input PrepareLaunchInput, store LedgerAppender) (PreparedLaunch, error) {
	if input.ResumeNativeID != "" && (input.ResumeProof == nil || input.ResumeProof.RootNativeID != input.ResumeNativeID) {
		return PreparedLaunch{}, ErrResumeUnauthorized
	}
	launch, err := store.Append(input.LedgerPath, sessionledger.Record{
		Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: input.Owner.ScopeKey, Tag: input.Owner.Tag, Agent: input.Owner.Agent,
		PairLogOffset: input.PairLogOffset, LaunchArtifactBoundaries: append([]sessionledger.LaunchArtifactBoundary(nil), input.ArtifactBoundaries...),
	})
	prepared := PreparedLaunch{}
	if launch.Ordinal != 0 {
		prepared.Launch = launch
	}
	err = reconcileLedgerAppend(store, input.LedgerPath, launch, err)
	var warning error
	if err != nil {
		if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendCommitted {
			return prepared, err
		}
		warning = err
	}
	prepared.Launch = launch
	if input.ResumeProof == nil {
		return prepared, warning
	}
	binding, err := store.AppendBindingProofIfCurrent(input.LedgerPath, input.Owner, launch.Ordinal, *input.ResumeProof)
	err = reconcileLedgerAppend(store, input.LedgerPath, binding, err)
	if err != nil {
		if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendCommitted {
			return prepared, errors.Join(warning, err)
		}
		warning = errors.Join(warning, err)
	}
	prepared.Binding = &binding
	return prepared, warning
}

func reconcileLedgerAppend(store LedgerAppender, path string, record sessionledger.Record, err error) error {
	if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendIndeterminate {
		return err
	}
	reconcileErr := store.Reconcile(path, record)
	if reconcileErr == nil || sessionledger.AppendOutcomeOf(reconcileErr) == sessionledger.AppendCommitted {
		return reconcileErr
	}
	return errors.Join(err, reconcileErr)
}

func containsNativeID(byRoot map[string]string, nativeID string) bool {
	for _, candidate := range byRoot {
		if candidate == nativeID {
			return true
		}
	}
	return false
}

// ObserveAndPersist resolves completed live rounds, persists only a unique
// scanner-authorized root against the still-current launch, then refreshes the
// config compatibility cache. Cache failure cannot weaken the durable binding.
func ObserveAndPersist(input ObserveInput, store LedgerAppender, writeConfig ConfigWriter) (sessioninventory.Inventory, error) {
	agent := sessioninventory.Agent(input.Owner.Agent)
	bindingInput := sessioninventory.BindingInput{
		ScopeKey: input.Owner.ScopeKey, Tag: input.Owner.Tag, Agent: agent,
		LaunchPresent: true, LiveRounds: input.LiveRounds,
	}
	resolved := sessioninventory.ResolveBindings(input.Inventory, []sessioninventory.BindingInput{bindingInput})
	if len(resolved.Bindings) != 1 {
		return resolved, nil
	}
	binding := resolved.Bindings[0]
	if binding.Status != sessioninventory.BindingProvisional || binding.RootNodeID == nil {
		return resolved, nil
	}
	nativeID := nativeIDForRoot(input.Inventory.Forests, agent, *binding.RootNodeID)
	if nativeID == "" {
		return resolved, nil
	}
	var appended sessionledger.Record
	var err error
	if proof, ok := input.Proofs[*binding.RootNodeID]; ok {
		appended, err = store.AppendBindingProofIfCurrent(input.LedgerPath, input.Owner, input.LaunchOrdinal, proof)
	} else {
		appended, err = store.AppendBindingIfCurrent(input.LedgerPath, input.Owner, input.LaunchOrdinal, nativeID)
	}
	err = reconcileLedgerAppend(store, input.LedgerPath, appended, err)
	if err != nil && sessionledger.AppendOutcomeOf(err) != sessionledger.AppendCommitted {
		return resolved, err
	}
	warning := err
	bindingInput.LedgerRootNodeID = *binding.RootNodeID
	resolved = sessioninventory.ResolveBindings(input.Inventory, []sessioninventory.BindingInput{bindingInput})
	if writeConfig != nil {
		if err := writeConfig(ConfigPayload{Agent: input.Owner.Agent, Args: StripResumeArgs(input.Owner.Agent, input.Args), SessionID: nativeID}); err != nil {
			resolved.Diagnostics = append(resolved.Diagnostics, sessioninventory.Diagnostic{
				Code: sessioninventory.DiagnosticBindingStale, Agent: agent,
				Detail: "durable binding established but config cache refresh failed",
			})
			resolved = sessioninventory.SortInventory(resolved)
		}
	}
	return resolved, warning
}

func nativeIDForRoot(forests []sessioninventory.Forest, agent sessioninventory.Agent, rootNodeID string) string {
	for _, forest := range forests {
		if forest.Agent != agent {
			continue
		}
		for _, root := range forest.Roots {
			if root.StableID == rootNodeID {
				return root.NativeID
			}
		}
	}
	return ""
}

// PrepareOSLaunch is the shared thin IO shell used by both the outer launcher
// and an in-pane fresh-agent restart before either can accept new input.
func PrepareOSLaunch(home, dataDir string, owner sessionledger.Owner, resumeNativeID string) (PreparedLaunch, error) {
	paths, err := artifactpath.ResolveScoped(dataDir, owner.Tag)
	if err != nil {
		return PreparedLaunch{}, err
	}
	pairLogOffset := uint64(0)
	if info, statErr := os.Stat(paths.Log()); statErr == nil {
		pairLogOffset = uint64(info.Size())
	} else if !os.IsNotExist(statErr) {
		return PreparedLaunch{}, statErr
	}
	return prepareRuntimeLaunch(paths.Ledger(), owner, resumeNativeID, pairLogOffset, sessioninventory.NewOSRuntime(home, dataDir), sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}})
}

// PrepareRuntimeLaunch is the injected metadata-only launch seam used by the
// stateful corpus tests.
func PrepareRuntimeLaunch(dataDir string, owner sessionledger.Owner, resumeNativeID string, pairLogOffset uint64, nativeRuntime sessioninventory.Runtime, store LedgerAppender) (PreparedLaunch, error) {
	paths, err := artifactpath.ResolveScoped(dataDir, owner.Tag)
	if err != nil {
		return PreparedLaunch{}, err
	}
	return prepareRuntimeLaunch(paths.Ledger(), owner, resumeNativeID, pairLogOffset, nativeRuntime, store)
}

func prepareRuntimeLaunch(ledgerPath string, owner sessionledger.Owner, resumeNativeID string, pairLogOffset uint64, nativeRuntime sessioninventory.Runtime, store LedgerAppender) (PreparedLaunch, error) {
	agent := sessioninventory.Agent(owner.Agent)
	observations, diagnostics := sessioninventory.ObserveAgentMetadata(nativeRuntime, agent)
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == sessioninventory.DiagnosticStorageUnreadable {
			return PreparedLaunch{}, fmt.Errorf("capture native launch metadata: %s", diagnostic.Detail)
		}
	}
	boundaries := make([]sessionledger.LaunchArtifactBoundary, 0, len(observations))
	for _, observation := range observations {
		entry := observation.Entry
		boundaries = append(boundaries, sessionledger.LaunchArtifactBoundary{
			StorageRoot: entry.Artifact.StorageRoot, RelativePath: entry.Artifact.RelativePath, StableFileID: string(entry.StableFileID),
			GenerationToken: string(entry.GenerationToken), MutationToken: string(entry.MutationToken), RawSize: entry.Size,
		})
	}
	var proof *sessionledger.AuthorizationProof
	if resumeNativeID != "" {
		selection := sessioninventory.SelectTargetWork(sessioninventory.TargetRequest{Mode: sessioninventory.TargetExplicitResume, Agent: agent, NativeID: resumeNativeID}, observations)
		validations, _ := sessioninventory.ValidateTargetWork(nativeRuntime, agent, selection.Eligible)
		for _, validation := range validations {
			if validation.State.NativeID == resumeNativeID && validation.State.Role == sessioninventory.RoleRoot {
				candidate, err := authorizationProof(validation)
				if err != nil {
					return PreparedLaunch{}, err
				}
				proof = &candidate
				break
			}
		}
		if proof == nil {
			return PreparedLaunch{}, ErrResumeUnauthorized
		}
	}
	return PrepareLaunch(PrepareLaunchInput{Owner: owner, LedgerPath: ledgerPath, PairLogOffset: pairLogOffset, ArtifactBoundaries: boundaries, ResumeNativeID: resumeNativeID, ResumeProof: proof}, store)
}

func authorizationProof(validation sessioninventory.TargetValidation) (sessionledger.AuthorizationProof, error) {
	state, err := json.Marshal(validation.State)
	if err != nil {
		return sessionledger.AuthorizationProof{}, err
	}
	proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: validation.State.NativeID, ScannerSchema: validation.State.ScannerSchema, ScannerState: state}
	for _, observation := range validation.Observations {
		entry := observation.Entry
		fingerprint := sessioninventory.ArtifactFingerprint{StableFileID: entry.StableFileID, GenerationToken: entry.GenerationToken, MutationToken: entry.MutationToken, Size: entry.Size}
		parserOffset := entry.Size
		if result, ok := validation.Results[entry.Artifact.StorageRoot+"\x00"+entry.Artifact.RelativePath]; ok {
			fingerprint = result.Fingerprint
			parserOffset = result.FrameState.ParserCompleteOffset
		}
		proof.Artifacts = append(proof.Artifacts, sessionledger.ArtifactProof{
			StorageRoot: entry.Artifact.StorageRoot, RelativePath: entry.Artifact.RelativePath, StableFileID: string(fingerprint.StableFileID),
			GenerationToken: string(fingerprint.GenerationToken), MutationToken: string(fingerprint.MutationToken), Size: fingerprint.Size, ParserCompleteOffset: parserOffset,
		})
	}
	if err := sessionledger.ValidateAuthorizationProof(proof, proof.RootNativeID); err != nil {
		return sessionledger.AuthorizationProof{}, err
	}
	return proof, nil
}

func fatalLaunchBaselineDiagnostic(diagnostic sessioninventory.Diagnostic) bool {
	return diagnostic.Code == sessioninventory.DiagnosticTurnUnusable &&
		(strings.Contains(diagnostic.Detail, "unreadable") || strings.Contains(diagnostic.Detail, "exactly one"))
}
