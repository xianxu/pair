package launcher

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// RecoveryNativeOwnerCheck permits only the exact source wrapper during
// preflight, proved by direct ancestry or the session's agent readiness. The post-retirement and Pair gates exclude nobody.
// This is an observation, not a lock against simultaneous cross-thread creates.
type RecoveryNativeOwnerCheck struct {
	DataDir, RepoScope, Tag, Agent, NativeID string
	SourcePID                                int
	SourceIdentity                           string
	SourceSession, SourceNonce               string
	Admissions                               NativeAdmissionOwners
	io                                       *nativeOwnerCheckIO
	heldFiles                                nativeHeldFiles
	ledgers                                  *nativeOwnerLedgerMemo
}

type recoveryNativeWrapper struct {
	Process procutil.Process
	Owner   artifactpath.StorageOwner
	Agent   string
}

func CheckRecoveryNativeOwner(ctx context.Context, check RecoveryNativeOwnerCheck) error {
	if ctx == nil {
		return errors.New("native ownership check requires context")
	}
	if check.NativeID == "" {
		return ctx.Err()
	}
	ops := nativeOwnerCheckIO{store: NativeAdmissionStore{Home: os.Getenv("HOME")}, observe: newRecoveryTerminalOS().Observe, read: readNativeOwnerMetadata, discover: discoverLegacyNativeOwners}
	if check.io != nil {
		ops = *check.io
	}
	run := func(ctx context.Context, owners NativeAdmissionOwners) error {
		check.Admissions = owners
		return checkRecoveryNativeOwnersWithDiscovery(ctx, check, ops.observe, ops.read, ops.discover)
	}
	if check.Admissions.active != nil && check.Admissions.active.Load() {
		return run(ctx, check.Admissions)
	}
	return ops.store.withStore(ctx, func(ctx context.Context, state *nativeAdmissionFile, io nativeAdmissionIO) error {
		return withNativeAdmissionOwners(ctx, state, io, run)
	})
}

func checkRecoveryNativeOwners(ctx context.Context, check RecoveryNativeOwnerCheck, observe func(context.Context, string) (recoveryTerminalSnapshot, error), read func(context.Context, string, int64) ([]byte, error)) error {
	return checkRecoveryNativeOwnersWithDiscovery(ctx, check, observe, read, discoverLegacyNativeOwners)
}

var errNativeOwnerUnbound = errors.New("live wrapper native ownership is unbound")

var errNativeOwnerObservation = errors.New("native owner observation incomplete or changed")

func checkRecoveryNativeOwnersWithDiscovery(ctx context.Context, check RecoveryNativeOwnerCheck, observe func(context.Context, string) (recoveryTerminalSnapshot, error), read func(context.Context, string, int64) ([]byte, error), discover func(context.Context, string) ([]artifactpath.StorageOwner, error)) error {
	if ctx == nil {
		return errors.New("native ownership check requires context")
	}
	if check.NativeID == "" {
		return ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, zjTimeout)
	defer cancel()
	check.ledgers = &nativeOwnerLedgerMemo{values: make(map[nativeOwnerLedgerKey]nativeOwnerLedgerResult)}
	// Re-observe once under the SAME budget. A process arriving between the
	// argv snapshot and final table is uncertainty, not a permanent blocker or
	// permission to admit it unchecked. No launch/signal effects occur here.
	for attempt := 0; ; attempt++ {
		err := checkRecoveryNativeOwnersOnce(ctx, check, observe, read, discover)
		if attempt == 1 || !errors.Is(err, errNativeOwnerObservation) {
			return err
		}
	}
}

