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
	Agent           string
	Tag             string
	ScopeKey        string
	LaunchOrdinal   uint64
	Cwd             string
	RepoRoot        string
	RepoName        string
	Args            []string
	Home            string
	DataDir         string
	PIDWait         time.Duration
	Timeout         time.Duration
	Poll            time.Duration
	SlowPoll        time.Duration
	ActivePoll      time.Duration // cadence while a Pair send awaits its round (#316)
	ActiveWindow    time.Duration // how long after a send ActivePoll applies
	IdleExpiry      time.Duration // an unbound watcher with no send or agent output for this long exits (#420)
	PIDNotBefore    time.Time
	FollowLifecycle bool
	AppendLifecycle func(string, LifecycleRecord) error
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
	// The log's state at start is the baseline: earlier launches' sends are not
	// this launch's, so only a later change counts as a send.
	logSeen, _ := rt.ModTime(paths.Log())
	var lastSend time.Time
	rootPID, rootIdentity := waitForPairProcess(paths.AgentPID(), opts, watchStart, rt)
	corroborationPID := rootPID
	if rootIdentity == "" {
		corroborationPID = ""
	}
	owner := sessionledger.Owner{ScopeKey: opts.ScopeKey, Tag: opts.Tag, Agent: opts.Agent}
	nativeRuntime := rt.NativeRuntime(opts.Home, opts.DataDir)
	trackedTargets := map[string]sessioninventory.TargetValidation{}
	boundRootNodeID := ""
	lifecycleWatermark := uint64(0)
	deadline := watchStart.Add(opts.Timeout)
	// In-memory observation phases: an incomplete launch first acquires a
	// conservative epoch; complete epochs admit handshake/correlation; a unique
	// confirmation ends probation and optionally follows lifecycle telemetry.
	// No epoch or parser state can replace the durable requested identity.
	var observationEpoch *sessionledger.Record
	// quietAt is the last scan start at which the Pair log held no turn past
	// the launch offset. A round's native events are written after its turn,
	// so a candidate last modified before it cannot hold one (#420).
	var quietAt time.Time
	// quietLogMod is the Pair log's mtime when it was last read with no turn
	// pending; an unchanged log is not re-read (#420).
	var quietLogMod time.Time
	// outputSeen is the agent pane recording's mtime at the last chosen-id
	// walk. The chosen transcript appears with the agent's first round, which
	// writes output, so an idle chosen launch walks again only after output.
	// Without a recording it keeps walking each pass (#420).
	var outputSeen time.Time
	// outputActivitySeen/lastOutput feed the idle expiry. The first observation
	// sets only the baseline; output that existed before launch is not activity.
	var outputActivitySeen, lastOutput time.Time
	if mod, err := rt.ModTime(paths.ScrollbackRaw(opts.Agent)); err == nil {
		outputActivitySeen = mod
	}
	outputMoved := func() bool {
		mod, err := rt.ModTime(paths.ScrollbackRaw(opts.Agent))
		if err != nil {
			return true
		}
		if mod.Equal(outputSeen) {
			return false
		}
		outputSeen = mod
		return true
	}
	// waitFor ends one unbound pass: the startup deadline, the idle expiry, then
	// the send-aware sleep (delay 0 means the scanDelay cadence). Every unbound
	// pass that sleeps goes through it, so the funeral covers each one. It
	// reports whether the watcher should exit.
	var waitFor func(time.Duration) bool
	wait := func() bool { return waitFor(0) }
	waitFor = func(delay time.Duration) bool {
		if rootPID == "" && !rt.Now().Before(deadline) {
			rt.Log(adapt.Fail, "no completed native round within startup deadline (agent="+opts.Agent+")")
			return true
		}
		if mod, err := rt.ModTime(paths.Log()); err == nil && !mod.Equal(logSeen) {
			logSeen, lastSend = mod, rt.Now()
		}
		// Agent output is activity too: an operator typing straight into the
		// pane binds a chosen launch through the handshake, with no send.
		if mod, err := rt.ModTime(paths.ScrollbackRaw(opts.Agent)); err == nil && !mod.Equal(outputActivitySeen) {
			outputActivitySeen, lastOutput = mod, rt.Now()
		}
		lastActivity := lastSend
		if lastOutput.After(lastActivity) {
			lastActivity = lastOutput
		}
		if idleExpired(rt.Now(), watchStart, lastActivity, opts.IdleExpiry) {
			rt.Log(adapt.NearMiss, "unbound watcher idle past expiry; exiting (agent="+opts.Agent+")")
			return true
		}
		if delay == 0 {
			delay = scanDelay(rt.Now(), watchStart, lastSend, opts)
		}
		waitForScan(rt, paths.Log(), logSeen, delay, opts.ActivePoll)
		return false
	}
	for {
		scanStart := rt.Now()
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
		if current.Binding != nil && boundRootNodeID == "" && current.Binding.AuthorizationProof == nil && current.Binding.Version < 3 {
			if err := migrateProoflessBinding(rt, nativeRuntime, owner, paths.Ledger(), current); err != nil {
				rt.Log(adapt.NearMiss, "legacy binding proof migration unavailable: "+err.Error())
			}
			return nil
		}
		if current.Binding != nil && boundRootNodeID == "" {
			if opts.FollowLifecycle && agent == sessioninventory.AgentCodex {
				validation, err := validateCurrentLifecycleTarget(rt, nativeRuntime, paths.SessionInventoryCatalog(), agent, *current.Binding)
				if err != nil {
					rt.Log(adapt.NearMiss, "bound lifecycle target unavailable: "+err.Error())
					return nil
				}
				trackedTargets[validation.State.NativeID] = validation
				boundRootNodeID = sessioninventory.StableID("node", string(agent), validation.State.NativeID)
				for _, event := range validation.Events {
					if event.Position > lifecycleWatermark {
						lifecycleWatermark = event.Position
					}
				}
				rt.Sleep(opts.Poll)
				continue
			}
			return nil
		}
		if current.Launch.Version != 2 && current.Launch.Version != 3 {
			rt.Log(adapt.NearMiss, "legacy unbound launch has no metadata boundary; watcher failed closed")
			return nil
		}
		// The catalog is read only by passes that observe the native store; a
		// gated pass skips it with the walk (#420).
		newIncremental := func() sessioninventory.IncrementalInventory {
			catalog := sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}
			if saved, readErr := rt.CatalogStore().Read(paths.SessionInventoryCatalog()); readErr == nil {
				catalog = saved
			} else if !errors.Is(readErr, os.ErrNotExist) && !errors.Is(readErr, sessioninventory.ErrCatalogCorrupt) {
				rt.Log(adapt.NearMiss, "session inventory catalog unavailable: "+readErr.Error())
			}
			return sessioninventory.NewIncrementalInventory(nativeRuntime, catalog)
		}
		effectiveLaunch := current.Launch
		if current.Launch.Version == 3 && !current.Launch.BaselineComplete {
			if observationEpoch == nil {
				snapshot := newIncremental().Observe(agent)
				boundaries, complete := launchBoundaries(snapshot)
				pairLog, readErr := rt.ReadFile(paths.Log())
				if complete && (readErr == nil || errors.Is(readErr, os.ErrNotExist)) {
					epoch := current.Launch
					epoch.BaselineComplete = true
					epoch.LaunchArtifactBoundaries = boundaries
					epoch.PairLogOffset = uint64(len(pairLog))
					epoch.RequestOrigin = ""
					epoch.RequestedNativeID = ""
					observationEpoch = &epoch
				}
				if waitFor(opts.SlowPoll) {
					return nil
				}
				continue
			}
			effectiveLaunch = *observationEpoch
		}
		// Send gate (#420): correlation matches Pair-log turns past the launch
		// offset, so before one exists nothing can bind and the native store is
		// not worth validating. An unreadable log stays on the full path.
		var pairLog []byte
		var logDiagnostics []sessioninventory.Diagnostic
		turnPending := false
		logMod, logModErr := rt.ModTime(paths.Log())
		if logModErr != nil || quietLogMod.IsZero() || !logMod.Equal(quietLogMod) || boundRootNodeID != "" {
			pairLog, logDiagnostics = readPairLog(rt, paths.Log(), agent)
			turnPending = len(logDiagnostics) != 0 || uint64(len(pairLog)) > effectiveLaunch.PairLogOffset
			if !turnPending && logModErr == nil {
				quietLogMod = logMod
			}
		}
		chosen := effectiveLaunch.Version == 3 && effectiveLaunch.RequestOrigin == sessionledger.RequestOriginChosen
		if !turnPending && boundRootNodeID == "" {
			quietAt = scanStart
			clear(trackedTargets)
			if !chosen || !outputMoved() {
				if wait() {
					return nil
				}
				continue
			}
		}
		incremental := newIncremental()
		snapshot := incremental.Observe(agent)
		if chosen {
			_, complete := launchBoundaries(snapshot)
			if complete && sessioninventory.ChosenFilenameHandshake(agent, effectiveLaunch.RequestedNativeID, targetBoundaries(effectiveLaunch), snapshot.Observations) != nil {
				confirmer, ok := rt.LedgerAppender().(interface {
					ConfirmIfCurrent(string, sessionledger.Owner, uint64, string, sessionledger.ConfirmationReason, *sessionledger.AuthorizationProof) (sessionledger.Record, error)
				})
				if !ok {
					return errors.New("ledger does not support confirmation")
				}
				record, err := confirmer.ConfirmIfCurrent(paths.Ledger(), owner, opts.LaunchOrdinal, effectiveLaunch.RequestedNativeID, sessionledger.ConfirmationChosen, nil)
				if err = reconcileLedgerAppend(rt.LedgerAppender(), paths.Ledger(), record, err); err != nil {
					return err
				}
				rt.Log(adapt.Fired, "session_id="+effectiveLaunch.RequestedNativeID+" reason=chosen-id")
				return nil
			}
		}
		if !turnPending && boundRootNodeID == "" {
			if wait() {
				return nil
			}
			continue
		}
		var inventory sessioninventory.Inventory
		var events []sessioninventory.NativeEventFact
		proofs := map[string]sessionledger.AuthorizationProof{}

		floor := time.Time{}
		if boundRootNodeID == "" && !quietAt.IsZero() {
			floor = quietAt.Add(-candidateMTimeSlack)
		}
		inventory, events, proofs = incrementalWatcherInventorySnapshot(nativeRuntime, incremental, agent, effectiveLaunch, trackedTargets, snapshot, floor)
		if err := persistTrackedCatalog(rt.CatalogStore(), paths.SessionInventoryCatalog(), trackedTargets); err != nil {

			inventory.Diagnostics = append(inventory.Diagnostics, sessioninventory.DiagnosticWithSource(sessioninventory.DiagnosticStorageUnreadable, agent, nil, "session inventory catalog", "catalog publication failed; durable observation remains usable"))
		}
		if boundRootNodeID != "" {
			var publishErr error
			lifecycleWatermark, publishErr = publishAuthorizedLifecycle(opts, rt, paths.LifecycleJournal(), trackedTargets, events, boundRootNodeID, lifecycleWatermark)
			if publishErr != nil {
				return publishErr
			}
			rt.Sleep(opts.Poll)
			continue
		}
		beforeRoots, beforeAvailable := processAuthorizedRoots(nativeRuntime, inventory, agent, corroborationPID)
		inventory.Diagnostics = append(inventory.Diagnostics, logDiagnostics...)
		for _, diagnostic := range logDiagnostics {
			rt.Log(adapt.NearMiss, string(diagnostic.Code)+": Pair log unavailable for causal matching")
		}
		rounds, roundDiagnostics := sessioninventory.RoundsAfterLaunch(inventory, opts.ScopeKey, opts.Tag, agent, pairLog, effectiveLaunch, events)
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
			Inventory: inventory, LiveRounds: rounds, Proofs: proofs, RequireProof: current.Launch.Version < 3, Args: opts.Args,
			ConfirmationReason: sessionledger.ConfirmationCorrelation,
		}, rt.LedgerAppender(), func(payload ConfigPayload) error {
			if saved, readErr := rt.ReadFile(configPath); readErr == nil {
				var previous ConfigPayload
				if json.Unmarshal(saved, &previous) != nil {
					return errors.New("saved launch config is unreadable")
				}
				if previous.Agent == payload.Agent {
					payload.Args = previous.Args
				}
			} else if !errors.Is(readErr, os.ErrNotExist) {
				return readErr
			}
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
			boundRootNodeID = valueOrEmpty(resolved.Bindings[0].RootNodeID)
			rt.Log(adapt.Fired, "session_id="+nativeIDForRoot(inventory.Forests, agent, boundRootNodeID))
			if !opts.FollowLifecycle || agent != sessioninventory.AgentCodex {
				return nil
			}
			var publishErr error
			lifecycleWatermark, publishErr = publishAuthorizedLifecycle(opts, rt, paths.LifecycleJournal(), trackedTargets, events, boundRootNodeID, lifecycleWatermark)
			if publishErr != nil {
				return publishErr
			}
			rt.Sleep(opts.Poll)
			continue
		}
		if wait() {
			return nil
		}
	}
}

