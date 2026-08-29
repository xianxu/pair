package sessionwatch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type Options struct {
	Agent         string
	Tag           string
	ScopeKey      string
	LaunchOrdinal uint64
	Cwd           string
	RepoRoot      string
	RepoName      string
	Args          []string
	Home          string
	DataDir       string
	PIDWait       time.Duration
	Timeout       time.Duration
	Poll          time.Duration
	SlowPoll      time.Duration
	PIDNotBefore  time.Time
}

// Runtime keeps only watcher scheduling, Pair cache IO, and access to the two
// shared production seams. Native enumeration belongs to sessioninventory;
// ledger mutation belongs to sessionledger.
type Runtime interface {
	Now() time.Time
	Sleep(time.Duration)
	ReadFile(path string) ([]byte, error)
	ModTime(path string) (time.Time, error)
	ProcessIdentity(pid string) string
	AtomicWrite(path string, data []byte) error
	Log(outcome adapt.Outcome, detail string)
	NativeRuntime(home, dataDir string) sessioninventory.Runtime
	LedgerAppender() LedgerAppender
	CatalogStore() sessioninventory.CatalogStore
	ProofMigrator() *sessioninventory.ProofMigrator
}

// Run monitors the complete post-launch suffix. It never chooses a first or
// newest native artifact: only a unique exact completed causal round may bind.
func Run(opts Options, rt Runtime) error {
	agent := sessioninventory.Agent(opts.Agent)
	if !SupportsAgent(opts.Agent) || opts.Tag == "" || opts.ScopeKey == "" || opts.DataDir == "" || opts.LaunchOrdinal == 0 {
		return nil
	}
	paths, err := artifactpath.ResolveScoped(opts.DataDir, opts.Tag)
	if err != nil {
		return err
	}
	configPath, err := paths.ConfigChecked(opts.Agent)
	if err != nil {
		return err
	}
	applyWatcherDefaults(&opts)
	watchStart := rt.Now()
	rootPID, rootIdentity := waitForPairProcess(paths.AgentPID(), opts, watchStart, rt)
	corroborationPID := rootPID
	if rootIdentity == "" {
		corroborationPID = ""
	}
	owner := sessionledger.Owner{ScopeKey: opts.ScopeKey, Tag: opts.Tag, Agent: opts.Agent}
	nativeRuntime := rt.NativeRuntime(opts.Home, opts.DataDir)
	trackedTargets := map[string]sessioninventory.TargetValidation{}
	deadline := watchStart.Add(opts.Timeout)
	for {
		if rootIdentity != "" && rt.ProcessIdentity(rootPID) != rootIdentity {
			rt.Log(adapt.NearMiss, "process identity changed before completed round")
			return nil
		}
		current, ok, err := readCurrentLaunch(rt, paths.Ledger(), owner)
		if err != nil {
			return err
		}
		if !ok || current.Launch.Ordinal != opts.LaunchOrdinal {
			return sessionledger.ErrStaleLaunch
		}
		if current.Binding != nil && current.Binding.AuthorizationProof == nil {
			if err := migrateProoflessBinding(rt, nativeRuntime, owner, paths.Ledger(), current); err != nil {
				rt.Log(adapt.NearMiss, "legacy binding proof migration unavailable: "+err.Error())
			}
			return nil
		}
		if current.Binding != nil {
			return nil
		}
		if current.Launch.Version != 2 {
			rt.Log(adapt.NearMiss, "legacy unbound launch has no metadata boundary; watcher failed closed")
			return nil
		}
		var inventory sessioninventory.Inventory
		var events []sessioninventory.NativeEventFact
		proofs := map[string]sessionledger.AuthorizationProof{}
		catalog := sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}
		if saved, readErr := rt.CatalogStore().Read(paths.SessionInventoryCatalog()); readErr == nil {
			catalog = saved
		} else if !errors.Is(readErr, os.ErrNotExist) && !errors.Is(readErr, sessioninventory.ErrCatalogCorrupt) {
			rt.Log(adapt.NearMiss, "session inventory catalog unavailable: "+readErr.Error())
		}
		incremental := sessioninventory.NewIncrementalInventory(nativeRuntime, catalog)
		inventory, events, proofs = incrementalWatcherInventory(nativeRuntime, incremental, agent, current.Launch, trackedTargets)
		if err := persistTrackedCatalog(rt.CatalogStore(), paths.SessionInventoryCatalog(), trackedTargets); err != nil {
			proofs = map[string]sessionledger.AuthorizationProof{}
			inventory.Diagnostics = append(inventory.Diagnostics, sessioninventory.DiagnosticWithSource(sessioninventory.DiagnosticStorageUnreadable, agent, nil, "session inventory catalog", "catalog publication failed; binding authority withheld"))
		}
		beforeRoots, beforeAvailable := processAuthorizedRoots(nativeRuntime, inventory, agent, corroborationPID)
		pairLog, logDiagnostics := readPairLog(rt, paths.Log(), agent)
		inventory.Diagnostics = append(inventory.Diagnostics, logDiagnostics...)
		for _, diagnostic := range logDiagnostics {
			rt.Log(adapt.NearMiss, string(diagnostic.Code)+": Pair log unavailable for causal matching")
		}
		rounds, roundDiagnostics := sessioninventory.RoundsAfterLaunch(inventory, opts.ScopeKey, opts.Tag, agent, pairLog, current.Launch, events)
		inventory.Diagnostics = append(inventory.Diagnostics, roundDiagnostics...)
		afterRoots, afterAvailable := processAuthorizedRoots(nativeRuntime, inventory, agent, corroborationPID)
		if rootIdentity != "" && rt.ProcessIdentity(rootPID) != rootIdentity {
			rt.Log(adapt.NearMiss, "process identity changed during native scan")
			return nil
		}
		if beforeAvailable && afterAvailable {
			rounds = retainCorroboratedRounds(rounds, beforeRoots, afterRoots)
		}
		resolved, persistErr := ObserveAndPersist(ObserveInput{
			Owner: owner, LedgerPath: paths.Ledger(), LaunchOrdinal: opts.LaunchOrdinal,
			Inventory: inventory, LiveRounds: rounds, Proofs: proofs, Args: opts.Args,
		}, rt.LedgerAppender(), func(payload ConfigPayload) error {
			raw, err := ConfigJSON(payload)
			if err != nil {
				return err
			}
			return rt.AtomicWrite(configPath, raw)
		})
		if persistErr != nil {
			if sessionledger.AppendOutcomeOf(persistErr) != sessionledger.AppendCommitted {
				return persistErr
			}
			rt.Log(adapt.NearMiss, "binding committed with cleanup warning: "+persistErr.Error())
		}
		if len(resolved.Bindings) == 1 && resolved.Bindings[0].Status == sessioninventory.BindingEstablished {
			rt.Log(adapt.Fired, "session_id="+nativeIDForRoot(inventory.Forests, agent, valueOrEmpty(resolved.Bindings[0].RootNodeID)))
			return nil
		}
		if rootPID == "" && !rt.Now().Before(deadline) {
			rt.Log(adapt.Fail, "no completed native round within startup deadline (agent="+opts.Agent+")")
			return nil
		}
		poll := opts.Poll
		if !rt.Now().Before(deadline) {
			poll = opts.SlowPoll
		}
		rt.Sleep(poll)
	}
}