func checkRecoveryNativeOwnersOnce(ctx context.Context, check RecoveryNativeOwnerCheck, observe func(context.Context, string) (recoveryTerminalSnapshot, error), read func(context.Context, string, int64) ([]byte, error), discover func(context.Context, string) ([]artifactpath.StorageOwner, error)) error {
	if !filepath.IsAbs(check.DataDir) || check.Agent == "" {
		return errors.New("native ownership scope unavailable")
	}
	before, err := observe(ctx, "")
	if err != nil {
		return fmt.Errorf("native ownership observation: %w", err)
	}
	wrappers, err := recoveryNativeWrappers(check.DataDir, before)
	if err != nil {
		return err
	}
	beforeOrphans, err := observeLegacyNativeOrphans(ctx, check, before, wrappers, read, discover)
	if err != nil {
		return err
	}
	excluded := 0
	admitted := map[int]bool{}
	proofs := map[int]recoveryNativeProof{}
	for _, wrapper := range wrappers {
		if wrapper.Agent == check.Agent && check.Admissions.coversDifferentWrapper(check.Agent, check.NativeID, wrapper.Process, before) {
			admitted[wrapper.Process.PID] = true
		}
	}
	err = runOrderedNativeProofs(ctx, wrappers, func(workerCtx context.Context, wrapper recoveryNativeWrapper) (recoveryNativeProof, error) {
		if wrapper.Agent != check.Agent || admitted[wrapper.Process.PID] {
			return recoveryNativeProof{}, nil
		}
		return recoveryNativeProofForWrapper(workerCtx, check, before, wrapper, read)
	}, func(wrapper recoveryNativeWrapper, proof recoveryNativeProof) error {
		if wrapper.Agent != check.Agent || admitted[wrapper.Process.PID] {
			return nil
		}
		proofs[wrapper.Process.PID] = proof
		if recoverySourceWrapper(check, before, wrapper, proof.Ready) {
			excluded++
			if excluded > 1 {
				return errors.New("multiple source wrappers make native ownership uncertain")
			}
			return nil
		}
		if proof.NativeID == check.NativeID || slices.Contains(proof.HeldIDs, check.NativeID) {
			return fmt.Errorf("native conversation %s already has a live wrapper (%s:%s)", check.NativeID, wrapper.Owner.RepoScope, wrapper.Owner.Tag)
		}
		return nil
	})
	if err != nil {
		return err
	}
	after, err := observe(ctx, "")
	if err != nil {
		return err
	}
	next, err := recoveryNativeWrappers(check.DataDir, after)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(wrappers, next) {
		return fmt.Errorf("%w: native owner process generation changed; retry", errNativeOwnerObservation)
	}
	afterOrphans, err := observeLegacyNativeOrphans(ctx, check, after, next, read, discover)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(beforeOrphans, afterOrphans) {
		return errors.New("retained native owner generation changed; retry")
	}
	err = runOrderedNativeProofs(ctx, next, func(workerCtx context.Context, wrapper recoveryNativeWrapper) (recoveryNativeProof, error) {
		if wrapper.Agent != check.Agent || admitted[wrapper.Process.PID] {
			return recoveryNativeProof{}, nil
		}
		return recoveryNativeProofForWrapper(workerCtx, check, after, wrapper, read)
	}, func(wrapper recoveryNativeWrapper, proof recoveryNativeProof) error {
		if wrapper.Agent != check.Agent {
			return nil
		}
		if admitted[wrapper.Process.PID] {
			if !check.Admissions.coversDifferentWrapper(check.Agent, check.NativeID, wrapper.Process, after) {
				return errors.New("admitted native writer changed; retry")
			}
			return nil
		}
		if !reflect.DeepEqual(proofs[wrapper.Process.PID], proof) {
			return errors.New("native owner readiness generation changed; retry")
		}
		return nil
	})
	if err != nil {
		return err
	}
	if excluded > 0 {
		confirmed := 0
		for _, wrapper := range next {
			if recoverySourceWrapper(check, after, wrapper, proofs[wrapper.Process.PID].Ready) {
				confirmed++
			}
		}
		if confirmed != excluded {
			return errors.New("source wrapper identity changed; retry")
		}
	}

	return ctx.Err()
}

func recoverySourceAncestor(table []procutil.Process, p procutil.Process, pid int, identity string) bool {
	if pid <= 0 || identity == "" {
		return false
	}
	byPID := map[int]procutil.Process{}
	for _, row := range table {
		byPID[row.PID] = row
	}
	seen := map[int]bool{}
	for p.PID > 0 && !seen[p.PID] {
		if p.PID == pid {
			return p.Identity == identity
		}
		seen[p.PID] = true
		p = byPID[p.PPID]
	}
	return false
}