// candidateMTimeSlack widens the quiet-moment floor for filesystem timestamp
// granularity and for a turn whose native write races the log append.
const candidateMTimeSlack = time.Minute

// idleExpired is the unbound watcher's end (#420, ARCH-FUNERAL): a day with no
// activity (a send or agent output) since the later of launch and the last
// activity. A thread used after that
// binds on its next relaunch, which spawns a fresh watcher.
func idleExpired(now, watchStart, lastActivity time.Time, expiry time.Duration) bool {
	since := watchStart
	if lastActivity.After(since) {
		since = lastActivity
	}
	return !now.Before(since.Add(expiry))
}

// scanDelay is the watcher's cadence. A proof needs a Pair send and then the
// agent finishing that round, both at arbitrary times after launch, so the fast
// cadence follows the latest send rather than the launch alone (#316). SlowPoll
// remains the #143 backstop for a thread nobody has written to.
func scanDelay(now, watchStart, lastSend time.Time, opts Options) time.Duration {
	if now.Before(watchStart.Add(opts.Timeout)) {
		return opts.Poll
	}
	if !lastSend.IsZero() && now.Before(lastSend.Add(opts.ActiveWindow)) {
		return opts.ActivePoll
	}
	return opts.SlowPoll
}

// waitForScan sleeps delay in slice-sized steps and returns early once the Pair
// log differs from seen: a send during a slow wait is exactly the event a proof
// waits on, so it must not sit out the rest of that wait.
func waitForScan(rt Runtime, logPath string, seen time.Time, delay, slice time.Duration) {
	for delay > slice {
		rt.Sleep(slice)
		delay -= slice
		if mod, err := rt.ModTime(logPath); err == nil && !mod.Equal(seen) {
			return
		}
	}
	rt.Sleep(delay)
}

