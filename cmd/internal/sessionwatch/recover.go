package sessionwatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type RecoveryOptions struct {
	Agent, Tag, ScopeKey, Home, DataDir string
	Apply                               bool
}

type RecoveryResult struct {
	ScopeKey      string                        `json:"scope_key"`
	Tag           string                        `json:"tag"`
	Agent         string                        `json:"agent"`
	LaunchOrdinal uint64                        `json:"launch_ordinal"`
	NativeID      string                        `json:"native_id,omitempty"`
	Status        string                        `json:"status"`
	Reason        string                        `json:"reason"`
	Confidence    string                        `json:"confidence"`
	Applied       bool                          `json:"applied"`
	Diagnostics   []sessioninventory.Diagnostic `json:"diagnostics,omitempty"`
}

type RecoveryConfirmer interface {
	ConfirmIfCurrent(string, sessionledger.Owner, uint64, string, sessionledger.ConfirmationReason, *sessionledger.AuthorizationProof) (sessionledger.Record, error)
}

// Recover previews a current-launch correlation without touching config or
// catalog. Applying publishes only the guarded ledger confirmation.
func Recover(opts RecoveryOptions, rt Runtime, store RecoveryConfirmer) (RecoveryResult, error) {
	result := RecoveryResult{ScopeKey: opts.ScopeKey, Tag: opts.Tag, Agent: opts.Agent, Status: "unavailable", Confidence: "none"}
	if !SupportsAgent(opts.Agent) || opts.Tag == "" || opts.ScopeKey == "" || opts.DataDir == "" {
		return result, errors.New("repair requires agent, tag, scope key and data directory")
	}
	paths, err := artifactpath.ResolveScoped(opts.DataDir, opts.Tag)
	if err != nil {
		return result, err
	}
	owner := sessionledger.Owner{ScopeKey: opts.ScopeKey, Tag: opts.Tag, Agent: opts.Agent}
	current, ok, err := readCurrentLaunch(rt, paths.Ledger(), owner)
	if err != nil {
		return result, err
	}
	if !ok {
		result.Reason = "no launch for this owner"
		return result, nil
	}
	result.LaunchOrdinal = current.Launch.Ordinal
	if current.Conflict {
		result.Status = "ambiguous"
		result.Reason = "conflicting saved bindings"
		return result, nil
	}
	if current.Binding != nil {
		result.Status = "already-bound"
		result.NativeID = current.Binding.RootNativeID
		result.Confidence = "recorded"
		result.Reason = "current launch already has a binding"
		return result, nil
	}
	if current.Launch.Version < 2 || (current.Launch.Version >= 3 && !current.Launch.BaselineComplete) {
		result.Reason = "launch has no complete observation baseline"
		return result, nil
	}
	native := rt.NativeRuntime(opts.Home, opts.DataDir)
	inventory, events, proofs := incrementalWatcherInventory(native, sessioninventory.NewIncrementalInventory(native, sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}), sessioninventory.Agent(opts.Agent), current.Launch, map[string]sessioninventory.TargetValidation{})
	result.Diagnostics = inventory.Diagnostics
	log, diagnostics := readPairLog(rt, paths.Log(), sessioninventory.Agent(opts.Agent))
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	rounds, diagnostics := sessioninventory.RoundsAfterLaunch(inventory, opts.ScopeKey, opts.Tag, sessioninventory.Agent(opts.Agent), log, current.Launch, events)
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	root, status := recoveryRoot(inventory, owner, rounds)
	if root == "" {
		result.Status = "unresolved"
		result.Reason = "no unique current-launch correlation"
		if status == sessioninventory.BindingAmbiguous {
			result.Status = "ambiguous"
		}
		return result, nil
	}
	proof, ok := proofs[root]
	if !ok {
		result.Reason = "correlated root has no usable transcript proof"
		return result, nil
	}
	result.NativeID = proof.RootNativeID
	result.Status = "repairable"
	result.Reason = "unique current-launch prompt and progress correlation"
	result.Confidence = "correlation"
	if !opts.Apply {
		return result, nil
	}
	record, err := store.ConfirmIfCurrent(paths.Ledger(), owner, current.Launch.Ordinal, proof.RootNativeID, sessionledger.ConfirmationCorrelation, &proof)
	if appender, ok := store.(LedgerAppender); ok {
		err = reconcileLedgerAppend(appender, paths.Ledger(), record, err)
	}
	if err != nil {
		return result, err
	}
	result.Applied = record.Ordinal != 0
	result.Status = "repaired"
	return result, nil
}

func RunRepairCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	usage := "usage: pair session-repair <agent> <tag> --scope-key <scope> [--apply]"
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	if len(args) < 2 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	opts := RecoveryOptions{Agent: args[0], Tag: args[1], Home: getenv("HOME"), DataDir: getenv("PAIR_DATA_DIR")}
	if opts.DataDir == "" {
		opts.DataDir = adapt.DataDir()
	}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--apply":
			opts.Apply = true
		case "--scope-key":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, usage)
				return 2
			}
			i++
			opts.ScopeKey = args[i]
		default:
			fmt.Fprintln(stderr, usage)
			return 2
		}
	}
	if opts.ScopeKey == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if opts.Apply {
		selectedEnv := func(key string) string {
			switch key {
			case "PAIR_TAG":
				return opts.Tag
			case "PAIR_DATA_DIR":
				return opts.DataDir
			case "PAIR_SCOPE_KEY":
				return opts.ScopeKey
			case "PAIR_RETENTION_START_ID":
				return ""
			}
			return getenv(key)
		}
		lease, err := storagegc.AcquireSelectedProcess(context.Background(), selectedEnv, "session-repair")
		if err != nil {
			fmt.Fprintln(stderr, "session-repair: retention:", err)
			return 1
		}
		defer lease.Close()
	}
	result, err := Recover(opts, OSRuntime{}, sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}})
	if err != nil {
		fmt.Fprintln(stderr, "session-repair:", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func recoveryRoot(inventory sessioninventory.Inventory, owner sessionledger.Owner, rounds []sessioninventory.RoundObservation) (string, sessioninventory.BindingStatus) {
	resolved := sessioninventory.ResolveBindings(inventory, []sessioninventory.BindingInput{{ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: sessioninventory.Agent(owner.Agent), LaunchPresent: true, LiveRounds: rounds}})
	if len(resolved.Bindings) != 1 {
		return "", sessioninventory.BindingUnbound
	}
	binding := resolved.Bindings[0]
	if binding.Status == sessioninventory.BindingProvisional && binding.RootNodeID != nil {
		return *binding.RootNodeID, binding.Status
	}
	return "", binding.Status
}
