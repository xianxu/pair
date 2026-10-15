package launcher

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// LegacyNativeOwnerError preserves uncertain legacy ownership and its exact
// migration target. It never grants authority to stop the named process.
type LegacyNativeOwnerError struct {
	Owner  artifactpath.StorageOwner
	Agent  string
	PID    int
	Reason string
}

func (e *LegacyNativeOwnerError) Error() string {
	return fmt.Sprintf("legacy %s owner %s:%s (child pid %d) lacks proven agent_identity: %s; relaunch its slot to migrate", e.Agent, e.Owner.RepoScope, e.Owner.Tag, e.PID, e.Reason)
}

type nativeHeldFiles func(context.Context, string) ([]string, error)

type recoveryNativeProof struct {
	Ready    readiness.ReadyRecord
	NativeID string
	Child    *procutil.Process
	// A legacy exemption is tied to this exact file-holding process too.
	Holders []procutil.Process
	HeldIDs []string
}

func recoveryNativeProofForWrapper(ctx context.Context, check RecoveryNativeOwnerCheck, snapshot recoveryTerminalSnapshot, wrapper recoveryNativeWrapper, read func(context.Context, string, int64) ([]byte, error)) (recoveryNativeProof, error) {
	ready, err := recoveryWrapperReady(ctx, snapshot, wrapper, read)
	var legacy *LegacyNativeOwnerError
	if err != nil && !errors.As(err, &legacy) {
		return recoveryNativeProof{}, err
	}
	id, err := readRecoveryNativeID(ctx, check, wrapper.Owner, ready.LaunchOrdinal, read)
	unbound := errors.Is(err, errNativeOwnerUnbound)
	if err != nil && !unbound {
		if legacy != nil && ctx.Err() == nil {
			legacy.Reason = err.Error()
			return recoveryNativeProof{}, legacy
		}
		return recoveryNativeProof{}, err
	}
	proof := recoveryNativeProof{Ready: ready, NativeID: id}
	if legacy == nil && !unbound {
		return proof, nil
	}
	// A legacy record does not carry a birth token. Its PID is usable only
	// while it is a direct live child of this exact owner-naming wrapper.
	var child procutil.Process
	for _, p := range snapshot.Processes {
		if p.PID == ready.PID && p.PPID == wrapper.Process.PID && p.Identity != "" && !p.Zombie {
			child = p
			break
		}
	}
	if child.PID == 0 {
		return recoveryNativeProof{}, nativeOwnerProofError(legacy, wrapper, ready, "readiness does not name a direct live child of the observed wrapper")
	}
	proof.Child = &child
	if id != "" && id != check.NativeID {
		// Approved negative collision exclusion, not a proven source exemption.
		return proof, nil
	}
	ids, holders, err := checkLegacyHeldNative(ctx, check, snapshot, child)
	if err == nil && id != "" && !slices.Contains(ids, id) {
		err = errors.New("no held writable transcript proves the owner ledger ID")
	}
	if err != nil {
		if ctx.Err() != nil {
			return recoveryNativeProof{}, ctx.Err()
		}
		return recoveryNativeProof{}, nativeOwnerProofError(legacy, wrapper, ready, err.Error())
	}
	proof.Ready.AgentIdentity = child.Identity
	proof.Holders, proof.HeldIDs = holders, ids
	return proof, nil
}

func nativeOwnerProofError(legacy *LegacyNativeOwnerError, wrapper recoveryNativeWrapper, ready readiness.ReadyRecord, reason string) error {
	if legacy != nil {
		legacy.Reason = reason
		return legacy
	}
	return fmt.Errorf("%s native owner %s:%s (child pid %d) has no proven native binding: %s; relaunch its slot to migrate", wrapper.Agent, wrapper.Owner.RepoScope, wrapper.Owner.Tag, ready.PID, reason)
}

func checkLegacyHeldNative(ctx context.Context, check RecoveryNativeOwnerCheck, snapshot recoveryTerminalSnapshot, child procutil.Process) ([]string, []procutil.Process, error) {
	holders := []procutil.Process{child}
	// The Codex node launcher remains the direct PTY child. Only its immediate
	// native engine qualifies; tool descendants opening transcripts do not.
	if check.Agent == "codex" && child.Name == "node" {
		for _, p := range snapshot.Processes {
			if p.PPID != child.PID || p.Zombie || p.Identity == "" || p.Name != "codex" {
				continue
			}
			args := snapshot.Args[p.PID]
			if len(args) == 0 {
				args, _ = splitObservedCommand(snapshot.Commands[p.PID])
			}
			if len(args) > 0 && filepath.Base(args[0]) == "codex" {
				holders = append(holders, p)
			}
		}
	}
	read := check.heldFiles
	if read == nil {
		read = procutil.LsofWritableNames
	}
	roots := sessioninventory.NewOSRuntime(os.Getenv("HOME"), check.DataDir).NativeRoots(sessioninventory.Agent(check.Agent))
	ids := []string{}
	for _, holder := range holders {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		paths, err := read(ctx, strconv.Itoa(holder.PID))
		if err != nil {
			return nil, nil, err // Partial inspection cannot exclude another writer.
		}
		for _, path := range paths {
			for _, root := range roots {
				rel, err := filepath.Rel(root.Path, path)
				if err != nil || !filepath.IsAbs(path) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					continue
				}
				a := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(rel)}
				if id := sessioninventory.NativeIDForArtifact(sessioninventory.Agent(check.Agent), a); id != "" {
					ids = append(ids, id)
				} else if filepath.Ext(path) == ".jsonl" {
					return nil, nil, errors.New("held native transcript identity is unrecognized")
				}
			}
		}
	}
	if len(ids) == 0 {
		return nil, nil, errors.New("no held writable native transcript proves this legacy owner's IDs")
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	slices.SortFunc(holders, func(a, b procutil.Process) int { return cmp.Compare(a.PID, b.PID) })
	return ids, holders, nil
}