func validateCurrentLifecycleTarget(rt Runtime, nativeRuntime sessioninventory.Runtime, catalogPath string, agent sessioninventory.Agent, binding sessionledger.Record) (sessioninventory.TargetValidation, error) {
	if binding.AuthorizationProof != nil {
		return validateBoundLifecycleTarget(rt, nativeRuntime, catalogPath, agent, *binding.AuthorizationProof)
	}
	catalog := sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}
	if saved, err := rt.CatalogStore().Read(catalogPath); err == nil {
		catalog = saved
	}
	validation, _, err := sessioninventory.NewIncrementalInventory(nativeRuntime, catalog).ValidateNamedContent(agent, binding.RootNativeID)
	return validation, err
}

func validateBoundLifecycleTarget(rt Runtime, nativeRuntime sessioninventory.Runtime, catalogPath string, agent sessioninventory.Agent, proof sessionledger.AuthorizationProof) (sessioninventory.TargetValidation, error) {
	catalog := sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}
	if saved, err := rt.CatalogStore().Read(catalogPath); err == nil {
		catalog = saved
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, sessioninventory.ErrCatalogCorrupt) {
		return sessioninventory.TargetValidation{}, err
	}
	validation, _, err := sessioninventory.NewIncrementalInventory(nativeRuntime, catalog).ValidateBindingProof(agent, proof)
	return validation, err
}