func recoveryNativeWrappers(root string, snapshot recoveryTerminalSnapshot) ([]recoveryNativeWrapper, error) {
	wrappers := []recoveryNativeWrapper{}
	for _, p := range snapshot.Processes {
		// Metadata selects possible wrappers BEFORE requiring command evidence.
		// Unknown names remain candidates. argv is still the matching authority;
		// unreadable system services and terminated rows are not native writers.
		if p.Zombie || (p.Name != "" && p.Name != "pair" && p.Name != "pair-wrap") {
			continue
		}
		command, ok := snapshot.Commands[p.PID]
		if !ok || p.Identity == "" {
			return nil, fmt.Errorf("%w: process %d (%s) command ownership is incomplete; retry", errNativeOwnerObservation, p.PID, p.Name)
		}
		args, err := splitObservedCommand(command)
		if err != nil {
			if strings.Contains(command, "pair-wrap") || strings.Contains(command, "pair wrap") {
				return nil, errors.New("wrapper command ownership is ambiguous")
			}
			continue
		}
		if len(args) > 0 && args[0] == "exec" {
			args = args[1:]
		}
		if len(args) == 0 {
			continue
		}
		switch filepath.Base(args[0]) {
		case "pair-wrap":
			args = args[1:]
		case "pair":
			if len(args) < 2 || args[1] != "wrap" {
				continue
			}
			args = args[2:]
		default:
			continue
		}
		paths := wrapperOwnerPaths(args)
		if len(paths) != 1 {
			return nil, errors.New("live wrapper has no exact storage owner")
		}
		owner, ok := artifactpath.SessionArtifactOwner(root, paths[0], AgentInventory())
		if !ok {
			return nil, errors.New("live wrapper storage owner is unavailable")
		}
		resolved, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
		if err != nil {
			return nil, err
		}
		agent := ""
		for _, name := range AgentInventory() {
			if resolved.ScrollbackRaw(name) == paths[0] {
				agent = name
				break
			}
		}
		if agent == "" {
			return nil, errors.New("live wrapper agent ownership is unavailable")
		}
		wrappers = append(wrappers, recoveryNativeWrapper{Process: p, Owner: owner, Agent: agent})
	}
	// Process table ordering is not identity.
	slices.SortFunc(wrappers, func(a, b recoveryNativeWrapper) int { return cmp.Compare(a.Process.PID, b.Process.PID) })
	return wrappers, nil
}

func recoverySourceWrapper(check RecoveryNativeOwnerCheck, snapshot recoveryTerminalSnapshot, wrapper recoveryNativeWrapper, ready readiness.ReadyRecord) bool {
	if wrapper.Owner.RepoScope != check.RepoScope || wrapper.Owner.Tag != check.Tag || wrapper.Agent != check.Agent || check.SourcePID <= 0 || check.SourceIdentity == "" {
		return false
	}
	var helper procutil.Process
	for _, p := range snapshot.Processes {
		if p.PID == check.SourcePID {
			helper = p
			break
		}
	}
	if helper.Identity != check.SourceIdentity {
		return false
	}
	if check.SourceNonce != "" && ready.Nonce != check.SourceNonce {
		return false
	}
	if check.SourceSession != "" && ready.Session != check.SourceSession {
		return false
	}
	if recoverySourceAncestor(snapshot.Processes, wrapper.Process, check.SourcePID, check.SourceIdentity) {
		return true
	}
	// Zellij servers daemonize; their wrappers are not descendants of the Pair
	// client. The creating session nonce and live agent generation bridge that
	// boundary without exempting other wrappers that share the same tag.
	if check.SourceSession == "" || check.SourceNonce == "" {
		return false
	}
	return ready.Session == check.SourceSession && ready.Nonce == check.SourceNonce
}

// A tag-owned file is insufficient: its exact agent generation must descend
// from this observed wrapper generation before its launch ordinal is authority.
func recoveryWrapperReady(ctx context.Context, snapshot recoveryTerminalSnapshot, wrapper recoveryNativeWrapper, read func(context.Context, string, int64) ([]byte, error)) (readiness.ReadyRecord, error) {
	raw, err := read(ctx, AgentReadyPath(wrapper.Owner.Directory(), wrapper.Owner.Tag, wrapper.Agent), maxAgentReadyBytes)
	if err != nil {
		return readiness.ReadyRecord{}, fmt.Errorf("live wrapper launch association unavailable: %w", err)
	}
	ready, err := readiness.Decode(string(raw))
	if err != nil {
		return readiness.ReadyRecord{}, fmt.Errorf("live wrapper launch association invalid: %w", err)
	}
	if ready.Tag != wrapper.Owner.Tag || ready.Agent != wrapper.Agent || ready.LaunchOrdinal == 0 {
		return readiness.ReadyRecord{}, errors.New("live wrapper launch association is unresolved")
	}
	if ready.AgentIdentity == "" {
		return ready, &LegacyNativeOwnerError{Owner: wrapper.Owner, Agent: wrapper.Agent, PID: ready.PID, Reason: "legacy readiness has no child birth"}
	}
	for _, agent := range snapshot.Processes {
		if !agent.Zombie && agent.PID == ready.PID && agent.Identity == ready.AgentIdentity && agent.PID != wrapper.Process.PID && recoverySourceAncestor(snapshot.Processes, agent, wrapper.Process.PID, wrapper.Process.Identity) {
			return ready, nil
		}
	}
	return readiness.ReadyRecord{}, errors.New("live wrapper readiness does not prove its exact process generation")
}

