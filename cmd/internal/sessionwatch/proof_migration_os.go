package sessionwatch

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// MigrateProoflessBindings is the bounded post-upgrade lifecycle. It scans
// Pair's small ledger set, then validates only each ledger-named native root;
// it never inventories transcript bodies to discover an owner.
func MigrateProoflessBindings(home, dataDir string) error {
	scope, err := artifactpath.ResolveSelectedScope(dataDir)
	if err != nil {
		return err
	}
	globs := scope.HistoryGlobs()
	if len(globs) < 3 {
		return errors.New("session ledger history glob is unavailable")
	}
	paths, err := filepath.Glob(globs[2])
	if err != nil {
		return err
	}
	nativeRuntime := sessioninventory.NewOSRuntime(home, dataDir)
	store := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	var migrationErr error
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			migrationErr = errors.Join(migrationErr, readErr)
			continue
		}
		parsed := sessionledger.ParseLedger(raw)
		owners := map[sessionledger.Owner]bool{}
		for _, record := range parsed.Records {
			if record.Kind == sessionledger.RecordLaunch {
				owners[sessionledger.Owner{ScopeKey: record.ScopeKey, Tag: record.Tag, Agent: record.Agent}] = true
			}
		}
		for owner := range owners {
			current, ok := sessionledger.CurrentLaunch(parsed.Records, owner)
			if !ok || current.Conflict || current.Binding == nil || current.Binding.AuthorizationProof != nil {
				continue
			}
			proof, proofErr := proofForNamedRoot(nativeRuntime, sessioninventory.Agent(owner.Agent), current.Binding.RootNativeID)
			if proofErr != nil {
				migrationErr = errors.Join(migrationErr, proofErr)
				continue
			}
			if _, appendErr := store.AppendBindingProofIfCurrent(path, owner, current.Launch.Ordinal, proof); appendErr != nil && !errors.Is(appendErr, sessionledger.ErrStaleLaunch) {
				migrationErr = errors.Join(migrationErr, appendErr)
			}
		}
	}
	return migrationErr
}

func proofForNamedRoot(runtime sessioninventory.Runtime, agent sessioninventory.Agent, nativeID string) (sessionledger.AuthorizationProof, error) {
	inventory := sessioninventory.NewIncrementalInventory(runtime, sessioninventory.Catalog{Version: sessioninventory.CatalogVersion})
	snapshot := inventory.Observe(agent)
	selected := inventory.Select(sessioninventory.TargetRequest{Mode: sessioninventory.TargetExplicitResume, Agent: agent, NativeID: nativeID}, snapshot)
	if selected.Unavailable {
		return sessionledger.AuthorizationProof{}, sessioninventory.ErrArtifactChanged
	}
	validations, _ := sessioninventory.ValidateTargetWork(runtime, agent, selected.Eligible)
	for _, validation := range validations {
		if validation.State.NativeID == nativeID && validation.State.Role == sessioninventory.RoleRoot {
			return authorizationProof(validation)
		}
	}
	return sessionledger.AuthorizationProof{}, sessioninventory.ErrArtifactChanged
}