func migrateProoflessBinding(rt Runtime, nativeRuntime sessioninventory.Runtime, owner sessionledger.Owner, ledgerPath string, current sessionledger.Current) error {
	binding := current.Binding
	if binding == nil || binding.RootNativeID == "" || binding.AuthorizationProof != nil {
		return nil
	}
	key := sessioninventory.ProofMigrationKey{ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: sessioninventory.Agent(owner.Agent), NativeID: binding.RootNativeID}
	result := <-rt.ProofMigrator().Request(key, func() (*sessionledger.AuthorizationProof, error) {
		inventory := sessioninventory.NewIncrementalInventory(nativeRuntime, sessioninventory.Catalog{Version: sessioninventory.CatalogVersion})
		snapshot := inventory.Observe(key.Agent)
		selected := inventory.Select(sessioninventory.TargetRequest{Mode: sessioninventory.TargetExplicitResume, Agent: key.Agent, NativeID: key.NativeID}, snapshot)
		if selected.Unavailable {
			return nil, sessioninventory.ErrArtifactChanged
		}
		validations, _ := sessioninventory.ValidateTargetWork(nativeRuntime, key.Agent, selected.Eligible)
		for _, validation := range validations {
			if validation.State.NativeID == key.NativeID && validation.State.Role == sessioninventory.RoleRoot {
				proof, err := authorizationProof(validation)
				return &proof, err
			}
		}
		return nil, sessioninventory.ErrArtifactChanged
	})
	if result.Err != nil {
		return result.Err
	}
	record, err := rt.LedgerAppender().AppendBindingProofIfCurrent(ledgerPath, owner, current.Launch.Ordinal, *result.Proof)
	return reconcileLedgerAppend(rt.LedgerAppender(), ledgerPath, record, err)
}