func migrateProoflessBinding(rt Runtime, nativeRuntime sessioninventory.Runtime, owner sessionledger.Owner, ledgerPath string, current sessionledger.Current) error {
	binding := current.Binding
	if binding == nil || binding.Version >= 3 || binding.RootNativeID == "" || binding.AuthorizationProof != nil {
		return nil
	}
	key := sessioninventory.ProofMigrationKey{ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: sessioninventory.Agent(owner.Agent), NativeID: binding.RootNativeID}
	result := <-rt.ProofMigrator().Request(key, func() (*sessionledger.AuthorizationProof, error) {
		proof, err := proofForNamedRoot(nativeRuntime, key.Agent, key.NativeID)
		return &proof, err
	})
	if result.Err != nil {
		return result.Err
	}
	record, err := rt.LedgerAppender().AppendBindingProofIfCurrent(ledgerPath, owner, current.Launch.Ordinal, *result.Proof)
	return reconcileLedgerAppend(rt.LedgerAppender(), ledgerPath, record, err)
}

func persistTrackedCatalog(store sessioninventory.CatalogStore, path string, tracked map[string]sessioninventory.TargetValidation) error {
	validations := make([]sessioninventory.TargetValidation, 0, len(tracked))
	for _, validation := range tracked {
		validations = append(validations, validation)
	}
	return sessioninventory.PublishTargetValidations(store, path, validations)
}

