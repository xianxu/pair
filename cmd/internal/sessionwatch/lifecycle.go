package sessionwatch

import (
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
	Reconcile(string, sessionledger.Record) error
}

type PrepareLaunchInput struct {
	Owner          sessionledger.Owner
	LedgerPath     string
	PairLogOffset  uint64
	Inventory      sessioninventory.Inventory
	NativeEvents   []sessioninventory.NativeEventFact
	ResumeNativeID string
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
	appended, err := store.AppendBindingIfCurrent(input.LedgerPath, input.Owner, input.LaunchOrdinal, nativeID)
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
	nativeRuntime := sessioninventory.NewOSRuntime(home, dataDir)
	agent := sessioninventory.Agent(owner.Agent)
	inventory := sessioninventory.InventoryWithRuntime(nativeRuntime, sessioninventory.ScannerForAgent(agent))
	events, diagnostics := sessioninventory.NativeEventsWithRuntime(nativeRuntime, inventory, agent)
	for _, diagnostic := range diagnostics {
		if fatalLaunchBaselineDiagnostic(diagnostic) {
			return PreparedLaunch{}, fmt.Errorf("capture native launch baseline: %s", diagnostic.Detail)
		}
	}
	return PrepareLaunch(PrepareLaunchInput{
		Owner: owner, LedgerPath: paths.Ledger(), PairLogOffset: pairLogOffset,
		Inventory: inventory, NativeEvents: events, ResumeNativeID: resumeNativeID,
	}, sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}})
}

func fatalLaunchBaselineDiagnostic(diagnostic sessioninventory.Diagnostic) bool {
	return diagnostic.Code == sessioninventory.DiagnosticTurnUnusable &&
		(strings.Contains(diagnostic.Detail, "unreadable") || strings.Contains(diagnostic.Detail, "exactly one"))
}