func persistTrackedCatalog(store sessioninventory.CatalogStore, path string, tracked map[string]sessioninventory.TargetValidation) error {
	entries := map[string]sessioninventory.CatalogEntry{}
	for _, validation := range tracked {
		stateRaw, err := json.Marshal(validation.State)
		if err != nil {
			return err
		}
		for _, observation := range validation.Observations {
			entry := observation.Entry
			fingerprint := sessioninventory.ArtifactFingerprint{StableFileID: entry.StableFileID, GenerationToken: entry.GenerationToken, MutationToken: entry.MutationToken, Size: entry.Size, BirthTime: entry.BirthTime, ModTime: entry.ModTime}
			rawOffset, parserOffset := entry.Size, entry.Size
			key := entry.Artifact.StorageRoot + "\x00" + entry.Artifact.RelativePath
			if result, ok := validation.Results[key]; ok {
				fingerprint = result.Fingerprint
				rawOffset = result.RawObservedOffset
				parserOffset = result.FrameState.ParserCompleteOffset
			}
			entries[string(observation.Agent)+"\x00"+key] = sessioninventory.CatalogEntry{
				Agent: observation.Agent, Artifact: entry.Artifact, Fingerprint: fingerprint, Authorization: sessioninventory.AuthorizationAuthorized,
				Facts: []sessioninventory.Fact{validation.Fact}, ScannerSchema: observation.ScannerSchema, ProviderContract: observation.ProviderContract,
				RawObservedOffset: rawOffset, ParserCompleteOffset: parserOffset, ScannerState: append(json.RawMessage(nil), stateRaw...),
			}
		}
	}
	merge := func(current sessioninventory.Catalog) (sessioninventory.Catalog, error) {
		byKey := map[string]sessioninventory.CatalogEntry{}
		for _, entry := range current.Entries {
			byKey[string(entry.Agent)+"\x00"+entry.Artifact.StorageRoot+"\x00"+entry.Artifact.RelativePath] = entry
		}
		for key, entry := range entries {
			if existing, ok := byKey[key]; ok {
				byKey[key] = sessioninventory.MergeCatalogPublication(existing, entry)
			} else {
				byKey[key] = entry
			}
		}
		current.Entries = current.Entries[:0]
		for _, entry := range byKey {
			current.Entries = append(current.Entries, entry)
		}
		return current, nil
	}
	_, err := store.Update(path, merge)
	if errors.Is(err, sessioninventory.ErrCatalogCorrupt) {
		_, err = store.Repair(path, merge)
	}
	return err
}

func incrementalWatcherInventory(runtime sessioninventory.Runtime, incremental sessioninventory.IncrementalInventory, agent sessioninventory.Agent, launch sessionledger.Record, tracked map[string]sessioninventory.TargetValidation) (sessioninventory.Inventory, []sessioninventory.NativeEventFact, map[string]sessionledger.AuthorizationProof) {
	snapshot := incremental.Observe(agent)
	diagnostics := snapshot.Diagnostics
	baseline := make([]sessioninventory.TargetArtifactBoundary, 0, len(launch.LaunchArtifactBoundaries))
	for _, boundary := range launch.LaunchArtifactBoundaries {
		baseline = append(baseline, sessioninventory.TargetArtifactBoundary{
			StorageRoot: boundary.StorageRoot, RelativePath: boundary.RelativePath,
			StableFileID: sessioninventory.StableFileID(boundary.StableFileID), GenerationToken: sessioninventory.GenerationToken(boundary.GenerationToken),
			MutationToken: sessioninventory.MutationToken(boundary.MutationToken), RawSize: boundary.RawSize,
		})
	}
	selection := incremental.Select(sessioninventory.TargetRequest{Mode: sessioninventory.TargetNewLaunch, Agent: agent, Baseline: baseline}, snapshot)
	handled := map[string]bool{}
	for nativeID, prior := range tracked {
		artifacts := make([]sessioninventory.Artifact, 0, len(prior.Observations))
		for _, observation := range prior.Observations {
			artifacts = append(artifacts, observation.Entry.Artifact)
		}
		current := incremental.Select(sessioninventory.TargetRequest{Mode: sessioninventory.TargetEstablished, Agent: agent, NativeID: nativeID, AuthorizedArtifacts: artifacts}, snapshot)
		if current.Unavailable || len(current.Eligible) != len(artifacts) {
			delete(tracked, nativeID)
			continue
		}
		advanced, found, err := sessioninventory.AdvanceTargetValidation(runtime, prior, current.Eligible)
		diagnostics = append(diagnostics, found...)
		if err != nil {
			delete(tracked, nativeID)
			continue
		}
		tracked[nativeID] = advanced
		for _, observation := range current.Eligible {
			handled[observation.Entry.Artifact.StorageRoot+"\x00"+observation.Entry.Artifact.RelativePath] = true
		}
	}
	var untracked []sessioninventory.ArtifactObservation
	for _, observation := range selection.Eligible {
		if !handled[observation.Entry.Artifact.StorageRoot+"\x00"+observation.Entry.Artifact.RelativePath] {
			untracked = append(untracked, observation)
		}
	}
	validated, found := sessioninventory.ValidateTargetWork(runtime, agent, untracked)
	diagnostics = append(diagnostics, found...)
	for _, validation := range validated {
		tracked[validation.State.NativeID] = validation
	}
	var facts []sessioninventory.Fact
	var events []sessioninventory.NativeEventFact
	proofs := map[string]sessionledger.AuthorizationProof{}
	for nativeID, validation := range tracked {
		facts = append(facts, validation.Fact)
		events = append(events, validation.Events...)
		if validation.State.Role == sessioninventory.RoleRoot {
			if proof, err := authorizationProof(validation); err == nil {
				proofs[sessioninventory.StableID("node", string(agent), nativeID)] = proof
			}
		}
	}
	inventory := sessioninventory.BuildForest(facts)
	inventory.Diagnostics = append(inventory.Diagnostics, diagnostics...)
	return sessioninventory.SortInventory(inventory), events, proofs
}

