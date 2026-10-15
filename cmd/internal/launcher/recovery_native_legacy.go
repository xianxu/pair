package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	Holder *procutil.Process
}

func recoveryNativeProofForWrapper(ctx context.Context, check RecoveryNativeOwnerCheck, snapshot recoveryTerminalSnapshot, wrapper recoveryNativeWrapper, read func(context.Context, string, int64) ([]byte, error)) (recoveryNativeProof, error) {
	ready, err := recoveryWrapperReady(ctx, snapshot, wrapper, read)
	var legacy *LegacyNativeOwnerError
	if err != nil && !errors.As(err, &legacy) {
		return recoveryNativeProof{}, err
	}
	id, err := readRecoveryNativeID(ctx, wrapper.Owner, wrapper.Agent, ready.LaunchOrdinal, read)
	if err != nil {
		if legacy != nil && ctx.Err() == nil {
			legacy.Reason = err.Error()
			return recoveryNativeProof{}, legacy
		}
		return recoveryNativeProof{}, err
	}
	proof := recoveryNativeProof{Ready: ready, NativeID: id}
	if legacy == nil {
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
		legacy.Reason = "readiness does not name a direct live child of the observed wrapper"
		return recoveryNativeProof{}, legacy
	}
	proof.Child = &child
	if id != check.NativeID {
		// Approved negative collision exclusion, not a proven source exemption.
		return proof, nil
	}
	holder, err := checkLegacyHeldNative(ctx, check, snapshot, child, id)
	if err != nil {
		if ctx.Err() != nil {
			return recoveryNativeProof{}, ctx.Err()
		}
		legacy.Reason = err.Error()
		return recoveryNativeProof{}, legacy
	}
	proof.Ready.AgentIdentity = child.Identity
	proof.Holder = &holder
	return proof, nil
}

func checkLegacyHeldNative(ctx context.Context, check RecoveryNativeOwnerCheck, snapshot recoveryTerminalSnapshot, child procutil.Process, id string) (procutil.Process, error) {
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
	for _, holder := range holders {
		if err := ctx.Err(); err != nil {
			return procutil.Process{}, err
		}
		paths, err := read(ctx, strconv.Itoa(holder.PID))
		if err != nil {
			continue // One positive held owner transcript is sufficient, never absence.
		}
		for _, path := range paths {
			for _, root := range roots {
				rel, err := filepath.Rel(root.Path, path)
				if err != nil || !filepath.IsAbs(path) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					continue
				}
				a := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(rel)}
				if sessioninventory.NativeIDForArtifact(sessioninventory.Agent(check.Agent), a) == id {
					return holder, nil
				}
			}
		}
	}
	return procutil.Process{}, errors.New("no held writable transcript proves the owner ledger ID")
}
