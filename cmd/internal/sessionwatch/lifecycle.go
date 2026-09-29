package sessionwatch

import (
	"encoding/json"
	"errors"
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
	proof, hasProof := input.Proofs[*binding.RootNodeID]
	if input.RequireProof && !hasProof {
		resolved.Diagnostics = append(resolved.Diagnostics, sessioninventory.Diagnostic{
			Code: sessioninventory.DiagnosticBindingStale, Agent: agent,
			Detail: "proof-bearing binding authority is unavailable",
		})
		return sessioninventory.SortInventory(resolved), nil
	}
	var appended sessionledger.Record
	var err error
	if input.ConfirmationReason != "" {
		confirmer, ok := store.(interface {
			ConfirmIfCurrent(string, sessionledger.Owner, uint64, string, string, *sessionledger.AuthorizationProof) (sessionledger.Record, error)
		})
		if !ok {
			return resolved, errors.New("ledger does not support confirmation")
		}
		var optionalProof *sessionledger.AuthorizationProof
		if hasProof {
			optionalProof = &proof
		}
		appended, err = confirmer.ConfirmIfCurrent(input.LedgerPath, input.Owner, input.LaunchOrdinal, nativeID, input.ConfirmationReason, optionalProof)
	} else if hasProof {
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
		write := func() error {
			return writeConfig(ConfigPayload{Agent: input.Owner.Agent, Args: StripResumeArgs(input.Owner.Agent, input.Args), SessionID: nativeID})
		}
		var configErr error
		if input.ConfirmationReason != "" {
			if guarded, ok := store.(interface {
				WithCurrentConfirmation(string, sessionledger.Owner, uint64, string, func() error) error
			}); ok {
				configErr = guarded.WithCurrentConfirmation(input.LedgerPath, input.Owner, input.LaunchOrdinal, nativeID, write)
			} else {
				configErr = errors.New("unguarded compatibility publication refused")
			}
		} else {
			configErr = write()
		}
		if configErr != nil {
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
	origin := sessionledger.RequestOrigin("")
	if resumeNativeID != "" {
		origin = sessionledger.RequestOriginResume
	}
	return PrepareOSLaunchRequest(home, dataDir, owner, resumeNativeID, origin)
}

func PrepareOSLaunchRequest(home, dataDir string, owner sessionledger.Owner, requestedNativeID string, origin sessionledger.RequestOrigin) (PreparedLaunch, error) {
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
	return prepareRuntimeLaunchRequest(paths.Ledger(), owner, requestedNativeID, origin, pairLogOffset, sessioninventory.NewOSRuntime(home, dataDir), sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}})
}

// PrepareRuntimeLaunch is the injected metadata-only launch seam used by the
// stateful corpus tests.
func PrepareRuntimeLaunch(dataDir string, owner sessionledger.Owner, resumeNativeID string, pairLogOffset uint64, nativeRuntime sessioninventory.Runtime, store LedgerAppender) (PreparedLaunch, error) {
	origin := sessionledger.RequestOrigin("")
	if resumeNativeID != "" {
		origin = sessionledger.RequestOriginResume
	}
	return PrepareRuntimeLaunchRequest(dataDir, owner, resumeNativeID, origin, pairLogOffset, nativeRuntime, store)
}

func PrepareRuntimeLaunchRequest(dataDir string, owner sessionledger.Owner, requestedNativeID string, origin sessionledger.RequestOrigin, pairLogOffset uint64, nativeRuntime sessioninventory.Runtime, store LedgerAppender) (PreparedLaunch, error) {
	paths, err := artifactpath.ResolveScoped(dataDir, owner.Tag)
	if err != nil {
		return PreparedLaunch{}, err
	}
	return prepareRuntimeLaunchRequest(paths.Ledger(), owner, requestedNativeID, origin, pairLogOffset, nativeRuntime, store)
}

func prepareRuntimeLaunchRequest(ledgerPath string, owner sessionledger.Owner, requestedNativeID string, origin sessionledger.RequestOrigin, pairLogOffset uint64, nativeRuntime sessioninventory.Runtime, store LedgerAppender) (PreparedLaunch, error) {
	snapshot := sessioninventory.NewIncrementalInventory(nativeRuntime, sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}).Observe(sessioninventory.Agent(owner.Agent))
	boundaries, complete := launchBoundaries(snapshot)
	launch, err := store.Append(ledgerPath, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent, PairLogOffset: pairLogOffset, LaunchArtifactBoundaries: boundaries, RequestedNativeID: requestedNativeID, RequestOrigin: origin, BaselineComplete: complete})
	err = reconcileLedgerAppend(store, ledgerPath, launch, err)
	return PreparedLaunch{Launch: launch}, err
}

func launchBoundaries(snapshot sessioninventory.IncrementalSnapshot) ([]sessionledger.LaunchArtifactBoundary, bool) {
	complete := true
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Code == sessioninventory.DiagnosticStorageUnreadable || diagnostic.Code == sessioninventory.DiagnosticArtifactPathInvalid {
			complete = false
		}
	}
	boundaries := make([]sessionledger.LaunchArtifactBoundary, 0, len(snapshot.Observations))
	for _, observation := range snapshot.Observations {
		entry := observation.Entry
		boundaries = append(boundaries, sessionledger.LaunchArtifactBoundary{StorageRoot: entry.Artifact.StorageRoot, RelativePath: entry.Artifact.RelativePath, StableFileID: string(entry.StableFileID), GenerationToken: string(entry.GenerationToken), MutationToken: string(entry.MutationToken), RawSize: entry.Size})
	}
	return boundaries, complete
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