func applyWatcherDefaults(opts *Options) {
	if opts.PIDWait <= 0 {
		opts.PIDWait = 2 * time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.Poll <= 0 {
		opts.Poll = 100 * time.Millisecond
	}
	if opts.SlowPoll <= 0 {
		opts.SlowPoll = 60 * time.Second
	}
}

func waitForPairProcess(pidFile string, opts Options, watchStart time.Time, rt Runtime) (string, string) {
	deadline := watchStart.Add(opts.PIDWait)
	for {
		if fresh, _ := pidFileCurrent(pidFile, opts.PIDNotBefore, watchStart, rt); fresh {
			if raw, err := rt.ReadFile(pidFile); err == nil {
				pid := strings.TrimSpace(string(raw))
				return pid, rt.ProcessIdentity(pid)
			}
		}
		if !rt.Now().Before(deadline) {
			return "", ""
		}
		rt.Sleep(opts.Poll)
	}
}

func readCurrentLaunch(rt Runtime, path string, owner sessionledger.Owner) (sessionledger.Current, bool, error) {
	raw, err := rt.ReadFile(path)
	if err != nil {
		return sessionledger.Current{}, false, err
	}
	current, ok := sessionledger.CurrentLaunch(sessionledger.ParseLedger(raw).Records, owner)
	return current, ok, nil
}

func processAuthorizedRoots(runtime sessioninventory.Runtime, inventory sessioninventory.Inventory, agent sessioninventory.Agent, rootPID string) (map[string]bool, bool) {
	if rootPID == "" {
		return nil, false
	}
	children := runtime.ProcessChildren()
	queue := []string{rootPID}
	seen := map[string]bool{}
	open := map[string]bool{}
	for len(queue) != 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		for _, path := range runtime.OpenFiles(pid) {
			open[filepath.Clean(path)] = true
		}
		queue = append(queue, children[pid]...)
	}
	if len(open) == 0 {
		return nil, false
	}
	rootPaths := map[string]string{}
	for _, root := range runtime.NativeRoots(agent) {
		rootPaths[root.Name] = root.Path
	}
	authorized := map[string]bool{}
	for _, forest := range inventory.Forests {
		if forest.Agent != agent {
			continue
		}
		for _, root := range forest.Roots {
			for _, artifact := range root.Artifacts {
				base := rootPaths[artifact.StorageRoot]
				if base != "" && open[filepath.Clean(filepath.Join(base, filepath.FromSlash(artifact.RelativePath)))] {
					authorized[root.StableID] = true
				}
			}
		}
	}
	return authorized, len(authorized) != 0
}

func readPairLog(rt Runtime, path string, agent sessioninventory.Agent) ([]byte, []sessioninventory.Diagnostic) {
	raw, err := rt.ReadFile(path)
	if err == nil {
		return raw, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, []sessioninventory.Diagnostic{sessioninventory.DiagnosticWithSource(
		sessioninventory.DiagnosticStorageUnreadable, agent, nil, "pair-log", "Pair log is unreadable during causal-round observation",
	)}
}

func retainCorroboratedRounds(rounds []sessioninventory.RoundObservation, before, after map[string]bool) []sessioninventory.RoundObservation {
	result := make([]sessioninventory.RoundObservation, 0, len(rounds))
	for _, round := range rounds {
		if before[round.RootNodeID] && after[round.RootNodeID] {
			result = append(result, round)
		}
	}
	return result
}

func pidFileCurrent(pidFile string, pidNotBefore, watchStart time.Time, rt Runtime) (bool, time.Time) {
	mod, err := rt.ModTime(pidFile)
	if err != nil {
		return false, time.Time{}
	}
	return pidFileFresh(mod, pidNotBefore, watchStart), mod
}

func pidFileFresh(mod, pidNotBefore, watchStart time.Time) bool {
	if !pidNotBefore.IsZero() {
		return !mod.Before(pidNotBefore)
	}
	return mod.Unix() >= watchStart.Unix()
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
