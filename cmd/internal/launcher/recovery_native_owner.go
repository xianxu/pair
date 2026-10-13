package launcher

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
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
}

type recoveryNativeWrapper struct {
	Process procutil.Process
	Owner   artifactpath.StorageOwner
	Agent   string
}

func CheckRecoveryNativeOwner(ctx context.Context, check RecoveryNativeOwnerCheck) error {
	return checkRecoveryNativeOwners(ctx, check, newRecoveryTerminalOS().Observe, osfs.ReadRegularMetadata)
}

func checkRecoveryNativeOwners(ctx context.Context, check RecoveryNativeOwnerCheck, observe func(context.Context, string) (recoveryTerminalSnapshot, error), read func(context.Context, string, int64) ([]byte, error)) error {
	if ctx == nil {
		return errors.New("native ownership check requires context")
	}
	if check.NativeID == "" {
		return ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, zjTimeout)
	defer cancel()
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
	excluded := 0
	proofs := map[int]readiness.ReadyRecord{}
	for _, wrapper := range wrappers {
		if wrapper.Agent != check.Agent {
			continue
		}
		ready, err := recoveryWrapperReady(ctx, before, wrapper, read)
		if err != nil {
			return err
		}
		proofs[wrapper.Process.PID] = ready
		paths, err := artifactpath.ResolveScoped(wrapper.Owner.Directory(), wrapper.Owner.Tag)
		if err != nil {
			return err
		}
		raw, err := read(ctx, paths.Ledger(), 8<<20)
		if err != nil {
			return fmt.Errorf("live %s wrapper native ownership unavailable: %w", wrapper.Agent, err)
		}
		parsed := sessionledger.ParseLedger(raw)
		if len(parsed.MalformedOrdinals) != 0 {
			return errors.New("live wrapper ledger has malformed native ownership")
		}
		// Only the readiness-proved physical generation may supply native identity.
		generation := make([]sessionledger.Record, 0)
		for _, record := range parsed.Records {
			if (record.Kind == sessionledger.RecordLaunch && record.Ordinal == ready.LaunchOrdinal) || (record.Kind == sessionledger.RecordBinding && record.LaunchOrdinal == ready.LaunchOrdinal) {
				generation = append(generation, record)
			}
		}
		current, found := sessionledger.CurrentLaunch(generation, sessionledger.Owner{ScopeKey: wrapper.Owner.RepoScope, Tag: wrapper.Owner.Tag, Agent: wrapper.Agent})
		if !found || current.Conflict {
			return errors.New("live wrapper native ownership is unresolved or conflicting")
		}
		target := sessioninventory.ResumeTargetForLaunch(current)
		native := target.NativeID
		// A Pair-chosen UUID is already reserved by this running wrapper even
		// before the native filename handshake confirms its first authored turn.
		if native == "" {
			native = target.RequestedNativeID
		}
		if native == "" {
			return errors.New("live wrapper native ownership is unresolved")
		}
		if recoverySourceWrapper(check, before, wrapper, ready) {
			excluded++
			if excluded > 1 {
				return errors.New("multiple source wrappers make native ownership uncertain")
			}
			continue
		}
		if native == check.NativeID {
			return fmt.Errorf("native conversation %s already has a live wrapper (%s:%s)", native, wrapper.Owner.RepoScope, wrapper.Owner.Tag)
		}
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
		return errors.New("native owner process generation changed; retry")
	}
	for _, wrapper := range next {
		if wrapper.Agent != check.Agent {
			continue
		}
		ready, err := recoveryWrapperReady(ctx, after, wrapper, read)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(proofs[wrapper.Process.PID], ready) {
			return errors.New("native owner readiness generation changed; retry")
		}
	}
	if excluded > 0 {
		confirmed := 0
		for _, wrapper := range next {
			if recoverySourceWrapper(check, after, wrapper, proofs[wrapper.Process.PID]) {
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
		command, ok := snapshot.Commands[p.PID]
		if !ok || p.Identity == "" {
			return nil, errors.New("process command ownership is incomplete; retry")
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
	if ready.Tag != wrapper.Owner.Tag || ready.Agent != wrapper.Agent || ready.LaunchOrdinal == 0 || ready.AgentIdentity == "" {
		return readiness.ReadyRecord{}, errors.New("live wrapper launch association is unresolved")
	}
	for _, agent := range snapshot.Processes {
		if agent.PID == ready.PID && agent.Identity == ready.AgentIdentity && agent.PID != wrapper.Process.PID && recoverySourceAncestor(snapshot.Processes, agent, wrapper.Process.PID, wrapper.Process.Identity) {
			return ready, nil
		}
	}
	return readiness.ReadyRecord{}, errors.New("live wrapper readiness does not prove its exact process generation")
}
