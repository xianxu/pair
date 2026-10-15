package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// A durable claim supplies a missing ready association only for its exact
// writer. Running proof includes the child: Fresh exec may retain wrapper birth.
func (v NativeAdmissionOwners) coversDifferentWrapper(agent, id string, wrapper procutil.Process, snapshot recoveryTerminalSnapshot) bool {
	if v.active == nil || !v.active.Load() || v.process == nil {
		return false
	}
	for _, e := range v.entries {
		if e.Request.Agent != agent || e.Request.NativeID == id || e.Wrapper.PID != wrapper.PID {
			continue
		}
		current, _, err := v.process(wrapper.PID)
		if err != nil || current != e.Wrapper {
			continue
		}
		if e.Phase == NativeAdmissionSpawning {
			return true
		}
		if e.Phase != NativeAdmissionRunning {
			continue
		}
		child, parent, err := v.process(e.Child.PID)
		if err != nil || child != e.Child || parent != wrapper.PID {
			continue
		}
		for _, row := range snapshot.Processes {
			if row.PID == child.PID && row.PPID == wrapper.PID && row.Identity != "" {
				return true
			}
		}
	}
	return false
}

type retainedNativeOwner struct {
	Owner    artifactpath.StorageOwner
	Ready    readiness.ReadyRecord
	Process  procutil.Process
	NativeID string
}

func observeLegacyNativeOrphans(ctx context.Context, check RecoveryNativeOwnerCheck, snapshot recoveryTerminalSnapshot, wrappers []recoveryNativeWrapper, read func(context.Context, string, int64) ([]byte, error), discover func(context.Context, string) ([]artifactpath.StorageOwner, error)) ([]retainedNativeOwner, error) {
	owners, err := discover(ctx, check.DataDir)
	if err != nil {
		return nil, err
	}
	rows, err := recoveryRows(snapshot)
	if err != nil {
		return nil, err
	}
	var facts []retainedNativeOwner
	for _, owner := range owners {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		expected, err := artifactpath.NewStorageOwner(check.DataDir, owner.RepoScope, owner.Tag)
		if err != nil || owner != expected {
			return nil, errors.New("retained native owner namespace invalid")
		}
		raw, err := read(ctx, AgentReadyPath(owner.Directory(), owner.Tag, check.Agent), maxAgentReadyBytes)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		ready, err := readiness.Decode(string(raw))
		if err != nil || ready.Tag != owner.Tag || ready.Agent != check.Agent {
			return nil, errors.New("retained native readiness invalid")
		}
		child, found := rows[ready.PID]
		if !found || child.Zombie {
			continue
		}
		if child.Identity == "" || ready.AgentIdentity == "" {
			return nil, errors.New("retained native child generation unknown")
		}
		if child.Identity != ready.AgentIdentity {
			continue
		} // Positive PID reuse is not the old writer.
		attached := false
		for _, wrapper := range wrappers {
			if wrapper.Owner == owner && wrapper.Agent == check.Agent && recoverySourceAncestor(snapshot.Processes, child, wrapper.Process.PID, wrapper.Process.Identity) {
				attached = true
				break
			}
		}
		if attached {
			continue
		} // The wrapper path handles readiness, source exclusion and admission proof.
		if ready.LaunchOrdinal == 0 {
			return nil, errors.New("retained native launch association unresolved")
		}
		paths, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
		if err != nil {
			return nil, err
		}
		raw, err = read(ctx, paths.Ledger(), 8<<20)
		if err != nil {
			return nil, err
		}
		parsed := sessionledger.ParseLedger(raw)
		if len(parsed.MalformedOrdinals) != 0 {
			return nil, errors.New("retained native ledger malformed")
		}
		var generation []sessionledger.Record
		for _, record := range parsed.Records {
			if record.Kind == sessionledger.RecordLaunch && record.Ordinal == ready.LaunchOrdinal || record.Kind == sessionledger.RecordBinding && record.LaunchOrdinal == ready.LaunchOrdinal {
				generation = append(generation, record)
			}
		}
		current, found := sessionledger.CurrentLaunch(generation, sessionledger.Owner{ScopeKey: owner.RepoScope, Tag: owner.Tag, Agent: check.Agent})
		if !found || current.Conflict {
			return nil, errors.New("retained native ownership unresolved")
		}
		target := sessioninventory.ResumeTargetForLaunch(current)
		id := target.NativeID
		if id == "" {
			id = target.RequestedNativeID
		}
		if id == "" {
			return nil, errors.New("retained native writer has no resolved root")
		}
		if id == check.NativeID {
			return nil, fmt.Errorf("native conversation %s still has its retained child (%s:%s)", id, owner.RepoScope, owner.Tag)
		}
		facts = append(facts, retainedNativeOwner{Owner: owner, Ready: ready, Process: child, NativeID: id})
	}
	slices.SortFunc(facts, func(a, b retainedNativeOwner) int { return strings.Compare(a.Owner.Key(), b.Owner.Key()) })
	return facts, nil
}

// The scan visits only namespace entries: flat root, repos, and scope roots.
// Existing artifact codecs discover owners from exact ledger/history anchors;
// unrelated native executable names never become speculative writer blockers.
func discoverLegacyNativeOwners(ctx context.Context, root string) ([]artifactpath.StorageOwner, error) {
	const limit = 16384
	budget := limit
	names := func(dir string) ([]string, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		st, err := os.Lstat(dir)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("unsafe retained native namespace")
		}
		f, err := os.Open(dir)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		var result []string
		for {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			batch, err := f.Readdirnames(min(256, budget+1))
			budget -= len(batch)
			if budget < 0 {
				return nil, errors.New("retained native owner inventory exceeds bound")
			}
			result = append(result, batch...)
			if errors.Is(err, io.EOF) {
				return result, nil
			}
			if err != nil {
				return nil, err
			}
		}
	}
	flat, err := names(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	owners, err := artifactpath.DiscoverStorageOwners(root, "", flat)
	if err != nil {
		return nil, err
	}
	scopes, err := names(filepath.Join(root, "repos"))
	if errors.Is(err, os.ErrNotExist) {
		return owners, nil
	}
	if err != nil {
		return nil, err
	}
	for _, scope := range scopes {
		if _, err := artifactpath.NewStorageOwner(root, scope, "validation"); err != nil {
			return nil, err
		}
		entries, err := names(filepath.Join(root, "repos", scope))
		if err != nil {
			return nil, err
		}
		found, err := artifactpath.DiscoverStorageOwners(root, scope, entries)
		if err != nil {
			return nil, err
		}
		owners = append(owners, found...)
	}
	slices.SortFunc(owners, func(a, b artifactpath.StorageOwner) int { return strings.Compare(a.Key(), b.Key()) })
	return owners, nil
}