type nativeOwnerCheckIO struct {
	store    NativeAdmissionStore
	observe  func(context.Context, string) (recoveryTerminalSnapshot, error)
	read     func(context.Context, string, int64) ([]byte, error)
	discover func(context.Context, string) ([]artifactpath.StorageOwner, error)
}

func readRecoveryNativeID(ctx context.Context, check RecoveryNativeOwnerCheck, owner artifactpath.StorageOwner, ordinal uint64, read func(context.Context, string, int64) ([]byte, error)) (string, error) {
	paths, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	if err != nil {
		return "", err
	}
	raw, err := read(ctx, paths.Ledger(), -1)
	if err != nil {
		return "", fmt.Errorf("live %s wrapper native ownership unavailable: %w", check.Agent, err)
	}
	// Every observation still reads and validates the physical file. Only
	// identical bytes reuse their pure projection during this one admission.
	key := nativeOwnerLedgerKey{Owner: owner, Agent: check.Agent, Ordinal: ordinal}
	digest := sha256.Sum256(raw)
	if check.ledgers != nil {
		check.ledgers.mu.Lock()
		prior, ok := check.ledgers.values[key]
		check.ledgers.mu.Unlock()
		if ok && prior.Digest == digest {
			return prior.ID, prior.Err
		}
	}
	id, err := projectRecoveryNativeID(raw, owner, check.Agent, ordinal)
	if check.ledgers != nil {
		check.ledgers.mu.Lock()
		check.ledgers.values[key] = nativeOwnerLedgerResult{Digest: digest, ID: id, Err: err}
		check.ledgers.mu.Unlock()
	}
	return id, err
}

type nativeOwnerLedgerMemo struct {
	mu     sync.Mutex
	values map[nativeOwnerLedgerKey]nativeOwnerLedgerResult
}

type nativeOwnerLedgerKey struct {
	Owner   artifactpath.StorageOwner
	Agent   string
	Ordinal uint64
}
type nativeOwnerLedgerResult struct {
	Digest [32]byte
	ID     string
	Err    error
}

func projectRecoveryNativeID(raw []byte, owner artifactpath.StorageOwner, agent string, ordinal uint64) (string, error) {
	parsed := sessionledger.ParseLedger(raw)
	if len(parsed.MalformedOrdinals) != 0 {
		return "", errors.New("live wrapper ledger has malformed native ownership")
	}
	// Only the readiness-proved physical generation may supply native identity.
	generation := make([]sessionledger.Record, 0)
	for _, record := range parsed.Records {
		if (record.Kind == sessionledger.RecordLaunch && record.Ordinal == ordinal) || (record.Kind == sessionledger.RecordBinding && record.LaunchOrdinal == ordinal) {
			generation = append(generation, record)
		}
	}
	current, found := sessionledger.CurrentLaunch(generation, sessionledger.Owner{ScopeKey: owner.RepoScope, Tag: owner.Tag, Agent: agent})
	if !found || current.Conflict {
		return "", errors.New("live wrapper native ownership is unresolved or conflicting")
	}
	target := sessioninventory.ResumeTargetForLaunch(current)
	native := target.NativeID
	// A Pair-chosen UUID is already reserved by this running wrapper even
	// before the native filename handshake confirms its first authored turn.
	if native == "" {
		native = target.RequestedNativeID
	}
	if native == "" {
		return "", fmt.Errorf("%w (%s:%s)", errNativeOwnerUnbound, owner.RepoScope, owner.Tag)
	}
	return native, nil
}