func incrementalWatcherInventory(runtime sessioninventory.Runtime, incremental sessioninventory.IncrementalInventory, agent sessioninventory.Agent, launch sessionledger.Record, tracked map[string]sessioninventory.TargetValidation) (sessioninventory.Inventory, []sessioninventory.NativeEventFact, map[string]sessionledger.AuthorizationProof) {
	snapshot := incremental.Observe(agent)
	return incrementalWatcherInventorySnapshot(runtime, incremental, agent, launch, tracked, snapshot, time.Time{})
}

func incrementalWatcherInventorySnapshot(runtime sessioninventory.Runtime, incremental sessioninventory.IncrementalInventory, agent sessioninventory.Agent, launch sessionledger.Record, tracked map[string]sessioninventory.TargetValidation, snapshot sessioninventory.IncrementalSnapshot, floor time.Time) (sessioninventory.Inventory, []sessioninventory.NativeEventFact, map[string]sessionledger.AuthorizationProof) {
	diagnostics := snapshot.Diagnostics
	baseline := targetBoundaries(launch)
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
		if handled[observation.Entry.Artifact.StorageRoot+"\x00"+observation.Entry.Artifact.RelativePath] {
			continue
		}
		// Older than the floor: written before any pending turn, so it holds
		// no round. Its next write brings it back.
		if modTime := observation.Entry.ModTime; !floor.IsZero() && modTime != nil && modTime.Before(floor) {
			continue
		}
		untracked = append(untracked, observation)
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
	if opts.ActivePoll <= 0 {
		opts.ActivePoll = time.Second
	}
	if opts.ActiveWindow <= 0 {
		opts.ActiveWindow = 30 * time.Minute
	}
	if opts.IdleExpiry <= 0 {
		opts.IdleExpiry = 24 * time.Hour
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

func targetBoundaries(launch sessionledger.Record) []sessioninventory.TargetArtifactBoundary {
	baseline := make([]sessioninventory.TargetArtifactBoundary, 0, len(launch.LaunchArtifactBoundaries))
	for _, boundary := range launch.LaunchArtifactBoundaries {
		baseline = append(baseline, sessioninventory.TargetArtifactBoundary{
			StorageRoot: boundary.StorageRoot, RelativePath: boundary.RelativePath,
			StableFileID: sessioninventory.StableFileID(boundary.StableFileID), GenerationToken: sessioninventory.GenerationToken(boundary.GenerationToken),
			MutationToken: sessioninventory.MutationToken(boundary.MutationToken), RawSize: boundary.RawSize,
		})
	}
	return baseline
}
